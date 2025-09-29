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

// Package fisher implements the Fisher (Fish shell plugin manager) adapter.
//
// Fisher manages Fish shell plugins from GitHub repositories using the format:
// owner/repo[@version] or local paths for development plugins.
package fisher

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/jamesainslie/terraform-provider-package/internal/adapters"
	"github.com/jamesainslie/terraform-provider-package/internal/executor"
)

// FisherAdapter implements the PackageManager interface for Fisher.
type FisherAdapter struct {
	executor           executor.Executor
	fishPath           string
	configDir          string
	pluginsFile        string
	latestVersionCache map[string]string
}

// NewFisherAdapter creates a new Fisher adapter.
func NewFisherAdapter(exec executor.Executor, fishPath, configDir string) *FisherAdapter {
	if fishPath == "" {
		fishPath = "fish"
	}

	if configDir == "" {
		// Default Fish config directory
		homeDir, err := os.UserHomeDir()
		if err == nil {
			configDir = filepath.Join(homeDir, ".config", "fish")
		}
	}

	pluginsFile := ""
	if configDir != "" {
		pluginsFile = filepath.Join(configDir, "fish_plugins")
	}

	return &FisherAdapter{
		executor:           exec,
		fishPath:           fishPath,
		configDir:          configDir,
		pluginsFile:        pluginsFile,
		latestVersionCache: make(map[string]string),
	}
}

// GetManagerName returns the name of this package manager.
func (f *FisherAdapter) GetManagerName() string {
	return "fisher"
}

// IsAvailable checks if Fisher is available on the system.
func (f *FisherAdapter) IsAvailable(ctx context.Context) bool {
	// Fisher requires Fish shell, which is available on multiple platforms
	// but we need to ensure both Fish and Fisher are installed

	// Check if fish command is available
	_, err := exec.LookPath(f.fishPath)
	if err != nil {
		return false
	}

	// Test fish command execution
	result, err := f.executor.Run(ctx, f.fishPath, []string{"--version"}, executor.ExecOpts{
		Timeout: 10 * time.Second,
	})
	if err != nil || result.ExitCode != 0 {
		return false
	}

	// Check if Fisher is installed by looking for fisher function
	result, err = f.executor.Run(ctx, f.fishPath, []string{"-c", "functions -q fisher"}, executor.ExecOpts{
		Timeout: 5 * time.Second,
	})

	return err == nil && result.ExitCode == 0
}

// DetectInstalled checks if a plugin is installed and returns its information.
func (f *FisherAdapter) DetectInstalled(ctx context.Context, name string) (*adapters.PackageInfo, error) {
	tflog.Debug(ctx, "FisherAdapter.DetectInstalled starting", map[string]interface{}{
		"plugin_name": name,
		"fish_path":   f.fishPath,
		"config_dir":  f.configDir,
	})

	// Parse the plugin name to understand its format
	pluginRef, err := ParsePluginName(name)
	if err != nil {
		return nil, fmt.Errorf("invalid plugin name '%s': %w", name, err)
	}

	// Check if plugin is installed by querying Fisher's plugin list
	installedPlugins, err := f.getInstalledPlugins(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get installed plugins: %w", err)
	}

	// Create basic package info
	packageInfo := &adapters.PackageInfo{
		Name:      name,
		Type:      adapters.PackageTypePlugin,
		Installed: false,
	}

	// Check if plugin is in the installed list
	if installedPlugin, found := f.findInstalledPlugin(pluginRef, installedPlugins); found {
		packageInfo.Installed = true
		packageInfo.Version = installedPlugin.Version
		if pluginRef.IsLocal {
			packageInfo.Repository = "local"
		} else {
			packageInfo.Repository = pluginRef.GitHubURL()
		}

		tflog.Debug(ctx, "Plugin found in installed list", map[string]interface{}{
			"plugin_name": name,
			"version":     installedPlugin.Version,
			"repository":  packageInfo.Repository,
		})
	} else {
		tflog.Debug(ctx, "Plugin not found in installed list", map[string]interface{}{
			"plugin_name": name,
		})
	}

	return packageInfo, nil
}

// Install installs a plugin with optional version specification.
func (f *FisherAdapter) Install(ctx context.Context, name, version string) error {
	return f.InstallWithType(ctx, name, version, adapters.PackageTypePlugin)
}

