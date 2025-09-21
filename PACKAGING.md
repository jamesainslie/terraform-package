# Terraform Package Provider - Packaging & Distribution Guide

## Overview

This document explains how to package and distribute the Terraform Package Provider for consumption by Terraform users.

## 📦 Packaging Methods

### 1. Terraform Registry (Primary Distribution)

The **recommended approach** for public distribution:

#### **Registry Structure:**
```
registry.terraform.io/jamesainslie/package
├── versions/
│   ├── 0.1.0/
│   │   ├── download/
│   │   │   ├── darwin_amd64/
│   │   │   ├── darwin_arm64/
│   │   │   ├── linux_amd64/
│   │   │   ├── linux_arm64/
│   │   │   ├── windows_amd64/
│   │   │   └── freebsd_amd64/
│   │   └── docs/
│   └── latest/
```

#### **User Experience:**
```hcl
terraform {
  required_providers {
    package = {
      source  = "jamesainslie/package"
      version = "~> 0.1"
    }
  }
}
```

### 2. GitHub Releases (Secondary Distribution)

For direct downloads and pre-release testing:

#### **Release Assets:**
```
terraform-provider-package_0.1.0_darwin_amd64.zip
terraform-provider-package_0.1.0_darwin_arm64.zip
terraform-provider-package_0.1.0_linux_amd64.zip
terraform-provider-package_0.1.0_linux_arm64.zip
terraform-provider-package_0.1.0_windows_amd64.zip
terraform-provider-package_0.1.0_freebsd_amd64.zip
terraform-provider-package_0.1.0_SHA256SUMS
terraform-provider-package_0.1.0_SHA256SUMS.sig
terraform-provider-package_0.1.0_manifest.json
```

#### **Manual Installation:**
```bash
# Download and extract to Terraform plugin directory
mkdir -p ~/.terraform.d/plugins/jamesainslie/package/0.1.0/darwin_amd64/
unzip terraform-provider-package_0.1.0_darwin_amd64.zip -d ~/.terraform.d/plugins/jamesainslie/package/0.1.0/darwin_amd64/
```

### 3. Local Development

For development and testing:

#### **Dev Override:**
```hcl
# ~/.terraformrc
provider_installation {
  dev_overrides {
    "jamesainslie/package" = "/path/to/terraform-package"
  }
  direct {}
}
```

## 🚀 Release Process

### Automated Release (Recommended)

1. **Create and push a tag:**
   ```bash
   git tag -a v0.1.0 -m "Release v0.1.0"
   git push origin v0.1.0
   ```

2. **GitHub Actions automatically:**
   - Runs full test suite
   - Builds multi-platform binaries
   - Signs artifacts with GPG
   - Creates GitHub release
   - Publishes to Terraform Registry

### Manual Release Process

```bash
# 1. Prepare release
make clean
make test
make security

# 2. Build for all platforms
make build-all

# 3. Create release with goreleaser
goreleaser release --clean

# 4. Publish to registry (if configured)
```

## 📋 Required Files for Distribution

### **Essential Files:**
- ✅ `terraform-registry-manifest.json` - Registry metadata
- ✅ `.goreleaser.yml` - Build configuration
- ✅ `LICENSE` - MIT license
- ✅ `README.md` - Usage documentation
- ✅ `CHANGELOG.md` - Version history
- ✅ `docs/` - Generated provider documentation
- ✅ `examples/` - Usage examples

### **Security Files:**
- ✅ `terraform-provider-package_VERSION_SHA256SUMS` - Checksums
- ✅ `terraform-provider-package_VERSION_SHA256SUMS.sig` - GPG signature
- ✅ GPG public key for verification

## 🔐 Security & Signing

### **GPG Key Setup:**
```bash
# Generate GPG key for signing
gpg --full-generate-key

# Export public key
gpg --armor --export YOUR_EMAIL > public-key.asc

# Set up GitHub secrets:
# - GPG_PRIVATE_KEY: Private key content
# - PASSPHRASE: Key passphrase
```

### **Verification Process:**
```bash
# Users can verify downloads
gpg --verify terraform-provider-package_0.1.0_SHA256SUMS.sig terraform-provider-package_0.1.0_SHA256SUMS
sha256sum -c terraform-provider-package_0.1.0_SHA256SUMS
```

## 📚 Documentation Packaging

### **Generated Documentation:**
```bash
# Auto-generated from provider schema
make generate

# Creates:
docs/
├── index.md
├── resources/
│   ├── pkg_package.md
│   └── pkg_repo.md
└── data-sources/
    ├── pkg_package_info.md
    ├── pkg_package_search.md
    ├── pkg_registry_lookup.md
    ├── pkg_manager_info.md
    ├── pkg_installed_packages.md
    ├── pkg_outdated_packages.md
    ├── pkg_repository_packages.md
    ├── pkg_dependencies.md
    ├── pkg_version_history.md
    └── pkg_security_info.md
```

## 🏷️ Versioning Strategy

### **Semantic Versioning:**
- **v0.x.x**: Pre-1.0 development (breaking changes allowed)
- **v1.x.x**: Stable API (backward compatibility required)
- **Patch**: Bug fixes (0.1.1)
- **Minor**: New features (0.2.0)
- **Major**: Breaking changes (1.0.0)

### **Release Cadence:**
- **Patch releases**: As needed for bug fixes
- **Minor releases**: Monthly for new features
- **Major releases**: When breaking changes are necessary

## 🎯 Distribution Targets

### **Phase 2 (Current):**
- ✅ **macOS support** via Homebrew
- ✅ **Terraform Registry** publishing
- ✅ **GitHub Releases** for direct download

### **Phase 3 (Linux):**
- 🔄 **Linux support** via APT
- 🔄 **Multi-platform testing** in CI

### **Phase 4 (Windows):**
- 🔄 **Windows support** via winget/chocolatey
- 🔄 **Complete cross-platform** distribution

## 📊 Usage Analytics

### **Registry Analytics:**
- Download counts per version
- Platform distribution
- Geographic usage patterns

### **GitHub Analytics:**
- Release download statistics
- Issue and PR metrics
- Community engagement

## 🔧 Development Workflow

### **Local Testing:**
```bash
# Build and install locally
make install

# Test with examples
make example-plan
make example-apply
make example-destroy
```

### **CI/CD Testing:**
```bash
# Run all checks locally
make ci

# Run acceptance tests
make test-acc
```

This packaging approach ensures our provider is **production-ready** and follows **Terraform ecosystem best practices**!
