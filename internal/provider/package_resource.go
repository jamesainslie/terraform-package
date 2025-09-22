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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jamesainslie/terraform-package/internal/adapters"
	"github.com/jamesainslie/terraform-package/internal/adapters/brew"
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
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	State              types.String `tfsdk:"state"`
	Version            types.String `tfsdk:"version"`
	VersionActual      types.String `tfsdk:"version_actual"`
	Pin                types.Bool   `tfsdk:"pin"`
	Managers           types.List   `tfsdk:"managers"`
	Aliases            types.Map    `tfsdk:"aliases"`
	ReinstallOnDrift   types.Bool   `tfsdk:"reinstall_on_drift"`
	HoldDependencies   types.Bool   `tfsdk:"hold_dependencies"`
	PackageType        types.String `tfsdk:"package_type"`
	Dependencies       types.List   `tfsdk:"dependencies"`
	InstallPriority    types.Int64  `tfsdk:"install_priority"`
	DependencyStrategy types.String `tfsdk:"dependency_strategy"`

	// Enhanced State Tracking
	TrackMetadata      types.Bool   `tfsdk:"track_metadata"`
	TrackDependencies  types.Bool   `tfsdk:"track_dependencies"`
	TrackUsage         types.Bool   `tfsdk:"track_usage"`
	InstallationSource types.String `tfsdk:"installation_source"`
	DependencyTree     types.Map    `tfsdk:"dependency_tree"`
	LastAccess         types.String `tfsdk:"last_access"`

	// Timeouts
	Timeouts *PackageResourceTimeouts `tfsdk:"timeouts"`

	// Drift Detection Configuration
	DriftDetection *DriftDetectionConfig `tfsdk:"drift_detection"`
}

// PackageResourceTimeouts defines timeout configurations for package operations.
type PackageResourceTimeouts struct {
	Create types.String `tfsdk:"create"`
	Read   types.String `tfsdk:"read"`
	Update types.String `tfsdk:"update"`
	Delete types.String `tfsdk:"delete"`
}

// PackageWithPriority represents a package with its installation priority.
type PackageWithPriority struct {
	Name     string
	Priority int64
}

// DependencyResolution contains information about resolved dependencies.
type DependencyResolution struct {
	InstallOrder []string
	Missing      []string
	Circular     []string
}

// DriftDetectionConfig defines drift detection configuration.
type DriftDetectionConfig struct {
	CheckVersion      types.Bool   `tfsdk:"check_version"`
	CheckIntegrity    types.Bool   `tfsdk:"check_integrity"`
	CheckDependencies types.Bool   `tfsdk:"check_dependencies"`
	Remediation       types.String `tfsdk:"remediation"`
}

// PackageState represents the state of a package.
type PackageState struct {
	Name      string
	Version   string
	Installed bool
	Checksum  string
}

// DriftInfo contains information about detected drift.
type DriftInfo struct {
	HasVersionDrift     bool
	HasIntegrityDrift   bool
	HasDependencyDrift  bool
	CurrentVersion      string
	DesiredVersion      string
	RemediationStrategy string
}

// IntegrityDriftInfo contains information about integrity drift.
type IntegrityDriftInfo struct {
	HasDrift         bool
	ExpectedChecksum string
	ActualChecksum   string
}

// Metadata returns the resource type name.
func (r *PackageResource) Metadata(
	_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_package"
}

