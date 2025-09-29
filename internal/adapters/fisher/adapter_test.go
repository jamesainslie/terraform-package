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

package fisher

import (
	"context"
	"errors"
	"testing"

	"github.com/jamesainslie/terraform-provider-package/internal/adapters"
	"github.com/jamesainslie/terraform-provider-package/internal/executor"
)

// MockExecutor implements executor.Executor for testing
type MockExecutor struct {
	runFunc func(ctx context.Context, command string, args []string, opts executor.ExecOpts) (executor.ExecResult, error)
}

func (m *MockExecutor) Run(ctx context.Context, command string, args []string, opts executor.ExecOpts) (executor.ExecResult, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, command, args, opts)
	}
	return executor.ExecResult{ExitCode: 0, Stdout: "", Stderr: ""}, nil
}

func TestNewFisherAdapter(t *testing.T) {
	mockExec := &MockExecutor{}
	
	tests := []struct {
		name      string
		fishPath  string
		configDir string
		want      *FisherAdapter
	}{
		{
			name:     "default paths",
			fishPath: "",
			configDir: "",
			want: &FisherAdapter{
				executor:  mockExec,
				fishPath:  "fish",
				configDir: "",
				pluginsFile: "",
			},
		},
		{
			name:     "custom paths",
			fishPath: "/usr/local/bin/fish",
			configDir: "/custom/.config/fish",
			want: &FisherAdapter{
				executor:  mockExec,
				fishPath:  "/usr/local/bin/fish",
				configDir: "/custom/.config/fish",
				pluginsFile: "/custom/.config/fish/fish_plugins",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewFisherAdapter(mockExec, tt.fishPath, tt.configDir)
			
			if got.executor != tt.want.executor {
				t.Errorf("NewFisherAdapter() executor = %v, want %v", got.executor, tt.want.executor)
			}
			if got.fishPath != tt.want.fishPath {
				t.Errorf("NewFisherAdapter() fishPath = %v, want %v", got.fishPath, tt.want.fishPath)
			}
			if tt.want.configDir != "" && got.configDir != tt.want.configDir {
				t.Errorf("NewFisherAdapter() configDir = %v, want %v", got.configDir, tt.want.configDir)
			}
			if tt.want.pluginsFile != "" && got.pluginsFile != tt.want.pluginsFile {
				t.Errorf("NewFisherAdapter() pluginsFile = %v, want %v", got.pluginsFile, tt.want.pluginsFile)
			}
		})
	}
}

func TestFisherAdapter_GetManagerName(t *testing.T) {
	adapter := NewFisherAdapter(&MockExecutor{}, "", "")
	
	got := adapter.GetManagerName()
	want := "fisher"
	
	if got != want {
		t.Errorf("GetManagerName() = %v, want %v", got, want)
	}
}

