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
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/jamesainslie/terraform-package/internal/executor"
	"github.com/jamesainslie/terraform-package/internal/services"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &ServiceResource{}
	_ resource.ResourceWithConfigure   = &ServiceResource{}
	_ resource.ResourceWithImportState = &ServiceResource{}
)

// NewServiceResource is a helper function to simplify the provider implementation.
func NewServiceResource() resource.Resource {
	return &ServiceResource{}
}

// ServiceResource defines the resource implementation.
type ServiceResource struct {
	executor       executor.Executor
	serviceManager services.ServiceManager
}

// ServiceResourceModel describes the resource data model.
type ServiceResourceModel struct {
	ID              types.String `tfsdk:"id"`
	ServiceName     types.String `tfsdk:"service_name"`
	State           types.String `tfsdk:"state"`   // "running", "stopped"
	Startup         types.String `tfsdk:"startup"` // "enabled", "disabled"
	ValidatePackage types.Bool   `tfsdk:"validate_package"`
	PackageName     types.String `tfsdk:"package_name"`
	HealthCheck     types.Object `tfsdk:"health_check"`
	Timeout         types.String `tfsdk:"timeout"`
	WaitForHealthy  types.Bool   `tfsdk:"wait_for_healthy"`
	WaitTimeout     types.String `tfsdk:"wait_timeout"`
	// Strategy configuration
	ManagementStrategy types.String `tfsdk:"management_strategy"`
	CustomCommands     types.Object `tfsdk:"custom_commands"`
	// Computed attributes
	Running     types.Bool   `tfsdk:"running"`
	Healthy     types.Bool   `tfsdk:"healthy"`
	Enabled     types.Bool   `tfsdk:"enabled"`
	Version     types.String `tfsdk:"version"`
	ProcessID   types.String `tfsdk:"process_id"`
	StartTime   types.String `tfsdk:"start_time"`
	ManagerType types.String `tfsdk:"manager_type"`
	Package     types.Object `tfsdk:"package"`
	Ports       types.List   `tfsdk:"ports"`
	Metadata    types.Map    `tfsdk:"metadata"`
	LastUpdated types.String `tfsdk:"last_updated"`
}

// ServiceHealthCheckModel describes the health check configuration for service resource.
type ServiceHealthCheckModel struct {
	Type         types.String `tfsdk:"type"`
	Command      types.String `tfsdk:"command"`
	URL          types.String `tfsdk:"url"`
	Port         types.Int64  `tfsdk:"port"`
	Timeout      types.String `tfsdk:"timeout"`
	ExpectedCode types.Int64  `tfsdk:"expected_code"`
	Interval     types.String `tfsdk:"interval"`
}

// CustomCommandsModel describes custom commands for service management.
type CustomCommandsModel struct {
	Start   types.List `tfsdk:"start"`
	Stop    types.List `tfsdk:"stop"`
	Restart types.List `tfsdk:"restart"`
	Status  types.List `tfsdk:"status"`
}

// PackageModel describes the package information.
type PackageModel struct {
	Name    types.String `tfsdk:"name"`
	Manager types.String `tfsdk:"manager"`
	Version types.String `tfsdk:"version"`
}

