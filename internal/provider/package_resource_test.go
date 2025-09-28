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
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestNewPackageResource(t *testing.T) {
	r := NewPackageResource()
	if r == nil {
		t.Error("NewPackageResource() should not return nil")
	}

	// Check that it implements the required interfaces
	// r is already of type resource.Resource from NewPackageResource(), so this check is satisfied

	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("NewPackageResource() should implement resource.ResourceWithImportState")
	}
}

func TestPackageResource_Metadata(t *testing.T) {
	r := &PackageResource{}
	ctx := context.Background()

	req := resource.MetadataRequest{
		ProviderTypeName: "pkg",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(ctx, req, resp)

	if resp.TypeName != "pkg_package" {
		t.Errorf("Expected TypeName 'pkg_package', got '%s'", resp.TypeName)
	}
}

func TestPackageResource_Schema(t *testing.T) {
	r := &PackageResource{}
	ctx := context.Background()

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(ctx, req, resp)

	// Check that required attributes are present
	if resp.Schema.Attributes["name"] == nil {
		t.Error("Schema should include 'name' attribute")
	}

	if resp.Schema.Attributes["state"] == nil {
		t.Error("Schema should include 'state' attribute")
	}
}

func TestPackageResource_GetTimeout(t *testing.T) {
	r := &PackageResource{}

	tests := []struct {
		name           string
		timeoutValue   types.String
		expectedResult time.Duration
	}{
		{
			name:           "null_timeout_uses_default",
			timeoutValue:   types.StringNull(),
			expectedResult: 10 * time.Minute,
		},
		{
			name:           "empty_timeout_uses_default",
			timeoutValue:   types.StringValue(""),
			expectedResult: 10 * time.Minute,
		},
		{
			name:           "valid_timeout_parsed_correctly",
			timeoutValue:   types.StringValue("5m"),
			expectedResult: 5 * time.Minute,
		},
		{
			name:           "invalid_timeout_uses_default",
			timeoutValue:   types.StringValue("invalid"),
			expectedResult: 10 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.getTimeout(tt.timeoutValue, "10m")
			if result != tt.expectedResult {
				t.Errorf("Expected timeout %v, got %v", tt.expectedResult, result)
			}
		})
	}
}

func TestPackageResource_Configure(t *testing.T) {
	r := &PackageResource{}
	ctx := context.Background()

	req := resource.ConfigureRequest{}
	resp := &resource.ConfigureResponse{}

	r.Configure(ctx, req, resp)

	// Should complete without errors since we handle nil ProviderData
	if resp.Diagnostics.HasError() {
		t.Errorf("Configure should not return errors for nil ProviderData")
	}
}

// TestPackageResource_Schema_PackageType tests that the schema includes package_type attribute
func TestPackageResource_Schema_PackageType(t *testing.T) {
	r := &PackageResource{}
	ctx := context.Background()

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(ctx, req, resp)

	// Check that package_type attribute is present
	if resp.Schema.Attributes["package_type"] == nil {
		t.Error("Schema should include 'package_type' attribute")
	}

	// Verify it's optional (not required)
	packageTypeAttr := resp.Schema.Attributes["package_type"]
	if packageTypeAttr.IsRequired() {
		t.Error("package_type attribute should be optional, not required")
	}
}

// TestPackageResourceModel_PackageType tests the model includes PackageType field
func TestPackageResourceModel_PackageType(t *testing.T) {
	// Create a model instance to verify the struct includes PackageType field
	model := PackageResourceModel{
		ID:          types.StringValue("brew:test"),
		Name:        types.StringValue("test"),
		State:       types.StringValue("present"),
		PackageType: types.StringValue("cask"), // This should compile if field exists
	}

	if model.PackageType.ValueString() != "cask" {
		t.Errorf("Expected PackageType to be 'cask', got '%s'", model.PackageType.ValueString())
	}
}

