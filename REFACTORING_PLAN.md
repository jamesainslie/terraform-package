# Terraform Package Provider - Refactoring Plan for Service Management Support

## Executive Summary

This refactoring plan addresses the critical bug where the `pkg_service_status` data source and `pkg_service` resource are not recognized during `terraform plan`, blocking declarative service management in configurations like the `terraform-devenv` module. The root cause is incomplete implementation, schema registration gaps, and potential build artifacts in the development override setup.

The plan follows Test-Driven Development (TDD) principles: write tests first, ensure >80% coverage, address linters, and make atomic commits using conventional commits (e.g., `feat(provider): register service resource schema`). It builds on the existing partial implementations in `internal/provider/` and `internal/services/`, expanding to full CRUD support for services (start/stop/enable/disable, health checks) integrated with package management (e.g., install Colima → manage colima service).

**Estimated Timeline**: 1-2 weeks (solo developer), assuming macOS focus first (Homebrew/launchd), with cross-platform extensions deferred to post-v0.3.0.
- Week 1: Complete schema, Read/Create/Update/Delete (CRUD), unit tests.
- Week 2: Acceptance tests, integration with `pkg_package`, docs/examples, release prep.

**Success Criteria**:
- `terraform plan` succeeds without schema errors for the repro config.
- Full lifecycle: Install package → query status → manage service (start/enable with custom commands/health check) → verify state (e.g., `running=true`, `healthy=true`).
- Tests pass (unit, integration, acceptance) with no regressions.
- No breaking changes; backward-compatible with existing `pkg_package`/`pkg_repo`.

## Current State Analysis

### Existing Code Review
- **Provider Registration** (`internal/provider/provider.go`):
  - Resources: Includes `NewServiceResource` (partial).
  - Data Sources: Includes `NewServiceStatusDataSource` and `NewServicesOverviewDataSource`.
  - Issue: Factories registered, but implementations may lack full schema or have conditional skips (e.g., Phase 2 macOS-only).
- **Service Resource** (`internal/provider/service_resource.go`):
  - Partial: Basic schema for `service_name`, `state` (running/stopped), `startup` (enabled/disabled), but missing `custom_commands`, `management_strategy`, `wait_for_healthy`, `health_check` block.
  - CRUD: Stubs for `Create`, `Read`, `Update`, `Delete`; no integration with service detectors or package dependencies.
  - Computed Fields: Partial (e.g., `running`, `healthy`); missing `ports`, `start_time`, `process_id`.
- **Service Status Data Source** (`internal/provider/service_status_data_source.go`):
  - Schema: Basic (`name`, `required_package`, `package_manager`, `timeout`, `health_check` block).
  - Read: Stub; uses `services.ServiceDetector` but no full status query (e.g., `colima status`).
  - Outputs: Missing computed attrs like `ports`, `start_time`.
- **Services Module** (`internal/services/`):
  - Strong: Platform detectors (Darwin: launchd/brew services; Linux: systemd; Windows: sc.exe), health checker (command/HTTP), mapping (e.g., colima → colima package).
  - Gaps: Custom commands not supported; health checks basic (no timeout/status assertion).
- **Adapters Integration**: `pkg_service` needs to depend on `pkg_package` (e.g., ensure Colima installed before starting service).
- **Tests**:
  - Unit: Basic for detectors/health checker; none for resource/data source.
  - Acceptance: None for services; existing `pkg_package` tests macOS-only.
- **Build/Dev Override**: Likely outdated binary; `go build` needed after changes.
- **Docs/Examples**: Partial in `docs/resources/service.md` and examples/; no Colima-specific.

### Root Causes Confirmed
1. **Schema Gaps**: Missing attributes/blocks in resource/data source models (e.g., no `custom_commands` map).
2. **Implementation Stubs**: Read/Create/etc. methods throw errors or return empty state.
3. **Registration**: Factories exist but may not be imported correctly (check `main.go` imports).
4. **Platform Limits**: Phase 2 code skips non-macOS; remove for service ops.
5. **Build Artifacts**: Dev override loads old binary; ensure `go install` or Makefile rebuild.

