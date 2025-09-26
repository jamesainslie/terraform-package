// MIT License
//
// Copyright (c) 2025 Terraform Package Provider Contributors
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// Package brew implements the Homebrew package manager adapter.
//
// IMPORTANT NOTE ABOUT ERROR MESSAGES:
// This adapter implements Homebrew's dual-type detection logic since packages can be
// either formulae (command-line tools) or casks (GUI applications). During normal
// operation, you will see expected stderr messages like:
//   - "Error: Cask 'jq' is unavailable: No Cask with this name exists."
//   - "Error: No available formula with the name 'firefox'"
//
// These messages are NORMAL and indicate the detection logic is working correctly.
// The adapter tries both types and uses whichever succeeds.
package brew

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/jamesainslie/terraform-package/internal/adapters"
	"github.com/jamesainslie/terraform-package/internal/executor"
)

// BrewAdapter implements the PackageManager interface for Homebrew.
type BrewAdapter struct {
	executor executor.Executor
	brewPath string
}

// NewBrewAdapter creates a new Homebrew adapter.
func NewBrewAdapter(exec executor.Executor, brewPath string) *BrewAdapter {
	if brewPath == "" {
		brewPath = "brew"
	}
	return &BrewAdapter{
		executor: exec,
		brewPath: brewPath,
	}
}

// GetManagerName returns the name of this package manager.
func (b *BrewAdapter) GetManagerName() string {
	return "brew"
}

// IsAvailable checks if Homebrew is available on the system.
func (b *BrewAdapter) IsAvailable(ctx context.Context) bool {
	if runtime.GOOS != "darwin" {
		return false
	}

	// Check if brew command is available
	_, err := exec.LookPath(b.brewPath)
	if err != nil {
		return false
	}

	// Test brew command execution
	result, err := b.executor.Run(ctx, b.brewPath, []string{"--version"}, executor.ExecOpts{
		Timeout: 10 * time.Second,
	})

	return err == nil && result.ExitCode == 0
}

// brewInfo represents the JSON structure returned by 'brew info --json'.
type brewInfo struct {
	Name       string             `json:"name"`
	FullName   string             `json:"full_name"`
	Versions   versions           `json:"versions"`
	Installed  []installedVersion `json:"installed"`
	LinkedKeg  string             `json:"linked_keg,omitempty"`
	Pinned     bool               `json:"pinned"`
	Outdated   bool               `json:"outdated"`
	Deprecated bool               `json:"deprecated"`
	Disabled   bool               `json:"disabled"`
	Desc       string             `json:"desc"`
	Homepage   string             `json:"homepage"`
	Tap        string             `json:"tap"`
	Cask       string             `json:"cask,omitempty"`
}

type versions struct {
	Stable string `json:"stable"`
	Head   string `json:"head"`
}

type installedVersion struct {
	Version               string       `json:"version"`
	UsedOptions           []string     `json:"used_options"`
	BuiltAsBottle         bool         `json:"built_as_bottle"`
	PouredFromBottle      bool         `json:"poured_from_bottle"`
	RuntimeDependencies   []dependency `json:"runtime_dependencies"`
	InstalledAsDependency bool         `json:"installed_as_dependency"`
	InstalledOnRequest    bool         `json:"installed_on_request"`
}

type dependency struct {
	FullName string `json:"full_name"`
	Version  string `json:"version"`
}

// DetectInstalled checks if a package is installed and returns its information.
// This method implements Homebrew's dual-type detection logic since packages can be
// either formulae (command-line tools) or casks (GUI applications).
func (b *BrewAdapter) DetectInstalled(ctx context.Context, name string) (*adapters.PackageInfo, error) {
	// First try as formula, then as cask
	// Note: This will produce expected "stderr" messages when the wrong type is tried
	// (e.g., "Error: Cask 'jq' is unavailable" when jq is actually a formula)
	// These messages are normal and indicate the detection logic is working correctly
	info, err := b.getPackageInfo(ctx, name, false)
	if err != nil {
		// Try as cask - this may also produce expected error messages
		info, err = b.getPackageInfo(ctx, name, true)
		if err != nil {
			// Both formula and cask failed, package not found
			return &adapters.PackageInfo{
				Name:      name,
				Installed: false,
			}, fmt.Errorf("package %s not found as formula or cask", name)
		}
	}

	return info, nil
}

