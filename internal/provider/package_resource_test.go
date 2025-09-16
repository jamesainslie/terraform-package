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
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestNewPackageResource(t *testing.T) {
	resource := NewPackageResource()

	if resource == nil {
		t.Fatal("NewPackageResource should not return nil")
	}

	// Verify it implements the correct interfaces
	if _, ok := resource.(*PackageResource); !ok {
		t.Error("NewPackageResource should return *PackageResource")
	}

	// Note: Interface compliance is verified at compile time via var _ declarations in the resource file
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

	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema should not have errors: %v", resp.Diagnostics.Errors())
	}

	schema := resp.Schema

	// Check that required attributes exist
	requiredAttrs := []string{
		"id", "name", "state", "version", "version_actual", "pin",
		"managers", "aliases", "reinstall_on_drift", "hold_dependencies",
	}

	for _, attr := range requiredAttrs {
		if _, exists := schema.Attributes[attr]; !exists {
			t.Errorf("Expected attribute '%s' to exist in schema", attr)
		}
	}

	// Check that timeouts block exists
	if _, exists := schema.Blocks["timeouts"]; !exists {
		t.Error("Expected 'timeouts' block to exist in schema")
	}

	// Verify name is required
	if nameAttr, exists := schema.Attributes["name"]; exists {
		if !nameAttr.IsRequired() {
			t.Error("Expected 'name' attribute to be required")
		}
	}

	// Verify id is computed
	if idAttr, exists := schema.Attributes["id"]; exists {
		if !idAttr.IsComputed() {
			t.Error("Expected 'id' attribute to be computed")
		}
	}
}

func TestPackageResource_GetTimeout(t *testing.T) {
	r := &PackageResource{}

	tests := []struct {
		name           string
		timeoutStr     types.String
		defaultTimeout string
		expected       time.Duration
	}{
		{
			name:           "null timeout uses default",
			timeoutStr:     types.StringNull(),
			defaultTimeout: "5m",
			expected:       5 * time.Minute,
		},
		{
			name:           "empty timeout uses default",
			timeoutStr:     types.StringValue(""),
			defaultTimeout: "10m",
			expected:       10 * time.Minute,
		},
		{
			name:           "valid timeout parsed correctly",
			timeoutStr:     types.StringValue("30s"),
			defaultTimeout: "5m",
			expected:       30 * time.Second,
		},
		{
			name:           "invalid timeout uses default",
			timeoutStr:     types.StringValue("invalid"),
			defaultTimeout: "2m",
			expected:       2 * time.Minute,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := r.getTimeout(test.timeoutStr, test.defaultTimeout)
			if result != test.expected {
				t.Errorf("Expected timeout %v, got %v", test.expected, result)
			}
		})
	}
}

func TestPackageResource_Configure(t *testing.T) {
	r := &PackageResource{}
	ctx := context.Background()

	// Test with nil provider data
	req := resource.ConfigureRequest{
		ProviderData: nil,
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(ctx, req, resp)

	// Should not error with nil provider data
	if resp.Diagnostics.HasError() {
		t.Error("Configure should not error with nil provider data")
	}

	// Test with wrong type
	req.ProviderData = "wrong type"
	resp = &resource.ConfigureResponse{}

	r.Configure(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Configure should error with wrong provider data type")
	}

	// Test with correct provider data
	providerData := &ProviderData{}
	req.ProviderData = providerData
	resp = &resource.ConfigureResponse{}

	r.Configure(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("Configure should not error with correct provider data: %v", resp.Diagnostics.Errors())
	}

	if r.providerData != providerData {
		t.Error("Configure should set provider data correctly")
	}
}