// Metadata returns the resource type name.
func (r *ServiceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

// Schema defines the schema for the resource.
func (r *ServiceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		Description: "Manages service lifecycle (start, stop, restart) and startup configuration.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the service resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"service_name": schema.StringAttribute{
				Description: "Name of the service to manage.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"state": schema.StringAttribute{
				Description: "Desired state of the service. Valid values: 'running', 'stopped'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("running"),
				Validators: []validator.String{
					stringvalidator.OneOf("running", "stopped"),
				},
			},
			"startup": schema.StringAttribute{
				Description: "Startup configuration for the service. Valid values: 'enabled', 'disabled'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("enabled"),
				Validators: []validator.String{
					stringvalidator.OneOf("enabled", "disabled"),
				},
			},
			"validate_package": schema.BoolAttribute{
				Description: "Whether to validate that the associated package is installed.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"package_name": schema.StringAttribute{
				Description: "Name of the package associated with this service. Used for validation if validate_package is true.",
				Optional:    true,
			},
			"health_check": schema.SingleNestedAttribute{
				Description: "Health check configuration for the service.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: "Type of health check. Valid values: 'command', 'http', 'tcp'.",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("command", "http", "tcp"),
						},
					},
					"command": schema.StringAttribute{
						Description: "Command to execute for command-based health checks.",
						Optional:    true,
					},
					"url": schema.StringAttribute{
						Description: "URL for HTTP-based health checks.",
						Optional:    true,
					},
					"port": schema.Int64Attribute{
						Description: "Port number for TCP-based health checks.",
						Optional:    true,
					},
					"timeout": schema.StringAttribute{
						Description: "Timeout for the health check (e.g., '30s').",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("30s"),
					},
					"expected_code": schema.Int64Attribute{
						Description: "Expected HTTP status code for HTTP health checks.",
						Optional:    true,
						Computed:    true,
						Default:     int64default.StaticInt64(200),
					},
					"interval": schema.StringAttribute{
						Description: "Interval between health checks (e.g., '10s').",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("10s"),
					},
				},
			},
			"timeout": schema.StringAttribute{
				Description: "Timeout for service operations (e.g., '60s').",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("60s"),
			},
			"wait_for_healthy": schema.BoolAttribute{
				Description: "Whether to wait for the service to be healthy after starting.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"wait_timeout": schema.StringAttribute{
				Description: "Timeout for waiting for service to be healthy (e.g., '120s').",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("120s"),
			},
			"management_strategy": schema.StringAttribute{
				Description: "Strategy for managing the service. Valid values: 'auto', 'brew_services', 'direct_command', 'launchd', 'process_only'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("auto"),
				Validators: []validator.String{
					stringvalidator.OneOf("auto", "brew_services", "direct_command", "launchd", "process_only"),
				},
			},
			"custom_commands": schema.SingleNestedAttribute{
				Description: "Custom commands for service management. Used when management_strategy is 'direct_command'.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"start": schema.ListAttribute{
						Description: "Command and arguments to start the service.",
						Optional:    true,
						ElementType: types.StringType,
					},
					"stop": schema.ListAttribute{
						Description: "Command and arguments to stop the service.",
						Optional:    true,
						ElementType: types.StringType,
					},
					"restart": schema.ListAttribute{
						Description: "Command and arguments to restart the service.",
						Optional:    true,
						ElementType: types.StringType,
					},
					"status": schema.ListAttribute{
						Description: "Command and arguments to check service status.",
						Optional:    true,
						ElementType: types.StringType,
					},
				},
			},
			// Computed attributes
			"running": schema.BoolAttribute{
				Description: "Whether the service is currently running.",
				Computed:    true,
			},
			"healthy": schema.BoolAttribute{
				Description: "Whether the service is currently healthy.",
				Computed:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the service is enabled for automatic startup.",
				Computed:    true,
			},
			"version": schema.StringAttribute{
				Description: "Version of the service.",
				Computed:    true,
			},
			"process_id": schema.StringAttribute{
				Description: "Process ID of the service.",
				Computed:    true,
			},
			"start_time": schema.StringAttribute{
				Description: "Start time of the service.",
				Computed:    true,
			},
			"manager_type": schema.StringAttribute{
				Description: "Type of service manager (e.g., 'systemd', 'launchd').",
				Computed:    true,
			},
			"package": schema.SingleNestedAttribute{
				Description: "Information about the associated package.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Description: "Name of the package.",
						Computed:    true,
					},
					"manager": schema.StringAttribute{
						Description: "Package manager (e.g., 'brew', 'apt').",
						Computed:    true,
					},
					"version": schema.StringAttribute{
						Description: "Version of the package.",
						Computed:    true,
					},
				},
			},
			"ports": schema.ListAttribute{
				Description: "List of ports the service is listening on.",
				Computed:    true,
				ElementType: types.Int64Type,
			},
			"metadata": schema.MapAttribute{
				Description: "Additional metadata about the service.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last update.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *ServiceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(*ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ProviderData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.executor = providerData.Executor
	r.serviceManager = services.NewServiceManager(providerData.Executor)
}