// TestPackageResource_ResolvePackageManager_CaskType tests cask package type resolution
func TestPackageResource_ResolvePackageManager_CaskType(t *testing.T) {
	// This test verifies that when package_type = "cask" is specified,
	// the resolvePackageManager method correctly handles the package type field
	r := &PackageResource{}

	data := PackageResourceModel{
		Name:        types.StringValue("cursor"),
		PackageType: types.StringValue("cask"),
		State:       types.StringValue("present"),
	}

	ctx := context.Background()

	// We expect this to fail gracefully due to missing provider data
	// But it should not panic and should be able to handle the PackageType field
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("resolvePackageManager should not panic, but got: %v", r)
		}
	}()

	_, _, err := r.resolvePackageManager(ctx, data)

	// We expect an error due to missing provider data
	if err == nil {
		t.Error("Expected error due to missing provider data, but got none")
	}

	// The error should NOT be about PackageType being undefined - that would indicate our changes are working
	if strings.Contains(err.Error(), "PackageType undefined") {
		t.Error("PackageType field should be properly implemented")
	}
}

// TestPackageResourceModel_DependencyFields tests the model includes dependency fields
func TestPackageResourceModel_DependencyFields(t *testing.T) {
	// Create a model instance to verify the struct includes dependency fields
	model := PackageResourceModel{
		ID:                 types.StringValue("brew:test"),
		Name:               types.StringValue("test"),
		State:              types.StringValue("present"),
		Dependencies:       types.ListValueMust(types.StringType, []attr.Value{types.StringValue("dep1")}),
		InstallPriority:    types.Int64Value(10),
		DependencyStrategy: types.StringValue("install_missing"),
	}

	if model.Dependencies.IsNull() {
		t.Error("Dependencies field should not be null")
	}

	if model.InstallPriority.ValueInt64() != 10 {
		t.Errorf("Expected InstallPriority to be 10, got %d", model.InstallPriority.ValueInt64())
	}

	if model.DependencyStrategy.ValueString() != "install_missing" {
		t.Errorf("Expected DependencyStrategy to be 'install_missing', got '%s'", model.DependencyStrategy.ValueString())
	}
}

// TestPackageResource_Dependencies_InstallOrder tests installation ordering
func TestPackageResource_Dependencies_InstallOrder(t *testing.T) {
	// This test will verify that packages with higher install_priority
	// are installed before packages with lower priority

	packages := []PackageWithPriority{
		{Name: "low-priority", Priority: 1},
		{Name: "high-priority", Priority: 10},
		{Name: "medium-priority", Priority: 5},
	}

	r := &PackageResource{}
	ordered := r.sortPackagesByPriority(packages)

	// Should sort by priority descending (high first)
	if len(ordered) != 3 {
		t.Errorf("Expected 3 packages, got %d", len(ordered))
	}

	if len(ordered) >= 3 {
		if ordered[0].Name != "high-priority" {
			t.Errorf("Expected high-priority first, got %s", ordered[0].Name)
		}
		if ordered[1].Name != "medium-priority" {
			t.Errorf("Expected medium-priority second, got %s", ordered[1].Name)
		}
		if ordered[2].Name != "low-priority" {
			t.Errorf("Expected low-priority last, got %s", ordered[2].Name)
		}
	}
}

// TestPackageResource_DriftDetection_CheckVersion tests version drift detection
func TestPackageResource_DriftDetection_CheckVersion(t *testing.T) {
	// This test will verify version drift detection functionality

	r := &PackageResource{}

	// Test drift detection method structure
	current := PackageState{
		Name:      "terraform",
		Version:   "1.5.0",
		Installed: true,
	}

	desired := PackageState{
		Name:      "terraform",
		Version:   "1.6.0",
		Installed: true,
	}

	drift := r.detectVersionDrift(current, desired)

	// Should detect version drift
	if !drift.HasVersionDrift {
		t.Error("Should detect version drift between 1.5.0 and 1.6.0")
	}

	if drift.CurrentVersion != "1.5.0" {
		t.Errorf("Expected current version '1.5.0', got '%s'", drift.CurrentVersion)
	}

	if drift.DesiredVersion != "1.6.0" {
		t.Errorf("Expected desired version '1.6.0', got '%s'", drift.DesiredVersion)
	}
}
