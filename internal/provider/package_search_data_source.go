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

	"github.com/geico-private/terraform-provider-pkg/internal/adapters/brew"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &PackageSearchDataSource{}

func NewPackageSearchDataSource() datasource.DataSource {
	return &PackageSearchDataSource{}
}

// PackageSearchDataSource defines the data source implementation.
type PackageSearchDataSource struct {
	providerData *ProviderData
}

// PackageSearchDataSourceModel describes the data source data model.
type PackageSearchDataSourceModel struct {
	ID      types.String `tfsdk:"id"`
	Query   types.String `tfsdk:"query"`
	Manager types.String `tfsdk:"manager"`
	Results types.List   `tfsdk:"results"`
}

// PackageSearchResult represents a single search result.
type PackageSearchResult struct {
	Name       types.String `tfsdk:"name"`
	Repository types.String `tfsdk:"repository"`
}

func (d *PackageSearchDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_package_search"
}

func (d *PackageSearchDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Searches for packages in the package manager catalog.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Data source identifier.",
			},
			"query": schema.StringAttribute{
				MarkdownDescription: "Search query string to find packages.",
				Required:            true,
			},
			"manager": schema.StringAttribute{
				MarkdownDescription: "Package manager to search. Valid values: 'auto', 'brew'. Defaults to 'auto' which auto-detects based on OS.",
				Optional:            true,
			},
			"results": schema.ListNestedAttribute{
				MarkdownDescription: "List of packages matching the search query.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "Package name.",
							Computed:            true,
						},
						"repository": schema.StringAttribute{
							MarkdownDescription: "Repository or tap that provides this package.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *PackageSearchDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(*ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *ProviderData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.providerData = providerData
}

func (d *PackageSearchDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PackageSearchDataSourceModel

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
	if managerName == "auto" {
		if runtime.GOOS != "darwin" {
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

	// Create Homebrew adapter
	brewPath := d.providerData.Config.BrewPath.ValueString()
	manager := brew.NewBrewAdapter(d.providerData.Executor, brewPath)

	// Check if manager is available
	if !manager.IsAvailable(ctx) {
		resp.Diagnostics.AddError(
			"Package Manager Not Available",
			fmt.Sprintf("Package manager %s is not available on this system", managerName),
		)
		return
	}

	// Perform the search
	query := data.Query.ValueString()
	searchResults, err := manager.Search(ctx, query)
	if err != nil {
		resp.Diagnostics.AddError(
			"Package Search Failed",
			fmt.Sprintf("Failed to search for packages with query '%s': %v", query, err),
		)
		return
	}

	// Convert search results to data source model
	results := make([]PackageSearchResult, len(searchResults))
	for i, result := range searchResults {
		results[i] = PackageSearchResult{
			Name:       types.StringValue(result.Name),
			Repository: types.StringValue(result.Repository),
		}
	}

	// Set computed values
	data.ID = types.StringValue(fmt.Sprintf("%s:%s", managerName, query))
	data.Manager = types.StringValue(managerName)

	// Convert results to list
	resultsList, diags := types.ListValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"name":       types.StringType,
			"repository": types.StringType,
		},
	}, results)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Results = resultsList

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