// getPackageInfo retrieves package information from Homebrew.
func (b *BrewAdapter) getPackageInfo(ctx context.Context, name string, isCask bool) (*adapters.PackageInfo, error) {
	args := []string{"info"}
	if isCask {
		args = append(args, "--json=v2", "--cask", name)
	} else {
		args = append(args, "--json", name)
	}

	result, err := b.executor.Run(ctx, b.brewPath, args, executor.ExecOpts{
		Timeout: 30 * time.Second,
	})

	if err != nil || result.ExitCode != 0 {
		return nil, fmt.Errorf("failed to get package info for %s: %w", name, err)
	}

	var info brewInfo

	if isCask {
		// Parse v2 JSON format for casks
		var v2Response struct {
			Casks []map[string]interface{} `json:"casks"`
		}
		if err := json.Unmarshal([]byte(result.Stdout), &v2Response); err != nil {
			return nil, fmt.Errorf("failed to parse brew info v2 JSON: %w", err)
		}

		if len(v2Response.Casks) == 0 {
			return nil, fmt.Errorf("no cask info found for %s", name)
		}

		cask := v2Response.Casks[0]
		info = brewInfo{
			Name:     getStringValue(cask, "token"),
			FullName: getStringValue(cask, "full_token"),
			Tap:      getStringValue(cask, "tap"),
			Desc:     getStringValue(cask, "desc"),
			Homepage: getStringValue(cask, "homepage"),
		}

		// Handle version for casks
		if version, ok := cask["version"].(string); ok {
			info.Versions = versions{Stable: version}
		}

		// Handle installed status for casks
		if installed := cask["installed"]; installed != nil {
			info.Installed = []installedVersion{{Version: info.Versions.Stable}}
		}
	} else {
		// Parse v1 JSON format for formulae
		var infos []brewInfo
		if err := json.Unmarshal([]byte(result.Stdout), &infos); err != nil {
			return nil, fmt.Errorf("failed to parse brew info JSON: %w", err)
		}

		if len(infos) == 0 {
			return nil, fmt.Errorf("no package info found for %s", name)
		}

		info = infos[0]
	}

	packageInfo := &adapters.PackageInfo{
		Name:      info.Name,
		Installed: len(info.Installed) > 0,
		Pinned:    info.Pinned,
	}

	// Set package type based on detection method
	if isCask {
		packageInfo.Type = adapters.PackageTypeCask
	} else {
		packageInfo.Type = adapters.PackageTypeFormula
	}

	// Set current version if installed
	if len(info.Installed) > 0 {
		packageInfo.Version = info.Installed[0].Version
	}

	// Set available versions
	if info.Versions.Stable != "" {
		packageInfo.AvailableVersions = []string{info.Versions.Stable}
	}
	if info.Versions.Head != "" && info.Versions.Head != info.Versions.Stable {
		packageInfo.AvailableVersions = append(packageInfo.AvailableVersions, info.Versions.Head)
	}

	// Set repository (tap)
	if info.Tap != "" {
		packageInfo.Repository = info.Tap
	}

	return packageInfo, nil
}

// Install installs a package with optional version specification.
func (b *BrewAdapter) Install(ctx context.Context, name, version string) error {
	// Use auto-detection for backward compatibility
	return b.InstallWithType(ctx, name, version, adapters.PackageTypeAuto)
}