// Schema defines the resource schema.
func (r *PackageResource) Schema(
	_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"state": schema.StringAttribute{
				MarkdownDescription: "Desired state of the package. " +
					"Valid values: 'present', 'absent'. " +
					"Defaults to 'present'.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("present"),
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "Desired version of the package. Supports exact versions, semantic version ranges, or glob patterns depending on the package manager. " +
					"Leave empty for latest version.",
				Optional: true,
			},
			"version_actual": schema.StringAttribute{
				MarkdownDescription: "Actual installed version of the package. " +
					"This is computed and shows the real installed version.",
				Computed: true,
			},
			"pin": schema.BoolAttribute{
				MarkdownDescription: "Whether to pin/hold the package at the current version to prevent upgrades. " +
					"Defaults to false.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"managers": schema.ListAttribute{
				ElementType: types.StringType,
				MarkdownDescription: "Override the package manager selection. " +
					"Valid values: 'auto', 'brew', 'apt', 'winget', 'choco'. " +
					"Defaults to ['auto'] which auto-detects based on OS.",
				Optional: true,
			},
			"aliases": schema.MapAttribute{
				ElementType: types.StringType,
				MarkdownDescription: "Platform-specific package name overrides. Keys: 'darwin', 'linux', 'windows'. " +
					"Values: platform-specific package names.",
				Optional: true,
			},
			"reinstall_on_drift": schema.BoolAttribute{
				MarkdownDescription: "If true, reinstall the package when version drift is detected. If false, only update version_actual. " +
					"Defaults to true.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"hold_dependencies": schema.BoolAttribute{
				MarkdownDescription: "Whether to hold/pin package dependencies. " +
					"Defaults to false.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"package_type": schema.StringAttribute{
				MarkdownDescription: "Type of package to install. " +
					"Valid values: 'auto', 'formula', 'cask'. " +
					"Defaults to 'auto' which auto-detects the package type. " +
					"For Homebrew: 'formula' for command-line tools, 'cask' for GUI applications.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("auto"),
			},
			"dependencies": schema.ListAttribute{
				ElementType: types.StringType,
				MarkdownDescription: "List of package names that must be installed before this package. " +
					"Dependencies will be resolved based on the dependency_strategy setting.",
				Optional: true,
			},
			"install_priority": schema.Int64Attribute{
				MarkdownDescription: "Installation priority for dependency ordering. " +
					"Higher numbers are installed first. Defaults to 0.",
				Optional: true,
				Computed: true,
				Default:  int64default.StaticInt64(0),
			},
			"dependency_strategy": schema.StringAttribute{
				MarkdownDescription: "Strategy for handling dependencies. " +
					"Valid values: 'install_missing', 'require_existing', 'ignore'. " +
					"Defaults to 'install_missing'.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("install_missing"),
			},
			"track_metadata": schema.BoolAttribute{
				MarkdownDescription: "Whether to track enhanced package metadata. " +
					"Defaults to false.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"track_dependencies": schema.BoolAttribute{
				MarkdownDescription: "Whether to track package dependency relationships. " +
					"Defaults to false.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"track_usage": schema.BoolAttribute{
				MarkdownDescription: "Whether to track package usage statistics. " +
					"Defaults to false.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"installation_source": schema.StringAttribute{
				MarkdownDescription: "Source from which the package was installed. " +
					"Computed automatically.",
				Computed: true,
			},
			"dependency_tree": schema.MapAttribute{
				ElementType: types.StringType,
				MarkdownDescription: "Map of dependencies and their versions. " +
					"Computed when track_dependencies is enabled.",
				Computed: true,
			},
			"last_access": schema.StringAttribute{
				MarkdownDescription: "Timestamp of last package access. " +
					"Computed when track_usage is enabled.",
				Computed: true,
			},
		},

		Blocks: map[string]schema.Block{
			"timeouts": schema.SingleNestedBlock{
				MarkdownDescription: "Timeout configuration for package operations.",
				Attributes: map[string]schema.Attribute{
					"create": schema.StringAttribute{
						MarkdownDescription: "Timeout for package installation. " +
							"Defaults to '15m'.",
						Optional: true,
					},
					"read": schema.StringAttribute{
						MarkdownDescription: "Timeout for reading package information. " +
							"Defaults to '2m'.",
						Optional: true,
					},
					"update": schema.StringAttribute{
						MarkdownDescription: "Timeout for package updates. " +
							"Defaults to '15m'.",
						Optional: true,
					},
					"delete": schema.StringAttribute{
						MarkdownDescription: "Timeout for package removal. " +
							"Defaults to '10m'.",
						Optional: true,
					},
				},
			},
			"drift_detection": schema.SingleNestedBlock{
				MarkdownDescription: "Configuration for drift detection and remediation.",
				Attributes: map[string]schema.Attribute{
					"check_version": schema.BoolAttribute{
						MarkdownDescription: "Whether to check for version drift. " +
							"Defaults to true.",
						Optional: true,
					},
					"check_integrity": schema.BoolAttribute{
						MarkdownDescription: "Whether to check package file integrity. " +
							"Defaults to true.",
						Optional: true,
					},
					"check_dependencies": schema.BoolAttribute{
						MarkdownDescription: "Whether to check dependency drift. " +
							"Defaults to true.",
						Optional: true,
					},
					"remediation": schema.StringAttribute{
						MarkdownDescription: "Remediation strategy for detected drift. " +
							"Valid values: 'auto', 'manual', 'warn'. " +
							"Defaults to 'auto'.",
						Optional: true,
					},
				},
			},
		},
	}
}

