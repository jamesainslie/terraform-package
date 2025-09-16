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

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/geico-private/terraform-provider-pkg/internal/executor"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &VersionHistoryDataSource{}

// NewVersionHistoryDataSource creates a new version history data source.
// NewVersionHistoryDataSource creates a new version history data source.
func NewVersionHistoryDataSource() datasource.DataSource {
	return &VersionHistoryDataSource{}
}

// VersionHistoryDataSource defines the data source implementation.
type VersionHistoryDataSource struct {
	providerData *ProviderData
}

// VersionHistoryDataSourceModel describes the data source data model.
type VersionHistoryDataSourceModel struct {
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Manager  types.String `tfsdk:"manager"`
	Versions types.List   `tfsdk:"versions"`
}

// Metadata returns the data source type name.
func (d *VersionHistoryDataSource) Metadata(
		ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_version_history"
}

// Schema defines the data source schema.
func (d *VersionHistoryDataSource) Schema(
		ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves available versions for a package.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Data source identifier.",
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Package name to get version history for.",
				Required:            true,
			},
			"manager": schema.StringAttribute{
				MarkdownDescription: "Package manager to query. " +
					"Valid values: 'auto', 'brew'. " +
					"Defaults to 'auto'.",
				Optional:            true,
			},
			"versions": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "List of available versions for the package.",
				Computed:            true,
			},
		},
	}
}

// Configure configures the data source with provider data.
func (d *VersionHistoryDataSource) Configure(
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

func (d *VersionHistoryDataSource) Read(
		ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data VersionHistoryDataSourceModel

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

	// Get version history
	packageName := data.Name.ValueString()
	versions := d.getBrewVersionHistory(ctx, packageName)

	// Set computed values
	data.ID = types.StringValue(fmt.Sprintf("%s:versions:%s", managerName, packageName))
	data.Manager = types.StringValue(managerName)

	// Convert versions to list
	versionsList, diags := types.ListValueFrom(ctx, types.StringType, versions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Versions = versionsList

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *VersionHistoryDataSource) getBrewVersionHistory(ctx context.Context, packageName string) []string {
	brewPath := d.providerData.Config.BrewPath.ValueString()
	if brewPath == "" {
		brewPath = "brew"
	}

	var versions []string

	// First, check for versioned formulas (e.g., python@3.9, python@3.10)
	result, err := d.providerData.Executor.Run(ctx, brewPath, []string{"search", packageName + "@"}, executor.ExecOpts{
		Timeout: 30,
	})

	if err == nil && result.ExitCode == 0 {
		lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.Contains(line, "==>") {
				continue
			}

			// Parse versioned package names
			names := strings.Fields(line)
			for _, name := range names {
				if strings.HasPrefix(name, packageName+"@") {
					version := strings.TrimPrefix(name, packageName+"@")
					versions = append(versions, version)
				}
			}
		}
	}

	// Also get the current stable version
	result, err = d.providerData.Executor.Run(ctx, brewPath, []string{"info", packageName}, executor.ExecOpts{
		Timeout: 30,
	})

	if err == nil && result.ExitCode == 0 {
		lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
		for _, line := range lines {
			if strings.Contains(line, ": stable ") {
				// Parse "package: stable X.Y.Z" format
				parts := strings.Split(line, ": stable ")
				if len(parts) == 2 {
					stableVersion := strings.Fields(parts[1])[0]
					// Add stable version if not already in list
					found := false
					for _, v := range versions {
						if v == stableVersion {
							found = true
							break
						}
					}
					if !found {
						versions = append([]string{stableVersion}, versions...)
					}
				}
				break
			}
		}
	}

	// If no versions found, return empty list
	if len(versions) == 0 {
		return []string{}
	}

	return versions
}
