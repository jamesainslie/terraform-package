# Terraform Package Provider Design

A a clean, battle-tested way to design a Terraform "package" provider that abstracts Homebrew (macOS), APT (Ubuntu), and a Windows manager (I recommend winget first, choco as optional fallback).

## High Level Roadmap

1. Implement for MAC OSX
2. Implement for Ubuntu
3. Implement for Windows

## Reada the following documentation

https://developer.hashicorp.com/terraform/plugin/sdkv2
https://developer.hashicorp.com/terraform/plugin/framework
https://developer.hashicorp.com/terraform/plugin/sdkv2/testing
https://developer.hashicorp.com/terraform/plugin/sdkv2/schemas
https://developer.hashicorp.com/terraform/plugin/sdkv2/resources

## Implementation Reference

Used Terraform Plugin Framework documentation and best practices from:
- https://developer.hashicorp.com/terraform/plugin/framework
- https://developer.hashicorp.com/terraform/plugin/framework/resources
- https://developer.hashicorp.com/terraform/plugin/framework/data-sources

## High-level design

- **Goal**: A single package resource with consistent semantics: ensure package X is installed at version Y (optionally pinned), across OSes.
- **Approach**: Write a Terraform provider in Go (Terraform Plugin Framework) with an execution layer that dispatches to brew, apt, and winget/choco. Keep a package-name mapping so HCL uses a logical name by default, but allow per-OS overrides.
- **Philosophy**: Be idempotent, non-interactive, and state-aware. Never mutate outside Terraform without detecting drift (via Read).

## Provider surface (v1)

### Provider block

```hcl
provider "pkg" {
  # Global toggles and execution options
  default_manager = "auto"   # auto | brew | apt | winget | choco
  assume_yes      = true     # run non-interactively
  sudo_enabled    = true     # use sudo on Unix
  brew_path       = "/opt/homebrew/bin/brew"   # optional
  apt_get_path    = "/usr/bin/apt-get"         # optional
  winget_path     = "C:\\Windows\\System32\\winget.exe" # optional
  choco_path      = "C:\\ProgramData\\chocolatey\\bin\\choco.exe" # optional
  update_cache    = "on_change" # never | on_change | always
  lock_timeout    = "10m"       # apt/dpkg lock wait
}
```

### Resources

#### pkg_package

Ensures a single package is installed/uninstalled, with optional version pinning.

```hcl
resource "pkg_package" "git" {
  name           = "git"          # logical name
  state          = "present"      # present | absent
  version        = "2.46.*"       # optional (supports exact/semver/glob where feasible)
  pin            = false          # pin/hold (brew pin, apt-mark hold, winget pin not native -> emulate)
  managers       = ["auto"]       # override to force brew/apt/winget/choco

  # Per-OS overrides (optional) if names differ
  aliases = {
    darwin  = "git"               # brew formula
    linux   = "git"               # apt package
    windows = "Git.Git"           # winget id
  }

  # Advanced
  reinstall_on_drift = true        # if version drifted, force reinstall
  hold_dependencies  = false

  # Timeouts
  timeouts {
    create = "15m"
    read   = "2m"
    update = "15m"
    delete = "10m"
  }
}
```

#### pkg_repo (optional, but useful)

- **brew**: taps
- **apt**: apt repositories/keys
- **winget/choco**: usually not needed (winget uses catalogs; choco sources possible)

```hcl
resource "pkg_repo" "grafana" {
  manager = "apt"
  name    = "grafana"
  uri     = "https://apt.grafana.com stable main"
  gpg_key = "https://apt.grafana.com/gpg.key"
}
```

### Data sources

- `pkg_package_info` — discover installed version, candidate version, repo/source
- `pkg_search` — search manager catalogs (best-effort)

## Core behaviors & semantics

### 1. OS & manager detection

- Default auto chooses:
  - **darwin** → brew (formula vs cask inferred by name or fallback flag)
  - **linux** (Ubuntu/Debian) → apt
  - **windows** → winget (fallback to choco if configured)
- Allow `managers = ["apt"]` etc. to force

### 2. Name resolution

- Prefer aliases if provided
- Else look up a package map (see registry section)
- Else use name as-is

### 3. Versioning