// InstallWithType installs a plugin. Fisher only supports plugin type.
// Implements idempotency by checking if the plugin is already installed before attempting installation.
func (f *FisherAdapter) InstallWithType(ctx context.Context, name, version string, packageType adapters.PackageType) error {
	if packageType != adapters.PackageTypePlugin && packageType != adapters.PackageTypeAuto {
		return fmt.Errorf("fisher only supports plugin package type, got: %s", packageType)
	}

	// Validate Fish shell version first
	if err := f.validateFishVersion(ctx); err != nil {
		return fmt.Errorf("fish shell compatibility check failed: %w", err)
	}

	tflog.Debug(ctx, "FisherAdapter.InstallWithType starting", map[string]interface{}{
		"plugin_name":  name,
		"version":      version,
		"package_type": string(packageType),
		"fish_path":    f.fishPath,
	})

	// Parse the plugin reference
	pluginRef, err := ParsePluginName(name)
	if err != nil {
		return fmt.Errorf("invalid plugin name '%s': %w", name, err)
	}

	// Resolve version if needed (e.g., "latest" -> actual version)
	resolvedVersion := version
	if !pluginRef.IsLocal && version != "" {
		resolved, err := f.resolveLatestVersion(ctx, &PluginRef{
			Owner:   pluginRef.Owner,
			Repo:    pluginRef.Repo,
			Version: version,
		})
		if err != nil {
			return fmt.Errorf("version resolution failed for %s: %w", name, err)
		}
		resolvedVersion = resolved

		tflog.Debug(ctx, "Version resolved", map[string]interface{}{
			"plugin":           name,
			"original_version": version,
			"resolved_version": resolvedVersion,
		})
	}

	// IDEMPOTENCY CHECK: Check if plugin is already installed
	packageInfo, err := f.DetectInstalled(ctx, name)
	if err == nil && packageInfo.Installed {
		// Check version compatibility
		if resolvedVersion == "" || resolvedVersion == packageInfo.Version {
			tflog.Debug(ctx, "Plugin already installed with correct version, skipping installation", map[string]interface{}{
				"plugin_name":       name,
				"installed_version": packageInfo.Version,
				"requested_version": resolvedVersion,
			})
			return nil
		}
		// Different version requested - continue with installation
		tflog.Debug(ctx, "Different version requested, proceeding with installation", map[string]interface{}{
			"plugin_name":       name,
			"installed_version": packageInfo.Version,
			"requested_version": resolvedVersion,
		})
	}

	// Determine the plugin reference to install
	pluginToInstall := f.buildPluginReference(pluginRef, resolvedVersion)

	tflog.Debug(ctx, "Installing Fisher plugin", map[string]interface{}{
		"plugin_name":      name,
		"plugin_reference": pluginToInstall,
		"is_local":         pluginRef.IsLocal,
	})

	// Execute fisher install command
	args := []string{"-c", fmt.Sprintf("fisher install %s", pluginToInstall)}
	result, err := f.executor.Run(ctx, f.fishPath, args, executor.ExecOpts{
		Timeout: 120 * time.Second, // 2 minutes for plugin installation
	})

	if err != nil || result.ExitCode != 0 {
		// Check for "already installed" errors and handle gracefully
		if f.isAlreadyInstalledError(result.Stderr) {
			tflog.Debug(ctx, "Plugin already installed, treating as success", map[string]interface{}{
				"plugin_name": name,
				"stderr":      result.Stderr,
			})
			return nil
		}

		tflog.Debug(ctx, "Fisher install failed", map[string]interface{}{
			"plugin_name": name,
			"exit_code":   result.ExitCode,
			"error":       err,
			"stderr":      result.Stderr,
			"stdout":      result.Stdout,
		})

		return fmt.Errorf("failed to install Fisher plugin %s: exit code %d, error: %w, stderr: %s",
			name, result.ExitCode, err, result.Stderr)
	}

	tflog.Debug(ctx, "Fisher plugin installed successfully", map[string]interface{}{
		"plugin_name":      name,
		"plugin_reference": pluginToInstall,
	})

	return nil
}

// Remove uninstalls a plugin.
func (f *FisherAdapter) Remove(ctx context.Context, name string) error {
	return f.RemoveWithType(ctx, name, adapters.PackageTypePlugin)
}

