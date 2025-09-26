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
	"runtime"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jamesainslie/terraform-package/internal/adapters"
	"github.com/jamesainslie/terraform-package/internal/adapters/apt"
	"github.com/jamesainslie/terraform-package/internal/adapters/brew"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &PackageInfoDataSource{}

// NewPackageInfoDataSource creates a new package info data source.
func NewPackageInfoDataSource() datasource.DataSource {
	return &PackageInfoDataSource{}
}

// PackageInfoDataSource defines the data source implementation.
type PackageInfoDataSource struct {
	providerData *ProviderData
}

// PackageInfoDataSourceModel describes the data source data model.
type PackageInfoDataSourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Manager           types.String `tfsdk:"manager"`
	Installed         types.Bool   `tfsdk:"installed"`
	Version           types.String `tfsdk:"version"`
	AvailableVersions types.List   `tfsdk:"available_versions"`
	Pinned            types.Bool   `tfsdk:"pinned"`
	Repository        types.String `tfsdk:"repository"`
}

// Metadata returns the data source type name.
// Metadata returns the data source type name.
func (d *PackageInfoDataSource) Metadata(
	_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_package_info"
}

// Schema defines the data source schema.
// Schema defines the data source schema.
func (d *PackageInfoDataSource) Schema(
	_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves information about a package from the package manager.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Data source identifier.",
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Package name to query information for.",
				Required:            true,
			},
			"manager": schema.StringAttribute{
				MarkdownDescription: "Package manager to query. " +
					"Valid values: 'auto', 'brew'. " +
					"Defaults to 'auto' which auto-detects based on OS.",
				Optional: true,
			},
			"installed": schema.BoolAttribute{
				MarkdownDescription: "Whether the package is currently installed.",
				Computed:            true,
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "Currently installed version of the package. " +
					"Empty if not installed.",
				Computed: true,
			},
			"available_versions": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "List of available versions for the package.",
				Computed:            true,
			},
			"pinned": schema.BoolAttribute{
				MarkdownDescription: "Whether the package is currently pinned/held.",
				Computed:            true,
			},
			"repository": schema.StringAttribute{
				MarkdownDescription: "Repository or tap that provides this package.",
				Computed:            true,
			},
		},
	}
}

// Configure configures the data source with provider data.
// Configure configures the data source with provider data.
func (d *PackageInfoDataSource) Configure(
	_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Data sources are read-only by design, making them inherently idempotent - multiple reads return the same info
// without side effects.
func (d *PackageInfoDataSource) Read(
	ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PackageInfoDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine package manager
	managerName := "auto"
	if !data.Manager.IsNull() {
		managerName = data.Manager.ValueString()
	}

	// Auto-detect manager based on OS
	if managerName == "auto" {
		switch runtime.GOOS {
		case "darwin":
			managerName = "brew"
		case "linux":
			managerName = "apt"
		default:
			resp.Diagnostics.AddError(
				"Unsupported Operating System",
				fmt.Sprintf("Only macOS (darwin) and Linux are supported, got: %s", runtime.GOOS),
			)
			return
		}
	}

	// Create manager based on name
	var manager adapters.PackageManager
	switch managerName {
	case "brew":
		brewPath := d.providerData.Config.BrewPath.ValueString()
		manager = brew.NewBrewAdapter(d.providerData.Executor, brewPath)
	case "apt":
		aptPath := d.providerData.Config.AptGetPath.ValueString()
		manager = apt.NewAptAdapter(d.providerData.Executor, aptPath, "", aptPath)
	default:
		resp.Diagnostics.AddError(
			"Unsupported Package Manager",
			fmt.Sprintf("Only 'brew' and 'apt' managers are supported, got: %s", managerName),
		)
		return
	}

	// Check if manager is available
	if !manager.IsAvailable(ctx) {
		resp.Diagnostics.AddError(
			"Package Manager Not Available",
			fmt.Sprintf("Package manager %s is not available on this system", managerName),
		)
		return
	}

	// Get package information
	packageName := data.Name.ValueString()
	info, err := manager.DetectInstalled(ctx, packageName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to Get Package Information",
			fmt.Sprintf("Failed to get information for package %s: %v", packageName, err),
		)
		return
	}

	// Set data source state
	data.ID = types.StringValue(packageName)
	data.Name = types.StringValue(info.Name)
	data.Installed = types.BoolValue(info.Installed)
	data.Version = types.StringValue(info.Version)
	// Convert string slice to attr.Value slice for ListValueMust
	versionValues := make([]attr.Value, len(info.AvailableVersions))
	for i, version := range info.AvailableVersions {
		versionValues[i] = types.StringValue(version)
	}
	data.AvailableVersions = types.ListValueMust(types.StringType, versionValues)
	data.Pinned = types.BoolValue(info.Pinned)
	data.Repository = types.StringValue(info.Repository)
	// data.Description = types.StringValue(info.Description) // Assume added to interface if needed
	// data.Homepage = types.StringValue(info.Homepage)     // Assume added

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
