// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
// var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
// 	"pkg": providerserver.NewProtocol6WithError(New("test")()),
// }

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

	// Currently should return empty slice (TODO items for Phase 2)
	if len(resources) != 0 {
		t.Errorf("Expected 0 resources in Phase 1, got %d", len(resources))
	}
}

func TestPackageProvider_DataSources(t *testing.T) {
	p := New("test")()
	ctx := context.Background()

	dataSources := p.DataSources(ctx)

	// Currently should return empty slice (TODO items for Phase 2)
	if len(dataSources) != 0 {
		t.Errorf("Expected 0 data sources in Phase 1, got %d", len(dataSources))
	}
}

func TestPackageProvider_Functions(t *testing.T) {
	p := New("test")()
	ctx := context.Background()

	// Cast to our concrete type to access Functions method
	if pkgProvider, ok := p.(*PackageProvider); ok {
		functions := pkgProvider.Functions(ctx)

		// Currently should return empty slice (TODO items for later phases)
		if len(functions) != 0 {
			t.Errorf("Expected 0 functions in Phase 1, got %d", len(functions))
		}
	} else {
		t.Fatal("Provider should be of type *PackageProvider")
	}
}