- **APT**: exact version strings supported (from apt-cache policy), or latest if omitted. For semver/glob, resolve to the newest matching candidate
- **Brew**: brew info --json → parse versions; install specific versions via formula versioned taps where available; else fail with clear message
- **Winget**: supports --version; if not available for a package, install latest and record that exact build in state
- **Pin/Hold**:
  - APT: `apt-mark hold <pkg>` / `unhold`
  - Brew: `brew pin` / `unpin`
  - Winget: no native pin → emulate by failing plan if desired version differs and pin = true

### 4. Idempotency & drift

- **Read**: query installed version & pin/hold status. If state = "present" and not installed, Read marks "needs create." If installed but version differs:
  - If `reinstall_on_drift = true`, plan replacement (force new)
  - Else set `version_actual` attribute and let user decide (optionally surface a plan diff)
- **Update cache**:
  - `never`: skip apt-get update / brew update / winget source update
  - `on_change`: run before install when package not present or version changed
  - `always`: run on every operation

### 5. Privileges

- **APT** usually needs root. If `sudo_enabled = true`, prefix commands with `sudo -n` and provide actionable error if password needed
- **Brew** generally unprivileged, except casks affecting `/Applications`
- **Windows**: winget may need elevated shell. Detect and fail fast with guidance

### 6. Non-interactive

- Always pass flags: `-y` (apt), `--silent/--accept-source-agreements --accept-package-agreements` (winget), `--quiet` (brew where applicable)

### 7. Execution model

- Use an executor interface with context, timeouts, and structured stderr/stdout capture:

```go
type ExecResult struct { 
    Stdout, Stderr string
    ExitCode int 
}

type Executor interface {
    Run(ctx context.Context, cmd string, args []string, opts ExecOpts) (ExecResult, error)
}
```

- Implement platform adapters:
  - `brewAdapter`, `aptAdapter`, `wingetAdapter`, `chocoAdapter`
  - Each implements: `DetectInstalled()`, `Install(version)`, `Remove()`, `Pin(bool)`, `UpdateCache()`, `Search()`, `Info()`

## Minimal Go skeleton (Terraform Plugin Framework)

```go
// go.mod: require github.com/hashicorp/terraform-plugin-framework v1.x

package main

import (
    "context"
    "os"
    "github.com/hashicorp/terraform-plugin-framework/providerserver"
)

func main() {
    providerserver.Serve(context.Background(), NewProvider, providerserver.ServeOpts{
        Address: "registry.terraform.io/you/pkg",
    })
}

// provider.go
type providerModel struct {
    DefaultManager types.String `tfsdk:"default_manager"`
    AssumeYes      types.Bool   `tfsdk:"assume_yes"`
    SudoEnabled    types.Bool   `tfsdk:"sudo_enabled"`
    BrewPath       types.String `tfsdk:"brew_path"`
    AptGetPath     types.String `tfsdk:"apt_get_path"`
    WingetPath     types.String `tfsdk:"winget_path"`
    ChocoPath      types.String `tfsdk:"choco_path"`
    UpdateCache    types.String `tfsdk:"update_cache"`
    LockTimeout    types.String `tfsdk:"lock_timeout"`
}

// Configure() wires up adapters based on OS and paths. Store in provider data.

// resource_package.go (outline)
type packageResourceModel struct {
    ID               types.String `tfsdk:"id"`
    Name             types.String `tfsdk:"name"`
    State            types.String `tfsdk:"state"`
    Version          types.String `tfsdk:"version"`
    VersionActual    types.String `tfsdk:"version_actual"`
    Pin              types.Bool   `tfsdk:"pin"`
    Managers         types.List   `tfsdk:"managers"`
    Aliases          types.Map    `tfsdk:"aliases"`
    ReinstallOnDrift types.Bool   `tfsdk:"reinstall_on_drift"`
    // timeouts...
}

func (r *packageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    // Resolve manager, name → adapter
    // Optionally UpdateCache
    // Install version (best-effort resolution) with non-interactive flags
    // Optionally pin/hold
    // Read back installed version and set state.Id to manager:name
}

func (r *packageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    // Query installed; if absent and state=="present", mark not found
    // Set version_actual. Detect drift
}

func (r *packageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    // Handle version change or pin change. If reinstall_on_drift, do uninstall+install
}

func (r *packageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    // Remove package; unpin if necessary
}
```

**Tip**: In Plugin Framework, use PlanModifiers to force replacement when version changes (unless you opt into in-place upgrade).

## Package-name registry (crucial for UX)

