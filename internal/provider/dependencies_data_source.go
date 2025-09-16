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
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/geico-private/terraform-provider-pkg/internal/executor"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &DependenciesDataSource{}

func NewDependenciesDataSource() datasource.DataSource {
	return &DependenciesDataSource{}
}

// DependenciesDataSource defines the data source implementation.
type DependenciesDataSource struct {
	providerData *ProviderData
}

// DependenciesDataSourceModel describes the data source data model.
type DependenciesDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Manager      types.String `tfsdk:"manager"`
	Type         types.String `tfsdk:"type"`
	Dependencies types.List   `tfsdk:"dependencies"`
}

// DependencyInfo represents information about a package dependency.
type DependencyInfo struct {
	Name     types.String `tfsdk:"name"`
	Version  types.String `tfsdk:"version"`
	Type     types.String `tfsdk:"type"`
	Optional types.Bool   `tfsdk:"optional"`
}

func (d *DependenciesDataSource) Metadata(
		ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dependencies"
}

func (d *DependenciesDataSource) Schema(
		ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves dependency information for a package.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Data source identifier.",
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Package name to get dependencies for.",
				Required:            true,
			},
			"manager": schema.StringAttribute{
				MarkdownDescription: "Package manager to query. " +
					"Valid values: 'auto', 'brew'. " +
					"Defaults to 'auto'.",
				Optional:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Type of dependencies to retrieve. " +
					"Valid values: 'runtime', 'build', " +
					"'optional', 'all'. Defaults to 'runtime'.",
				Optional: true,
			},
			"dependencies": schema.ListNestedAttribute{
				MarkdownDescription: "List of package dependencies.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "Dependency package name.",
							Computed:            true,
						},
						"version": schema.StringAttribute{
							MarkdownDescription: "Required version or version constraint.",
							Computed:            true,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "Dependency type (runtime, build, optional).",
							Computed:            true,
						},
						"optional": schema.BoolAttribute{
							MarkdownDescription: "Whether this dependency is optional.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *DependenciesDataSource) Configure(
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

func (d *DependenciesDataSource) Read(
		ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DependenciesDataSourceModel

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

	// Get dependency type
	depType := "runtime"
	if !data.Type.IsNull() {
		depType = data.Type.ValueString()
	}

	// Get dependencies
	packageName := data.Name.ValueString()
	dependencies, err := d.getBrewDependencies(ctx, packageName, depType)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to Get Dependencies",
			fmt.Sprintf("Failed to get dependencies for package %s: %v", packageName, err),
		)
		return
	}

	// Set computed values
	data.ID = types.StringValue(fmt.Sprintf("%s:deps:%s:%s", managerName, packageName, depType))
	data.Manager = types.StringValue(managerName)
	data.Type = types.StringValue(depType)

	// Convert dependencies to list
	depsList, diags := types.ListValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"name":     types.StringType,
			"version":  types.StringType,
			"type":     types.StringType,
			"optional": types.BoolType,
		},
	}, dependencies)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Dependencies = depsList

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *DependenciesDataSource) getBrewDependencies(ctx context.Context, packageName, depType string) ([]DependencyInfo, error) {
	brewPath := d.providerData.Config.BrewPath.ValueString()
	if brewPath == "" {
		brewPath = "brew"
	}

	// Get detailed package information with dependencies
	result, err := d.providerData.Executor.Run(ctx, brewPath, []string{"info", "--json", packageName}, executor.ExecOpts{
		Timeout: 30,
	})

	if err != nil || result.ExitCode != 0 {
		return nil, fmt.Errorf("failed to get package info: exit code %d, error: %w", result.ExitCode, err)
	}

	// Parse JSON response
	var infos []map[string]interface{}
	if err := json.Unmarshal([]byte(result.Stdout), &infos); err != nil {
		return nil, fmt.Errorf("failed to parse brew info JSON: %w", err)
	}

	if len(infos) == 0 {
		return []DependencyInfo{}, nil
	}

	info := infos[0]
	var dependencies []DependencyInfo

	// Extract dependencies based on type
	switch depType {
	case "runtime", "all":
		if deps, ok := info["dependencies"].([]interface{}); ok {
			for _, dep := range deps {
				if depStr, ok := dep.(string); ok {
					dependencies = append(dependencies, DependencyInfo{
						Name:     types.StringValue(depStr),
						Version:  types.StringValue(""), // Brew doesn't specify version constraints
						Type:     types.StringValue("runtime"),
						Optional: types.BoolValue(false),
					})
				}
			}
		}

	case "build":
		if deps, ok := info["build_dependencies"].([]interface{}); ok {
			for _, dep := range deps {
				if depStr, ok := dep.(string); ok {
					dependencies = append(dependencies, DependencyInfo{
						Name:     types.StringValue(depStr),
						Version:  types.StringValue(""),
						Type:     types.StringValue("build"),
						Optional: types.BoolValue(false),
					})
				}
			}
		}

	case "optional":
		if deps, ok := info["optional_dependencies"].([]interface{}); ok {
			for _, dep := range deps {
				if depStr, ok := dep.(string); ok {
					dependencies = append(dependencies, DependencyInfo{
						Name:     types.StringValue(depStr),
						Version:  types.StringValue(""),
						Type:     types.StringValue("optional"),
						Optional: types.BoolValue(true),
					})
				}
			}
		}
	}

	return dependencies, nil
}
