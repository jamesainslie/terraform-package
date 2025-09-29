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
	"fmt"
	"testing"

	"github.com/jamesainslie/terraform-provider-package/internal/executor"
)

// TestResolveLatestVersion tests the GitHub API integration for resolving "latest" version
func TestResolveLatestVersion(t *testing.T) {
	tests := []struct {
		name           string
		owner          string
		repo           string
		inputVersion   string
		expectedResult string
		expectedError  bool
	}{
		{
			name:           "latest version for ilancosman/tide",
			owner:          "ilancosman",
			repo:           "tide",
			inputVersion:   "latest",
			expectedResult: "v6.0.1", // Mock expected latest
			expectedError:  false,
		},
		{
			name:           "specific version unchanged",
			owner:          "ilancosman", 
			repo:           "tide",
			inputVersion:   "v5.0.0",
			expectedResult: "v5.0.0", // Should remain unchanged
			expectedError:  false,
		},
		{
			name:           "latest version for jorgebucaran/fisher",
			owner:          "jorgebucaran",
			repo:           "fisher",
			inputVersion:   "latest",
			expectedResult: "4.4.4", // Mock expected latest
			expectedError:  false,
		},
		{
			name:          "invalid repository",
			owner:         "nonexistent",
			repo:          "repo",
			inputVersion:  "latest",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create adapter with proper constructor to initialize cache
			mockExecutor := &MockExecutor{
				runFunc: func(ctx context.Context, command string, args []string, opts executor.ExecOpts) (executor.ExecResult, error) {
					return executor.ExecResult{ExitCode: 0}, nil
				},
			}
			adapter := NewFisherAdapter(mockExecutor, "fish", "")
			
			// Mock the GitHub API response for testing
			if !tt.expectedError {
				adapter.latestVersionCache[fmt.Sprintf("%s/%s", tt.owner, tt.repo)] = tt.expectedResult
			}

			pluginRef := &PluginRef{
				Owner:   tt.owner,
				Repo:    tt.repo,
				Version: tt.inputVersion,
			}

			ctx := context.Background()
			result, err := adapter.resolveLatestVersion(ctx, pluginRef)

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error for %s/%s, got none", tt.owner, tt.repo)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != tt.expectedResult {
				t.Errorf("Expected version %s, got %s", tt.expectedResult, result)
			}
		})
	}
}

// TestFishVersionValidation tests Fish shell version compatibility checking
func TestFishVersionValidation(t *testing.T) {
	tests := []struct {
		name          string
		fishVersion   string
		expectedError bool
		errorContains string
	}{
		{
			name:          "fish 3.6.0 - compatible", 
			fishVersion:   "fish, version 3.6.0",
			expectedError: false,
		},
		{
			name:          "fish 3.4.0 - minimum compatible",
			fishVersion:   "fish, version 3.4.0", 
			expectedError: false,
		},
		{
			name:          "fish 3.3.1 - too old",
			fishVersion:   "fish, version 3.3.1",
			expectedError: true,
			errorContains: "requires Fish shell version 3.4.0 or later",
		},
		{
			name:          "fish 4.0.0 - future version compatible",
			fishVersion:   "fish, version 4.0.0",
			expectedError: false,
		},
		{
			name:          "invalid version format",
			fishVersion:   "fish, invalid version",
			expectedError: true,
			errorContains: "unable to parse Fish shell version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock executor that returns the specified version
			mockExecutor := &MockExecutor{
				runFunc: func(ctx context.Context, command string, args []string, opts executor.ExecOpts) (executor.ExecResult, error) {
					if command == "fish" && len(args) > 0 && args[0] == "--version" {
						return executor.ExecResult{
							ExitCode: 0,
							Stdout:   tt.fishVersion,
							Stderr:   "",
						}, nil
					}
					return executor.ExecResult{ExitCode: 1}, nil
				},
			}

			adapter := NewFisherAdapter(mockExecutor, "fish", "")
			ctx := context.Background()
			
			err := adapter.validateFishVersion(ctx)

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error for version %s, got none", tt.fishVersion)
					return
				}
				if tt.errorContains != "" && !stringContains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for version %s: %v", tt.fishVersion, err)
			}
		})
	}
}

// TestEnhancedInstallWithVersionResolution tests installation with version resolution
func TestEnhancedInstallWithVersionResolution(t *testing.T) {
	tests := []struct {
		name            string
		pluginName      string
		inputVersion    string
		resolvedVersion string
		expectedCommand []string
	}{
		{
			name:            "install tide with latest version",
			pluginName:      "ilancosman/tide",
			inputVersion:    "latest", 
			resolvedVersion: "v6.0.1",
			expectedCommand: []string{"-c", "fisher install ilancosman/tide@v6.0.1"},
		},
		{
			name:            "install with specific version unchanged",
			pluginName:      "ilancosman/tide",
			inputVersion:    "v5.0.0",
			resolvedVersion: "v5.0.0",
			expectedCommand: []string{"-c", "fisher install ilancosman/tide@v5.0.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedCommand []string
			
			mockExecutor := &MockExecutor{
				runFunc: func(ctx context.Context, command string, args []string, opts executor.ExecOpts) (executor.ExecResult, error) {
					// Mock Fish version check
					if command == "fish" && len(args) > 0 && args[0] == "--version" {
						return executor.ExecResult{ExitCode: 0, Stdout: "fish, version 3.6.0"}, nil
					}
					// Mock fisher availability
					if len(args) > 1 && args[1] == "type -q fisher" {
						return executor.ExecResult{ExitCode: 0}, nil
					}
					// Mock plugin install
					if len(args) > 1 && stringContains(args[1], "fisher install") {
						capturedCommand = args
						return executor.ExecResult{ExitCode: 0, Stdout: "Installing..."}, nil
					}
					return executor.ExecResult{ExitCode: 0}, nil
				},
			}

			adapter := NewFisherAdapter(mockExecutor, "fish", "")
			
			// Mock the resolveLatestVersion method result
			adapter.latestVersionCache = map[string]string{
				tt.pluginName: tt.resolvedVersion,
			}
			
			ctx := context.Background()
			err := adapter.Install(ctx, tt.pluginName, tt.inputVersion)

			if err != nil {
				t.Fatalf("Install failed: %v", err)
			}

			// Verify the correct command was called
			if len(capturedCommand) != len(tt.expectedCommand) {
				t.Errorf("Expected command %v, got %v", tt.expectedCommand, capturedCommand)
				return
			}

			for i, expected := range tt.expectedCommand {
				if !stringContains(capturedCommand[i], expected) {
					t.Errorf("Expected command to contain %s, got %s", expected, capturedCommand[i])
				}
			}
		})
	}
}

// Helper function to check if string contains substring
func stringContains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || findSubstringInString(s, substr))
}

func findSubstringInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