// InstallWithType installs a package with explicit type and optional version specification.
// Implements idempotency by checking if the package is already installed before attempting installation.
func (b *BrewAdapter) InstallWithType(ctx context.Context, name, version string, packageType adapters.PackageType) error {
	// DEBUG: Log installation request
	tflog.Debug(ctx, "BrewAdapter.InstallWithType starting",
		map[string]interface{}{
			"package_name": name,
			"version":      version,
			"package_type": string(packageType),
			"brew_path":    b.brewPath,
		})

	var isCask bool
	var err error

	switch packageType {
	case adapters.PackageTypeCask:
		isCask = true
		tflog.Debug(ctx, "Installing as Homebrew cask", map[string]interface{}{
			"package_name": name,
			"is_cask":      true,
		})
	case adapters.PackageTypeFormula:
		isCask = false
		tflog.Debug(ctx, "Installing as Homebrew formula", map[string]interface{}{
			"package_name": name,
			"is_cask":      false,
		})
	case adapters.PackageTypeAuto:
		// Auto-detect as before
		tflog.Debug(ctx, "Auto-detecting package type", map[string]interface{}{
			"package_name": name,
		})
		isCask, err = b.isCask(ctx, name)
		if err != nil {
			// If we can't determine, try as formula first
			isCask = false
			tflog.Debug(ctx, "Auto-detection failed, defaulting to formula", map[string]interface{}{
				"package_name": name,
				"error":        err.Error(),
				"is_cask":      false,
			})
		} else {
			tflog.Debug(ctx, "Auto-detection completed", map[string]interface{}{
				"package_name": name,
				"is_cask":      isCask,
			})
		}
	default:
		tflog.Debug(ctx, "Unsupported package type", map[string]interface{}{
			"package_name": name,
			"package_type": string(packageType),
		})
		return fmt.Errorf("unsupported package type: %s", packageType)
	}

	// IDEMPOTENCY CHECK: Check if package is already installed to avoid unnecessary operations
	// This is especially important for casks, which fail with exit code 1 if already installed
	tflog.Debug(ctx, "Performing idempotency check", map[string]interface{}{
		"package_name": name,
		"is_cask":      isCask,
	})

	info, err := b.DetectInstalled(ctx, name)
	if err == nil && info.Installed {
		tflog.Debug(ctx, "Package already installed, checking version compatibility", map[string]interface{}{
			"package_name":      name,
			"installed_version": info.Version,
			"requested_version": version,
		})

		// Package is already installed - check version compatibility
		if version == "" || version == info.Version {
			// Already installed with correct version, no action needed
			tflog.Debug(ctx, "Package already installed with correct version, skipping installation", map[string]interface{}{
				"package_name":      name,
				"installed_version": info.Version,
				"requested_version": version,
			})
			return nil
		}
		// Different version requested - continue with installation (may upgrade/downgrade)
		tflog.Debug(ctx, "Different version requested, proceeding with installation", map[string]interface{}{
			"package_name":      name,
			"installed_version": info.Version,
			"requested_version": version,
		})
	} else if err != nil {
		tflog.Debug(ctx, "Could not detect installation status, proceeding with installation", map[string]interface{}{
			"package_name": name,
			"error":        err.Error(),
		})
	} else {
		tflog.Debug(ctx, "Package not installed, proceeding with installation", map[string]interface{}{
			"package_name": name,
		})
	}

	args := []string{"install"}
	if isCask {
		args = append(args, "--cask")
	}

	// For Homebrew, version specification is complex
	// Most packages don't support specific versions directly
	packageName := name
	if version != "" && !isCask {
		// Try versioned formula naming convention
		packageName = fmt.Sprintf("%s@%s", name, version)
		tflog.Debug(ctx, "Using versioned formula naming", map[string]interface{}{
			"original_name": name,
			"package_name":  packageName,
			"version":       version,
		})
	}

	args = append(args, packageName)

	// DEBUG: Log the final command being executed
	tflog.Debug(ctx, "Executing brew install command", map[string]interface{}{
		"command":      b.brewPath,
		"args":         args,
		"package_name": packageName,
		"is_cask":      isCask,
		"timeout":      "300s",
	})

	result, err := b.executor.Run(ctx, b.brewPath, args, executor.ExecOpts{
		Timeout: 300 * time.Second, // 5 minutes for package installation
	})

	// DEBUG: Log execution results
	tflog.Debug(ctx, "Brew install command completed", map[string]interface{}{
		"package_name": packageName,
		"exit_code":    result.ExitCode,
		"has_error":    err != nil,
		"stdout_len":   len(result.Stdout),
		"stderr_len":   len(result.Stderr),
	})

	if err != nil || result.ExitCode != 0 {
		// Handle cask-specific "already installed" error gracefully
		if isCask && result.ExitCode == 1 &&
			(strings.Contains(result.Stderr, "already an App at") ||
				strings.Contains(result.Stderr, "already installed")) {
			// This is actually success - the cask is already installed
			tflog.Debug(ctx, "Cask already installed error detected, treating as success", map[string]interface{}{
				"package_name": packageName,
				"stderr":       result.Stderr,
			})
			return nil
		}

		tflog.Debug(ctx, "Installation failed", map[string]interface{}{
			"package_name": packageName,
			"exit_code":    result.ExitCode,
			"error":        err,
			"stderr":       result.Stderr,
		})

		return fmt.Errorf("failed to install %s: exit code %d, error: %w, stderr: %s",
			packageName, result.ExitCode, err, result.Stderr)
	}

	tflog.Debug(ctx, "BrewAdapter.InstallWithType completed successfully", map[string]interface{}{
		"package_name": packageName,
		"is_cask":      isCask,
	})

	return nil
}

