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
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &RegistryLookupDataSource{}

func NewRegistryLookupDataSource() datasource.DataSource {
	return &RegistryLookupDataSource{}
}

// RegistryLookupDataSource defines the data source implementation.
type RegistryLookupDataSource struct {
	providerData *ProviderData
}

// RegistryLookupDataSourceModel describes the data source data model.
type RegistryLookupDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	LogicalName types.String `tfsdk:"logical_name"`
	Darwin      types.String `tfsdk:"darwin"`
	Linux       types.String `tfsdk:"linux"`
	Windows     types.String `tfsdk:"windows"`
	Found       types.Bool   `tfsdk:"found"`
}

func (d *RegistryLookupDataSource) Metadata(
		ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_registry_lookup"
}

func (d *RegistryLookupDataSource) Schema(
		ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up platform-specific package names from the package registry.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Data source identifier.",
			},
			"logical_name": schema.StringAttribute{
				MarkdownDescription: "Logical package name to look up in the registry.",
				Required:            true,
			},
			"darwin": schema.StringAttribute{
				MarkdownDescription: "Package name for macOS/Homebrew. " +
					"Empty if not found.",
				Computed:            true,
			},
			"linux": schema.StringAttribute{
				MarkdownDescription: "Package name for Linux/APT. " +
					"Empty if not found.",
				Computed:            true,
			},
			"windows": schema.StringAttribute{
				MarkdownDescription: "Package name for Windows/winget. " +
					"Empty if not found.",
				Computed:            true,
			},
			"found": schema.BoolAttribute{
				MarkdownDescription: "Whether the logical package name was found in the registry.",
				Computed:            true,
			},
		},
	}
}

func (d *RegistryLookupDataSource) Configure(
		ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(*ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *ProviderData, got: %T. Please report this issue to the provider developers.",
				req.ProviderData),
		)
		return
	}

	d.providerData = providerData
}

func (d *RegistryLookupDataSource) Read(
		ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RegistryLookupDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	logicalName := data.LogicalName.ValueString()

	// Look up the package mapping in the registry
	mapping, err := d.providerData.Registry.GetPackageMapping(ctx, logicalName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Registry Lookup Failed",
			fmt.Sprintf("Failed to lookup package %s in registry: %v", logicalName, err),
		)
		return
	}

	// Set computed values
	data.ID = types.StringValue(fmt.Sprintf("registry:%s", logicalName))

	if mapping != nil {
		// Package found in registry
		data.Found = types.BoolValue(true)
		data.Darwin = types.StringValue(mapping.Darwin)
		data.Linux = types.StringValue(mapping.Linux)
		data.Windows = types.StringValue(mapping.Windows)
	} else {
		// Package not found in registry
		data.Found = types.BoolValue(false)
		data.Darwin = types.StringValue("")
		data.Linux = types.StringValue("")
		data.Windows = types.StringValue("")
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
