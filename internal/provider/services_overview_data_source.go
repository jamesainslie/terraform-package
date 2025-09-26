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

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jamesainslie/terraform-package/internal/services"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ServicesOverviewDataSource{}

// NewServicesOverviewDataSource creates a new services overview data source.
func NewServicesOverviewDataSource() datasource.DataSource {
	return &ServicesOverviewDataSource{}
}

// ServicesOverviewDataSource defines the data source implementation.
type ServicesOverviewDataSource struct {
	providerData *ProviderData
	detector     services.ServiceDetector
}

// ServicesOverviewDataSourceModel describes the data source data model.
type ServicesOverviewDataSourceModel struct {
	ID                    types.String `tfsdk:"id"`
	Services              types.List   `tfsdk:"services"`
	PackageManager        types.String `tfsdk:"package_manager"`
	InstalledPackagesOnly types.Bool   `tfsdk:"installed_packages_only"`
	IncludeMetrics        types.Bool   `tfsdk:"include_metrics"`
	HealthChecks          types.Bool   `tfsdk:"health_checks"`
	Timeout               types.String `tfsdk:"timeout"`

	// Computed results
	ServicesData    types.Map    `tfsdk:"services_data"`
	TotalServices   types.Int64  `tfsdk:"total_services"`
	RunningServices types.Int64  `tfsdk:"running_services"`
	HealthyServices types.Int64  `tfsdk:"healthy_services"`
	Summary         types.Object `tfsdk:"summary"`
}

// ServiceOverviewModel describes individual service information in the overview.
type ServiceOverviewModel struct {
	Name        types.String `tfsdk:"name"`
	Running     types.Bool   `tfsdk:"running"`
	Healthy     types.Bool   `tfsdk:"healthy"`
	Version     types.String `tfsdk:"version"`
	ProcessID   types.String `tfsdk:"process_id"`
	StartTime   types.String `tfsdk:"start_time"`
	ManagerType types.String `tfsdk:"manager_type"`
	Ports       types.List   `tfsdk:"ports"`
	Package     types.Object `tfsdk:"package"`
}

// ServicePackageModel describes package information for a service.
type ServicePackageModel struct {
	Name    types.String `tfsdk:"name"`
	Manager types.String `tfsdk:"manager"`
	Version types.String `tfsdk:"version"`
}

// ServicesSummaryModel describes the summary of services status.
type ServicesSummaryModel struct {
	TotalServices       types.Int64 `tfsdk:"total_services"`
	RunningServices     types.Int64 `tfsdk:"running_services"`
	HealthyServices     types.Int64 `tfsdk:"healthy_services"`
	StoppedServices     types.Int64 `tfsdk:"stopped_services"`
	UnhealthyServices   types.Int64 `tfsdk:"unhealthy_services"`
	RunningServiceNames types.List  `tfsdk:"running_service_names"`
	StoppedServiceNames types.List  `tfsdk:"stopped_service_names"`
}

// Metadata returns the data source type name.
func (d *ServicesOverviewDataSource) Metadata(
	_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_services_overview"
}

// Schema defines the data source schema.
func (d *ServicesOverviewDataSource) Schema(
	_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves status information for multiple system services at once.",

		Attributes: map[string]schema.Attribute{
			// Required
			"services": schema.ListAttribute{
				Required:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of service names to check.",
			},

			// Optional configuration
			"package_manager": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "Package manager to check for service packages. " +
					"Valid values: 'brew', 'apt', 'winget', 'choco'. " +
					"Defaults to auto-detection based on OS.",
				Validators: []validator.String{
					stringvalidator.OneOf("brew", "apt", "winget", "choco"),
				},
			},
			"installed_packages_only": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Only check services from installed packages. Defaults to false.",
			},
			"include_metrics": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Include performance metrics in the output. Defaults to false.",
			},
			"health_checks": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Perform health checks on running services. Defaults to true.",
			},
			"timeout": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Timeout for service status checks (e.g., '10s', '1m'). Defaults to '30s'.",
			},

			// Computed outputs
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Data source identifier.",
			},
			"services_data": schema.MapAttribute{
				Computed: true,
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"name":         types.StringType,
						"running":      types.BoolType,
						"healthy":      types.BoolType,
						"version":      types.StringType,
						"process_id":   types.StringType,
						"start_time":   types.StringType,
						"manager_type": types.StringType,
						"ports":        types.ListType{ElemType: types.Int64Type},
						"package": types.ObjectType{
							AttrTypes: map[string]attr.Type{
								"name":    types.StringType,
								"manager": types.StringType,
								"version": types.StringType,
							},
						},
					},
				},
				MarkdownDescription: "Map of service names to their detailed information.",
			},
			"total_services": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Total number of services checked.",
			},
			"running_services": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Number of services currently running.",
			},
			"healthy_services": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Number of services that are running and healthy.",
			},
			"summary": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Summary of services status.",
				Attributes: map[string]schema.Attribute{
					"total_services": schema.Int64Attribute{
						Computed:            true,
						MarkdownDescription: "Total number of services checked.",
					},
					"running_services": schema.Int64Attribute{
						Computed:            true,
						MarkdownDescription: "Number of services currently running.",
					},
					"healthy_services": schema.Int64Attribute{
						Computed:            true,
						MarkdownDescription: "Number of services that are running and healthy.",
					},
					"stopped_services": schema.Int64Attribute{
						Computed:            true,
						MarkdownDescription: "Number of services that are stopped.",
					},
					"unhealthy_services": schema.Int64Attribute{
						Computed:            true,
						MarkdownDescription: "Number of services that are running but unhealthy.",
					},
					"running_service_names": schema.ListAttribute{
						Computed:            true,
						ElementType:         types.StringType,
						MarkdownDescription: "Names of services that are currently running.",
					},
					"stopped_service_names": schema.ListAttribute{
						Computed:            true,
						ElementType:         types.StringType,
						MarkdownDescription: "Names of services that are stopped.",
					},
				},
			},
		},
	}
}

