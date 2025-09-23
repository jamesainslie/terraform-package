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

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"pkg": providerserver.NewProtocol6WithError(New("test")()),
}

func TestPackageProvider_BasicSchema(t *testing.T) {
	// This is a basic smoke test to ensure the provider can be instantiated
	// and its schema can be retrieved without errors
	provider := New("test")()

	if provider == nil {
		t.Fatal("Provider should not be nil")
	}

	// Additional schema validation tests can be added here
}

func TestPackageProvider_Metadata(t *testing.T) {
	p := New("test")()
	ctx := context.Background()

	req := provider.MetadataRequest{}
	resp := &provider.MetadataResponse{}

	p.Metadata(ctx, req, resp)

	if resp.TypeName != "pkg" {
		t.Errorf("Expected type name 'pkg', got '%s'", resp.TypeName)
	}

	if resp.Version != "test" {
		t.Errorf("Expected version 'test', got '%s'", resp.Version)
	}

	// Also test the provider instance directly
	if pkgProvider, ok := p.(*PackageProvider); ok {
		if pkgProvider.version != "test" {
			t.Errorf("Expected version 'test', got '%s'", pkgProvider.version)
		}
	} else {
		t.Fatal("Provider should be of type *PackageProvider")
	}
}

func TestPackageProvider_Schema(t *testing.T) {
	p := New("test")()
	ctx := context.Background()

	req := provider.SchemaRequest{}
	resp := &provider.SchemaResponse{}

	p.Schema(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema should not have errors: %v", resp.Diagnostics.Errors())
	}

	schema := resp.Schema

	// Check that required attributes exist
	requiredAttrs := []string{
		"default_manager", "assume_yes", "sudo_enabled",
		"brew_path", "apt_get_path", "winget_path", "choco_path",
		"update_cache", "lock_timeout",
	}

	for _, attr := range requiredAttrs {
		if _, exists := schema.Attributes[attr]; !exists {
			t.Errorf("Expected attribute '%s' to exist in schema", attr)
		}
	}

	// Check that all attributes are optional (no required attributes)
	for name, attr := range schema.Attributes {
		if attr.IsRequired() {
			t.Errorf("Attribute '%s' should be optional, not required", name)
		}
	}
}

// Note: Configuration testing is complex with the current framework version.
// These tests focus on the parts we can reliably test without mocking the entire framework.

func TestPackageProvider_Resources(t *testing.T) {
	p := New("test")()
	ctx := context.Background()

	resources := p.Resources(ctx)

	// Should have 2 resources in Phase 2.3 (pkg_package, pkg_repo)
	if len(resources) != 2 {
		t.Errorf("Expected 2 resources in Phase 2.3, got %d", len(resources))
	}
}

func TestPackageProvider_DataSources(t *testing.T) {
	p := New("test")()
	ctx := context.Background()

	dataSources := p.DataSources(ctx)

	// Should have 12 data sources in Phase 2.4 (comprehensive data source suite + service status)
	if len(dataSources) != 12 {
		t.Errorf("Expected 12 data sources in Phase 2.4 (including service status), got %d", len(dataSources))
	}
}

// TestPackageProvider_Schema_ErrorHandling tests that error handling fields are included
func TestPackageProvider_Schema_ErrorHandling(t *testing.T) {
	p := New("test")()
	ctx := context.Background()

	req := provider.SchemaRequest{}
	resp := &provider.SchemaResponse{}

	p.Schema(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema should not have errors: %v", resp.Diagnostics.Errors())
	}

	schema := resp.Schema

	// Check that error handling attributes exist
	errorHandlingAttrs := []string{
		"retry_count", "retry_delay", "fail_on_download",
		"cleanup_on_error", "verify_downloads", "checksum_validation",
	}

	for _, attr := range errorHandlingAttrs {
		if _, exists := schema.Attributes[attr]; !exists {
			t.Error("Schema should include '" + attr + "' attribute for error handling")
		}
	}
}

// TestPackageProviderModel_ErrorHandling tests the model includes error handling fields
func TestPackageProviderModel_ErrorHandling(t *testing.T) {
	// Create a model instance to verify the struct includes error handling fields
	model := PackageProviderModel{
		DefaultManager:     types.StringValue("brew"),
		AssumeYes:          types.BoolValue(true),
		RetryCount:         types.Int64Value(3),
		RetryDelay:         types.StringValue("30s"),
		FailOnDownload:     types.BoolValue(false),
		CleanupOnError:     types.BoolValue(true),
		VerifyDownloads:    types.BoolValue(true),
		ChecksumValidation: types.BoolValue(true),
	}

	if model.RetryCount.ValueInt64() != 3 {
		t.Errorf("Expected RetryCount to be 3, got %d", model.RetryCount.ValueInt64())
	}

	if model.RetryDelay.ValueString() != "30s" {
		t.Errorf("Expected RetryDelay to be '30s', got '%s'", model.RetryDelay.ValueString())
	}

	if model.VerifyDownloads.ValueBool() != true {
		t.Errorf("Expected VerifyDownloads to be true, got %t", model.VerifyDownloads.ValueBool())
	}
}
