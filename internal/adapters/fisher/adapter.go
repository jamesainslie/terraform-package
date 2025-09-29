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
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/jamesainslie/terraform-provider-package/internal/adapters"
	"github.com/jamesainslie/terraform-provider-package/internal/executor"
)

// FisherAdapter implements the PackageManager interface for Fisher.
type FisherAdapter struct {
	executor    executor.Executor
	fishPath    string
	configDir   string
	pluginsFile string
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
		executor:    exec,
		fishPath:    fishPath,
		configDir:   configDir,
		pluginsFile: pluginsFile,
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

	// TODO: Implement actual detection logic
	// For now, return a basic structure indicating the plugin is not installed
	return &adapters.PackageInfo{
		Name:      name,
		Installed: false,
		Type:      adapters.PackageTypePlugin,
	}, nil
}

// Install installs a plugin with optional version specification.
func (f *FisherAdapter) Install(ctx context.Context, name, version string) error {
	return f.InstallWithType(ctx, name, version, adapters.PackageTypePlugin)
}

// InstallWithType installs a plugin. Fisher only supports plugin type.
func (f *FisherAdapter) InstallWithType(ctx context.Context, name, version string, packageType adapters.PackageType) error {
	if packageType != adapters.PackageTypePlugin && packageType != adapters.PackageTypeAuto {
		return fmt.Errorf("fisher only supports plugin package type, got: %s", packageType)
	}

	tflog.Debug(ctx, "FisherAdapter.InstallWithType starting", map[string]interface{}{
		"plugin_name":  name,
		"version":      version,
		"package_type": string(packageType),
		"fish_path":    f.fishPath,
	})

	// TODO: Implement actual installation logic
	return fmt.Errorf("fisher plugin installation not yet implemented")
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

	// TODO: Implement actual removal logic
	return fmt.Errorf("fisher plugin removal not yet implemented")
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
