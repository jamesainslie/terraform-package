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
)


// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &SecurityInfoDataSource{}

// NewSecurityInfoDataSource creates a new security info data source.
func NewSecurityInfoDataSource() datasource.DataSource {
	return &SecurityInfoDataSource{}
}

// SecurityInfoDataSource defines the data source implementation.
type SecurityInfoDataSource struct {
	providerData *ProviderData
}

// SecurityInfoDataSourceModel describes the data source data model.
type SecurityInfoDataSourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Manager       types.String `tfsdk:"manager"`
	HasAdvisories types.Bool   `tfsdk:"has_advisories"`
	Advisories    types.List   `tfsdk:"advisories"`
	LastChecked   types.String `tfsdk:"last_checked"`
}

// SecurityAdvisory represents a security advisory for a package.
type SecurityAdvisory struct {
	ID          types.String `tfsdk:"id"`
	Severity    types.String `tfsdk:"severity"`
	Title       types.String `tfsdk:"title"`
	Description types.String `tfsdk:"description"`
	CVE         types.String `tfsdk:"cve"`
	URL         types.String `tfsdk:"url"`
}

// Metadata returns the data source type name.
// Metadata returns the data source type name.
func (d *SecurityInfoDataSource) Metadata(
	_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_security_info"
}

// Schema defines the data source schema.
// Schema defines the data source schema.
func (d *SecurityInfoDataSource) Schema(
	_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves security information and advisories for a package.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Data source identifier.",
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Package name to get security information for.",
				Required:            true,
			},
			"manager": schema.StringAttribute{
				MarkdownDescription: "Package manager to query. " +
					"Valid values: 'auto', 'brew'. " +
					"Defaults to 'auto'.",
				Optional: true,
			},
			"has_advisories": schema.BoolAttribute{
				MarkdownDescription: "Whether the package has any known security advisories.",
				Computed:            true,
			},
			"last_checked": schema.StringAttribute{
				MarkdownDescription: "Timestamp when security information was last checked.",
				Computed:            true,
			},
			"advisories": schema.ListNestedAttribute{
				MarkdownDescription: "List of security advisories for the package.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Advisory identifier.",
							Computed:            true,
						},
						"severity": schema.StringAttribute{
							MarkdownDescription: "Severity level (low, medium, high, critical).",
							Computed:            true,
						},
						"title": schema.StringAttribute{
							MarkdownDescription: "Advisory title.",
							Computed:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "Advisory description.",
							Computed:            true,
						},
						"cve": schema.StringAttribute{
							MarkdownDescription: "CVE identifier if available.",
							Computed:            true,
						},
						"url": schema.StringAttribute{
							MarkdownDescription: "URL to advisory details.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

// Configure configures the data source with provider data.
// Configure configures the data source with provider data.
func (d *SecurityInfoDataSource) Configure(
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

func (d *SecurityInfoDataSource) Read(
	ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SecurityInfoDataSourceModel

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

	// Auto-detect manager based on OS (Phase 2: only macOS supported)
	if managerName == managerAuto {
		if runtime.GOOS != platformDarwin {
			resp.Diagnostics.AddError(
				"Unsupported Operating System",
				fmt.Sprintf("Only macOS (darwin) is supported in Phase 2, got: %s", runtime.GOOS),
			)
			return
		}
		managerName = "brew"
	}

	// Only support brew in Phase 2
	if managerName != "brew" {
		resp.Diagnostics.AddError(
			"Unsupported Package Manager",
			fmt.Sprintf("Only 'brew' manager is supported in Phase 2, got: %s", managerName),
		)
		return
	}

	packageName := data.Name.ValueString()

	// Note: Homebrew doesn't have built-in security advisory checking
	// This is a placeholder implementation that could be enhanced to check
	// external security databases or integrate with tools like `brew audit`

	// Set computed values
	data.ID = types.StringValue(fmt.Sprintf("%s:security:%s", managerName, packageName))
	data.Manager = types.StringValue(managerName)
	data.HasAdvisories = types.BoolValue(false)
	data.LastChecked = types.StringValue("not implemented")

	// Empty advisories list for now
	emptyList, diags := types.ListValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"id":          types.StringType,
			"severity":    types.StringType,
			"title":       types.StringType,
			"description": types.StringType,
			"cve":         types.StringType,
			"url":         types.StringType,
		},
	}, []SecurityAdvisory{})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Advisories = emptyList

	// Add a warning that this is not fully implemented
	resp.Diagnostics.AddWarning(
		"Security Info Not Fully Implemented",
		"Security advisory checking is not fully implemented for Homebrew. This data source returns empty results and could be enhanced to integrate with security databases.",
	)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
