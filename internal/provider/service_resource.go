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
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/jamesainslie/terraform-package/internal/executor"
	"github.com/jamesainslie/terraform-package/internal/services"
	"github.com/jamesainslie/terraform-package/internal/services/detectors"
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
	executor          executor.Executor
	serviceManager    services.ServiceManager
	dependencyManager *services.DependencyManager
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

	// Initialize dependency manager with detectors
	r.dependencyManager = services.NewDependencyManager()
	r.dependencyManager.RegisterDetector(detectors.NewDockerColimaDetector())
	r.dependencyManager.RegisterDetector(detectors.NewContainerRuntimeDetector())
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

	// Detect and validate service dependencies
	serviceName := plan.ServiceName.ValueString()
	dependencies, err := r.dependencyManager.DetectDependencies(ctx, serviceName)
	if err != nil {
		resp.Diagnostics.AddWarning(
			"Dependency Detection Warning",
			fmt.Sprintf("Failed to detect dependencies for service %s: %v", serviceName, err),
		)
	}

	// Validate dependency chain
	if err := r.dependencyManager.ValidateDependencyChain(ctx, serviceName); err != nil {
		resp.Diagnostics.AddError(
			"Dependency Validation Error",
			fmt.Sprintf("Failed to validate dependency chain for service %s: %v", serviceName, err),
		)
		return
	}

	// Log detected dependencies
	if len(dependencies) > 0 {
		resp.Diagnostics.AddWarning(
			"Dependencies Detected",
			fmt.Sprintf("Service %s has %d dependencies: %v", serviceName, len(dependencies), dependencies),
		)
	}

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
// Idempotency: Before starting/stopping, GetServiceInfo (via strategy) checks current state.
// If already in desired state (e.g., running and state="running"), the strategy methods are no-ops.
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

	// Create lifecycle strategy (supports both management and health checks)
	strategyFactory := services.NewServiceStrategyFactory(r.executor)
	lifecycleStrategy := strategyFactory.CreateLifecycleStrategy(strategy, customCommands, serviceName)

	// Apply startup configuration (only for strategies that support it)
	if strategy != services.StrategyProcessOnly {
		enabled := desiredStartup == "enabled"
		if err := r.serviceManager.SetServiceStartup(ctx, serviceName, enabled); err != nil {
			return fmt.Errorf("failed to set service startup: %w", err)
		}
	}

	// Handle service dependencies before applying state
	if desiredState == "running" {
		if err := r.handleServiceDependencies(ctx, serviceName); err != nil {
			return fmt.Errorf("failed to handle service dependencies: %w", err)
		}
	}

	// Apply service state using the chosen strategy
	switch desiredState {
	case "running":
		if err := lifecycleStrategy.StartService(ctx, serviceName); err != nil {
			return fmt.Errorf("failed to start service with strategy %s: %w", strategy, err)
		}
	case "stopped":
		if err := lifecycleStrategy.StopService(ctx, serviceName); err != nil {
			return fmt.Errorf("failed to stop service with strategy %s: %w", strategy, err)
		}
	}

	// Wait for healthy if requested
	if model.WaitForHealthy.ValueBool() && desiredState == "running" {
		if err := r.waitForHealthy(ctx, model, lifecycleStrategy); err != nil {
			return fmt.Errorf("failed to wait for healthy: %w", err)
		}
	}

	return nil
}