## Refactoring Strategy

Adopt incremental refactoring: Fix schema first (to unblock plan), then CRUD, then integration/tests. Use dependency injection (ProviderData with executor, detectors). Ensure idempotency (check state before ops), non-interactive (brew services -q), and error diagnostics (e.g., "Service colima not found; install package first").

### Phase 1: Schema and Registration (Day 1-2)
- **Goal**: Unblock `terraform plan`; validate config without execution errors.
- **Steps**:
  1. **Update Resource Schema** (`internal/provider/service_resource.go`):
     - Model: Add `custom_commands` (map[string]list[string]), `management_strategy` (string: "direct_command", "brew_services", "systemd"), `wait_for_healthy` (bool), `wait_timeout` (string).
     - Health Check Block: Nested object with `command` (string), `http_endpoint` (string), `expected_status` (int), `timeout` (string).
     - Computed: `id` (string), `running` (bool), `healthy` (bool), `version` (string), `process_id` (string), `start_time` (string), `ports` (list[int]), `manager_type` (string).
     - Validation: Ensure `service_name` matches mapping; require `package_name` if `validate_package=true`.
     - MarkdownDescription: Detailed for each attr (e.g., "Custom start command for direct management, e.g., ['colima', 'start', '--cpu', '4']").
  2. **Update Data Source Schema** (`internal/provider/service_status_data_source.go`):
     - Similar to resource: Input `name`, `required_package`, `package_manager`, `timeout`; optional `health_check`.
     - Computed: Same as resource (focus on status query).
  3. **Registration Verification** (`internal/provider/provider.go`):
     - Ensure `Resources()` includes `NewServiceResource()`.
     - `DataSources()` includes `NewServiceStatusDataSource()`.
     - Import: Add `internal/provider` imports in `main.go` if missing.
  4. **Remove Phase 2 Limits**:
     - In Configure/Read: Allow "darwin" only initially; comment non-macOS errors.
  5. **Build and Test**:
     - `make build && terraform init -upgrade` in repro workspace.
     - `terraform plan`: Should validate schema; no "Invalid data source/resource" errors.
     - Commit: `fix(provider): complete service schema registration`.

### Phase 2: Implement CRUD Operations (Day 3-5)
- **Goal**: Enable full lifecycle; support repro config (query status, manage Colima).
- **Service Resource** (`pkg_service`):
  1. **Create**:
     - Validate: Check package installed via `pkg_package` dependency or detector.
     - Install Package: If `validate_package=true`, trigger `pkg_package` import or error.
     - Start/Enable: Use detector (Darwin: `brew services start ${service_name}` or custom command).
     - Health Wait: Poll `health_check` until healthy or timeout.
     - State: Set computed fields via `Read`.
  2. **Read**:
     - Query Status: Use `ServiceDetector.Status(ctx, service_name)` (implement if missing: parse `brew services list` or `colima status`).
     - Populate Computed: `running` from status, `healthy` from health checker, `ports` (parse output, e.g., netstat), `process_id`/`start_time` (ps/aux).
     - Drift Detection: Compare desired `state`/`startup` vs actual.
  3. **Update**:
     - State Change: Stop/start/restart based on diff (e.g., running→stopped: `brew services stop`).
     - Config Change: Restart if `custom_commands` or health check updated.
     - Re-validate Package: If version changed.
  4. **Delete**:
     - Stop/Disable: `brew services stop/disable`.
     - Optional Cleanup: Remove package if `cleanup_package=true` (new attr).
     - Idempotent: No-op if absent.
  5. **Import**:
     - Detect Existing: Read status; populate state from actual (e.g., import colima service).
- **Service Status Data Source** (`pkg_service_status`):
  1. **Read**:
     - Similar to resource Read: Query detector, run health check, populate computed.
     - No mutations; pure query.
     - Depends_On: Support `depends_on` for package (e.g., wait for install).