// Configure configures the data source with provider data.
func (d *ServicesOverviewDataSource) Configure(
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

// Read reads the services overview and performs health checks.
func (d *ServicesOverviewDataSource) Read(
	ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ServicesOverviewDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract service names
	var serviceNames []string
	resp.Diagnostics.Append(data.Services.ElementsAs(ctx, &serviceNames, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse timeout
	timeoutDuration := 30 * time.Second // default for bulk operations
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

	// Check health checks option
	performHealthChecks := true
	if !data.HealthChecks.IsNull() {
		performHealthChecks = data.HealthChecks.ValueBool()
	}

	// Collect service information
	servicesData := make(map[string]types.Object)
	var runningCount, healthyCount int64
	var runningServiceNames, stoppedServiceNames []string

	for _, serviceName := range serviceNames {
		serviceInfo, err := d.detector.GetServiceInfo(ctxTimeout, serviceName)
		if err != nil {
			// Log warning but continue with other services
			resp.Diagnostics.AddWarning(
				"Failed to Get Service Information",
				fmt.Sprintf("Failed to get information for service %s: %v", serviceName, err),
			)
			continue
		}

		// Perform health check if requested and service is running
		if performHealthChecks && serviceInfo.Running {
			if healthResult, err := d.detector.CheckHealth(ctxTimeout, serviceName, nil); err == nil {
				serviceInfo.Healthy = healthResult.Healthy
			}
		}

		// Convert to Terraform types
		serviceObj := d.convertServiceInfoToObject(ctx, serviceInfo, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}

		servicesData[serviceName] = serviceObj

		// Update counters
		if serviceInfo.Running {
			runningCount++
			runningServiceNames = append(runningServiceNames, serviceName)
			if serviceInfo.Healthy {
				healthyCount++
			}
		} else {
			stoppedServiceNames = append(stoppedServiceNames, serviceName)
		}
	}

	// Convert services data to map
	servicesMap, diags := types.MapValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"name":         types.StringType,
			"running":      types.BoolType,
			"healthy":      types.BoolType,
			"version":      types.StringType,
			"process_id":   types.StringType,
			"start_time":   types.StringType,
			"manager_type": types.StringType,
			"ports":        types.ListType{ElemType: types.Int64Type},
			"package": types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"name":    types.StringType,
					"manager": types.StringType,
					"version": types.StringType,
				},
			},
		},
	}, servicesData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create summary
	totalServices := int64(len(serviceNames))
	stoppedCount := totalServices - runningCount
	unhealthyCount := runningCount - healthyCount

	// Convert service name slices to types.List
	runningNamesList, diags := types.ListValueFrom(ctx, types.StringType, runningServiceNames)
	resp.Diagnostics.Append(diags...)
	stoppedNamesList, diags := types.ListValueFrom(ctx, types.StringType, stoppedServiceNames)
	resp.Diagnostics.Append(diags...)

	summary, diags := types.ObjectValueFrom(ctx, map[string]attr.Type{
		"total_services":        types.Int64Type,
		"running_services":      types.Int64Type,
		"healthy_services":      types.Int64Type,
		"stopped_services":      types.Int64Type,
		"unhealthy_services":    types.Int64Type,
		"running_service_names": types.ListType{ElemType: types.StringType},
		"stopped_service_names": types.ListType{ElemType: types.StringType},
	}, ServicesSummaryModel{
		TotalServices:       types.Int64Value(totalServices),
		RunningServices:     types.Int64Value(runningCount),
		HealthyServices:     types.Int64Value(healthyCount),
		StoppedServices:     types.Int64Value(stoppedCount),
		UnhealthyServices:   types.Int64Value(unhealthyCount),
		RunningServiceNames: runningNamesList,
		StoppedServiceNames: stoppedNamesList,
	})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set computed values
	data.ID = types.StringValue(fmt.Sprintf("services_overview:%d", len(serviceNames)))
	data.ServicesData = servicesMap
	data.TotalServices = types.Int64Value(totalServices)
	data.RunningServices = types.Int64Value(runningCount)
	data.HealthyServices = types.Int64Value(healthyCount)
	data.Summary = summary

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

	if data.InstalledPackagesOnly.IsNull() {
		data.InstalledPackagesOnly = types.BoolValue(false)
	}

	if data.IncludeMetrics.IsNull() {
		data.IncludeMetrics = types.BoolValue(false)
	}

	if data.HealthChecks.IsNull() {
		data.HealthChecks = types.BoolValue(true)
	}

	if data.Timeout.IsNull() {
		data.Timeout = types.StringValue("30s")
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// convertServiceInfoToObject converts ServiceInfo to Terraform object
func (d *ServicesOverviewDataSource) convertServiceInfoToObject(ctx context.Context, serviceInfo *services.ServiceInfo, diagnostics *diag.Diagnostics) types.Object {
	// Convert ports
	var portsList types.List
	if len(serviceInfo.Ports) > 0 {
		ports := make([]types.Int64, len(serviceInfo.Ports))
		for i, port := range serviceInfo.Ports {
			ports[i] = types.Int64Value(int64(port))
		}
		var diagsLocal diag.Diagnostics
		portsList, diagsLocal = types.ListValueFrom(ctx, types.Int64Type, ports)
		diagnostics.Append(diagsLocal...)
	} else {
		var diagsLocal diag.Diagnostics
		portsList, diagsLocal = types.ListValueFrom(ctx, types.Int64Type, []types.Int64{})
		diagnostics.Append(diagsLocal...)
	}

	// Convert package info
	var packageObj types.Object
	if serviceInfo.Package != nil {
		var diagsLocal diag.Diagnostics
		packageObj, diagsLocal = types.ObjectValueFrom(ctx, map[string]attr.Type{
			"name":    types.StringType,
			"manager": types.StringType,
			"version": types.StringType,
		}, ServicePackageModel{
			Name:    types.StringValue(serviceInfo.Package.Name),
			Manager: types.StringValue(serviceInfo.Package.Manager),
			Version: types.StringValue(serviceInfo.Package.Version),
		})
		diagnostics.Append(diagsLocal...)
	} else {
		packageObj = types.ObjectNull(map[string]attr.Type{
			"name":    types.StringType,
			"manager": types.StringType,
			"version": types.StringType,
		})
	}

	// Convert start time
	startTime := ""
	if serviceInfo.StartTime != nil {
		startTime = serviceInfo.StartTime.Format(time.RFC3339)
	}

	serviceObj, diagsLocal := types.ObjectValueFrom(ctx, map[string]attr.Type{
		"name":         types.StringType,
		"running":      types.BoolType,
		"healthy":      types.BoolType,
		"version":      types.StringType,
		"process_id":   types.StringType,
		"start_time":   types.StringType,
		"manager_type": types.StringType,
		"ports":        types.ListType{ElemType: types.Int64Type},
		"package": types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"name":    types.StringType,
				"manager": types.StringType,
				"version": types.StringType,
			},
		},
	}, ServiceOverviewModel{
		Name:        types.StringValue(serviceInfo.Name),
		Running:     types.BoolValue(serviceInfo.Running),
		Healthy:     types.BoolValue(serviceInfo.Healthy),
		Version:     types.StringValue(serviceInfo.Version),
		ProcessID:   types.StringValue(serviceInfo.ProcessID),
		StartTime:   types.StringValue(startTime),
		ManagerType: types.StringValue(serviceInfo.ManagerType),
		Ports:       portsList,
		Package:     packageObj,
	})
	diagnostics.Append(diagsLocal...)

	return serviceObj
}
