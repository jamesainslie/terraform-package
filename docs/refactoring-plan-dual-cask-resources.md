# Refactoring Plan: Implementing Dual Resource Types for Homebrew Cask Support

Based on the `terraform-package-provider-feature-requirements.md` document, the goal is to support Homebrew cask management in two ways: (1) enhancing the existing `pkg_package` resource with a `package_type` attribute (e.g., `"cask"`, `"formula"`, or `"auto"`), and (2) introducing a dedicated `pkg_cask` resource for cask-only operations. This dual approach provides flexibility: users can use the generic `pkg_package` for mixed workloads or `pkg_cask` for explicit GUI app management, eliminating `local-exec` provisioners.

The current local provider code (at `/Volumes/Development/terraform-provider-package`) already has partial support for casks in `pkg_package` via the `package_type` attribute and `BrewAdapter`'s `InstallWithType`/`RemoveWithType` methods (using `--cask` flags). However, to fully implement both resource types without duplication, we'll refactor for reusability while ensuring backward compatibility. The plan focuses on **no breaking changes** to `pkg_package`—cask handling via `package_type = "cask"` will continue working.

This plan is phased for incremental development, assuming Go 1.25.1 and Terraform Framework v1.15.1 (from `go.mod`). Estimated effort: 4-6 weeks for a single developer, including testing. All changes will be in the local override path.

## Phase 1: Preparation and Analysis (1 week)
1. **Audit Current Code for Reusability**:
   - Review `internal/provider/package_resource.go`: The schema already includes `package_type` (lines 227-235) with validation for `"auto"`, `"formula"`, `"cask"`. Methods like `Create`/`Update`/`Read` (lines 373-664) call `InstallWithType`/`RemoveWithType` on the manager, passing the resolved `packageType` (via `getPackageType` on lines 828-844). This is already cask-aware.
   - Review `internal/adapters/brew/adapter.go`: Cask logic is implemented in `InstallWithType` (lines 257-303, appends `--cask`), `RemoveWithType` (lines 311-348), and detection (`DetectInstalled` lines 130-151, `isCask` lines 470-497, which handles expected stderr like "Cask 'jq' unavailable").
   - Identify shared components: 
     - Common: Timeout handling (`getTimeout` lines 806-826), state reading (`readPackageState` lines 772-804), manager resolution (`resolvePackageManager` lines 698-770), dependency resolution (lines 846-936).
     - Cask-specific: Hardcode `PackageTypeCask` in new resource; no pinning for casks (already handled in `Pin` lines 350-382).
   - Check dependencies: No changes needed to `go.mod` (validators v0.18.0 can help with schema validation if adding constraints).
   - Output: Document reusable functions/structs (e.g., extract `BasePackageResource` struct with shared methods).

2. **Set Up Testing Baseline**:
   - Run existing acceptance tests (if any in `internal/provider/acceptance_test.go`).
   - Add a simple test for `pkg_package` with `package_type = "cask"` (e.g., install "cursor" cask).
   - Tools: Use `go test` and Terraform's testing framework (v1.13.3 from `go.mod`).

3. **Update TODO Tracking**:
   - I've already initialized a TODO list (via internal tool) for this refactoring. It includes tasks like analyzing reuse, creating schema, etc. I'll update it as we progress.

## Phase 2: Implement Dedicated `pkg_cask` Resource (1-2 weeks)
Create a new resource type that specializes `pkg_package` for casks, reducing boilerplate for GUI apps.

