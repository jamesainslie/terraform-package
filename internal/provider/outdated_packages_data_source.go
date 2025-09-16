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
var _ datasource.DataSource = &OutdatedPackagesDataSource{}

func NewOutdatedPackagesDataSource() datasource.DataSource {
	return &OutdatedPackagesDataSource{}
}

// OutdatedPackagesDataSource defines the data source implementation.
type OutdatedPackagesDataSource struct {
	providerData *ProviderData
}

// OutdatedPackagesDataSourceModel describes the data source data model.
type OutdatedPackagesDataSourceModel struct {
	ID       types.String `tfsdk:"id"`
	Manager  types.String `tfsdk:"manager"`
	Packages types.List   `tfsdk:"packages"`
}

// OutdatedPackageInfo represents information about an outdated package.
type OutdatedPackageInfo struct {
	Name           types.String `tfsdk:"name"`
	CurrentVersion types.String `tfsdk:"current_version"`
	LatestVersion  types.String `tfsdk:"latest_version"`
	Pinned         types.Bool   `tfsdk:"pinned"`
}

func (d *OutdatedPackagesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_outdated_packages"
}

func (d *OutdatedPackagesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists packages that have available updates.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Data source identifier.",
			},
			"manager": schema.StringAttribute{
				MarkdownDescription: "Package manager to query. Valid values: 'auto', 'brew'. Defaults to 'auto'.",
				Optional:            true,
			},
			"packages": schema.ListNestedAttribute{
				MarkdownDescription: "List of packages with available updates.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "Package name.",
							Computed:            true,
						},
						"current_version": schema.StringAttribute{
							MarkdownDescription: "Currently installed version.",
							Computed:            true,
						},
						"latest_version": schema.StringAttribute{
							MarkdownDescription: "Latest available version.",
							Computed:            true,
						},
						"pinned": schema.BoolAttribute{
							MarkdownDescription: "Whether the package is pinned (preventing updates).",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *OutdatedPackagesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *OutdatedPackagesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data OutdatedPackagesDataSourceModel

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

	// Get outdated packages
	packages, err := d.getOutdatedBrewPackages(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to List Outdated Packages",
			fmt.Sprintf("Failed to list outdated packages: %v", err),
		)
		return
	}

	// Set computed values
	data.ID = types.StringValue(fmt.Sprintf("%s:outdated", managerName))
	data.Manager = types.StringValue(managerName)

	// Convert packages to list
	packagesList, diags := types.ListValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"name":            types.StringType,
			"current_version": types.StringType,
			"latest_version":  types.StringType,
			"pinned":          types.BoolType,
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

func (d *OutdatedPackagesDataSource) getOutdatedBrewPackages(ctx context.Context) ([]OutdatedPackageInfo, error) {
	brewPath := d.providerData.Config.BrewPath.ValueString()
	if brewPath == "" {
		brewPath = "brew"
	}

	// Get list of outdated packages
	result, err := d.providerData.Executor.Run(ctx, brewPath, []string{"outdated", "--verbose"}, executor.ExecOpts{
		Timeout: 120,
	})

	if err != nil || result.ExitCode != 0 {
		return nil, fmt.Errorf("failed to list outdated packages: exit code %d, error: %v", result.ExitCode, err)
	}

	var packages []OutdatedPackageInfo
	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse line format: "package_name (current_version) < latest_version"
		// Example: "git (2.41.0) < 2.42.0"
		if strings.Contains(line, " < ") {
			parts := strings.Split(line, " < ")
			if len(parts) != 2 {
				continue
			}

			leftPart := strings.TrimSpace(parts[0])
			latestVersion := strings.TrimSpace(parts[1])

			// Parse package name and current version from left part
			// Format: "package_name (current_version)"
			if strings.Contains(leftPart, " (") && strings.HasSuffix(leftPart, ")") {
				nameEnd := strings.LastIndex(leftPart, " (")
				packageName := leftPart[:nameEnd]
				currentVersion := strings.TrimSuffix(leftPart[nameEnd+2:], ")")

				packages = append(packages, OutdatedPackageInfo{
					Name:           types.StringValue(packageName),
					CurrentVersion: types.StringValue(currentVersion),
					LatestVersion:  types.StringValue(latestVersion),
					Pinned:         types.BoolValue(false), // Pin status requires individual package queries
				})
			}
		}
	}

	return packages, nil
}
