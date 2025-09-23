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
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"

	"github.com/jamesainslie/terraform-package/internal/services"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ServiceStatusDataSource{}

// NewServiceStatusDataSource creates a new service status data source.
func NewServiceStatusDataSource() datasource.DataSource {
	return &ServiceStatusDataSource{}
}

// ServiceStatusDataSource defines the data source implementation.
type ServiceStatusDataSource struct {
	providerData *ProviderData
	detector     services.ServiceDetector
}

// ServiceStatusDataSourceModel describes the data source data model.
type ServiceStatusDataSourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	RequiredPackage types.String `tfsdk:"required_package"`
	PackageManager  types.String `tfsdk:"package_manager"`
	Timeout         types.String `tfsdk:"timeout"`
	Running         types.Bool   `tfsdk:"running"`
	Healthy         types.Bool   `tfsdk:"healthy"`
	Version         types.String `tfsdk:"version"`
	ProcessID       types.String `tfsdk:"process_id"`
	StartTime       types.String `tfsdk:"start_time"`
	ManagerType     types.String `tfsdk:"manager_type"`
	Ports           types.List   `tfsdk:"ports"`
	HealthCheck     types.Object `tfsdk:"health_check"`
}

// HealthCheckModel describes the health check configuration model.
type HealthCheckModel struct {
	Command        types.String `tfsdk:"command"`
	HTTPEndpoint   types.String `tfsdk:"http_endpoint"`
	ExpectedStatus types.Int64  `tfsdk:"expected_status"`
	Timeout        types.String `tfsdk:"timeout"`
}

// Metadata returns the data source type name.
func (d *ServiceStatusDataSource) Metadata(
	_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_status"
}

// Schema defines the data source schema.
func (d *ServiceStatusDataSource) Schema(
	_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Checks the status of a system service and performs health checks.",

		Attributes: map[string]schema.Attribute{
			// Required
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Service name to check.",
			},

			// Optional configuration
			"required_package": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Package that provides this service (validates installation).",
			},
			"package_manager": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "Package manager to check for service package. " +
					"Valid values: 'brew', 'apt', 'winget', 'choco'. " +
					"Defaults to 'brew' on macOS, 'apt' on Linux, 'winget' on Windows.",
				Validators: []validator.String{
					stringvalidator.OneOf("brew", "apt", "winget", "choco"),
				},
			},
			"timeout": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Timeout for service status checks (e.g., '10s', '1m'). Defaults to '10s'.",
			},

			// Computed outputs
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Data source identifier.",
			},
			"running": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the service is currently running.",
			},
			"healthy": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the service passes health checks.",
			},
			"version": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Service version (if detectable).",
			},
			"process_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Process ID of the running service.",
			},
			"start_time": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "When the service was started (RFC3339 format).",
			},
			"manager_type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Service manager type (launchd, systemd, brew, windows, process).",
			},
			"ports": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.Int64Type,
				MarkdownDescription: "Ports the service is listening on.",
			},
		},
		Blocks: map[string]schema.Block{
			"health_check": schema.SingleNestedBlock{
				MarkdownDescription: "Custom health check configuration.",
				Attributes: map[string]schema.Attribute{
					"command": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Custom health check command.",
					},
					"http_endpoint": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "HTTP endpoint for health checks.",
					},
					"expected_status": schema.Int64Attribute{
						Optional:            true,
						MarkdownDescription: "Expected HTTP status code. Defaults to 200.",
					},
					"timeout": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Health check timeout (e.g., '5s', '10s'). Defaults to '5s'.",
					},
				},
			},
		},
	}
}

// Configure configures the data source with provider data.
func (d *ServiceStatusDataSource) Configure(
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

	// Initialize service detector
	d.detector = services.NewServiceDetector(providerData.Executor)
}