// Create creates the resource and sets the initial Terraform state.
func (r *ServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ServiceResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set the ID
	plan.ID = types.StringValue(plan.ServiceName.ValueString())

	// Validate package if requested
	if plan.ValidatePackage.ValueBool() && !plan.PackageName.IsNull() && !plan.PackageName.IsUnknown() {
		packageName := plan.PackageName.ValueString()
		services, err := r.serviceManager.GetServicesForPackage(packageName)
		if err != nil {
			resp.Diagnostics.AddError(
				"Package Validation Error",
				fmt.Sprintf("Failed to validate package %s: %v", packageName, err),
			)
			return
		}
		if len(services) == 0 {
			resp.Diagnostics.AddWarning(
				"Package Validation Warning",
				fmt.Sprintf("No services found for package %s", packageName),
			)
		}
	}

	// Apply the desired state
	if err := r.applyServiceState(ctx, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Service State Error",
			fmt.Sprintf("Failed to apply service state: %v", err),
		)
		return
	}

	// Update computed attributes
	if err := r.updateComputedAttributes(ctx, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Service Info Error",
			fmt.Sprintf("Failed to get service information: %v", err),
		)
		return
	}

	// Set the last updated timestamp
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC3339))

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *ServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ServiceResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update computed attributes
	if err := r.updateComputedAttributes(ctx, &state); err != nil {
		resp.Diagnostics.AddError(
			"Service Info Error",
			fmt.Sprintf("Failed to get service information: %v", err),
		)
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *ServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ServiceResourceModel

	// Read Terraform plan and state data into the models
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Apply the desired state
	if err := r.applyServiceState(ctx, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Service State Error",
			fmt.Sprintf("Failed to apply service state: %v", err),
		)
		return
	}

	// Update computed attributes
	if err := r.updateComputedAttributes(ctx, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Service Info Error",
			fmt.Sprintf("Failed to get service information: %v", err),
		)
		return
	}

	// Set the last updated timestamp
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC3339))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *ServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ServiceResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Stop the service if it's running
	if state.State.ValueString() == "running" {
		if err := r.serviceManager.StopService(ctx, state.ServiceName.ValueString()); err != nil {
			resp.Diagnostics.AddWarning(
				"Service Stop Warning",
				fmt.Sprintf("Failed to stop service %s: %v", state.ServiceName.ValueString(), err),
			)
		}
	}

	// Disable the service if it's enabled
	if state.Startup.ValueString() == "enabled" {
		if err := r.serviceManager.DisableService(ctx, state.ServiceName.ValueString()); err != nil {
			resp.Diagnostics.AddWarning(
				"Service Disable Warning",
				fmt.Sprintf("Failed to disable service %s: %v", state.ServiceName.ValueString(), err),
			)
		}
	}
}

