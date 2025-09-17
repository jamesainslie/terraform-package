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

	"github.com/geico-private/terraform-provider-pkg/internal/adapters"
	"github.com/geico-private/terraform-provider-pkg/internal/executor"
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
func (b *BrewAdapter) DetectInstalled(ctx context.Context, name string) (*adapters.PackageInfo, error) {
	// First try as formula, then as cask
	info, err := b.getPackageInfo(ctx, name, false)
	if err != nil {
		// Try as cask
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
	// Check if it's a cask first
	isCask, err := b.isCask(ctx, name)
	if err != nil {
		// If we can't determine, try as formula first
		isCask = false
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
	}

	args = append(args, packageName)

	result, err := b.executor.Run(ctx, b.brewPath, args, executor.ExecOpts{
		Timeout: 300 * time.Second, // 5 minutes for package installation
	})

	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to install %s: exit code %d, error: %w, stderr: %s",
			packageName, result.ExitCode, err, result.Stderr)
	}

	return nil
}

// Remove uninstalls a package.
func (b *BrewAdapter) Remove(ctx context.Context, name string) error {
	// Check if it's a cask
	isCask, err := b.isCask(ctx, name)
	if err != nil {
		// If we can't determine, try as formula first
		isCask = false
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
func (b *BrewAdapter) isCask(ctx context.Context, name string) (bool, error) {
	// Try to get info as cask first - use correct syntax with --json=v2 --cask
	result, err := b.executor.Run(ctx, b.brewPath, []string{"info", "--json=v2", "--cask", name}, executor.ExecOpts{
		Timeout: 30 * time.Second,
	})

	// If cask info succeeds, it's a cask
	if err == nil && result.ExitCode == 0 {
		return true, nil
	}

	// Try as formula
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