// RemoveWithType uninstalls a plugin. Fisher only supports plugin type.
func (f *FisherAdapter) RemoveWithType(ctx context.Context, name string, packageType adapters.PackageType) error {
	if packageType != adapters.PackageTypePlugin && packageType != adapters.PackageTypeAuto {
		return fmt.Errorf("fisher only supports plugin package type, got: %s", packageType)
	}

	tflog.Debug(ctx, "FisherAdapter.RemoveWithType starting", map[string]interface{}{
		"plugin_name":  name,
		"package_type": string(packageType),
		"fish_path":    f.fishPath,
	})

	// Parse the plugin reference
	pluginRef, err := ParsePluginName(name)
	if err != nil {
		return fmt.Errorf("invalid plugin name '%s': %w", name, err)
	}

	// IDEMPOTENCY CHECK: Check if plugin is currently installed
	packageInfo, err := f.DetectInstalled(ctx, name)
	if err == nil && !packageInfo.Installed {
		tflog.Debug(ctx, "Plugin not installed, nothing to remove", map[string]interface{}{
			"plugin_name": name,
		})
		return nil // Already not installed - idempotent
	}

	// Use the plugin reference for removal
	pluginToRemove := pluginRef.String()

	tflog.Debug(ctx, "Removing Fisher plugin", map[string]interface{}{
		"plugin_name":      name,
		"plugin_reference": pluginToRemove,
		"is_local":         pluginRef.IsLocal,
	})

	// Execute fisher remove command
	args := []string{"-c", fmt.Sprintf("fisher remove %s", pluginToRemove)}
	result, err := f.executor.Run(ctx, f.fishPath, args, executor.ExecOpts{
		Timeout: 60 * time.Second, // 1 minute for plugin removal
	})

	if err != nil || result.ExitCode != 0 {
		// Check for "not installed" errors and handle gracefully
		if f.isNotInstalledError(result.Stderr) {
			tflog.Debug(ctx, "Plugin not installed, treating as success", map[string]interface{}{
				"plugin_name": name,
				"stderr":      result.Stderr,
			})
			return nil
		}

		tflog.Debug(ctx, "Fisher remove failed", map[string]interface{}{
			"plugin_name": name,
			"exit_code":   result.ExitCode,
			"error":       err,
			"stderr":      result.Stderr,
			"stdout":      result.Stdout,
		})

		return fmt.Errorf("failed to remove Fisher plugin %s: exit code %d, error: %w, stderr: %s",
			name, result.ExitCode, err, result.Stderr)
	}

	tflog.Debug(ctx, "Fisher plugin removed successfully", map[string]interface{}{
		"plugin_name":      name,
		"plugin_reference": pluginToRemove,
	})

	return nil
}

// Pin pins or unpins a plugin. Fisher doesn't support pinning.
func (f *FisherAdapter) Pin(ctx context.Context, name string, pin bool) error {
	if pin {
		return fmt.Errorf("fisher does not support pinning plugins")
	}
	// Unpinning is a no-op since Fisher doesn't support pinning
	return nil
}

// UpdateCache updates Fisher's plugin cache. Fisher uses Git, no explicit cache update needed.
func (f *FisherAdapter) UpdateCache(ctx context.Context) error {
	// Fisher doesn't have a cache to update - it works directly with Git repositories
	// This is essentially a no-op for Fisher
	tflog.Debug(ctx, "FisherAdapter.UpdateCache called - no cache to update for Fisher")
	return nil
}

// Search searches for plugins. This would require GitHub API integration.
func (f *FisherAdapter) Search(ctx context.Context, query string) ([]adapters.PackageInfo, error) {
	tflog.Debug(ctx, "FisherAdapter.Search starting", map[string]interface{}{
		"query": query,
	})

	// TODO: Implement GitHub API search for Fish shell plugins
	return []adapters.PackageInfo{}, fmt.Errorf("fisher plugin search not yet implemented")
}

// Info retrieves detailed information about a plugin.
func (f *FisherAdapter) Info(ctx context.Context, name string) (*adapters.PackageInfo, error) {
	return f.DetectInstalled(ctx, name)
}

// InstalledPlugin represents a Fisher plugin that is currently installed.
type InstalledPlugin struct {
	Name    string // Original plugin reference (e.g., "owner/repo@version")
	Version string // Git commit SHA or tag
	Path    string // Local path if applicable
}