// ImportState imports the resource from the given identifier.
func (r *ServiceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Set the service name from the import ID
	serviceName := req.ID

	// Create a basic model with the service name
	model := ServiceResourceModel{
		ID:          types.StringValue(serviceName),
		ServiceName: types.StringValue(serviceName),
		State:       types.StringValue("running"),
		Startup:     types.StringValue("enabled"),
	}

	// Update computed attributes
	if err := r.updateComputedAttributes(ctx, &model); err != nil {
		resp.Diagnostics.AddError(
			"Service Info Error",
			fmt.Sprintf("Failed to get service information: %v", err),
		)
		return
	}

	// Save the model to state
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

// applyServiceState applies the desired service state (running/stopped and enabled/disabled).
func (r *ServiceResource) applyServiceState(ctx context.Context, model *ServiceResourceModel) error {
	serviceName := model.ServiceName.ValueString()
	desiredState := model.State.ValueString()
	desiredStartup := model.Startup.ValueString()

	// Determine strategy
	strategy := services.StrategyAuto
	if !model.ManagementStrategy.IsNull() && !model.ManagementStrategy.IsUnknown() {
		strategy = services.ServiceManagementStrategy(model.ManagementStrategy.ValueString())
	}

	// Parse custom commands if provided
	var customCommands *services.CustomCommands
	if !model.CustomCommands.IsNull() && !model.CustomCommands.IsUnknown() {
		var customCommandsModel CustomCommandsModel
		diags := model.CustomCommands.As(ctx, &customCommandsModel, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return fmt.Errorf("failed to parse custom commands: %v", diags)
		}

		customCommands = &services.CustomCommands{}

		// Parse start command
		if !customCommandsModel.Start.IsNull() && !customCommandsModel.Start.IsUnknown() {
			var startCommands []types.String
			diags := customCommandsModel.Start.ElementsAs(ctx, &startCommands, false)
			if !diags.HasError() {
				customCommands.Start = make([]string, len(startCommands))
				for i, cmd := range startCommands {
					customCommands.Start[i] = cmd.ValueString()
				}
			}
		}

		// Parse stop command
		if !customCommandsModel.Stop.IsNull() && !customCommandsModel.Stop.IsUnknown() {
			var stopCommands []types.String
			diags := customCommandsModel.Stop.ElementsAs(ctx, &stopCommands, false)
			if !diags.HasError() {
				customCommands.Stop = make([]string, len(stopCommands))
				for i, cmd := range stopCommands {
					customCommands.Stop[i] = cmd.ValueString()
				}
			}
		}

		// Parse restart command
		if !customCommandsModel.Restart.IsNull() && !customCommandsModel.Restart.IsUnknown() {
			var restartCommands []types.String
			diags := customCommandsModel.Restart.ElementsAs(ctx, &restartCommands, false)
			if !diags.HasError() {
				customCommands.Restart = make([]string, len(restartCommands))
				for i, cmd := range restartCommands {
					customCommands.Restart[i] = cmd.ValueString()
				}
			}
		}

		// Parse status command
		if !customCommandsModel.Status.IsNull() && !customCommandsModel.Status.IsUnknown() {
			var statusCommands []types.String
			diags := customCommandsModel.Status.ElementsAs(ctx, &statusCommands, false)
			if !diags.HasError() {
				customCommands.Status = make([]string, len(statusCommands))
				for i, cmd := range statusCommands {
					customCommands.Status[i] = cmd.ValueString()
				}
			}
		}
	}

	// If no custom commands provided and strategy is direct_command, try to get defaults
	if customCommands == nil && strategy == services.StrategyDirectCommand {
		factory := services.NewServiceStrategyFactory(r.executor)
		customCommands = factory.GetDefaultCommandsForService(serviceName)
	}

	// Create strategy
	strategyFactory := services.NewServiceStrategyFactory(r.executor)
	serviceStrategy := strategyFactory.CreateStrategy(strategy, customCommands)

	// Apply startup configuration (only for strategies that support it)
	if strategy != services.StrategyProcessOnly {
		enabled := desiredStartup == "enabled"
		if err := r.serviceManager.SetServiceStartup(ctx, serviceName, enabled); err != nil {
			return fmt.Errorf("failed to set service startup: %w", err)
		}
	}

	// Apply service state using the chosen strategy
	switch desiredState {
	case "running":
		if err := serviceStrategy.StartService(ctx, serviceName); err != nil {
			return fmt.Errorf("failed to start service with strategy %s: %w", strategy, err)
		}
	case "stopped":
		if err := serviceStrategy.StopService(ctx, serviceName); err != nil {
			return fmt.Errorf("failed to stop service with strategy %s: %w", strategy, err)
		}
	}

	// Wait for healthy if requested
	if model.WaitForHealthy.ValueBool() && desiredState == "running" {
		if err := r.waitForHealthy(ctx, model); err != nil {
			return fmt.Errorf("failed to wait for healthy: %w", err)
		}
	}

	return nil
}

// updateComputedAttributes updates the computed attributes with current service information.
func (r *ServiceResource) updateComputedAttributes(ctx context.Context, model *ServiceResourceModel) error {
	serviceName := model.ServiceName.ValueString()

	// Get service information
	serviceInfo, err := r.serviceManager.GetServiceInfo(ctx, serviceName)
	if err != nil {
		return fmt.Errorf("failed to get service info: %w", err)
	}

	// Update computed attributes
	model.Running = types.BoolValue(serviceInfo.Running)
	model.Healthy = types.BoolValue(serviceInfo.Healthy)
	model.Enabled = types.BoolValue(serviceInfo.Enabled)
	model.Version = types.StringValue(serviceInfo.Version)
	model.ProcessID = types.StringValue(serviceInfo.ProcessID)
	model.ManagerType = types.StringValue(serviceInfo.ManagerType)

	// Always set start_time - use empty string if not available
	if serviceInfo.StartTime != nil {
		model.StartTime = types.StringValue(serviceInfo.StartTime.Format(time.RFC3339))
	} else {
		model.StartTime = types.StringValue("")
	}

	// Update package information - always set a value, even if empty
	var packageModel PackageModel
	if serviceInfo.Package != nil {
		packageModel = PackageModel{
			Name:    types.StringValue(serviceInfo.Package.Name),
			Manager: types.StringValue(serviceInfo.Package.Manager),
			Version: types.StringValue(serviceInfo.Package.Version),
		}
	} else {
		packageModel = PackageModel{
			Name:    types.StringValue(""),
			Manager: types.StringValue(""),
			Version: types.StringValue(""),
		}
	}
	packageObject, diags := types.ObjectValueFrom(ctx, map[string]attr.Type{
		"name":    types.StringType,
		"manager": types.StringType,
		"version": types.StringType,
	}, packageModel)
	if diags.HasError() {
		return fmt.Errorf("failed to create package object: %v", diags)
	}
	model.Package = packageObject

	// Update ports - always set a value, even if empty
	portValues := make([]attr.Value, len(serviceInfo.Ports))
	for i, port := range serviceInfo.Ports {
		portValues[i] = types.Int64Value(int64(port))
	}
	ports, diags := types.ListValue(types.Int64Type, portValues)
	if diags.HasError() {
		return fmt.Errorf("failed to create ports list: %v", diags)
	}
	model.Ports = ports

	// Update metadata - always set a value, even if empty
	metadataValues := make(map[string]attr.Value)
	for k, v := range serviceInfo.Metadata {
		metadataValues[k] = types.StringValue(v)
	}
	metadata, diags := types.MapValue(types.StringType, metadataValues)
	if diags.HasError() {
		return fmt.Errorf("failed to create metadata map: %v", diags)
	}
	model.Metadata = metadata

	return nil
}

// waitForHealthy waits for the service to become healthy.
func (r *ServiceResource) waitForHealthy(ctx context.Context, model *ServiceResourceModel) error {
	serviceName := model.ServiceName.ValueString()
	waitTimeout := model.WaitTimeout.ValueString()

	// Parse timeout
	timeout, err := time.ParseDuration(waitTimeout)
	if err != nil {
		return fmt.Errorf("invalid wait timeout: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Wait for service to be healthy
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for service to be healthy")
		case <-ticker.C:
			// Check if service is healthy
			serviceInfo, err := r.serviceManager.GetServiceInfo(ctx, serviceName)
			if err != nil {
				continue
			}
			if serviceInfo.Healthy {
				return nil
			}
		}
	}
}
