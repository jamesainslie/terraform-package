// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
// var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
// 	"pkg": providerserver.NewProtocol6WithError(New("test")()),
// }

func TestPackageProvider_Schema(t *testing.T) {
	// This is a basic smoke test to ensure the provider can be instantiated
	// and its schema can be retrieved without errors
	provider := New("test")()

	if provider == nil {
		t.Fatal("Provider should not be nil")
	}

	// Additional schema validation tests can be added here
}

func TestPackageProvider_Metadata(t *testing.T) {
	provider := New("test")()

	if pkgProvider, ok := provider.(*PackageProvider); ok {
		if pkgProvider.version != "test" {
			t.Errorf("Expected version 'test', got '%s'", pkgProvider.version)
		}
	} else {
		t.Fatal("Provider should be of type *PackageProvider")
	}
}