// updateComputedAttributes updates the computed attributes with current service information.
// Idempotency: This read-only method fetches real system state, allowing Terraform to detect if
// the desired state (e.g., running=true) already matches the actual state, resulting in no planned changes.
func (r *ServiceResource) updateComputedAttributes(ctx context.Context, model *ServiceResourceModel) error {
	serviceName := model.ServiceName.ValueString()

	// Create lifecycle strategy for strategy-aware health checks
	strategy := services.ServiceManagementStrategy(model.ManagementStrategy.ValueString())
	if strategy == "" {
		strategy = services.StrategyAuto // Default to auto if not specified
	}

	var customCommands *services.CustomCommands
	if !model.CustomCommands.IsNull() && !model.CustomCommands.IsUnknown() {
		customCommands = parseCustomCommands(ctx, model.CustomCommands)
	}

	strategyFactory := services.NewServiceStrategyFactory(r.executor)
	lifecycleStrategy := strategyFactory.CreateLifecycleStrategy(strategy, customCommands, serviceName)

	// Get service information using strategy-aware methods
	statusInfo, err := lifecycleStrategy.StatusCheck(ctx, serviceName)
	if err != nil {
		return fmt.Errorf("failed to get service status: %w", err)
	}

	healthInfo, err := lifecycleStrategy.HealthCheck(ctx, serviceName)
	if err != nil {
		return fmt.Errorf("failed to get service health: %w", err)
	}

	// Update computed attributes from real state using strategy-aware information
	model.Running = types.BoolValue(statusInfo.Running)
	model.Healthy = types.BoolValue(healthInfo.Healthy)
	model.Enabled = types.BoolValue(statusInfo.Enabled)
	model.Version = types.StringValue("") // Version info not available from strategy
	model.ProcessID = types.StringValue(statusInfo.ProcessID)
	model.ManagerType = types.StringValue(string(statusInfo.Strategy))

	// Handle start_time - not available from strategy, so leave empty
	model.StartTime = types.StringValue("")

	// Update package information - not available from strategy, so set empty
	packageModel := PackageModel{
		Name:    types.StringValue(""),
		Manager: types.StringValue(""),
		Version: types.StringValue(""),
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

	// Update ports - not available from strategy, so set empty
	ports, diags := types.ListValue(types.Int64Type, []attr.Value{})
	if diags.HasError() {
		return fmt.Errorf("failed to create ports list: %v", diags)
	}
	model.Ports = ports

	// Update metadata - use strategy details as metadata
	metadataValues := make(map[string]attr.Value)
	metadataValues["strategy"] = types.StringValue(string(statusInfo.Strategy))
	metadataValues["health_details"] = types.StringValue(healthInfo.Details)
	metadataValues["status_details"] = types.StringValue(statusInfo.Details)

	metadata, diags := types.MapValue(types.StringType, metadataValues)
	if diags.HasError() {
		return fmt.Errorf("failed to create metadata map: %v", diags)
	}
	model.Metadata = metadata

	return nil
}

// parseCustomCommands parses custom commands from the Terraform model
func parseCustomCommands(ctx context.Context, customCommandsObject types.Object) *services.CustomCommands {
	if customCommandsObject.IsNull() || customCommandsObject.IsUnknown() {
		return nil
	}

	var customCommandsModel CustomCommandsModel
	diags := customCommandsObject.As(ctx, &customCommandsModel, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil // Return nil if parsing fails
	}

	customCommands := &services.CustomCommands{}

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

	return customCommands
}

// waitForHealthy waits for the service to become healthy using strategy-aware health checks.
func (r *ServiceResource) waitForHealthy(ctx context.Context, model *ServiceResourceModel, lifecycleStrategy services.ServiceLifecycleStrategy) error {
	serviceName := model.ServiceName.ValueString()
	waitTimeout := model.WaitTimeout.ValueString()

	// Parse timeout
	timeout, err := time.ParseDuration(waitTimeout)
	if err != nil {
		return fmt.Errorf("invalid wait timeout: %w", err)
	}

	// DEBUG: Log health check start
	tflog.Debug(ctx, "Starting service health check", map[string]interface{}{
		"service_name": serviceName,
		"timeout":      timeout.String(),
	})

	// Create context with timeout for the overall operation
	operationCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Wait for service to be healthy
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-operationCtx.Done():
			tflog.Debug(ctx, "Service health check timed out", map[string]interface{}{
				"service_name": serviceName,
				"timeout":      timeout.String(),
			})
			return fmt.Errorf("timeout waiting for service to be healthy")
		case <-ticker.C:
			// Create fresh context for each health check command to avoid timeout chain reaction
			// This prevents expired contexts from causing immediate command failures
			checkCtx, checkCancel := context.WithTimeout(context.Background(), 30*time.Second)

			tflog.Debug(ctx, "Performing health check", map[string]interface{}{
				"service_name":  serviceName,
				"check_timeout": "30s",
			})

			// Check if service is healthy using strategy-aware health check
			healthInfo, err := lifecycleStrategy.HealthCheck(checkCtx, serviceName)
			checkCancel() // Always cancel the check context

			if err != nil {
				tflog.Debug(ctx, "Health check failed, retrying", map[string]interface{}{
					"service_name": serviceName,
					"error":        err.Error(),
				})
				continue
			}

			tflog.Debug(ctx, "Health check completed", map[string]interface{}{
				"service_name": serviceName,
				"healthy":      healthInfo.Healthy,
				"strategy":     healthInfo.Strategy,
				"details":      healthInfo.Details,
			})

			if healthInfo.Healthy {
				tflog.Debug(ctx, "Service is healthy", map[string]interface{}{
					"service_name": serviceName,
					"strategy":     healthInfo.Strategy,
					"details":      healthInfo.Details,
				})
				return nil
			}
		}
	}
}

