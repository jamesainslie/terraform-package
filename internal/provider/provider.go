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
	"runtime"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jamesainslie/terraform-package/internal/executor"
	"github.com/jamesainslie/terraform-package/internal/registry"
)

// Ensure PackageProvider satisfies various provider interfaces.
var _ provider.Provider = &PackageProvider{}
var _ provider.ProviderWithFunctions = &PackageProvider{}

// PackageProvider defines the provider implementation.
type PackageProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// PackageProviderModel describes the provider data model.
type PackageProviderModel struct {
	DefaultManager     types.String `tfsdk:"default_manager"`
	AssumeYes          types.Bool   `tfsdk:"assume_yes"`
	SudoEnabled        types.Bool   `tfsdk:"sudo_enabled"`
	BrewPath           types.String `tfsdk:"brew_path"`
	AptGetPath         types.String `tfsdk:"apt_get_path"`
	WingetPath         types.String `tfsdk:"winget_path"`
	ChocoPath          types.String `tfsdk:"choco_path"`
	UpdateCache        types.String `tfsdk:"update_cache"`
	LockTimeout        types.String `tfsdk:"lock_timeout"`
	RetryCount         types.Int64  `tfsdk:"retry_count"`
	RetryDelay         types.String `tfsdk:"retry_delay"`
	FailOnDownload     types.Bool   `tfsdk:"fail_on_download"`
	CleanupOnError     types.Bool   `tfsdk:"cleanup_on_error"`
	VerifyDownloads    types.Bool   `tfsdk:"verify_downloads"`
	ChecksumValidation types.Bool   `tfsdk:"checksum_validation"`
}

// ProviderData holds the configured provider data that will be passed to resources and data sources.
type ProviderData struct {
	Executor       executor.Executor
	Registry       registry.PackageRegistry
	Config         *PackageProviderModel
	DiagHelpers    *DiagnosticHelpers
	DetectedOS     string
	PrivilegeCheck bool
}

// Metadata returns the provider metadata.
func (p *PackageProvider) Metadata(
	_ context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "pkg"
	resp.Version = p.version
}

// Schema returns the provider schema.
func (p *PackageProvider) Schema(
	_ context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The pkg provider enables cross-platform package management using Homebrew, APT, winget, and Chocolatey.",
		Attributes: map[string]schema.Attribute{
			"default_manager": schema.StringAttribute{
				MarkdownDescription: "Default package manager to use. " +
					"Valid values: auto, brew, apt, winget, choco. " +
					"Defaults to 'auto' which auto-detects based on OS.",
				Optional: true,
			},
			"assume_yes": schema.BoolAttribute{
				MarkdownDescription: "Run package operations non-interactively, assuming 'yes' to all prompts. " +
					"Defaults to true.",
				Optional: true,
			},
			"sudo_enabled": schema.BoolAttribute{
				MarkdownDescription: "Enable sudo usage for operations that require elevated privileges on Unix systems. " +
					"Defaults to true.",
				Optional: true,
			},
			"brew_path": schema.StringAttribute{
				MarkdownDescription: "Path to the Homebrew binary. " +
					"If not specified, will use default system path.",
				Optional: true,
			},
			"apt_get_path": schema.StringAttribute{
				MarkdownDescription: "Path to the apt-get binary. " +
					"If not specified, will use default system path.",
				Optional: true,
			},
			"winget_path": schema.StringAttribute{
				MarkdownDescription: "Path to the winget binary. " +
					"If not specified, will use default system path.",
				Optional: true,
			},
			"choco_path": schema.StringAttribute{
				MarkdownDescription: "Path to the Chocolatey binary. " +
					"If not specified, will use default system path.",
				Optional: true,
			},
			"update_cache": schema.StringAttribute{
				MarkdownDescription: "When to update package manager cache. " +
					"Valid values: never, on_change, always. " +
					"Defaults to 'on_change'.",
				Optional: true,
			},
			"lock_timeout": schema.StringAttribute{
				MarkdownDescription: "Timeout for waiting on package manager locks (e.g., apt/dpkg). " +
					"Defaults to '10m'.",
				Optional: true,
			},
			"retry_count": schema.Int64Attribute{
				MarkdownDescription: "Number of times to retry failed operations. " +
					"Defaults to 3.",
				Optional: true,
			},
			"retry_delay": schema.StringAttribute{
				MarkdownDescription: "Delay between retry attempts (e.g., '30s', '1m'). " +
					"Defaults to '30s'.",
				Optional: true,
			},
			"fail_on_download": schema.BoolAttribute{
				MarkdownDescription: "Whether to fail immediately on download errors. " +
					"Defaults to false (retry on download failures).",
				Optional: true,
			},
			"cleanup_on_error": schema.BoolAttribute{
				MarkdownDescription: "Whether to clean up partial installations on error. " +
					"Defaults to true.",
				Optional: true,
			},
			"verify_downloads": schema.BoolAttribute{
				MarkdownDescription: "Whether to verify downloaded packages before installation. " +
					"Defaults to true.",
				Optional: true,
			},
			"checksum_validation": schema.BoolAttribute{
				MarkdownDescription: "Whether to validate package checksums when available. " +
					"Defaults to true.",
				Optional: true,
			},
		},
	}
}