1. **Define New Resource Schema** (`internal/provider/cask_resource.go`):
   - Mirror `PackageResourceModel` but simplify:
     - Required: `name` (string), `state` (defaults to `"present"`).
     - Optional: `version`, `pin` (but warn/ignore for casks, as they don't support pinning), `dependencies`, `install_priority`, `dependency_strategy`.
     - Remove: `package_type` (hardcode to `"cask"`), `managers` (force `"brew"` on darwin).
     - Retain: Timeouts block, drift_detection block, tracking attributes (e.g., `track_metadata`).
     - Add cask-specific (if needed): `cask_options` (list of strings for flags like `--no-quarantine`).
   - MarkdownDescription: "Manages Homebrew cask installations (GUI applications like Cursor or Firefox)."
   - Validation: Use framework validators to enforce darwin OS and cask names (no formulas).
   - Reuse: Inherit from a new `BasePackageResource` struct (see Phase 3).

2. **Implement Resource Methods**:
   - `NewCaskResource()`: Returns `&CaskResource{}` (similar to `NewPackageResource` line 52).
   - `Metadata()`: Sets `TypeName = req.ProviderTypeName + "_cask"`.
   - `Configure()`: Same as `pkg_package` (sets `providerData`).
   - `Create()`/`Update()`/`Read`/`Delete`: 
     - Call shared `resolvePackageManager` but force `managerName = "brew"`.
     - In install/remove: Call `manager.InstallWithType(ctx, name, version, adapters.PackageTypeCask)` (hardcoded type).
     - Handle deps: Reuse `resolveDependencies` and install loop (lines 414-461).
     - State: Use `readPackageState` to set `version_actual`, etc.
     - ID: Format as `"brew-cask:name"` for distinction.
   - `ImportState`: Similar to `pkg_package` (line 666-694), but validate cask format.
   - Edge cases: Skip pinning if `pin=true` (add warning diagnostic); handle cask-specific errors (e.g., app already running).

3. **Register in Provider** (`internal/provider/provider.go`):
   - In `New()` or schema builder: Add `framework.NewSchemaResourceType(pkgcask.NewCaskResource(), ...)` alongside `pkg_package`.
   - No changes to provider schema (cask is resource-level).

## Phase 3: Refactor for Shared Logic in `pkg_package` (1 week)
Enhance `pkg_package` to fully leverage the dual support without duplicating code.

1. **Extract Shared Base** (`internal/provider/base_resource.go`):
   - Create `BasePackageResource` struct embedding common fields (e.g., `providerData`).
   - Move shared methods: `resolvePackageManager`, `getTimeout`, `readPackageState`, `resolveDependencies`, drift helpers (lines 697-986).
   - `pkg_package`: Embed base; override `getPackageType` to use schema value.
   - `pkg_cask`: Embed base; override to always return `PackageTypeCask`; remove `package_type` from schema.

2. **Enhance `pkg_package` for Better Cask Integration**:
   - Add validation to `package_type`: Warn if `"cask"` on non-darwin (use validators).
   - Deprecate note (optional): In docs, suggest `pkg_cask` for pure cask workflows, but keep `pkg_package` flexible.
   - Update `Pin` handling: In base, check if cask via `DetectInstalled` and skip if so.

3. **Adapter Improvements** (Minimal):
   - In `brew/adapter.go`: Add cask-specific options support (e.g., parse `cask_options` into args like `--force`).
   - No major changes—existing type-aware methods are sufficient.

## Phase 4: Documentation, Testing, and Validation (1 week)
1. **Documentation**:
   - New file: `docs/resources/cask.md` – Examples like `resource "pkg_cask" "cursor" { name = "cursor" }`.
   - Update `docs/resources/package.md`: Add section on `package_type = "cask"` with migration note from `local-exec`.
   - Provider index (`docs/index.md`): Mention dual resources.
   - Requirements doc: Update with "Implemented" status for cask support.

2. **Testing**:
   - Unit tests: `internal/provider/cask_resource_test.go` – Mock executor for install/remove; test hardcoded cask type.
   - Acceptance tests: Extend existing (e.g., `internal/provider/acceptance_test.go`) to cover `pkg_cask "cursor"` and mixed `pkg_package` (formula + cask).
   - Integration: Run on darwin with real brew (e.g., install/uninstall test cask like "google-chrome"); check drift detection.
   - Compatibility: Verify `pkg_package` with `package_type = "cask"` still works (no regressions).
   - Edge: Test deps (e.g., cask depending on formula), timeouts, absent state.

3. **Examples Update**:
   - `examples/resources/package_management/resource.tf`: Add `pkg_cask` example alongside `pkg_package`.
   - `examples/package_discovery/`: Data source for cask search.

4. **Validation and Cleanup**:
   - Lint: Run `golangci-lint` or `go fmt`.
   - Build: `go build` and test provider binary.
   - Compatibility: Ensure import works for both (`terraform import pkg_cask.brew /path/to/id`).
   - Deprecate path: In future v0.2+, consider making `pkg_cask` preferred, but no deprecation now.

## Risks and Considerations
- **Duplication**: Mitigated by base struct—aim for 80% shared code.
- **Platform Limits**: Still macOS-only (Phase 2); cross-platform cask equiv (e.g., winget apps) for later.
- **Error Handling**: Leverage existing diagnostics; add cask-specific warnings (e.g., no version pinning).
- **Performance**: No impact—operations remain sequential.
- **Migration**: Users with existing `pkg_package` casks unchanged; new users get choice.
- **Next Steps**: After plan approval, start Phase 1. I can implement via `edit_file` tools if instructed.

This plan aligns with the requirements' "Phase 1: Core Package Types" and ensures the provider evolves toward enterprise readiness.
