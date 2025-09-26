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

// Package apt implements the APT package manager adapter for Debian/Ubuntu systems.
package apt

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/jamesainslie/terraform-package/internal/adapters"
	"github.com/jamesainslie/terraform-package/internal/executor"
)

// AptAdapter implements the PackageManager interface for APT.
type AptAdapter struct {
	executor     executor.Executor
	aptGetPath   string
	dpkgPath     string
	aptCachePath string
}

// NewAptAdapter creates a new APT adapter.
func NewAptAdapter(exec executor.Executor, aptGetPath, dpkgPath, aptCachePath string) *AptAdapter {
	if aptGetPath == "" {
		aptGetPath = "apt-get"
	}
	if dpkgPath == "" {
		dpkgPath = "dpkg-query"
	}
	if aptCachePath == "" {
		aptCachePath = "apt-cache"
	}
	return &AptAdapter{
		executor:     exec,
		aptGetPath:   aptGetPath,
		dpkgPath:     dpkgPath,
		aptCachePath: aptCachePath,
	}
}

// GetManagerName returns the name of this package manager.
func (a *AptAdapter) GetManagerName() string {
	return "apt"
}

// IsAvailable checks if APT is available on the system.
func (a *AptAdapter) IsAvailable(ctx context.Context) bool {
	if runtime.GOOS != "linux" {
		return false
	}

	// Check if apt-get command is available
	_, err := exec.LookPath(a.aptGetPath)
	if err != nil {
		return false
	}

	// Test apt-get command execution with --version
	result, err := a.executor.Run(ctx, a.aptGetPath, []string{"--version"}, executor.ExecOpts{
		Timeout: 10 * time.Second,
	})
	return err == nil && result.ExitCode == 0
}

// AptPackageInfo represents APT package information parsed from dpkg-query or apt-cache.
type AptPackageInfo struct {
	Name              string   `json:"name"`
	Version           string   `json:"version"`
	Installed         bool     `json:"installed"`
	AvailableVersions []string `json:"available_versions"`
	Repository        string   `json:"repository"`
	Status            string   `json:"status"` // e.g., "install ok installed"
}

// DetectInstalled checks if a package is installed and returns its information.
func (a *AptAdapter) DetectInstalled(ctx context.Context, name string) (*adapters.PackageInfo, error) {
	// Use dpkg-query to check installation status and version
	args := []string{"--showformat", "${Package} ${Version} ${Status} ${Maintainer}", "--show", name}
	result, err := a.executor.Run(ctx, a.dpkgPath, args, executor.ExecOpts{
		Timeout: 30 * time.Second,
	})
	if err != nil || result.ExitCode != 0 {
		// Package not installed or dpkg-query error
		return &adapters.PackageInfo{
			Name:      name,
			Installed: false,
		}, nil
	}

	// Parse output: package version status maintainer
	parts := strings.Fields(result.Stdout)
	if len(parts) < 3 {
		return &adapters.PackageInfo{
			Name:      name,
			Installed: false,
		}, fmt.Errorf("failed to parse dpkg-query output for %s", name)
	}

	version := parts[1]
	// Status can be multiple words (e.g., "install ok installed"), so check the entire output
	installed := strings.Contains(result.Stdout, "installed")

	info := &adapters.PackageInfo{
		Name:      name,
		Version:   version,
		Installed: installed,
		Type:      adapters.PackageTypeFormula, // APT doesn't have cask-like distinction
	}

	// Get available versions using apt-cache policy
	availResult, _ := a.executor.Run(ctx, a.aptCachePath, []string{"policy", name}, executor.ExecOpts{
		Timeout: 30 * time.Second,
	})
	if availResult.ExitCode == 0 {
		// Parse apt-cache policy output for versions
		lines := strings.Split(availResult.Stdout, "\n")
		for _, line := range lines {
			if strings.Contains(line, "Candidate") || strings.Contains(line, "Version table") {
				// Extract version numbers using regex
				re := regexp.MustCompile(`([0-9.+-]+)`)
				matches := re.FindAllString(line, -1)
				if len(matches) > 0 {
					info.AvailableVersions = append(info.AvailableVersions, matches...)
				}
				break
			}
		}
	}

	return info, nil
}

// Install installs a package with optional version specification.
func (a *AptAdapter) Install(ctx context.Context, name, version string) error {
	return a.InstallWithType(ctx, name, version, adapters.PackageTypeAuto)
}

