# Terraform Package Provider

A cross-platform Terraform provider for managing packages across macOS (Homebrew), Linux (APT), and Windows (winget/Chocolatey) with unified resource definitions and consistent behavior.

[![CI/CD Pipeline](https://github.com/jamesainslie/terraform-package/workflows/CI%2FCD%20Pipeline/badge.svg?branch=main)](https://github.com/jamesainslie/terraform-package/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/jamesainslie/terraform-package)](https://goreportcard.com/report/github.com/jamesainslie/terraform-package)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Release](https://github.com/jamesainslie/terraform-package/workflows/Release/badge.svg?branch=main)](https://github.com/jamesainslie/terraform-package/actions)
[![Latest Release](https://img.shields.io/github/v/release/jamesainslie/terraform-package)](https://github.com/jamesainslie/terraform-package/releases)

## Features

- **Cross-Platform Package Management**: Unified interface for Homebrew, APT, winget, and Chocolatey
- **Smart Package Resolution**: Automatic package name mapping across platforms
- **Version Management**: Support for exact versions, ranges, and pinning
- **Drift Detection**: Automatic detection of package state changes
- **Repository Management**: Support for custom package repositories and taps
- **Comprehensive Discovery**: 10 data sources for package information and auditing
- **Security-First**: Proper privilege handling and input validation

## Current Status

**macOS Support Complete**: Full Homebrew integration available

- ✅ **macOS (Homebrew)**: Complete support for formulas and casks
- 🔄 **Linux (APT)**: Planned for next milestone
- 🔄 **Windows (winget/choco)**: Planned for future milestone

## Quick Start

### Installation

```hcl
terraform {
  required_providers {
    pkg = {
      source  = "geico-private/pkg"
      version = "~> 0.1"
    }
  }
}

provider "pkg" {
  default_manager = "auto"  # auto-detects based on OS
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
```

## Resources

### pkg_package

Manages package installation across platforms.

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
    linux   = "git"
    windows = "Git.Git"
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

### pkg_repo

Manages package repositories and taps.

```hcl
resource "pkg_repo" "example" {
  manager = "brew"                    # brew | apt | winget | choco
  name    = "homebrew/cask-fonts"     # repository identifier
  uri     = "homebrew/cask-fonts"     # repository URI/tap name
  gpg_key = ""                        # GPG key (APT only)
}
```

## Data Sources

### Discovery and Information

- **`pkg_package_info`**: Get detailed package information
- **`pkg_package_search`**: Search for packages
- **`pkg_registry_lookup`**: Cross-platform name resolution
- **`pkg_manager_info`**: Package manager availability and version

### Auditing and Maintenance

- **`pkg_installed_packages`**: List all installed packages
- **`pkg_outdated_packages`**: Find packages needing updates
- **`pkg_repository_packages`**: List packages from repositories
- **`pkg_dependencies`**: Package dependency analysis
- **`pkg_version_history`**: Available package versions
- **`pkg_security_info`**: Security advisory information

## Provider Configuration

```hcl
provider "pkg" {
  default_manager = "auto"                              # auto | brew | apt | winget | choco
  assume_yes      = true                                # non-interactive mode
  sudo_enabled    = true                                # enable sudo on Unix
  update_cache    = "on_change"                         # never | on_change | always
  lock_timeout    = "10m"                               # package manager lock timeout
  
  # Custom binary paths (optional)
  brew_path       = "/opt/homebrew/bin/brew"
  apt_get_path    = "/usr/bin/apt-get"
  winget_path     = "C:\\Windows\\System32\\winget.exe"
  choco_path      = "C:\\ProgramData\\chocolatey\\bin\\choco.exe"
}
```

## Examples

See the [`examples/`](./examples/) directory for complete usage examples:

- **Basic Usage**: Simple package installation
- **Cross-Platform**: Platform-specific configurations
- **Advanced**: Repository management and version pinning
- **Discovery**: Using data sources for package discovery

## Development

### Prerequisites

- Go 1.21+
- Terraform 1.5+
- Platform-specific package managers (brew, apt, winget, choco)

### Building

```bash
# Clone the repository
git clone https://github.com/jamesainslie/terraform-package.git
cd terraform-package

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

### Current Milestone - macOS Support ✅
- Complete Homebrew integration
- Package and repository management
- Comprehensive data sources

### Next Milestone - Linux Support 🔄
- APT package manager integration
- Ubuntu/Debian support
- Repository and GPG key management

### Future Milestone - Windows Support 🔄
- winget and Chocolatey integration
- Windows package management
- Elevation handling

### Advanced Features Milestone 🔄
- Performance optimization
- Enterprise features
- Additional package managers

## Support

- **Documentation**: [Provider Documentation](./docs/)
- **Examples**: [Usage Examples](./examples/)
- **Issues**: [GitHub Issues](https://github.com/jamesainslie/terraform-package/issues)
- **Discussions**: [GitHub Discussions](https://github.com/jamesainslie/terraform-package/discussions)

## Acknowledgments

Built with:
- [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework)
- [GoReleaser](https://goreleaser.com/) for multi-platform builds
- [GitHub Actions](https://github.com/features/actions) for CI/CD

---

**Made with ❤️ for the Terraform community**