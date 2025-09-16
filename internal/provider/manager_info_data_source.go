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
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/geico-private/terraform-provider-pkg/internal/adapters/brew"
	"github.com/geico-private/terraform-provider-pkg/internal/executor"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ManagerInfoDataSource{}

// NewManagerInfoDataSource creates a new manager info data source.
func NewManagerInfoDataSource() datasource.DataSource {
	return &ManagerInfoDataSource{}
}

// ManagerInfoDataSource defines the data source implementation.
type ManagerInfoDataSource struct {
	providerData *ProviderData
}

// ManagerInfoDataSourceModel describes the data source data model.
type ManagerInfoDataSourceModel struct {
	ID              types.String `tfsdk:"id"`
	Manager         types.String `tfsdk:"manager"`
	DetectedManager types.String `tfsdk:"detected_manager"`
	Available       types.Bool   `tfsdk:"available"`
	Version         types.String `tfsdk:"version"`
	Path            types.String `tfsdk:"path"`
	Platform        types.String `tfsdk:"platform"`
}

// Metadata returns the data source type name.
// Metadata returns the data source type name.
func (d *ManagerInfoDataSource) Metadata(
	_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_manager_info"
}

// Schema defines the data source schema.
// Schema defines the data source schema.
func (d *ManagerInfoDataSource) Schema(
	_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves information about package manager availability and configuration.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Data source identifier.",
			},
			"manager": schema.StringAttribute{
				MarkdownDescription: "Package manager to query. " +
					"Valid values: 'auto', 'brew'. " +
					"Defaults to 'auto'.",
				Optional: true,
			},
			"detected_manager": schema.StringAttribute{
				MarkdownDescription: "The actual package manager that was detected or specified.",
				Computed:            true,
			},
			"available": schema.BoolAttribute{
				MarkdownDescription: "Whether the package manager is available on this system.",
				Computed:            true,
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "Version of the package manager. " +
					"Empty if not available.",
				Computed: true,
			},
			"path": schema.StringAttribute{
				MarkdownDescription: "Path to the package manager binary. " +
					"Empty if not available.",
				Computed: true,
			},
			"platform": schema.StringAttribute{
				MarkdownDescription: "Current operating system platform.",
				Computed:            true,
			},
		},
	}
}

// Configure configures the data source with provider data.
// Configure configures the data source with provider data.
func (d *ManagerInfoDataSource) Configure(
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

func (d *ManagerInfoDataSource) Read(
	ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ManagerInfoDataSourceModel

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
	detectedManager := managerName
	if managerName == "auto" {
		if runtime.GOOS != "darwin" {
			resp.Diagnostics.AddError(
				"Unsupported Operating System",
				fmt.Sprintf("Only macOS (darwin) is supported in Phase 2, got: %s", runtime.GOOS),
			)
			return
		}
		detectedManager = "brew"
	}

	// Only support brew in Phase 2
	if detectedManager != "brew" {
		resp.Diagnostics.AddError(
			"Unsupported Package Manager",
			fmt.Sprintf("Only 'brew' manager is supported in Phase 2, got: %s", detectedManager),
		)
		return
	}

	// Set basic computed values
	data.ID = types.StringValue(fmt.Sprintf("manager:%s", detectedManager))
	data.Manager = types.StringValue(managerName)
	data.DetectedManager = types.StringValue(detectedManager)
	data.Platform = types.StringValue(runtime.GOOS)

	// Get manager-specific information
	switch detectedManager {
	case "brew":
		d.getBrewInfo(ctx, &data, resp)
	default:
		resp.Diagnostics.AddError(
			"Unsupported Manager",
			fmt.Sprintf("Manager %s is not implemented", detectedManager),
		)
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *ManagerInfoDataSource) getBrewInfo(
	ctx context.Context, data *ManagerInfoDataSourceModel, _ *datasource.ReadResponse) {
	brewPath := d.providerData.Config.BrewPath.ValueString()
	manager := brew.NewBrewAdapter(d.providerData.Executor, brewPath)

	// Check availability
	available := manager.IsAvailable(ctx)
	data.Available = types.BoolValue(available)

	if !available {
		data.Version = types.StringValue("")
		data.Path = types.StringValue("")
		return
	}

	// Get version information
	if brewPath == "" {
		brewPath = "brew"
	}
	data.Path = types.StringValue(brewPath)

	// Get brew version
	result, err := d.providerData.Executor.Run(ctx, brewPath, []string{"--version"}, executor.ExecOpts{
		Timeout: 10 * time.Second,
	})

	if err != nil || result.ExitCode != 0 {
		data.Version = types.StringValue("unknown")
		return
	}

	// Parse version from output (first line usually contains version)
	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	if len(lines) > 0 {
		versionLine := strings.TrimSpace(lines[0])
		// Extract version from "Homebrew X.Y.Z" format
		if strings.HasPrefix(versionLine, "Homebrew ") {
			version := strings.TrimPrefix(versionLine, "Homebrew ")
			data.Version = types.StringValue(version)
		} else {
			data.Version = types.StringValue(versionLine)
		}
	} else {
		data.Version = types.StringValue("unknown")
	}
}
