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
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/geico-private/terraform-provider-pkg/internal/executor"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &RepositoryPackagesDataSource{}

// NewRepositoryPackagesDataSource creates a new repository packages data source.
// NewRepositoryPackagesDataSource creates a new repository packages data source.
func NewRepositoryPackagesDataSource() datasource.DataSource {
	return &RepositoryPackagesDataSource{}
}

// RepositoryPackagesDataSource defines the data source implementation.
type RepositoryPackagesDataSource struct {
	providerData *ProviderData
}

// RepositoryPackagesDataSourceModel describes the data source data model.
type RepositoryPackagesDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	Manager    types.String `tfsdk:"manager"`
	Repository types.String `tfsdk:"repository"`
	Limit      types.Int64  `tfsdk:"limit"`
	Packages   types.List   `tfsdk:"packages"`
}

// RepositoryPackageInfo represents information about a package in a repository.
type RepositoryPackageInfo struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Version     types.String `tfsdk:"version"`
}

// Metadata returns the data source type name.
func (d *RepositoryPackagesDataSource) Metadata(
		ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_packages"
}

// Schema defines the data source schema.
func (d *RepositoryPackagesDataSource) Schema(
		ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists packages available from a specific repository or tap.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Data source identifier.",
			},
			"manager": schema.StringAttribute{
				MarkdownDescription: "Package manager to query. " +
					"Valid values: 'auto', 'brew'. " +
					"Defaults to 'auto'.",
				Optional:            true,
			},
			"repository": schema.StringAttribute{
				MarkdownDescription: "Repository or tap name to list packages from (e.g., 'homebrew/cask-fonts').",
				Required:            true,
			},
			"limit": schema.Int64Attribute{
				MarkdownDescription: "Maximum number of packages to return. " +
					"Defaults to 100.",
				Optional:            true,
			},
			"packages": schema.ListNestedAttribute{
				MarkdownDescription: "List of packages available from the repository.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "Package name.",
							Computed:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "Package description.",
							Computed:            true,
						},
						"version": schema.StringAttribute{
							MarkdownDescription: "Latest available version.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

// Configure configures the data source with provider data.
func (d *RepositoryPackagesDataSource) Configure(
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

func (d *RepositoryPackagesDataSource) Read(
		ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RepositoryPackagesDataSourceModel

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

	// Get limit
	limit := int64(100) // default
	if !data.Limit.IsNull() {
		limit = data.Limit.ValueInt64()
	}

	// Get packages from repository
	repository := data.Repository.ValueString()
	packages, err := d.getRepositoryBrewPackages(ctx, repository, limit)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to List Repository Packages",
			fmt.Sprintf("Failed to list packages from repository %s: %v", repository, err),
		)
		return
	}

	// Set computed values
	data.ID = types.StringValue(fmt.Sprintf("%s:repo:%s", managerName, repository))
	data.Manager = types.StringValue(managerName)

	// Convert packages to list
	packagesList, diags := types.ListValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"name":        types.StringType,
			"description": types.StringType,
			"version":     types.StringType,
		},
	}, packages)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Packages = packagesList

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *RepositoryPackagesDataSource) getRepositoryBrewPackages(ctx context.Context, repository string, limit int64) ([]RepositoryPackageInfo, error) {
	brewPath := d.providerData.Config.BrewPath.ValueString()
	if brewPath == "" {
		brewPath = "brew"
	}

	// Search for packages in the specific tap
	// Use search with tap prefix to find packages from that tap
	result, err := d.providerData.Executor.Run(ctx, brewPath, []string{"search", "--formula", repository + "/"}, executor.ExecOpts{
		Timeout: 60,
	})

	if err != nil || result.ExitCode != 0 {
		return nil, fmt.Errorf("failed to search repository packages: exit code %d, error: %w", result.ExitCode, err)
	}

	var packages []RepositoryPackageInfo
	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	count := int64(0)

	for _, line := range lines {
		if count >= limit {
			break
		}

		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "==>") {
			continue
		}

		// Parse package names (they might be in columns)
		names := strings.Fields(line)
		for _, name := range names {
			if count >= limit {
				break
			}

			name = strings.TrimSpace(name)
			if name != "" && strings.HasPrefix(name, repository+"/") {
				// Remove the tap prefix to get just the package name
				packageName := strings.TrimPrefix(name, repository+"/")

				packages = append(packages, RepositoryPackageInfo{
					Name:        types.StringValue(packageName),
					Description: types.StringValue(""), // Description requires individual brew info calls
					Version:     types.StringValue(""), // Version requires individual brew info calls
				})
				count++
			}
		}
	}

	return packages, nil
}