// Remove uninstalls a package.
func (b *BrewAdapter) Remove(ctx context.Context, name string) error {
	// Use auto-detection for backward compatibility
	return b.RemoveWithType(ctx, name, adapters.PackageTypeAuto)
}

// RemoveWithType uninstalls a package with explicit type.
func (b *BrewAdapter) RemoveWithType(ctx context.Context, name string, packageType adapters.PackageType) error {
	var isCask bool
	var err error

	switch packageType {
	case adapters.PackageTypeCask:
		isCask = true
	case adapters.PackageTypeFormula:
		isCask = false
	case adapters.PackageTypeAuto:
		// Auto-detect as before
		isCask, err = b.isCask(ctx, name)
		if err != nil {
			// If we can't determine, try as formula first
			isCask = false
		}
	default:
		return fmt.Errorf("unsupported package type: %s", packageType)
	}

	args := []string{"uninstall"}
	if isCask {
		args = append(args, "--cask")
	}
	args = append(args, name)

	result, err := b.executor.Run(ctx, b.brewPath, args, executor.ExecOpts{
		Timeout: 120 * time.Second, // 2 minutes for uninstall
	})

	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to uninstall %s: exit code %d, error: %w, stderr: %s",
			name, result.ExitCode, err, result.Stderr)
	}

	return nil
}

// Pin pins or unpins a package at its current version.
func (b *BrewAdapter) Pin(ctx context.Context, name string, pin bool) error {
	// Casks don't support pinning
	isCask, err := b.isCask(ctx, name)
	if err == nil && isCask {
		if pin {
			return fmt.Errorf("cask %s does not support pinning", name)
		}
		return nil // Unpinning a cask is a no-op
	}

	var args []string
	if pin {
		args = []string{"pin", name}
	} else {
		args = []string{"unpin", name}
	}

	result, err := b.executor.Run(ctx, b.brewPath, args, executor.ExecOpts{
		Timeout: 30 * time.Second,
	})

	if err != nil || result.ExitCode != 0 {
		action := "pin"
		if !pin {
			action = "unpin"
		}
		return fmt.Errorf("failed to %s %s: exit code %d, error: %w, stderr: %s",
			action, name, result.ExitCode, err, result.Stderr)
	}

	return nil
}

