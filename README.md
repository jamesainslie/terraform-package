# Terraform Package Provider

A cross-platform Terraform provider for managing packages and services across macOS (Homebrew), with planned support for Linux (APT) and Windows (winget/Chocolatey). Provides unified resource definitions and consistent behavior across platforms.

[![CI/CD Pipeline](https://github.com/jamesainslie/terraform-provider-package/actions/workflows/test.yml/badge.svg)](https://github.com/jamesainslie/terraform-provider-package/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/jamesainslie/terraform-provider-package)](https://goreportcard.com/report/github.com/jamesainslie/terraform-provider-package)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Latest Release](https://img.shields.io/github/v/release/jamesainslie/terraform-provider-package)](https://github.com/jamesainslie/terraform-provider-package/releases)

## Features

- **Package Management**: Complete macOS Homebrew integration with formulas and casks
- **Service Management**: Comprehensive service lifecycle management with health checks
- **Repository Management**: Support for Homebrew taps and custom repositories
- **Smart Package Resolution**: Automatic package name mapping across platforms
- **Version Management**: Support for exact versions, ranges, and pinning
- **Drift Detection**: Automatic detection of package and service state changes
- **Comprehensive Discovery**: 12 data sources for package information and auditing
- **Service Monitoring**: Real-time service status, health checks, and port monitoring
- **Security-First**: Proper privilege handling and input validation

## Current Status

**Phase 2: macOS Support Complete** - Full Homebrew and service management

- ✅ **macOS (Homebrew)**: Complete support for formulas and casks
- ✅ **Service Management**: Full lifecycle management for macOS services
- 🔄 **Linux (APT)**: Planned for next milestone  
- 🔄 **Windows (winget/choco)**: Planned for future milestone

## Quick Start

### Installation

```hcl
terraform {
  required_providers {
    pkg = {
      source  = "jamesainslie/package" 
      version = "~> 0.1"
    }
  }
}

provider "pkg" {
  default_manager = "auto"  # auto-detects based on OS (currently macOS only)
  assume_yes      = true    # non-interactive mode
  sudo_enabled    = true    # enable privilege escalation when needed
}
```

### Basic Usage

```hcl
# Install a package
resource "pkg_package" "git" {
  name    = "git"
  state   = "present"
  version = "2.42.*"
  pin     = false
}

# Manage a service 
resource "pkg_service" "nginx" {
  service_name = "nginx"
  state        = "running"
  startup      = "enabled"
  
  health_check = {
    type = "http"
    url  = "http://localhost:80"
    timeout = "10s"
  }
  
  wait_for_healthy = true
}

# Add a custom repository
resource "pkg_repo" "fonts" {
  manager = "brew"
  name    = "homebrew/cask-fonts"
  uri     = "homebrew/cask-fonts"
}

# Get package information
data "pkg_package_info" "git_info" {
  name = "git"
}

# Search for packages
data "pkg_package_search" "fonts" {
  query = "font"
}

# Check service status
data "pkg_service_status" "nginx_status" {
  service_name = "nginx"
}
```

## Resources

### pkg_package

Manages package installation across platforms (currently macOS/Homebrew only).

```hcl
resource "pkg_package" "example" {
  name                = "git"
  state               = "present"        # present | absent
  version             = "2.42.*"         # optional version specification
  pin                 = false            # pin at current version
  reinstall_on_drift  = true             # reinstall if version drifts
  
  # Platform-specific overrides
  aliases = {
    darwin  = "git"
    linux   = "git"      # planned
    windows = "Git.Git"  # planned
  }
  
  # Operation timeouts
  timeouts {
    create = "15m"
    read   = "2m"
    update = "15m"
    delete = "10m"
  }
}
```

### pkg_service

Manages service lifecycle and monitoring (macOS support).

```hcl
resource "pkg_service" "example" {
  service_name         = "nginx"
  state               = "running"        # running | stopped
  startup             = "enabled"        # enabled | disabled
  validate_package    = true             # validate associated package
  package_name        = "nginx"          # optional package validation
  
  # Health check configuration
  health_check = {
    type         = "http"              # http | tcp | command
    url          = "http://localhost:80"
    port         = 80
    timeout      = "10s"
    interval     = "30s"
    expected_code = 200
  }
  
  # Service management options
  wait_for_healthy    = true
  wait_timeout       = "5m"
  management_strategy = "auto"          # auto | launchd | systemd | service
}
```

### pkg_repo

Manages package repositories and taps (currently Homebrew taps only).

```hcl
resource "pkg_repo" "example" {
  manager = "brew"                    # currently only "brew" supported
  name    = "homebrew/cask-fonts"     # repository identifier
  uri     = "homebrew/cask-fonts"     # repository URI/tap name
  gpg_key = ""                        # GPG key (for future APT support)
}
```

## Data Sources

### Package Discovery and Information

- **`pkg_package_info`**: Get detailed package information and metadata
- **`pkg_package_search`**: Search for packages across repositories
- **`pkg_registry_lookup`**: Cross-platform name resolution and mapping
- **`pkg_manager_info`**: Package manager availability and version info

### Package Auditing and Maintenance  

- **`pkg_installed_packages`**: List all installed packages with versions
- **`pkg_outdated_packages`**: Find packages needing updates
- **`pkg_repository_packages`**: List packages from specific repositories
- **`pkg_dependencies`**: Package dependency analysis and resolution
- **`pkg_version_history`**: Available package versions and release info
- **`pkg_security_info`**: Security advisory information (placeholder)

### Service Management and Monitoring

- **`pkg_service_status`**: Real-time service status and health information
- **`pkg_services_overview`**: Overview of all detected and managed services

## Provider Configuration

```hcl
provider "pkg" {
  default_manager = "auto"                              # auto | brew (others planned)
  assume_yes      = true                                # non-interactive mode
  sudo_enabled    = true                                # enable sudo on Unix systems
  update_cache    = "on_change"                         # never | on_change | always
  lock_timeout    = "10m"                               # package manager lock timeout
  
  # Custom binary paths (optional)
  brew_path       = "/opt/homebrew/bin/brew"            # Custom Homebrew path
  # Future planned paths:
  # apt_get_path    = "/usr/bin/apt-get"                # Linux APT
  # winget_path     = "C:\\Windows\\System32\\winget.exe" # Windows winget
  # choco_path      = "C:\\ProgramData\\chocolatey\\bin\\choco.exe" # Windows Chocolatey
}
```

## Examples

See the [`examples/`](./examples/) directory for complete usage examples:

- **[Package Management](./examples/resources/package_management/)**: Package installation and management
- **[Service Management](./examples/service-management/)**: Service lifecycle management
- **[Service Management with Strategies](./examples/service-management-with-strategies/)**: Advanced service management
- **[Service Status Monitoring](./examples/service-status/)**: Real-time service monitoring
- **[Package Discovery](./examples/data-sources/package_discovery/)**: Using data sources for discovery
- **[Debug Logging](./examples/debug-logging/)**: Troubleshooting and debug configuration

## Development

### Prerequisites

- Go 1.25.1+
- Terraform 1.5+
- Platform-specific package managers (brew, apt, winget, choco)

### Building

```bash
# Clone the repository
git clone https://github.com/jamesainslie/terraform-provider-package.git
cd terraform-provider-package

# Build the provider
make build

# Run tests
make test

# Run acceptance tests (requires package managers)
make test-acc

# Install locally for development
make install
```

### Testing

```bash
# Unit tests (all platforms)
make test

# Acceptance tests (requires actual package managers)
TF_ACC=1 go test ./internal/provider/

# Coverage report
make test-coverage
```

## Contributing

1. **Fork the repository**
2. **Create a feature branch**: `git checkout -b feature/new-feature`
3. **Make changes and add tests**
4. **Run quality checks**: `make check`
5. **Commit using conventional commits**: `git commit -m "feat: add new feature"`
6. **Push and create pull request**

### Commit Convention

We use [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` New features
- `fix:` Bug fixes
- `docs:` Documentation changes
- `test:` Test additions or changes
- `refactor:` Code refactoring
- `chore:` Maintenance tasks

## Security

### Reporting Vulnerabilities

Please report security vulnerabilities privately to the maintainers.

### Security Features

- **Input Validation**: All user inputs are validated and sanitized
- **Privilege Handling**: Secure sudo/elevation handling
- **Non-Interactive Mode**: Prevents interactive prompts in automation
- **Timeout Protection**: All operations have configurable timeouts

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Roadmap

### ✅ Phase 2 Complete - macOS Foundation
- ✅ Complete Homebrew integration (formulas and casks)
- ✅ Package and repository management
- ✅ Service lifecycle management  
- ✅ Comprehensive data sources (12 total)
- ✅ Health checking and monitoring
- ✅ Registry sync and GPG signing

### 🔄 Phase 3 - Linux Support 
- APT package manager integration
- Ubuntu/Debian support
- Repository and GPG key management
- Systemd service management

### 🔄 Phase 4 - Windows Support 
- winget and Chocolatey integration
- Windows package management
- Windows Service management
- Elevation handling

### 🔄 Phase 5 - Advanced Features 
- Performance optimization
- Enterprise features  
- Additional package managers
- Multi-platform service orchestration

## Support

- **Documentation**: [Provider Documentation](./docs/)
- **Examples**: [Usage Examples](./examples/)
- **Issues**: [GitHub Issues](https://github.com/jamesainslie/terraform-provider-package/issues)
- **Discussions**: [GitHub Discussions](https://github.com/jamesainslie/terraform-provider-package/discussions)

## Acknowledgments

Built with:
- [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework)
- [GoReleaser](https://goreleaser.com/) for multi-platform builds
- [GitHub Actions](https://github.com/features/actions) for CI/CD

---

**Made with  for the Terraform community**