// handleServiceDependencies handles service dependencies before starting a service
func (r *ServiceResource) handleServiceDependencies(ctx context.Context, serviceName string) error {
	// Get dependencies for the service
	dependencies := r.dependencyManager.GetDependencies(serviceName)
	if len(dependencies) == 0 {
		return nil // No dependencies to handle
	}

	// Handle each dependency
	for _, dep := range dependencies {
		if dep.DependencyType == services.DependencyTypeRequired ||
			dep.DependencyType == services.DependencyTypeProxy ||
			dep.DependencyType == services.DependencyTypeContainer {

			// Check if target service is running
			targetServiceInfo, err := r.serviceManager.GetServiceInfo(ctx, dep.TargetService)
			if err != nil {
				return fmt.Errorf("failed to get info for dependency service %s: %w", dep.TargetService, err)
			}

			// If target service is not running, start it using the appropriate strategy
			if !targetServiceInfo.Running {
				if err := r.startDependencyService(ctx, dep.TargetService); err != nil {
					return fmt.Errorf("failed to start dependency service %s: %w", dep.TargetService, err)
				}
			}

			// Wait for target service to be healthy
			if dep.HealthCheck != nil {
				if err := r.waitForDependencyHealthy(ctx, dep); err != nil {
					return fmt.Errorf("dependency service %s is not healthy: %w", dep.TargetService, err)
				}
			}
		}
	}

	return nil
}

// startDependencyService starts a dependency service using the appropriate strategy
func (r *ServiceResource) startDependencyService(ctx context.Context, serviceName string) error {
	// Create dependency configuration manager
	configManager := services.NewDependencyConfigManager()

	// Try to detect service configuration from environment
	config, err := configManager.DetectServiceConfiguration(ctx, serviceName)
	if err != nil {
		// Fallback to default configuration
		defaultConfigs := configManager.GetDefaultConfigs()
		if defaultConfig, exists := defaultConfigs[serviceName]; exists {
			config = defaultConfig
		} else {
			// Use auto strategy as final fallback
			config = &services.DependencyConfig{
				ServiceName: serviceName,
				Strategy:    services.StrategyAuto,
				AutoDetect:  false,
			}
		}
	}

	// Create lifecycle strategy factory and start the service
	strategyFactory := services.NewServiceStrategyFactory(r.executor)
	lifecycleStrategy := strategyFactory.CreateLifecycleStrategy(config.Strategy, config.CustomCommands, serviceName)

	return lifecycleStrategy.StartService(ctx, serviceName)
}

// waitForDependencyHealthy waits for a dependency service to be healthy
func (r *ServiceResource) waitForDependencyHealthy(ctx context.Context, dep services.ServiceDependency) error {
	if dep.HealthCheck == nil {
		return nil // No health check defined
	}

	timeout := dep.HealthCheck.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second // Default timeout
	}

	// DEBUG: Log dependency health check start
	tflog.Debug(ctx, "Starting dependency health check", map[string]interface{}{
		"target_service": dep.TargetService,
		"timeout":        timeout.String(),
	})

	// Create context with timeout for the overall operation
	operationCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(dep.HealthCheck.Interval)
	if ticker == nil {
		ticker = time.NewTicker(5 * time.Second) // Default interval
	}
	defer ticker.Stop()

	for {
		select {
		case <-operationCtx.Done():
			tflog.Debug(ctx, "Dependency health check timed out", map[string]interface{}{
				"target_service": dep.TargetService,
				"timeout":        timeout.String(),
			})
			return fmt.Errorf("timeout waiting for dependency %s to be healthy", dep.TargetService)
		case <-ticker.C:
			// Create fresh context for each health check command to avoid timeout chain reaction
			checkCtx, checkCancel := context.WithTimeout(context.Background(), 15*time.Second)

			tflog.Debug(ctx, "Performing dependency health check", map[string]interface{}{
				"target_service": dep.TargetService,
				"check_timeout":  "15s",
			})

			// Perform health check using fresh context
			healthy, err := r.performHealthCheck(checkCtx, dep.HealthCheck)
			checkCancel() // Always cancel the check context

			if err != nil {
				tflog.Debug(ctx, "Dependency health check failed, retrying", map[string]interface{}{
					"target_service": dep.TargetService,
					"error":          err.Error(),
				})
				continue // Retry on error
			}

			tflog.Debug(ctx, "Dependency health check completed", map[string]interface{}{
				"target_service": dep.TargetService,
				"healthy":        healthy,
			})

			if healthy {
				tflog.Debug(ctx, "Dependency is healthy", map[string]interface{}{
					"target_service": dep.TargetService,
				})
				return nil // Dependency is healthy
			}
		}
	}
}

// performHealthCheck performs a health check based on the configuration
func (r *ServiceResource) performHealthCheck(ctx context.Context, healthCheck *services.DependencyHealthCheck) (bool, error) {
	switch healthCheck.Type {
	case "command":
		if healthCheck.Command == "" {
			return false, fmt.Errorf("command health check requires a command")
		}
		// Execute the health check command
		// This would integrate with the executor
		return true, nil // Simplified for now
	case "http":
		// HTTP health check would be implemented here
		return true, nil // Simplified for now
	case "tcp":
		// TCP health check would be implemented here
		return true, nil // Simplified for now
	case "socket":
		// Socket health check would be implemented here
		return true, nil // Simplified for now
	default:
		return false, fmt.Errorf("unsupported health check type: %s", healthCheck.Type)
	}
}
