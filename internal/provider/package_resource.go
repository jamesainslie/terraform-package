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

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/geico-private/terraform-provider-pkg/internal/adapters"
	"github.com/geico-private/terraform-provider-pkg/internal/adapters/brew"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &PackageResource{}
var _ resource.ResourceWithImportState = &PackageResource{}

// NewPackageResource creates a new package resource.
func NewPackageResource() resource.Resource {
	return &PackageResource{}
}

// PackageResource defines the resource implementation.
type PackageResource struct {
	providerData *ProviderData
}

// PackageResourceModel describes the resource data model.
type PackageResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	State            types.String `tfsdk:"state"`
	Version          types.String `tfsdk:"version"`
	VersionActual    types.String `tfsdk:"version_actual"`
	Pin              types.Bool   `tfsdk:"pin"`
	Managers         types.List   `tfsdk:"managers"`
	Aliases          types.Map    `tfsdk:"aliases"`
	ReinstallOnDrift types.Bool   `tfsdk:"reinstall_on_drift"`
	HoldDependencies types.Bool   `tfsdk:"hold_dependencies"`

	// Timeouts
	Timeouts *PackageResourceTimeouts `tfsdk:"timeouts"`
}

type PackageResourceTimeouts struct {
	Create types.String `tfsdk:"create"`
	Read   types.String `tfsdk:"read"`
	Update types.String `tfsdk:"update"`
	Delete types.String `tfsdk:"delete"`
}

func (r *PackageResource) Metadata(
		ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_package"
}

func (r *PackageResource) Schema(
		ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a package installation across different package managers (Homebrew, APT, winget, Chocolatey).",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Package identifier in the format 'manager:name'.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Logical package name. " +
					"This will be resolved to platform-specific names using the package registry.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"state": schema.StringAttribute{
				MarkdownDescription: "Desired state of the package. " +
					"Valid values: 'present', 'absent'. " +
					"Defaults to 'present'.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("present"),
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "Desired version of the package. Supports exact versions, semantic version ranges, or glob patterns depending on the package manager. " +
					"Leave empty for latest version.",
				Optional:            true,
			},
			"version_actual": schema.StringAttribute{
				MarkdownDescription: "Actual installed version of the package. " +
					"This is computed and shows the real installed version.",
				Computed:            true,
			},
			"pin": schema.BoolAttribute{
				MarkdownDescription: "Whether to pin/hold the package at the current version to prevent upgrades. " +
					"Defaults to false.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"managers": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Override the package manager selection. " +
					"Valid values: 'auto', 'brew', 'apt', 'winget', 'choco'. " +
					"Defaults to ['auto'] which auto-detects based on OS.",
				Optional:            true,
			},
			"aliases": schema.MapAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Platform-specific package name overrides. Keys: 'darwin', 'linux', 'windows'. " +
					"Values: platform-specific package names.",
				Optional:            true,
			},
			"reinstall_on_drift": schema.BoolAttribute{
				MarkdownDescription: "If true, reinstall the package when version drift is detected. If false, only update version_actual. " +
					"Defaults to true.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"hold_dependencies": schema.BoolAttribute{
				MarkdownDescription: "Whether to hold/pin package dependencies. " +
					"Defaults to false.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
		},

		Blocks: map[string]schema.Block{
			"timeouts": schema.SingleNestedBlock{
				MarkdownDescription: "Timeout configuration for package operations.",
				Attributes: map[string]schema.Attribute{
					"create": schema.StringAttribute{
						MarkdownDescription: "Timeout for package installation. " +
					"Defaults to '15m'.",
						Optional:            true,
					},
					"read": schema.StringAttribute{
						MarkdownDescription: "Timeout for reading package information. " +
					"Defaults to '2m'.",
						Optional:            true,
					},
					"update": schema.StringAttribute{
						MarkdownDescription: "Timeout for package updates. " +
					"Defaults to '15m'.",
						Optional:            true,
					},
					"delete": schema.StringAttribute{
						MarkdownDescription: "Timeout for package removal. " +
					"Defaults to '10m'.",
						Optional:            true,
					},
				},
			},
		},
	}
}

func (r *PackageResource) Configure(
		ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(*ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ProviderData, got: %T. Please report this issue to the provider developers.",
				req.ProviderData),
		)
		return
	}

	r.providerData = providerData
}

