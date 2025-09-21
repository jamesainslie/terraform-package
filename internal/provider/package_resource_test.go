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
	r := NewPackageResource()
	if r == nil {
		t.Error("NewPackageResource() should not return nil")
	}

	// Check that it implements the required interfaces
	if _, ok := r.(resource.Resource); !ok {
		t.Error("NewPackageResource() should implement resource.Resource")
	}

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