Create a small embedded registry (and allow user overrides):

```yaml
# internal/registry/packages.yaml
git:
  darwin: git
  linux: git
  windows: Git.Git

docker:
  darwin: colima # maybe, or docker -- note brew cask vs formula
  linux: docker.io
  windows: Docker.DockerDesktop
```

- Add a data source `pkg_registry_entry` to let users discover current mappings
- Allow override file via provider setting or aliases in resource

## Edge cases & differences to smooth over

- **Brew cask vs formula**: Add `kind = "auto|formula|cask"` attribute. Try formula first, then cask if not found
- **APT virtual packages**: Resolve via `apt-cache policy <name>`; if no exact match and multiple candidates, fail with clear diagnostics
- **Winget IDs vs "names"**: Prefer stable Id (e.g., Git.Git), not display names. Provide a `pkg_search` data source to look up IDs
- **Downgrades**: APT supports if version available; Brew often does not (unless specific versioned formula); Winget inconsistent. Make downgrades opt-in with `allow_downgrade = true`
- **Holds/pins & upgrades**: Document that pin blocks future upgrades until unset

## Idempotency, drift & planning

- Read sets:
  - `version_actual` (computed)
  - `pinned_actual` (computed)
- If `state = "present"` but not installed → Terraform shows create
- If version set and differs from `version_actual`:
  - If `reinstall_on_drift = true` → Plan Replace
  - Else keep and surface drift via `version_actual`
- Use DiffSuppressFunc/PlanModifiers to avoid spurious diffs from case or formatting (e.g., 1.2.3 vs 1.2.3-1ubuntu1)

## Security & privileges

- **Unix**: Use `sudo -n` only; if it fails, return a readable error: "Run Terraform as a user with sudo NOPASSWD for apt" (or disable `sudo_enabled` and manage privilege outside Terraform)
- **Windows**: Detect elevation on start. Fail fast with guidance to run elevated PowerShell or set your agent service to run as admin

## Testing strategy

- **Unit tests**: mock adapters (no shell)
- **Acceptance tests**:
  - Ubuntu: GitHub Actions ubuntu-latest runners
  - Windows: GitHub Actions windows-latest with winget preinstalled
  - macOS: Actions macos-latest (limited concurrency, budget accordingly)
- Use environment gates to skip destructive tests on CI if needed

## Example usage

```hcl
terraform {
  required_providers {
    pkg = {
      source  = "you/pkg"
      version = "~> 0.1"
    }
  }
}

provider "pkg" {
  default_manager = "auto"
  assume_yes      = true
  sudo_enabled    = true
  update_cache    = "on_change"
}

# Install Git everywhere
resource "pkg_package" "git" {
  name    = "git"
  state   = "present"
  version = ""      # latest
  pin     = false

  aliases = {
    windows = "Git.Git"
  }
}

# Pin Node.js 20.x on Ubuntu & macOS, latest on Windows
resource "pkg_package" "node" {
  name    = "nodejs"
  version = "20.*"
  pin     = true

  aliases = {
    darwin  = "node@20"
    linux   = "nodejs"   # with repo managed separately
    windows = "OpenJS.NodeJS.LTS"
  }
}

# APT repo for NodeSource
resource "pkg_repo" "nodesource" {
  manager = "apt"
  name    = "nodesource"
  uri     = "https://deb.nodesource.com/node_20.x nodistro main"
  gpg_key = "https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key"
}
```

## Implementation checklist

- [ ] Scaffold with terraform-plugin-framework
- [ ] Provider schema + config → build Executor and chosen adapter(s)
- [ ] pkg_package resource CRUD + timeouts + plan modifiers
- [ ] pkg_repo resource per-manager subtypes
- [ ] Registry (embedded YAML/JSON) + override mechanism + aliases
- [ ] Robust parser for versions (normalize 2.46.0-1ubuntu1 vs 2.46.0)
- [ ] Logging with TF SDK diagnostics; redact sensitive output
- [ ] Docs/examples for brew/apt/winget
- [ ] Acceptance test matrix (Ubuntu, macOS, Windows)
- [ ] Release to GitHub + TF Registry

If you want, I can spin up a starter repo skeleton (provider main, resource skeletons, adapter interfaces, and a couple of acceptance tests) that compiles and installs a harmless package (jq) on Ubuntu and Windows to prove the shape.