// getInstalledPlugins retrieves the list of currently installed Fisher plugins.
func (f *FisherAdapter) getInstalledPlugins(ctx context.Context) ([]InstalledPlugin, error) {
	// Query Fisher's $_fisher_plugins universal variable
	// This contains the list of currently installed plugins
	result, err := f.executor.Run(ctx, f.fishPath, []string{"-c", "echo $_fisher_plugins"}, executor.ExecOpts{
		Timeout: 10 * time.Second,
	})

	if err != nil || result.ExitCode != 0 {
		// Fisher might not be installed or $_fisher_plugins might not be set
		tflog.Debug(ctx, "Failed to query Fisher plugins", map[string]interface{}{
			"error":     err,
			"exit_code": result.ExitCode,
			"stderr":    result.Stderr,
		})
		return []InstalledPlugin{}, nil
	}

	// Parse the output - plugins are typically space-separated
	pluginList := strings.TrimSpace(result.Stdout)
	if pluginList == "" {
		return []InstalledPlugin{}, nil
	}

	var plugins []InstalledPlugin
	pluginNames := strings.Fields(pluginList)

	for _, pluginName := range pluginNames {
		if pluginName == "" {
			continue
		}

		plugin := InstalledPlugin{
			Name: pluginName,
		}

		// Try to get version information by checking Git metadata
		// This is more complex and might require accessing the plugin's directory
		// For now, we'll leave version empty and implement this in a future iteration
		plugin.Version = ""

		plugins = append(plugins, plugin)
	}

	tflog.Debug(ctx, "Retrieved installed plugins", map[string]interface{}{
		"plugin_count": len(plugins),
		"plugins":      pluginNames,
	})

	return plugins, nil
}

// findInstalledPlugin searches for a plugin reference in the list of installed plugins.
func (f *FisherAdapter) findInstalledPlugin(pluginRef *PluginRef, installedPlugins []InstalledPlugin) (*InstalledPlugin, bool) {
	for _, installed := range installedPlugins {
		if f.pluginMatches(pluginRef, installed) {
			return &installed, true
		}
	}
	return nil, false
}

// pluginMatches checks if a plugin reference matches an installed plugin.
func (f *FisherAdapter) pluginMatches(pluginRef *PluginRef, installed InstalledPlugin) bool {
	// Try exact match first
	if pluginRef.Raw == installed.Name {
		return true
	}

	// For GitHub plugins, try different variations
	if !pluginRef.IsLocal {
		// Check owner/repo format without version
		baseFormat := fmt.Sprintf("%s/%s", pluginRef.Owner, pluginRef.Repo)
		if baseFormat == installed.Name {
			return true
		}

		// Check with version if specified
		if pluginRef.Version != "" {
			withVersion := fmt.Sprintf("%s@%s", baseFormat, pluginRef.Version)
			if withVersion == installed.Name {
				return true
			}
		}
	}

	// For local plugins, check path matching
	if pluginRef.IsLocal {
		// Simple path comparison - could be enhanced for relative paths
		if pluginRef.Path == installed.Name || pluginRef.Raw == installed.Name {
			return true
		}
	}

	return false
}

// buildPluginReference constructs the plugin reference to use for installation.
func (f *FisherAdapter) buildPluginReference(pluginRef *PluginRef, version string) string {
	if pluginRef.IsLocal {
		return pluginRef.Path
	}

	// For GitHub plugins, use owner/repo[@version] format
	baseRef := fmt.Sprintf("%s/%s", pluginRef.Owner, pluginRef.Repo)

	// Use explicit version if provided, otherwise use plugin's version, otherwise just base
	if version != "" {
		return fmt.Sprintf("%s@%s", baseRef, version)
	} else if pluginRef.Version != "" {
		return fmt.Sprintf("%s@%s", baseRef, pluginRef.Version)
	}

	return baseRef
}

// isAlreadyInstalledError checks if the error message indicates the plugin is already installed.
func (f *FisherAdapter) isAlreadyInstalledError(stderr string) bool {
	lowerStderr := strings.ToLower(stderr)
	return strings.Contains(lowerStderr, "already") &&
		(strings.Contains(lowerStderr, "installed") || strings.Contains(lowerStderr, "exists"))
}

// isNotInstalledError checks if the error message indicates the plugin is not installed.
func (f *FisherAdapter) isNotInstalledError(stderr string) bool {
	lowerStderr := strings.ToLower(stderr)
	return strings.Contains(lowerStderr, "not installed") ||
		strings.Contains(lowerStderr, "not found") ||
		strings.Contains(lowerStderr, "no such plugin")
}
