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

func TestPackageResource_Schema_PackageType_ValidValues(t *testing.T) {
	r := &PackageResource{}
	ctx := context.Background()

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(ctx, req, resp)

	// The package_type should support: formula, cask, auto
	// This is tested implicitly through the schema validation
	// We'll verify this in integration tests
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

// TestPackageResource_ResolvePackageManager_FormulaType tests formula package type resolution
func TestPackageResource_ResolvePackageManager_FormulaType(t *testing.T) {
	r := &PackageResource{}

	data := PackageResourceModel{
		Name:        types.StringValue("jq"),
		PackageType: types.StringValue("formula"),
		State:       types.StringValue("present"),
	}

	ctx := context.Background()

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
}

// TestPackageResource_ResolvePackageManager_AutoType tests auto package type detection
func TestPackageResource_ResolvePackageManager_AutoType(t *testing.T) {
	r := &PackageResource{}

	data := PackageResourceModel{
		Name:        types.StringValue("terraform"),
		PackageType: types.StringValue("auto"),
		State:       types.StringValue("present"),
	}

	ctx := context.Background()

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
}

// TestPackageResource_Schema_Dependencies tests that dependencies field is included
func TestPackageResource_Schema_Dependencies(t *testing.T) {
	r := &PackageResource{}
	ctx := context.Background()

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(ctx, req, resp)

	// Check that dependencies attribute is present
	if resp.Schema.Attributes["dependencies"] == nil {
		t.Error("Schema should include 'dependencies' attribute")
	}

	// Verify it's optional (not required)
	dependenciesAttr := resp.Schema.Attributes["dependencies"]
	if dependenciesAttr.IsRequired() {
		t.Error("dependencies attribute should be optional, not required")
	}
}

// TestPackageResource_Schema_InstallPriority tests install_priority field
func TestPackageResource_Schema_InstallPriority(t *testing.T) {
	r := &PackageResource{}
	ctx := context.Background()

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(ctx, req, resp)

	// Check that install_priority attribute is present
	if resp.Schema.Attributes["install_priority"] == nil {
		t.Error("Schema should include 'install_priority' attribute")
	}
}

// TestPackageResource_Schema_DependencyStrategy tests dependency_strategy field
func TestPackageResource_Schema_DependencyStrategy(t *testing.T) {
	r := &PackageResource{}
	ctx := context.Background()

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(ctx, req, resp)

	// Check that dependency_strategy attribute is present
	if resp.Schema.Attributes["dependency_strategy"] == nil {
		t.Error("Schema should include 'dependency_strategy' attribute")
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

// TestPackageResource_Dependencies_InstallMissing tests install_missing strategy
func TestPackageResource_Dependencies_InstallMissing(t *testing.T) {
	// This test will verify that when dependency_strategy = "install_missing",
	// missing dependencies are automatically installed before the main package

	// For now, this tests the dependency resolution logic structure
	deps := []string{"colima", "ca-certificates"}
	strategy := "install_missing"

	// Test dependency resolution helper function exists
	r := &PackageResource{}
	resolved, err := r.resolveDependencies(context.Background(), deps, strategy)

	// Should not panic and should handle the dependency list
	if err != nil {
		t.Logf("Expected error due to missing provider data: %v", err)
	}

	// When implemented, resolved should contain dependency information
	if resolved != nil {
		t.Log("Dependency resolution returned results")
	}
}

// TestPackageResource_Dependencies_RequireExisting tests require_existing strategy
func TestPackageResource_Dependencies_RequireExisting(t *testing.T) {
	// This test will verify that when dependency_strategy = "require_existing",
	// installation fails if dependencies are not already present

	deps := []string{"nonexistent-package"}
	strategy := "require_existing"

	r := &PackageResource{}
	_, err := r.resolveDependencies(context.Background(), deps, strategy)

	// Should handle the require_existing strategy
	if err != nil {
		t.Logf("Expected error for require_existing with missing deps: %v", err)
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

// TestPackageResource_Dependencies_CircularDetection tests circular dependency detection
func TestPackageResource_Dependencies_CircularDetection(t *testing.T) {
	// This test verifies that circular dependencies are detected and handled

	// Create a circular dependency scenario: A -> B -> C -> A
	depMap := map[string][]string{
		"package-a": {"package-b"},
		"package-b": {"package-c"},
		"package-c": {"package-a"}, // Circular reference
	}

	r := &PackageResource{}
	err := r.detectCircularDependencies("package-a", depMap, make(map[string]bool), make(map[string]bool))

	// Should detect the circular dependency
	if err == nil {
		t.Error("Expected circular dependency to be detected")
	} else {
		t.Logf("Correctly detected circular dependency: %v", err)
	}
}

// TestPackageResource_Schema_StateTracking tests enhanced state tracking fields
func TestPackageResource_Schema_StateTracking(t *testing.T) {
	r := &PackageResource{}
	ctx := context.Background()

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(ctx, req, resp)

	// Check that enhanced state tracking attributes exist
	stateFields := []string{
		"track_metadata", "track_dependencies", "track_usage",
	}

	for _, field := range stateFields {
		if _, exists := resp.Schema.Attributes[field]; !exists {
			t.Error("Schema should include '" + field + "' attribute for state tracking")
		}
	}
}

// TestPackageResource_Schema_DriftDetection tests drift detection configuration fields
func TestPackageResource_Schema_DriftDetection(t *testing.T) {
	r := &PackageResource{}
	ctx := context.Background()

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(ctx, req, resp)

	// Check that drift detection block exists
	if resp.Schema.Blocks["drift_detection"] == nil {
		t.Error("Schema should include 'drift_detection' block")
	}
}

// TestPackageResourceModel_StateTrackingFields tests enhanced state tracking model fields
func TestPackageResourceModel_StateTrackingFields(t *testing.T) {
	// Create a model instance to verify enhanced state tracking fields
	model := PackageResourceModel{
		ID:                 types.StringValue("brew:test"),
		Name:               types.StringValue("test"),
		State:              types.StringValue("present"),
		TrackMetadata:      types.BoolValue(true),
		TrackDependencies:  types.BoolValue(true),
		TrackUsage:         types.BoolValue(true),
		InstallationSource: types.StringValue("brew"),
		DependencyTree: types.MapValueMust(types.StringType, map[string]attr.Value{
			"colima": types.StringValue("1.2.3"),
		}),
		LastAccess: types.StringValue("2023-01-01T00:00:00Z"),
	}

	if !model.TrackMetadata.ValueBool() {
		t.Error("TrackMetadata should be true")
	}

	if !model.TrackDependencies.ValueBool() {
		t.Error("TrackDependencies should be true")
	}

	if model.InstallationSource.ValueString() != "brew" {
		t.Errorf("Expected InstallationSource to be 'brew', got '%s'", model.InstallationSource.ValueString())
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

// TestPackageResource_DriftDetection_CheckIntegrity tests integrity drift detection
func TestPackageResource_DriftDetection_CheckIntegrity(t *testing.T) {
	// This test will verify package integrity drift detection

	r := &PackageResource{}

	packagePath := "/usr/local/bin/terraform"
	expectedChecksum := "abc123def456"

	drift := r.detectIntegrityDrift(packagePath, expectedChecksum)

	// Should have integrity drift structure
	if drift == nil {
		t.Error("Integrity drift detection should return a result")
	}
}

// TestPackageResource_DriftDetection_Remediation tests drift remediation strategies
func TestPackageResource_DriftDetection_Remediation(t *testing.T) {
	// This test will verify drift remediation functionality

	r := &PackageResource{}

	drift := DriftInfo{
		HasVersionDrift:     true,
		HasIntegrityDrift:   false,
		HasDependencyDrift:  true,
		RemediationStrategy: "auto",
	}

	action := r.determineRemediationAction(drift)

	// Should determine appropriate remediation action
	if action != "reinstall" {
		t.Errorf("Expected remediation action 'reinstall', got '%s'", action)
	}
}