// UpdateCache updates Homebrew's package cache.
func (b *BrewAdapter) UpdateCache(ctx context.Context) error {
	result, err := b.executor.Run(ctx, b.brewPath, []string{"update"}, executor.ExecOpts{
		Timeout: 120 * time.Second, // 2 minutes for update
	})

	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to update brew cache: exit code %d, error: %w, stderr: %s",
			result.ExitCode, err, result.Stderr)
	}

	return nil
}

// Search searches for packages matching a query.
func (b *BrewAdapter) Search(ctx context.Context, query string) ([]adapters.PackageInfo, error) {
	// Search formulas
	formulaResults, err := b.searchType(ctx, query, false)
	if err != nil {
		return nil, err
	}

	// Search casks
	caskResults, err := b.searchType(ctx, query, true)
	if err != nil {
		// If cask search fails, we just won't include cask results
		caskResults = []adapters.PackageInfo{}
	}

	// Combine results
	formulaResults = append(formulaResults, caskResults...)
	return formulaResults, nil
}

// searchType searches for a specific type (formula or cask).
func (b *BrewAdapter) searchType(ctx context.Context, query string, isCask bool) ([]adapters.PackageInfo, error) {
	args := []string{"search"}
	if isCask {
		args = append(args, "--cask")
	}
	args = append(args, query)

	result, err := b.executor.Run(ctx, b.brewPath, args, executor.ExecOpts{
		Timeout: 60 * time.Second,
	})

	if err != nil || result.ExitCode != 0 {
		return nil, fmt.Errorf("failed to search packages: exit code %d, error: %w", result.ExitCode, err)
	}

	var packages []adapters.PackageInfo
	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip header lines
		if strings.Contains(line, "==>") {
			continue
		}

		// Parse package names (they might be in columns)
		names := regexp.MustCompile(`\s+`).Split(line, -1)
		for _, name := range names {
			name = strings.TrimSpace(name)
			if name != "" {
				packages = append(packages, adapters.PackageInfo{
					Name:      name,
					Installed: false, // We don't know installation status from search
				})
			}
		}
	}

	return packages, nil
}

// Info retrieves detailed information about a package.
func (b *BrewAdapter) Info(ctx context.Context, name string) (*adapters.PackageInfo, error) {
	return b.DetectInstalled(ctx, name)
}

// isCask determines if a package is a cask or formula.
// This method tries both package types to determine the correct one.
// IMPORTANT: This function will generate expected stderr messages during normal operation:
// - "Error: Cask 'package' is unavailable" when testing if a formula is a cask
// - "Error: No available formula" when testing if a cask is a formula
// These error messages are NORMAL and indicate the detection process is working correctly.
func (b *BrewAdapter) isCask(ctx context.Context, name string) (bool, error) {
	// Try to get info as cask first - use correct syntax with --json=v2 --cask
	// NOTE: This will produce stderr like "Error: Cask 'jq' is unavailable" for formulae - this is expected!
	result, err := b.executor.Run(ctx, b.brewPath, []string{"info", "--json=v2", "--cask", name}, executor.ExecOpts{
		Timeout: 30 * time.Second,
	})

	// If cask info succeeds, it's a cask
	if err == nil && result.ExitCode == 0 {
		return true, nil
	}

	// Try as formula - this may also produce expected error messages for casks
	result, err = b.executor.Run(ctx, b.brewPath, []string{"info", "--json", name}, executor.ExecOpts{
		Timeout: 30 * time.Second,
	})

	if err == nil && result.ExitCode == 0 {
		return false, nil
	}

	return false, fmt.Errorf("package %s not found", name)
}

// getStringValue safely extracts a string value from a map[string]interface{}
func getStringValue(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}
