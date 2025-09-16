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

	"github.com/geico-private/terraform-provider-pkg/internal/adapters/brew"
	"github.com/geico-private/terraform-provider-pkg/internal/executor"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &InstalledPackagesDataSource{}

func NewInstalledPackagesDataSource() datasource.DataSource {
	return &InstalledPackagesDataSource{}
}

// InstalledPackagesDataSource defines the data source implementation.
type InstalledPackagesDataSource struct {
	providerData *ProviderData
}

// InstalledPackagesDataSourceModel describes the data source data model.
type InstalledPackagesDataSourceModel struct {
	ID       types.String `tfsdk:"id"`
	Manager  types.String `tfsdk:"manager"`
	Filter   types.String `tfsdk:"filter"`
	Packages types.List   `tfsdk:"packages"`
}

// InstalledPackageInfo represents information about an installed package.
type InstalledPackageInfo struct {
	Name       types.String `tfsdk:"name"`
	Version    types.String `tfsdk:"version"`
	Pinned     types.Bool   `tfsdk:"pinned"`
	Repository types.String `tfsdk:"repository"`
}

func (d *InstalledPackagesDataSource) Metadata(
		ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_installed_packages"
}

func (d *InstalledPackagesDataSource) Schema(
		ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all packages installed by the package manager.",

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
			"filter": schema.StringAttribute{
				MarkdownDescription: "Optional filter pattern to match package names (supports glob patterns).",
				Optional:            true,
			},
			"packages": schema.ListNestedAttribute{
				MarkdownDescription: "List of installed packages.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "Package name.",
							Computed:            true,
						},
						"version": schema.StringAttribute{
							MarkdownDescription: "Installed version.",
							Computed:            true,
						},
						"pinned": schema.BoolAttribute{
							MarkdownDescription: "Whether the package is pinned.",
							Computed:            true,
						},
						"repository": schema.StringAttribute{
							MarkdownDescription: "Repository or tap that provided this package.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *InstalledPackagesDataSource) Configure(
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

func (d *InstalledPackagesDataSource) Read(
		ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data InstalledPackagesDataSourceModel

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

	// Get installed packages
	packages, err := d.getInstalledBrewPackages(ctx, data.Filter.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to List Installed Packages",
			fmt.Sprintf("Failed to list installed packages: %v", err),
		)
		return
	}

	// Set computed values
	data.ID = types.StringValue(fmt.Sprintf("%s:installed", managerName))
	data.Manager = types.StringValue(managerName)

	// Convert packages to list
	packagesList, diags := types.ListValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"name":       types.StringType,
			"version":    types.StringType,
			"pinned":     types.BoolType,
			"repository": types.StringType,
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

func (d *InstalledPackagesDataSource) getInstalledBrewPackages(
		ctx context.Context, filter string) ([]InstalledPackageInfo, error) {
	brewPath := d.providerData.Config.BrewPath.ValueString()
	if brewPath == "" {
		brewPath = "brew"
	}

	// Get list of installed packages
	args := []string{"list", "--versions"}
	if filter != "" {
		args = append(args, filter)
	}

	result, err := d.providerData.Executor.Run(ctx, brewPath, args, executor.ExecOpts{
		Timeout: 60,
	})

	if err != nil || result.ExitCode != 0 {
		return nil, fmt.Errorf("failed to list installed packages: exit code %d, error: %w", result.ExitCode, err)
	}

	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	packages := make([]InstalledPackageInfo, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse line format: "package_name version1 version2..."
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		packageName := parts[0]
		version := parts[1] // Use first version

		// Get additional info for this package (pinned status, repository)
		manager := brew.NewBrewAdapter(d.providerData.Executor, brewPath)
		info, err := manager.DetectInstalled(ctx, packageName)
		if err != nil {
			// If we can't get detailed info, use basic info
			packages = append(packages, InstalledPackageInfo{
				Name:       types.StringValue(packageName),
				Version:    types.StringValue(version),
				Pinned:     types.BoolValue(false),
				Repository: types.StringValue(""),
			})
			continue
		}

		packages = append(packages, InstalledPackageInfo{
			Name:       types.StringValue(packageName),
			Version:    types.StringValue(version),
			Pinned:     types.BoolValue(info.Pinned),
			Repository: types.StringValue(info.Repository),
		})
	}

	return packages, nil
}