- **Integration with Services Module**:
  - Inject Detector: In ProviderData, add `services.ServiceDetector` (NewServiceDetector(executor)).
  - Custom Commands: Map to detector methods (e.g., `StartWithArgs(ctx, args []string)`).
  - Health Checker: Enhance to support HTTP (curl), command (exec), with assertion (exit 0 or status==200).
  - Logging: Slog.Info("Starting service", slog.String("name", service_name), slog.String("strategy", strategy)).
- **Error Handling**:
  - Diagnostics: User-friendly (e.g., "Service colima failed health check: command 'colima status' exited 1").
  - Retries: Use provider config (retry_count/delay) for transient failures (e.g., startup race).
  - Privileges: Check `sudo_enabled`; fallback to warning if needed.
- **Commit**: `feat(service): implement full CRUD for pkg_service resource`.

### Phase 3: Testing and Validation (Day 6-8)
- **Goal**: Ensure reliability; cover repro and edge cases.
- **Unit Tests** (`*_test.go`):
  - Resource: Mock detector/executor; test Create (success/fail), Read (running/stopped), Update (state change), Delete.
  - Data Source: Mock Read; test status query, health fail.
  - Coverage: >80% (use `make test-coverage`).
- **Integration Tests** (`internal/provider/integration/`):
  - Executor mocks for brew/colima commands.
  - Test custom commands: e.g., start with args, verify output.
- **Acceptance Tests** (`internal/provider/acceptance_test.go`):
  - Add TestAccServiceResource_Colima: Install pkg_package.colima → create pkg_service.colima (running/enabled) → query data.pkg_service_status.colima → update (restart) → destroy.
  - TestAccServiceStatusDataSource: Query before/after start; health check pass/fail.
  - Skip: Non-macOS (t.Skip if !darwin).
  - Use repro config: Extract to test fixture.
- **Repro Validation**:
  - In `/Volumes/Development/dotfiles/terraform-devenv`: `terraform plan/apply`; verify Colima starts with custom args (CPU/memory), Docker depends_on succeeds.
  - Debug: `TF_LOG=DEBUG` for traces.
- **Linter/Security**: `make lint && make security`; fix issues (no suppressions).
- **Commit**: `test(service): add comprehensive unit and acceptance tests`.

### Phase 4: Documentation, Polish, and Release (Day 9-10)
- **Goal**: User-ready; integrate with ecosystem.
- **Documentation**:
  - Update `docs/resources/service.md`: Full schema, examples (Colima/Docker).
  - `docs/data-sources/service_status.md`: Usage, computed attrs.
  - README.md: Add service management section; link to examples/service-management/.
  - Examples: Create `examples/service-colima/main.tf` mirroring repro.
- **Enhancements**:
  - Depends_On: Auto-add for `pkg_package.${package_name}`.
  - Observability: Log service events; add metrics (e.g., startup time).
  - Cross-Platform: Stub Linux/Windows (error: "Coming in v0.4.0"); full in future.
- **Release Prep**:
  - Version: Bump to 0.3.0 (`go.mod`, CHANGELOG.md: "feat: add pkg_service resource and pkg_service_status data source").
  - Build: `make release` (binaries for darwin/amd64).
  - Registry: Update manifest; test dev override.
  - Migration: Note in docs: "Replaces local-exec for services".
- **Commit**: `docs(service): update documentation and examples` + `chore(release): prepare v0.3.0`.

## Risks and Mitigations
- **Build Issues**: Mitigation: Use `make clean build`; verify with `terraform providers`.
- **Platform-Specific Bugs**: Focus macOS; test in VM for others.
- **State Incompat**: Use plan modifiers; test import.
- **Performance**: Timeout health polls; cache status.
- **Scope**: Defer advanced (e.g., clustering) to v1.0.

## Next Steps
1. Start Phase 1: Update schemas, commit, test plan in repro.
2. Review: Self-review code; address linters.
3. Merge: To main/feature branch; create PR if collaborative.
4. Monitor: After release, track issues in terraform-devenv.

This plan ensures robust, declarative service management, unblocking the dev env workflow.