// InstallWithType installs a package. APT doesn't support types like cask/formula.
// Implements idempotency by checking if the package is already installed before attempting installation.
func (a *AptAdapter) InstallWithType(ctx context.Context, name, version string, packageType adapters.PackageType) error {
	// IDEMPOTENCY CHECK: Check if package is already installed to avoid unnecessary operations
	info, err := a.DetectInstalled(ctx, name)
	if err == nil && info.Installed {
		// Package is already installed - check version compatibility
		if version == "" || version == info.Version {
			// Already installed with correct version, no action needed
			return nil
		}
		// Different version requested - continue with installation (may upgrade/downgrade)
	}

	// First, update cache to ensure latest package info
	if err := a.UpdateCache(ctx); err != nil {
		return fmt.Errorf("failed to update APT cache before install: %w", err)
	}

	args := []string{"install", "-y", "--no-install-recommends"}
	if version != "" {
		args = append(args, fmt.Sprintf("%s=%s", name, version))
	} else {
		args = append(args, name)
	}

	result, err := a.executor.Run(ctx, a.aptGetPath, args, executor.ExecOpts{
		Timeout: 600 * time.Second, // 10 minutes for installation
	})
	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to install %s: exit code %d, error: %w, stderr: %s",
			name, result.ExitCode, err, result.Stderr)
	}

	return nil
}

// Remove removes a package.
func (a *AptAdapter) Remove(ctx context.Context, name string) error {
	return a.RemoveWithType(ctx, name, adapters.PackageTypeAuto)
}

// RemoveWithType removes a package. APT doesn't support types.
func (a *AptAdapter) RemoveWithType(ctx context.Context, name string, packageType adapters.PackageType) error {
	args := []string{"remove", "-y", name}

	result, err := a.executor.Run(ctx, a.aptGetPath, args, executor.ExecOpts{
		Timeout: 300 * time.Second, // 5 minutes for removal
	})
	if err != nil || result.ExitCode != 0 {
		// APT remove returns non-zero if package not installed, but we treat as no-op for idempotency
		if strings.Contains(result.Stderr, "Package") && strings.Contains(result.Stderr, "is not installed") {
			return nil
		}
		return fmt.Errorf("failed to remove %s: exit code %d, error: %w, stderr: %s",
			name, result.ExitCode, err, result.Stderr)
	}

	return nil
}

// Pin pins or unpins a package.
func (a *AptAdapter) Pin(ctx context.Context, name string, pin bool) error {
	var args []string
	if pin {
		args = []string{"hold", name}
	} else {
		args = []string{"unhold", name}
	}

	result, err := a.executor.Run(ctx, a.aptGetPath, args, executor.ExecOpts{
		Timeout: 30 * time.Second,
	})
	if err != nil || result.ExitCode != 0 {
		action := "hold"
		if !pin {
			action = "unhold"
		}
		return fmt.Errorf("failed to %s %s: exit code %d, error: %w, stderr: %s",
			action, name, result.ExitCode, err, result.Stderr)
	}

	return nil
}

// UpdateCache updates the APT package cache.
func (a *AptAdapter) UpdateCache(ctx context.Context) error {
	result, err := a.executor.Run(ctx, a.aptGetPath, []string{"update"}, executor.ExecOpts{
		Timeout: 120 * time.Second, // 2 minutes for update
	})
	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to update APT cache: exit code %d, error: %w, stderr: %s",
			result.ExitCode, err, result.Stderr)
	}

	return nil
}

// Search searches for packages matching a query.
func (a *AptAdapter) Search(ctx context.Context, query string) ([]adapters.PackageInfo, error) {
	result, err := a.executor.Run(ctx, a.aptGetPath, []string{"search", "--no-install-recommends", query}, executor.ExecOpts{
		Timeout: 60 * time.Second,
	})
	if err != nil || result.ExitCode != 0 {
		return nil, fmt.Errorf("failed to search packages: exit code %d, error: %w", result.ExitCode, err)
	}

	var packages []adapters.PackageInfo
	lines := strings.Split(result.Stdout, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Listing...") {
			continue
		}
		// Parse package names from search output (first column)
		parts := strings.Fields(line)
		if len(parts) > 0 {
			packages = append(packages, adapters.PackageInfo{
				Name:      parts[0],
				Installed: false, // Search doesn't show install status
			})
		}
	}

	return packages, nil
}

// Info retrieves detailed information about a package.
func (a *AptAdapter) Info(ctx context.Context, name string) (*adapters.PackageInfo, error) {
	return a.DetectInstalled(ctx, name)
}
