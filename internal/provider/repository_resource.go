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

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/geico-private/terraform-provider-pkg/internal/adapters"
	"github.com/geico-private/terraform-provider-pkg/internal/adapters/brew"
)

const (
	managerBrew   = "brew"
	platformDarwin = "darwin"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &RepositoryResource{}
var _ resource.ResourceWithImportState = &RepositoryResource{}

// NewRepositoryResource creates a new repository resource.
func NewRepositoryResource() resource.Resource {
	return &RepositoryResource{}
}

// RepositoryResource defines the resource implementation.
type RepositoryResource struct {
	providerData *ProviderData
}

// RepositoryResourceModel describes the resource data model.
type RepositoryResourceModel struct {
	ID      types.String `tfsdk:"id"`
	Manager types.String `tfsdk:"manager"`
	Name    types.String `tfsdk:"name"`
	URI     types.String `tfsdk:"uri"`
	GPGKey  types.String `tfsdk:"gpg_key"`
	Enabled types.Bool   `tfsdk:"enabled"`
}

// Metadata returns the resource type name.
func (r *RepositoryResource) Metadata(
		_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repo"
}

// Schema defines the resource schema.
func (r *RepositoryResource) Schema(
		_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages package repositories (Homebrew taps, APT repositories, etc.).",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Repository identifier in the format 'manager:name'.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"manager": schema.StringAttribute{
				MarkdownDescription: "Package manager for this repository. " +
					"Valid values: 'brew', 'apt', 'winget', 'choco'. " +
					"Currently only 'brew' is supported.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Repository name or identifier.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"uri": schema.StringAttribute{
				MarkdownDescription: "Repository URI. For Homebrew taps, this is the tap name (e.g., 'homebrew/cask-fonts'). " +
					"For APT, this is the repository line.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"gpg_key": schema.StringAttribute{
				MarkdownDescription: "GPG key URL or content for repository verification. " +
					"Not used for Homebrew taps.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the repository is enabled. " +
					"This is computed for most package managers.",
				Computed:            true,
			},
		},
	}
}

// Configure configures the resource with provider data.
func (r *RepositoryResource) Configure(
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

func (r *RepositoryResource) Create(
		ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RepositoryResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate and get repository manager
	repoManager, err := r.getRepositoryManager(ctx, data.Manager.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Repository Manager Error", err.Error())
		return
	}

	// Add the repository
	name := data.Name.ValueString()
	uri := data.URI.ValueString()
	gpgKey := data.GPGKey.ValueString()

	if err := repoManager.AddRepository(ctx, name, uri, gpgKey); err != nil {
		resp.Diagnostics.AddError(
			"Repository Addition Failed",
			fmt.Sprintf("Failed to add repository %s: %v", name, err),
		)
		return
	}

	// Set computed values
	data.ID = types.StringValue(fmt.Sprintf("%s:%s", data.Manager.ValueString(), name))
	data.Enabled = types.BoolValue(true)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RepositoryResource) Read(
		ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RepositoryResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get repository manager
	repoManager, err := r.getRepositoryManager(ctx, data.Manager.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Repository Manager Error", err.Error())
		return
	}

	// List repositories to check if this one still exists
	repositories, err := repoManager.ListRepositories(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to List Repositories",
			fmt.Sprintf("Failed to list repositories: %v", err),
		)
		return
	}

	// Check if our repository is in the list
	found := false
	for _, repo := range repositories {
		if repo.Name == data.Name.ValueString() {
			found = true
			data.Enabled = types.BoolValue(repo.Enabled)
			break
		}
	}

	if !found {
		// Repository was removed outside of Terraform
		resp.Diagnostics.AddWarning(
			"Repository Not Found",
			fmt.Sprintf("Repository '%s' was not found. It may have been removed outside of Terraform.", data.Name.ValueString()),
		)
		resp.State.RemoveResource(ctx)
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RepositoryResource) Update(
		ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// For most package managers, repositories are immutable
	// Any changes require replacement (handled by plan modifiers)
	resp.Diagnostics.AddError(
		"Repository Update Not Supported",
		"Repository attributes cannot be updated. Changes require resource replacement.",
	)
}

func (r *RepositoryResource) Delete(
		ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RepositoryResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get repository manager
	repoManager, err := r.getRepositoryManager(ctx, data.Manager.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Repository Manager Error", err.Error())
		return
	}

	// Remove the repository
	name := data.Name.ValueString()
	if err := repoManager.RemoveRepository(ctx, name); err != nil {
		resp.Diagnostics.AddError(
			"Repository Removal Failed",
			fmt.Sprintf("Failed to remove repository %s: %v", name, err),
		)
		return
	}
}

func (r *RepositoryResource) ImportState(
		ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: "manager:repository_name"
	parts := strings.SplitN(req.ID, ":", 2)

	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Import ID must be in format 'manager:repository_name' (e.g., 'brew:homebrew/cask-fonts')",
		)
		return
	}

	manager := parts[0]
	name := parts[1]

	// Validate manager
	if manager != managerBrew {
		resp.Diagnostics.AddError(
			"Unsupported Manager",
			fmt.Sprintf("Only 'brew' manager is supported in Phase 2, got: %s", manager),
		)
		return
	}

	// Set the attributes
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("manager"), manager)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), name)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uri"), name)...)
}

// Helper methods

func (r *RepositoryResource) getRepositoryManager(_ context.Context, managerName string) (adapters.RepositoryManager, error) {
	// Only support Homebrew in Phase 2
	if managerName != managerBrew {
		return nil, fmt.Errorf("only 'brew' manager is supported in Phase 2, got: %s", managerName)
	}

	// Check OS compatibility
	if runtime.GOOS != platformDarwin {
		return nil, fmt.Errorf("homebrew is only supported on macOS, current OS: %s", runtime.GOOS)
	}

	// Create Homebrew repository manager
	brewPath := r.providerData.Config.BrewPath.ValueString()
	repoManager := brew.NewBrewRepositoryManager(r.providerData.Executor, brewPath)

	return repoManager, nil
}