// Read reads the service status and performs health checks.
func (d *ServiceStatusDataSource) Read(
	ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ServiceStatusDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceName := data.Name.ValueString()

	// Parse timeout
	timeoutDuration := 10 * time.Second // default
	if !data.Timeout.IsNull() {
		var err error
		timeoutDuration, err = time.ParseDuration(data.Timeout.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid Timeout",
				fmt.Sprintf("Failed to parse timeout %s: %v", data.Timeout.ValueString(), err),
			)
			return
		}
	}

	// Create context with timeout
	ctxTimeout, cancel := context.WithTimeout(ctx, timeoutDuration)
	defer cancel()

	// Get service information
	serviceInfo, err := d.detector.GetServiceInfo(ctxTimeout, serviceName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to Get Service Information",
			fmt.Sprintf("Failed to get information for service %s: %v", serviceName, err),
		)
		return
	}

	// Validate required package if specified
	if !data.RequiredPackage.IsNull() {
		requiredPackage := data.RequiredPackage.ValueString()

		// Check if the service's package matches the required package
		if serviceInfo.Package == nil || serviceInfo.Package.Name != requiredPackage {
			resp.Diagnostics.AddWarning(
				"Required Package Mismatch",
				fmt.Sprintf("Service %s is not provided by required package %s", serviceName, requiredPackage),
			)
		}
	}

	// Perform custom health check if configured
	var healthCheckConfig *services.HealthCheckConfig
	if !data.HealthCheck.IsNull() {
		var healthCheckData HealthCheckModel
		diags := data.HealthCheck.As(ctx, &healthCheckData, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		healthCheckConfig = &services.HealthCheckConfig{
			Timeout: 5 * time.Second, // default
		}

		if !healthCheckData.Command.IsNull() {
			healthCheckConfig.Command = healthCheckData.Command.ValueString()
		}
		if !healthCheckData.HTTPEndpoint.IsNull() {
			healthCheckConfig.HTTPEndpoint = healthCheckData.HTTPEndpoint.ValueString()
		}
		if !healthCheckData.ExpectedStatus.IsNull() {
			healthCheckConfig.ExpectedStatus = int(healthCheckData.ExpectedStatus.ValueInt64())
		} else if healthCheckConfig.HTTPEndpoint != "" {
			healthCheckConfig.ExpectedStatus = 200 // default for HTTP checks
		}
		if !healthCheckData.Timeout.IsNull() {
			if timeout, err := time.ParseDuration(healthCheckData.Timeout.ValueString()); err == nil {
				healthCheckConfig.Timeout = timeout
			}
		}

		// Perform custom health check
		if healthResult, err := d.detector.CheckHealth(ctxTimeout, serviceName, healthCheckConfig); err == nil {
			serviceInfo.Healthy = healthResult.Healthy
		}
	}

	// Set computed values
	data.ID = types.StringValue(fmt.Sprintf("service:%s", serviceName))
	data.Running = types.BoolValue(serviceInfo.Running)
	data.Healthy = types.BoolValue(serviceInfo.Healthy)
	data.ManagerType = types.StringValue(serviceInfo.ManagerType)

	// Set version
	if serviceInfo.Version != "" {
		data.Version = types.StringValue(serviceInfo.Version)
	} else {
		data.Version = types.StringValue("")
	}

	// Set process ID
	if serviceInfo.ProcessID != "" {
		data.ProcessID = types.StringValue(serviceInfo.ProcessID)
	} else {
		data.ProcessID = types.StringValue("")
	}

	// Set start time
	if serviceInfo.StartTime != nil {
		data.StartTime = types.StringValue(serviceInfo.StartTime.Format(time.RFC3339))
	} else {
		data.StartTime = types.StringValue("")
	}

	// Set ports
	if len(serviceInfo.Ports) > 0 {
		ports := make([]types.Int64, len(serviceInfo.Ports))
		for i, port := range serviceInfo.Ports {
			ports[i] = types.Int64Value(int64(port))
		}
		portsList, diags := types.ListValueFrom(ctx, types.Int64Type, ports)
		resp.Diagnostics.Append(diags...)
		if !resp.Diagnostics.HasError() {
			data.Ports = portsList
		}
	} else {
		// Empty list if no ports
		emptyList, diags := types.ListValueFrom(ctx, types.Int64Type, []types.Int64{})
		resp.Diagnostics.Append(diags...)
		if !resp.Diagnostics.HasError() {
			data.Ports = emptyList
		}
	}

	// Set default values for optional fields if not provided
	if data.PackageManager.IsNull() {
		// Auto-detect package manager based on OS
		switch d.providerData.DetectedOS {
		case "darwin":
			data.PackageManager = types.StringValue("brew")
		case "linux":
			data.PackageManager = types.StringValue("apt")
		case "windows":
			data.PackageManager = types.StringValue("winget")
		default:
			data.PackageManager = types.StringValue("auto")
		}
	}

	if data.Timeout.IsNull() {
		data.Timeout = types.StringValue("10s")
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