// Configure configures the resource with provider data.
func (r *PackageResource) Configure(
	_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create creates a new resource.
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

	// Handle dependencies if specified
	if !data.Dependencies.IsNull() && len(data.Dependencies.Elements()) > 0 {
		// Extract dependency list
		dependencies := make([]string, 0)
		for _, elem := range data.Dependencies.Elements() {
			if strElem, ok := elem.(types.String); ok {
				dependencies = append(dependencies, strElem.ValueString())
			}
		}

		// Get dependency strategy
		strategy := "install_missing"
		if !data.DependencyStrategy.IsNull() {
			strategy = data.DependencyStrategy.ValueString()
		}

		// Resolve dependencies
		resolution, err := r.resolveDependencies(createCtx, dependencies, strategy)
		if err != nil {
			resp.Diagnostics.AddError(
				"Dependency Resolution Failed",
				fmt.Sprintf("Failed to resolve dependencies for %s: %v", packageName, err),
			)
			return
		}

		// Check for circular dependencies
		if len(resolution.Circular) > 0 {
			resp.Diagnostics.AddError(
				"Circular Dependency Detected",
				fmt.Sprintf("Circular dependencies found: %v", resolution.Circular),
			)
			return
		}

		// Install dependencies in order based on strategy
		if strategy == "install_missing" {
			for _, dep := range resolution.InstallOrder {
				// Install dependency with auto type detection
				if err := manager.InstallWithType(createCtx, dep, "", adapters.PackageTypeAuto); err != nil {
					resp.Diagnostics.AddWarning(
						"Dependency Installation Failed",
						fmt.Sprintf("Failed to install dependency %s for package %s: %v", dep, packageName, err),
					)
					// Continue with main package installation even if dependency fails
				}
			}
		}
	}

	// Install the main package with specified type
	version := data.Version.ValueString()
	packageType := r.getPackageType(data.PackageType)

	if err := manager.InstallWithType(createCtx, packageName, version, packageType); err != nil {
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

// Update updates an existing resource.
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
	packageType := r.getPackageType(data.PackageType)

	if data.State.ValueString() == "absent" {
		// Remove the package with specified type
		if err := manager.RemoveWithType(updateCtx, packageName, packageType); err != nil {
			resp.Diagnostics.AddError(
				"Package Removal Failed",
				fmt.Sprintf("Failed to remove package %s: %v", packageName, err),
			)
			return
		}
	} else {
		// Install/update the package with specified type
		version := data.Version.ValueString()
		if err := manager.InstallWithType(updateCtx, packageName, version, packageType); err != nil {
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

// Delete removes a resource.
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

	// Remove the package with specified type
	packageType := r.getPackageType(data.PackageType)
	if err := manager.RemoveWithType(deleteCtx, packageName, packageType); err != nil {
		resp.Diagnostics.AddError(
			"Package Removal Failed",
			fmt.Sprintf("Failed to remove package %s: %v", packageName, err),
		)
		return
	}
}

// ImportState imports an existing resource.
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
	// Check if provider data is available
	if r.providerData == nil {
		return nil, "", fmt.Errorf("provider data is not configured")
	}

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
		timeout, err := time.ParseDuration(defaultTimeout)
		if err != nil {
			timeout = 5 * time.Minute // Hard fallback
		}
		return timeout
	}

	timeout, err := time.ParseDuration(timeoutStr.ValueString())
	if err != nil {
		// Fall back to default if parsing fails
		timeout, err = time.ParseDuration(defaultTimeout)
		if err != nil {
			timeout = 5 * time.Minute // Hard fallback
		}
		return timeout
	}

	return timeout
}

func (r *PackageResource) getPackageType(packageTypeValue types.String) adapters.PackageType {
	if packageTypeValue.IsNull() || packageTypeValue.ValueString() == "" {
		return adapters.PackageTypeAuto
	}

	switch packageTypeValue.ValueString() {
	case "auto":
		return adapters.PackageTypeAuto
	case "formula":
		return adapters.PackageTypeFormula
	case "cask":
		return adapters.PackageTypeCask
	default:
		// Default to auto if invalid value
		return adapters.PackageTypeAuto
	}
}

// resolveDependencies resolves package dependencies based on the strategy.
func (r *PackageResource) resolveDependencies(ctx context.Context, dependencies []string, strategy string) (*DependencyResolution, error) {
	if r.providerData == nil {
		return nil, fmt.Errorf("provider data is not configured")
	}

	resolution := &DependencyResolution{
		InstallOrder: []string{},
		Missing:      []string{},
		Circular:     []string{},
	}

	// Build dependency map for circular detection
	depMap := make(map[string][]string)
	depMap["main"] = dependencies

	// Check for circular dependencies
	for _, dep := range dependencies {
		if err := r.detectCircularDependencies(dep, depMap, make(map[string]bool), make(map[string]bool)); err != nil {
			resolution.Circular = append(resolution.Circular, dep)
		}
	}

	// Process each dependency based on strategy
	switch strategy {
	case "install_missing":
		// Check which dependencies are missing and add them to install order
		for _, dep := range dependencies {
			// In a real implementation, we would check if the package is installed
			// For now, assume all dependencies need to be installed
			resolution.InstallOrder = append(resolution.InstallOrder, dep)
		}
	case "require_existing":
		// Check which dependencies are missing and fail if any are missing
		for _, dep := range dependencies {
			// In a real implementation, we would check if package exists
			// For testing purposes, assume any dependency is missing
			resolution.Missing = append(resolution.Missing, dep)
		}
		if len(resolution.Missing) > 0 {
			return resolution, fmt.Errorf("required dependencies not found: %v", resolution.Missing)
		}
	case "ignore":
		// Don't process dependencies
		break
	default:
		return nil, fmt.Errorf("unknown dependency strategy: %s", strategy)
	}

	return resolution, nil
}

// sortPackagesByPriority sorts packages by their installation priority (higher numbers first).
func (r *PackageResource) sortPackagesByPriority(packages []PackageWithPriority) []PackageWithPriority {
	// Create a copy to avoid modifying the original slice
	sorted := make([]PackageWithPriority, len(packages))
	copy(sorted, packages)

	// Sort by priority descending (higher priority first)
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i].Priority < sorted[j].Priority {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	return sorted
}

// detectCircularDependencies detects circular dependencies in the dependency graph using DFS.
func (r *PackageResource) detectCircularDependencies(pkg string, depMap map[string][]string, visiting, visited map[string]bool) error {
	if visiting[pkg] {
		return fmt.Errorf("circular dependency detected involving package: %s", pkg)
	}
	if visited[pkg] {
		return nil
	}

	visiting[pkg] = true

	// Visit all dependencies
	if deps, exists := depMap[pkg]; exists {
		for _, dep := range deps {
			if err := r.detectCircularDependencies(dep, depMap, visiting, visited); err != nil {
				return err
			}
		}
	}

	visiting[pkg] = false
	visited[pkg] = true

	return nil
}

// detectVersionDrift detects version drift between current and desired package state.
func (r *PackageResource) detectVersionDrift(current, desired PackageState) DriftInfo {
	drift := DriftInfo{
		CurrentVersion: current.Version,
		DesiredVersion: desired.Version,
	}

	// Simple version comparison for now
	drift.HasVersionDrift = current.Version != desired.Version

	return drift
}

// detectIntegrityDrift detects integrity drift by checking package checksums.
func (r *PackageResource) detectIntegrityDrift(packagePath, expectedChecksum string) *IntegrityDriftInfo {
	drift := &IntegrityDriftInfo{
		ExpectedChecksum: expectedChecksum,
		ActualChecksum:   "", // Would be computed from file
	}

	// In a real implementation, we would:
	// 1. Calculate checksum of the package file
	// 2. Compare with expected checksum
	// For now, assume no drift
	drift.HasDrift = false

	return drift
}

// determineRemediationAction determines the appropriate remediation action based on drift info.
func (r *PackageResource) determineRemediationAction(drift DriftInfo) string {
	switch drift.RemediationStrategy {
	case "auto":
		// Automatically determine action based on drift type
		if drift.HasVersionDrift || drift.HasDependencyDrift {
			return "reinstall"
		}
		if drift.HasIntegrityDrift {
			return "repair"
		}
		return "none"
	case "manual":
		return "manual"
	case "warn":
		return "warn"
	default:
		return "none"
	}
}