// Configure configures the provider with user settings.
func (p *PackageProvider) Configure(
	ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data PackageProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Set defaults for optional values
	if data.DefaultManager.IsNull() {
		data.DefaultManager = types.StringValue("auto")
	}
	if data.AssumeYes.IsNull() {
		data.AssumeYes = types.BoolValue(true)
	}
	if data.SudoEnabled.IsNull() {
		data.SudoEnabled = types.BoolValue(true)
	}
	if data.UpdateCache.IsNull() {
		data.UpdateCache = types.StringValue("on_change")
	}
	if data.LockTimeout.IsNull() {
		data.LockTimeout = types.StringValue("10m")
	}

	// Set defaults for error handling
	if data.RetryCount.IsNull() {
		data.RetryCount = types.Int64Value(3)
	}
	if data.RetryDelay.IsNull() {
		data.RetryDelay = types.StringValue("30s")
	}
	if data.FailOnDownload.IsNull() {
		data.FailOnDownload = types.BoolValue(false)
	}
	if data.CleanupOnError.IsNull() {
		data.CleanupOnError = types.BoolValue(true)
	}
	if data.VerifyDownloads.IsNull() {
		data.VerifyDownloads = types.BoolValue(true)
	}
	if data.ChecksumValidation.IsNull() {
		data.ChecksumValidation = types.BoolValue(true)
	}

	// Validate configuration values
	validManagers := map[string]bool{
		"auto": true, "brew": true, "apt": true, "winget": true, "choco": true,
	}
	if !validManagers[data.DefaultManager.ValueString()] {
		resp.Diagnostics.AddError(
			"Invalid default_manager",
			"default_manager must be one of: auto, brew, apt, winget, choco",
		)
		return
	}

	validCacheOptions := map[string]bool{
		"never": true, "on_change": true, "always": true,
	}
	if !validCacheOptions[data.UpdateCache.ValueString()] {
		resp.Diagnostics.AddError(
			"Invalid update_cache",
			"update_cache must be one of: never, on_change, always",
		)
		return
	}

	// Create executor and registry
	exec := executor.NewSystemExecutor()
	reg := registry.NewDefaultRegistry()
	diagHelpers := NewDiagnosticHelpers()

	// Detect OS and perform privilege check
	detectedOS := runtime.GOOS
	privilegeCheck := false
	if data.SudoEnabled.ValueBool() {
		canElevate, err := executor.DetectPrivilegeEscalation(ctx)
		if err != nil {
			// Log warning but don't fail - privilege might not be needed
			resp.Diagnostics.AddWarning(
				"Privilege Detection Failed",
				"Could not detect privilege escalation capability: "+err.Error(),
			)
		}
		privilegeCheck = canElevate
	}

	// Create provider data
	providerData := &ProviderData{
		Executor:       exec,
		Registry:       reg,
		Config:         &data,
		DiagHelpers:    diagHelpers,
		DetectedOS:     detectedOS,
		PrivilegeCheck: privilegeCheck,
	}

	resp.DataSourceData = providerData
	resp.ResourceData = providerData
}

// Resources returns the list of resources supported by this provider.
func (p *PackageProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewPackageResource,
		NewRepositoryResource,
	}
}

// DataSources returns the list of data sources supported by this provider.
func (p *PackageProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewPackageInfoDataSource,
		NewPackageSearchDataSource,
		NewRegistryLookupDataSource,
		NewManagerInfoDataSource,
		NewInstalledPackagesDataSource,
		NewOutdatedPackagesDataSource,
		NewRepositoryPackagesDataSource,
		NewDependenciesDataSource,
		NewVersionHistoryDataSource,
		NewSecurityInfoDataSource,
	}
}

// Functions returns the list of functions supported by this provider.
func (p *PackageProvider) Functions(_ context.Context) []func() function.Function {
	return []func() function.Function{
		// No functions implemented - provider uses resources and data sources
	}
}

// New creates a new package provider instance with the specified version.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &PackageProvider{
			version: version,
		}
	}
}
