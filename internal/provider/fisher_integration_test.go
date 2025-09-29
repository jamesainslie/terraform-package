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

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jamesainslie/terraform-provider-package/internal/adapters"
	"github.com/jamesainslie/terraform-provider-package/internal/executor"
	"github.com/jamesainslie/terraform-provider-package/internal/registry"
)

// MockExecutor for testing Fisher integration
type MockExecutorFisher struct {
	runFunc func(ctx context.Context, command string, args []string, opts executor.ExecOpts) (executor.ExecResult, error)
}

func (m *MockExecutorFisher) Run(ctx context.Context, command string, args []string, opts executor.ExecOpts) (executor.ExecResult, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, command, args, opts)
	}
	return executor.ExecResult{ExitCode: 0, Stdout: "", Stderr: ""}, nil
}

// TestFisherIntegration_ProviderConfiguration tests that the provider can be configured with Fisher settings
func TestFisherIntegration_ProviderConfiguration(t *testing.T) {
	// Create provider configuration with Fisher settings
	config := &PackageProviderModel{
		DefaultManager:     types.StringValue("fisher"),
		FisherPath:         types.StringValue("/opt/homebrew/bin/fish"),
		FishConfigDir:      types.StringValue("/tmp/test-fish"),
		AssumeYes:          types.BoolValue(true),
		SudoEnabled:        types.BoolValue(false),
		UpdateCache:        types.StringValue("never"),
		LockTimeout:        types.StringValue("5m"),
		RetryCount:         types.Int64Value(2),
		RetryDelay:         types.StringValue("10s"),
		FailOnDownload:     types.BoolValue(false),
		CleanupOnError:     types.BoolValue(true),
		VerifyDownloads:    types.BoolValue(false),
		ChecksumValidation: types.BoolValue(false),
	}

	// Create provider data
	mockExecutor := &MockExecutorFisher{
		runFunc: func(ctx context.Context, command string, args []string, opts executor.ExecOpts) (executor.ExecResult, error) {
			// Mock fish availability check
			if command == "/opt/homebrew/bin/fish" && len(args) >= 2 && args[0] == "-c" {
				if args[1] == "type -q fisher" {
					return executor.ExecResult{ExitCode: 0, Stdout: "", Stderr: ""}, nil
				}
			}
			return executor.ExecResult{ExitCode: 0, Stdout: "", Stderr: ""}, nil
		},
	}

	providerData := &ProviderData{
		Executor:   mockExecutor,
		Registry:   registry.NewDefaultRegistry(),
		Config:     config,
		DetectedOS: "darwin",
	}

	// Create a package resource
	resourceImpl := NewPackageResource()
	resource := resourceImpl.(*PackageResource)
	resource.providerData = providerData

	// Test resolving Fisher as package manager
	packageData := PackageResourceModel{
		Name:        types.StringValue("ilancosman/tide"),
		Version:     types.StringValue("v6"),
		State:       types.StringValue("present"),
		PackageType: types.StringValue("plugin"),
		Managers:    types.ListValueMust(types.StringType, []attr.Value{types.StringValue("fisher")}),
	}

	ctx := context.Background()
	manager, packageName, err := resource.resolvePackageManager(ctx, packageData)

	if err != nil {
		t.Fatalf("Failed to resolve Fisher package manager: %v", err)
	}

	if manager.GetManagerName() != "fisher" {
		t.Errorf("Expected manager name 'fisher', got '%s'", manager.GetManagerName())
	}

	if packageName != "ilancosman/tide" {
		t.Errorf("Expected package name 'ilancosman/tide', got '%s'", packageName)
	}

	// Verify the manager is available
	if !manager.IsAvailable(ctx) {
		t.Error("Fisher manager should be available with mocked executor")
	}
}