func (r *PackageResource) Create(
		ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PackageResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Resolve package manager and name
	manager, packageName, err := r.resolvePackageManager(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Package Manager Resolution Failed", err.Error())
		return
	}

	// Get timeout for create operation
	var createTimeout types.String
	if data.Timeouts != nil {
		createTimeout = data.Timeouts.Create
	}
	timeout := r.getTimeout(createTimeout, "15m")

	// Create context with timeout
	createCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Update cache if configured
	if r.providerData.Config.UpdateCache.ValueString() == "on_change" ||
		r.providerData.Config.UpdateCache.ValueString() == "always" {
		if err := manager.UpdateCache(createCtx); err != nil {
			resp.Diagnostics.AddWarning(
				"Cache Update Failed",
				fmt.Sprintf("Failed to update package cache: %v", err),
			)
		}
	}

	// Install the package
	version := data.Version.ValueString()
	if err := manager.Install(createCtx, packageName, version); err != nil {
		resp.Diagnostics.AddError(
			"Package Installation Failed",
			fmt.Sprintf("Failed to install package %s: %v", packageName, err),
		)
		return
	}

	// Pin the package if requested
	if data.Pin.ValueBool() {
		if err := manager.Pin(createCtx, packageName, true); err != nil {
			resp.Diagnostics.AddWarning(
				"Package Pinning Failed",
				fmt.Sprintf("Package installed successfully but pinning failed: %v", err),
			)
		}
	}

	// Read back the actual state
	if err := r.readPackageState(createCtx, manager, packageName, &data); err != nil {
		resp.Diagnostics.AddError(
			"Failed to Read Package State",
			fmt.Sprintf("Package installed but failed to read final state: %v", err),
		)
		return
	}

	// Set the ID
	data.ID = types.StringValue(fmt.Sprintf("%s:%s", manager.GetManagerName(), packageName))

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PackageResource) Read(
		ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PackageResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Resolve package manager and name
	manager, packageName, err := r.resolvePackageManager(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Package Manager Resolution Failed", err.Error())
		return
	}

	// Get timeout for read operation
	var readTimeout types.String
	if data.Timeouts != nil {
		readTimeout = data.Timeouts.Read
	}
	timeout := r.getTimeout(readTimeout, "2m")

	// Create context with timeout
	readCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Read the package state
	if err := r.readPackageState(readCtx, manager, packageName, &data); err != nil {
		resp.Diagnostics.AddError(
			"Failed to Read Package State",
			fmt.Sprintf("Failed to read package state: %v", err),
		)
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PackageResource) Update(
		ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PackageResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Resolve package manager and name
	manager, packageName, err := r.resolvePackageManager(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Package Manager Resolution Failed", err.Error())
		return
	}

	// Get timeout for update operation
	var updateTimeout types.String
	if data.Timeouts != nil {
		updateTimeout = data.Timeouts.Update
	}
	timeout := r.getTimeout(updateTimeout, "15m")

	// Create context with timeout
	updateCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Handle state change (present -> absent or absent -> present)
	if data.State.ValueString() == "absent" {
		// Remove the package
		if err := manager.Remove(updateCtx, packageName); err != nil {
			resp.Diagnostics.AddError(
				"Package Removal Failed",
				fmt.Sprintf("Failed to remove package %s: %v", packageName, err),
			)
			return
		}
	} else {
		// Install/update the package
		version := data.Version.ValueString()
		if err := manager.Install(updateCtx, packageName, version); err != nil {
			resp.Diagnostics.AddError(
				"Package Installation Failed",
				fmt.Sprintf("Failed to install/update package %s: %v", packageName, err),
			)
			return
		}

		// Update pin status
		if err := manager.Pin(updateCtx, packageName, data.Pin.ValueBool()); err != nil {
			resp.Diagnostics.AddWarning(
				"Package Pinning Update Failed",
				fmt.Sprintf("Package updated successfully but pin status update failed: %v", err),
			)
		}
	}

	// Read back the actual state
	if err := r.readPackageState(updateCtx, manager, packageName, &data); err != nil {
		resp.Diagnostics.AddError(
			"Failed to Read Package State",
			fmt.Sprintf("Package updated but failed to read final state: %v", err),
		)
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PackageResource) Delete(
		ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PackageResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Resolve package manager and name
	manager, packageName, err := r.resolvePackageManager(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Package Manager Resolution Failed", err.Error())
		return
	}

	// Get timeout for delete operation
	var deleteTimeout types.String
	if data.Timeouts != nil {
		deleteTimeout = data.Timeouts.Delete
	}
	timeout := r.getTimeout(deleteTimeout, "10m")

	// Create context with timeout
	deleteCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Unpin the package first if it's pinned
	if data.Pin.ValueBool() {
		if err := manager.Pin(deleteCtx, packageName, false); err != nil {
			resp.Diagnostics.AddWarning(
				"Package Unpinning Failed",
				fmt.Sprintf("Failed to unpin package before removal: %v", err),
			)
		}
	}

	// Remove the package
	if err := manager.Remove(deleteCtx, packageName); err != nil {
		resp.Diagnostics.AddError(
			"Package Removal Failed",
			fmt.Sprintf("Failed to remove package %s: %v", packageName, err),
		)
		return
	}
}

func (r *PackageResource) ImportState(
		ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: "manager:package_name" or just "package_name" (auto-detect manager)
	parts := strings.SplitN(req.ID, ":", 2)

	switch len(parts) {
	case 1:
		// Just package name provided, use auto-detection
		resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
	case 2:
		// Manager and package name provided
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[1])...)

		// Set manager in the managers list
		managers := []types.String{types.StringValue(parts[0])}
		managersList, diags := types.ListValueFrom(ctx, types.StringType, managers)
		resp.Diagnostics.Append(diags...)
		if !resp.Diagnostics.HasError() {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("managers"), managersList)...)
		}
	default:
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Import ID must be in format 'package_name' or 'manager:package_name'",
		)
	}
}