func TestFisherAdapter_IsAvailable(t *testing.T) {
	tests := []struct {
		name     string
		runFunc  func(ctx context.Context, command string, args []string, opts executor.ExecOpts) (executor.ExecResult, error)
		want     bool
	}{
		{
			name: "fish and fisher available",
			runFunc: func(ctx context.Context, command string, args []string, opts executor.ExecOpts) (executor.ExecResult, error) {
				if len(args) > 0 && args[0] == "--version" {
					return executor.ExecResult{ExitCode: 0, Stdout: "fish, version 3.6.0"}, nil
				}
				if len(args) > 0 && args[0] == "-c" && args[1] == "functions -q fisher" {
					return executor.ExecResult{ExitCode: 0}, nil
				}
				return executor.ExecResult{ExitCode: 1}, errors.New("command failed")
			},
			want: true,
		},
		{
			name: "fish available but fisher not installed",
			runFunc: func(ctx context.Context, command string, args []string, opts executor.ExecOpts) (executor.ExecResult, error) {
				if len(args) > 0 && args[0] == "--version" {
					return executor.ExecResult{ExitCode: 0, Stdout: "fish, version 3.6.0"}, nil
				}
				if len(args) > 0 && args[0] == "-c" && args[1] == "functions -q fisher" {
					return executor.ExecResult{ExitCode: 1}, nil
				}
				return executor.ExecResult{ExitCode: 1}, errors.New("command failed")
			},
			want: false,
		},
		{
			name: "fish not available",
			runFunc: func(ctx context.Context, command string, args []string, opts executor.ExecOpts) (executor.ExecResult, error) {
				return executor.ExecResult{ExitCode: 1}, errors.New("fish: command not found")
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := &MockExecutor{runFunc: tt.runFunc}
			adapter := NewFisherAdapter(mockExec, "fish", "")
			
			got := adapter.IsAvailable(context.Background())
			if got != tt.want {
				t.Errorf("IsAvailable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFisherAdapter_DetectInstalled(t *testing.T) {
	tests := []struct {
		name     string
		plugin   string
		runFunc  func(ctx context.Context, command string, args []string, opts executor.ExecOpts) (executor.ExecResult, error)
		want     *adapters.PackageInfo
		wantErr  bool
	}{
		{
			name:   "plugin installed",
			plugin: "ilancosman/tide",
			runFunc: func(ctx context.Context, command string, args []string, opts executor.ExecOpts) (executor.ExecResult, error) {
				if len(args) > 0 && args[0] == "-c" && args[1] == "echo $_fisher_plugins" {
					return executor.ExecResult{ExitCode: 0, Stdout: "ilancosman/tide jorgebucaran/fisher"}, nil
				}
				return executor.ExecResult{ExitCode: 0}, nil
			},
			want: &adapters.PackageInfo{
				Name:       "ilancosman/tide",
				Type:       adapters.PackageTypePlugin,
				Installed:  true,
				Repository: "https://github.com/ilancosman/tide",
			},
			wantErr: false,
		},
		{
			name:   "plugin not installed",
			plugin: "nonexistent/plugin",
			runFunc: func(ctx context.Context, command string, args []string, opts executor.ExecOpts) (executor.ExecResult, error) {
				if len(args) > 0 && args[0] == "-c" && args[1] == "echo $_fisher_plugins" {
					return executor.ExecResult{ExitCode: 0, Stdout: "ilancosman/tide jorgebucaran/fisher"}, nil
				}
				return executor.ExecResult{ExitCode: 0}, nil
			},
			want: &adapters.PackageInfo{
				Name:      "nonexistent/plugin",
				Type:      adapters.PackageTypePlugin,
				Installed: false,
			},
			wantErr: false,
		},
		{
			name:   "no plugins installed",
			plugin: "ilancosman/tide",
			runFunc: func(ctx context.Context, command string, args []string, opts executor.ExecOpts) (executor.ExecResult, error) {
				if len(args) > 0 && args[0] == "-c" && args[1] == "echo $_fisher_plugins" {
					return executor.ExecResult{ExitCode: 0, Stdout: ""}, nil
				}
				return executor.ExecResult{ExitCode: 0}, nil
			},
			want: &adapters.PackageInfo{
				Name:      "ilancosman/tide",
				Type:      adapters.PackageTypePlugin,
				Installed: false,
			},
			wantErr: false,
		},
		{
			name:   "invalid plugin name",
			plugin: "invalid-name",
			runFunc: func(ctx context.Context, command string, args []string, opts executor.ExecOpts) (executor.ExecResult, error) {
				return executor.ExecResult{ExitCode: 0}, nil
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := &MockExecutor{runFunc: tt.runFunc}
			adapter := NewFisherAdapter(mockExec, "fish", "")
			
			got, err := adapter.DetectInstalled(context.Background(), tt.plugin)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("DetectInstalled() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				if got.Name != tt.want.Name {
					t.Errorf("DetectInstalled() Name = %v, want %v", got.Name, tt.want.Name)
				}
				if got.Type != tt.want.Type {
					t.Errorf("DetectInstalled() Type = %v, want %v", got.Type, tt.want.Type)
				}
				if got.Installed != tt.want.Installed {
					t.Errorf("DetectInstalled() Installed = %v, want %v", got.Installed, tt.want.Installed)
				}
				if tt.want.Repository != "" && got.Repository != tt.want.Repository {
					t.Errorf("DetectInstalled() Repository = %v, want %v", got.Repository, tt.want.Repository)
				}
			}
		})
	}
}

func TestFisherAdapter_Install(t *testing.T) {
	tests := []struct {
		name     string
		plugin   string
		version  string
		runFunc  func(ctx context.Context, command string, args []string, opts executor.ExecOpts) (executor.ExecResult, error)
		wantErr  bool
	}{
		{
			name:    "successful installation",
			plugin:  "ilancosman/tide",
			version: "",
			runFunc: func(ctx context.Context, command string, args []string, opts executor.ExecOpts) (executor.ExecResult, error) {
				if len(args) > 0 && args[0] == "-c" {
					if args[1] == "echo $_fisher_plugins" {
						return executor.ExecResult{ExitCode: 0, Stdout: ""}, nil // Not installed
					}
					if args[1] == "fisher install ilancosman/tide" {
						return executor.ExecResult{ExitCode: 0, Stdout: "Installing ilancosman/tide..."}, nil
					}
				}
				return executor.ExecResult{ExitCode: 0}, nil
			},
			wantErr: false,
		},
		{
			name:    "already installed - idempotent",
			plugin:  "ilancosman/tide",
			version: "",
			runFunc: func(ctx context.Context, command string, args []string, opts executor.ExecOpts) (executor.ExecResult, error) {
				if len(args) > 0 && args[0] == "-c" && args[1] == "echo $_fisher_plugins" {
					return executor.ExecResult{ExitCode: 0, Stdout: "ilancosman/tide"}, nil // Already installed
				}
				return executor.ExecResult{ExitCode: 0}, nil
			},
			wantErr: false,
		},
		{
			name:    "installation with version",
			plugin:  "ilancosman/tide",
			version: "v6",
			runFunc: func(ctx context.Context, command string, args []string, opts executor.ExecOpts) (executor.ExecResult, error) {
				if len(args) > 0 && args[0] == "-c" {
					if args[1] == "echo $_fisher_plugins" {
						return executor.ExecResult{ExitCode: 0, Stdout: ""}, nil // Not installed
					}
					if args[1] == "fisher install ilancosman/tide@v6" {
						return executor.ExecResult{ExitCode: 0, Stdout: "Installing ilancosman/tide@v6..."}, nil
					}
				}
				return executor.ExecResult{ExitCode: 0}, nil
			},
			wantErr: false,
		},
		{
			name:    "installation failure",
			plugin:  "nonexistent/plugin",
			version: "",
			runFunc: func(ctx context.Context, command string, args []string, opts executor.ExecOpts) (executor.ExecResult, error) {
				if len(args) > 0 && args[0] == "-c" {
					if args[1] == "echo $_fisher_plugins" {
						return executor.ExecResult{ExitCode: 0, Stdout: ""}, nil // Not installed
					}
					if args[1] == "fisher install nonexistent/plugin" {
						return executor.ExecResult{
							ExitCode: 1,
							Stderr:   "error: repository not found",
						}, nil
					}
				}
				return executor.ExecResult{ExitCode: 0}, nil
			},
			wantErr: true,
		},
		{
			name:    "invalid plugin name",
			plugin:  "invalid-name",
			version: "",
			runFunc: func(ctx context.Context, command string, args []string, opts executor.ExecOpts) (executor.ExecResult, error) {
				return executor.ExecResult{ExitCode: 0}, nil
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := &MockExecutor{runFunc: tt.runFunc}
			adapter := NewFisherAdapter(mockExec, "fish", "")
			
			err := adapter.Install(context.Background(), tt.plugin, tt.version)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("Install() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFisherAdapter_Remove(t *testing.T) {
	tests := []struct {
		name     string
		plugin   string
		runFunc  func(ctx context.Context, command string, args []string, opts executor.ExecOpts) (executor.ExecResult, error)
		wantErr  bool
	}{
		{
			name:   "successful removal",
			plugin: "ilancosman/tide",
			runFunc: func(ctx context.Context, command string, args []string, opts executor.ExecOpts) (executor.ExecResult, error) {
				if len(args) > 0 && args[0] == "-c" {
					if args[1] == "echo $_fisher_plugins" {
						return executor.ExecResult{ExitCode: 0, Stdout: "ilancosman/tide"}, nil // Installed
					}
					if args[1] == "fisher remove ilancosman/tide" {
						return executor.ExecResult{ExitCode: 0, Stdout: "Removing ilancosman/tide..."}, nil
					}
				}
				return executor.ExecResult{ExitCode: 0}, nil
			},
			wantErr: false,
		},
		{
			name:   "not installed - idempotent",
			plugin: "ilancosman/tide",
			runFunc: func(ctx context.Context, command string, args []string, opts executor.ExecOpts) (executor.ExecResult, error) {
				if len(args) > 0 && args[0] == "-c" && args[1] == "echo $_fisher_plugins" {
					return executor.ExecResult{ExitCode: 0, Stdout: ""}, nil // Not installed
				}
				return executor.ExecResult{ExitCode: 0}, nil
			},
			wantErr: false,
		},
		{
			name:   "removal failure",
			plugin: "ilancosman/tide",
			runFunc: func(ctx context.Context, command string, args []string, opts executor.ExecOpts) (executor.ExecResult, error) {
				if len(args) > 0 && args[0] == "-c" {
					if args[1] == "echo $_fisher_plugins" {
						return executor.ExecResult{ExitCode: 0, Stdout: "ilancosman/tide"}, nil // Installed
					}
					if args[1] == "fisher remove ilancosman/tide" {
						return executor.ExecResult{
							ExitCode: 1,
							Stderr:   "error: failed to remove plugin",
						}, nil
					}
				}
				return executor.ExecResult{ExitCode: 0}, nil
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := &MockExecutor{runFunc: tt.runFunc}
			adapter := NewFisherAdapter(mockExec, "fish", "")
			
			err := adapter.Remove(context.Background(), tt.plugin)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("Remove() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFisherAdapter_Pin(t *testing.T) {
	adapter := NewFisherAdapter(&MockExecutor{}, "", "")
	
	// Test pinning (should fail)
	err := adapter.Pin(context.Background(), "test/plugin", true)
	if err == nil {
		t.Error("Pin(true) should return error, got nil")
	}
	
	// Test unpinning (should succeed)
	err = adapter.Pin(context.Background(), "test/plugin", false)
	if err != nil {
		t.Errorf("Pin(false) should succeed, got error: %v", err)
	}
}

func TestFisherAdapter_UpdateCache(t *testing.T) {
	adapter := NewFisherAdapter(&MockExecutor{}, "", "")
	
	// UpdateCache should always succeed (no-op for Fisher)
	err := adapter.UpdateCache(context.Background())
	if err != nil {
		t.Errorf("UpdateCache() should succeed, got error: %v", err)
	}
}

func TestFisherAdapter_Search(t *testing.T) {
	adapter := NewFisherAdapter(&MockExecutor{}, "", "")
	
	// Search is not yet implemented
	results, err := adapter.Search(context.Background(), "test")
	if err == nil {
		t.Error("Search() should return error (not implemented), got nil")
	}
	if len(results) != 0 {
		t.Errorf("Search() should return empty results when not implemented, got %d results", len(results))
	}
}

func TestFisherAdapter_Info(t *testing.T) {
	mockExec := &MockExecutor{
		runFunc: func(ctx context.Context, command string, args []string, opts executor.ExecOpts) (executor.ExecResult, error) {
			if len(args) > 0 && args[0] == "-c" && args[1] == "echo $_fisher_plugins" {
				return executor.ExecResult{ExitCode: 0, Stdout: "ilancosman/tide"}, nil
			}
			return executor.ExecResult{ExitCode: 0}, nil
		},
	}
	
	adapter := NewFisherAdapter(mockExec, "fish", "")
	
	info, err := adapter.Info(context.Background(), "ilancosman/tide")
	if err != nil {
		t.Errorf("Info() error = %v", err)
		return
	}
	
	if !info.Installed {
		t.Error("Info() should show plugin as installed")
	}
}

func TestFisherAdapter_buildPluginReference(t *testing.T) {
	adapter := NewFisherAdapter(&MockExecutor{}, "", "")
	
	tests := []struct {
		name      string
		pluginRef *PluginRef
		version   string
		want      string
	}{
		{
			name: "local plugin",
			pluginRef: &PluginRef{
				IsLocal: true,
				Path:    "/path/to/plugin",
			},
			version: "",
			want:    "/path/to/plugin",
		},
		{
			name: "github plugin without version",
			pluginRef: &PluginRef{
				Owner: "ilancosman",
				Repo:  "tide",
			},
			version: "",
			want:    "ilancosman/tide",
		},
		{
			name: "github plugin with explicit version",
			pluginRef: &PluginRef{
				Owner: "ilancosman",
				Repo:  "tide",
			},
			version: "v6",
			want:    "ilancosman/tide@v6",
		},
		{
			name: "github plugin with plugin version when no explicit version",
			pluginRef: &PluginRef{
				Owner:   "ilancosman",
				Repo:    "tide",
				Version: "main",
			},
			version: "",
			want:    "ilancosman/tide@main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := adapter.buildPluginReference(tt.pluginRef, tt.version)
			if got != tt.want {
				t.Errorf("buildPluginReference() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFisherAdapter_isAlreadyInstalledError(t *testing.T) {
	adapter := NewFisherAdapter(&MockExecutor{}, "", "")
	
	tests := []struct {
		name   string
		stderr string
		want   bool
	}{
		{
			name:   "already installed",
			stderr: "error: plugin already installed",
			want:   true,
		},
		{
			name:   "already exists",
			stderr: "error: plugin already exists",
			want:   true,
		},
		{
			name:   "different error",
			stderr: "error: network connection failed",
			want:   false,
		},
		{
			name:   "empty stderr",
			stderr: "",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := adapter.isAlreadyInstalledError(tt.stderr)
			if got != tt.want {
				t.Errorf("isAlreadyInstalledError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFisherAdapter_isNotInstalledError(t *testing.T) {
	adapter := NewFisherAdapter(&MockExecutor{}, "", "")
	
	tests := []struct {
		name   string
		stderr string
		want   bool
	}{
		{
			name:   "not installed",
			stderr: "error: plugin not installed",
			want:   true,
		},
		{
			name:   "not found",
			stderr: "error: plugin not found",
			want:   true,
		},
		{
			name:   "no such plugin",
			stderr: "error: no such plugin",
			want:   true,
		},
		{
			name:   "different error",
			stderr: "error: network connection failed",
			want:   false,
		},
		{
			name:   "empty stderr",
			stderr: "",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := adapter.isNotInstalledError(tt.stderr)
			if got != tt.want {
				t.Errorf("isNotInstalledError() = %v, want %v", got, tt.want)
			}
		})
	}
}