// TestFisherIntegration_PackageTypePlugin tests that plugin package type works with Fisher
func TestFisherIntegration_PackageTypePlugin(t *testing.T) {
	config := &PackageProviderModel{
		DefaultManager:     types.StringValue("fisher"),
		FisherPath:         types.StringValue("fish"),
		FishConfigDir:      types.StringValue(""),
		AssumeYes:          types.BoolValue(true),
		SudoEnabled:        types.BoolValue(false),
		UpdateCache:        types.StringValue("never"),
		LockTimeout:        types.StringValue("5m"),
		RetryCount:         types.Int64Value(2),
		RetryDelay:         types.StringValue("10s"),
		FailOnDownload:     types.BoolValue(false),
		CleanupOnError:     types.BoolValue(true),
		VerifyDownloads:    types.BoolValue(false),
		ChecksumValidation: types.BoolValue(false),
	}

	mockExecutor := &MockExecutorFisher{
		runFunc: func(ctx context.Context, command string, args []string, opts executor.ExecOpts) (executor.ExecResult, error) {
			if command == "fish" && len(args) >= 2 && args[0] == "-c" {
				if args[1] == "type -q fisher" {
					return executor.ExecResult{ExitCode: 0, Stdout: "", Stderr: ""}, nil
				}
				if args[1] == "echo $_fisher_plugins" {
					return executor.ExecResult{ExitCode: 0, Stdout: "", Stderr: ""}, nil
				}
			}
			return executor.ExecResult{ExitCode: 0, Stdout: "", Stderr: ""}, nil
		},
	}

	providerData := &ProviderData{
		Executor:   mockExecutor,
		Registry:   registry.NewDefaultRegistry(),
		Config:     config,
		DetectedOS: "darwin",
	}

	resourceImpl := NewPackageResource()
	resource := resourceImpl.(*PackageResource)
	resource.providerData = providerData

	// Test plugin package with Fisher
	packageData := PackageResourceModel{
		Name:        types.StringValue("jorgebucaran/fisher"),
		State:       types.StringValue("present"),
		PackageType: types.StringValue("plugin"),
		Managers:    types.ListValueMust(types.StringType, []attr.Value{types.StringValue("fisher")}),
	}

	ctx := context.Background()
	manager, packageName, err := resource.resolvePackageManager(ctx, packageData)

	if err != nil {
		t.Fatalf("Failed to resolve Fisher for plugin package: %v", err)
	}

	if manager.GetManagerName() != "fisher" {
		t.Errorf("Expected manager name 'fisher', got '%s'", manager.GetManagerName())
	}

	if packageName != "jorgebucaran/fisher" {
		t.Errorf("Expected package name 'jorgebucaran/fisher', got '%s'", packageName)
	}

	// Test that we can detect if plugin is installed
	info, err := manager.DetectInstalled(ctx, packageName)
	if err != nil {
		t.Fatalf("Failed to detect installation: %v", err)
	}

	if info == nil {
		t.Error("Expected package info, got nil")
		return
	}

	if info.Name != packageName {
		t.Errorf("Expected package name '%s', got '%s'", packageName, info.Name)
	}
}

// TestFisherIntegration_ValidatePackageTypes tests that Fisher works with the plugin package type
func TestFisherIntegration_ValidatePackageTypes(t *testing.T) {
	// Test valid package types for Fisher
	validTypes := []adapters.PackageType{
		adapters.PackageTypeAuto,
		adapters.PackageTypePlugin,
	}

	for _, packageType := range validTypes {
		t.Run(string(packageType), func(t *testing.T) {
			config := &PackageProviderModel{
				FisherPath:    types.StringValue("fish"),
				FishConfigDir: types.StringValue(""),
			}

			mockExecutor := &MockExecutorFisher{
				runFunc: func(ctx context.Context, command string, args []string, opts executor.ExecOpts) (executor.ExecResult, error) {
					return executor.ExecResult{ExitCode: 0, Stdout: "", Stderr: ""}, nil
				},
			}

			providerData := &ProviderData{
				Executor: mockExecutor,
				Registry: registry.NewDefaultRegistry(),
				Config:   config,
			}

			resourceImpl := NewPackageResource()
			resource := resourceImpl.(*PackageResource)
			resource.providerData = providerData

			packageData := PackageResourceModel{
				Name:        types.StringValue("test/plugin"),
				PackageType: types.StringValue(string(packageType)),
				Managers:    types.ListValueMust(types.StringType, []attr.Value{types.StringValue("fisher")}),
			}

			ctx := context.Background()
			manager, _, err := resource.resolvePackageManager(ctx, packageData)

			if err != nil {
				t.Fatalf("Failed to resolve Fisher with package type %s: %v", packageType, err)
			}

			if manager.GetManagerName() != "fisher" {
				t.Errorf("Expected manager name 'fisher', got '%s'", manager.GetManagerName())
			}
		})
	}
}