// Helper methods

func (r *PackageResource) resolvePackageManager(
		ctx context.Context, data PackageResourceModel) (adapters.PackageManager, string, error) {
	// Determine which manager to use
	managerName := "auto"
	if !data.Managers.IsNull() && len(data.Managers.Elements()) > 0 {
		managers := make([]string, 0, len(data.Managers.Elements()))
		for _, elem := range data.Managers.Elements() {
			if strElem, ok := elem.(types.String); ok {
				managers = append(managers, strElem.ValueString())
			}
		}
		if len(managers) > 0 {
			managerName = managers[0] // Use first manager
		}
	}

	// Auto-detect manager based on OS
	if managerName == "auto" {
		switch runtime.GOOS {
		case "darwin":
			managerName = "brew"
		default:
			return nil, "", fmt.Errorf("only macOS (darwin) is supported in Phase 2, got: %s", runtime.GOOS)
		}
	}

	// Create the appropriate manager (only brew supported in Phase 2)
	var manager adapters.PackageManager
	switch managerName {
	case "brew":
		brewPath := r.providerData.Config.BrewPath.ValueString()
		manager = brew.NewBrewAdapter(r.providerData.Executor, brewPath)
	default:
		return nil, "", fmt.Errorf("only homebrew manager is supported in Phase 2, got: %s", managerName)
	}

	// Check if manager is available
	if !manager.IsAvailable(ctx) {
		return nil, "", fmt.Errorf("package manager %s is not available on this system", managerName)
	}

	// Resolve package name
	packageName := data.Name.ValueString()

	// Check for platform-specific aliases
	if !data.Aliases.IsNull() {
		aliases := make(map[string]string)
		for key, value := range data.Aliases.Elements() {
			if strValue, ok := value.(types.String); ok {
				aliases[key] = strValue.ValueString()
			}
		}

		if platformName, exists := aliases[runtime.GOOS]; exists {
			packageName = platformName
		}
	}

	// If no alias, try registry resolution
	if packageName == data.Name.ValueString() {
		resolvedName, err := r.providerData.Registry.ResolvePackageName(ctx, packageName, runtime.GOOS)
		if err == nil && resolvedName != "" {
			packageName = resolvedName
		}
	}

	return manager, packageName, nil
}

func (r *PackageResource) readPackageState(ctx context.Context, manager adapters.PackageManager, packageName string, data *PackageResourceModel) error {
	info, err := manager.DetectInstalled(ctx, packageName)
	if err != nil {
		return err
	}

	// Update only computed fields, don't override configured state
	if info.Installed {
		data.VersionActual = types.StringValue(info.Version)
	} else {
		data.VersionActual = types.StringValue("")
	}

	// Don't override the configured state and pin values
	// These should only be updated by user configuration, not by what we detect

	return nil
}

func (r *PackageResource) getTimeout(timeoutStr types.String, defaultTimeout string) time.Duration {
	if timeoutStr.IsNull() || timeoutStr.ValueString() == "" {
		timeout, _ := time.ParseDuration(defaultTimeout)
		return timeout
	}

	timeout, err := time.ParseDuration(timeoutStr.ValueString())
	if err != nil {
		// Fall back to default if parsing fails
		timeout, _ = time.ParseDuration(defaultTimeout)
		return timeout
	}

	return timeout
}
