# Terraform Package Provider Feature Requirements

## Document Overview

This document outlines the feature requirements for enhancing the `jamesainslie/package` Terraform provider based on real-world usage in the terraform-devenv module implementation. These requirements address current limitations and gaps that force users to resort to local-exec provisioners or workarounds.

## Current Provider Status

**Provider**: `jamesainslie/package`  
**Current Version**: v0.1.16  
**Primary Function**: Cross-platform package management via Homebrew, APT, YUM, etc.  
**GitHub Repository**: [jamesainslie/terraform-provider-package](https://github.com/jamesainslie/terraform-provider-package)

## Critical Feature Requirements

### 1. 🍺 Homebrew Cask Support (HIGH PRIORITY)

**Current Limitation:**
```hcl
# CURRENTLY BROKEN - Forces local-exec usage
resource "null_resource" "cask" {
  provisioner "local-exec" {
    command = "brew install --cask cursor"
  }
}
```

**Required Enhancement:**
```hcl
# DESIRED API
resource "pkg_package" "gui_apps" {
  name         = "cursor"
  state        = "present"
  package_type = "cask"  # NEW: Specify package type
  manager      = "brew"
}

# OR separate resource type
resource "pkg_cask" "cursor" {
  name    = "cursor"
  state   = "present"
  manager = "brew"
}
```

**Implementation Requirements:**
- Support for `brew install --cask` operations
- Cask-specific state tracking and drift detection
- Handle cask-specific update and removal operations
- Support for cask options and configurations
- Integration with existing package state management

**Impact:** Eliminates 2 local-exec provisioners in terraform-devenv module

### 2. 📦 Enhanced Package Type Support

**Current Limitation:**
Limited to basic package installation without type awareness.

**Required Enhancement:**
```hcl
resource "pkg_package" "development_tools" {
  name         = "docker"
  state        = "present"
  package_type = "formula"  # formula | cask | tap_package
  
  # Package type specific options
  cask_options = var.cask_options  # For casks only
  tap_source   = var.tap_name      # For tap packages only
  
  # Enhanced metadata
  category = "containers"  # For organization
  critical = true         # For dependency ordering
}
```

**Package Types to Support:**
- `formula` - Standard Homebrew packages
- `cask` - GUI applications
- `tap_package` - Packages from third-party taps
- `service` - Packages that include services (future)

### 3. 🔧 Dependency Management and Ordering

**Current Limitation:**
No built-in dependency resolution or installation ordering.

**Required Enhancement:**
```hcl
resource "pkg_package" "docker" {
  name  = "docker"
  state = "present"
  
  # NEW: Dependency management
  dependencies = [
    "colima",
    "ca-certificates"
  ]
  
  # NEW: Installation ordering
  install_priority = 10  # Higher numbers install first
  
  # NEW: Dependency behavior
  dependency_strategy = "install_missing"  # install_missing | require_existing | ignore
}
```

**Features:**
- Automatic dependency resolution
- Installation ordering based on priority
- Configurable dependency handling strategies
- Circular dependency detection

### 4. 🚨 Error Handling and Recovery

**Current Issue:**
```
Error while installing jamesainslie/package v0.1.14: zip: not a valid zip file
```

**Required Enhancement:**
```hcl
provider "pkg" {
  # NEW: Error handling configuration
  error_handling = {
    retry_count      = 3
    retry_delay      = "30s"
    fail_on_download = false
    cleanup_on_error = true
  }
  
  # NEW: Download validation
  verify_downloads = true
  checksum_validation = true
}

resource "pkg_package" "example" {
  name  = "terraform"
  state = "present"
  
  # NEW: Package-specific error handling
  error_handling = {
    ignore_version_conflicts = false
    rollback_on_failure     = true
    max_install_time       = "10m"
  }
}
```

**Features:**
- Automatic retry logic for transient failures
- Download verification and checksum validation
- Rollback capabilities on failed installations
- Detailed error reporting and logging

### 5. 📊 Enhanced State Management

**Current Limitation:**
Limited visibility into package state and drift detection.

**Required Enhancement:**
```hcl
resource "pkg_package" "monitored_package" {
  name  = "terraform"
  state = "present"
  
  # NEW: Enhanced state tracking
  track_metadata = true
  track_dependencies = true
  track_usage = true
  
  # NEW: Drift detection options
  drift_detection = {
    check_version     = true
    check_integrity   = true
    check_dependencies = true
    remediation      = "auto"  # auto | manual | warn
  }
}
```

**State Information to Track:**
- Installation source and method
- Dependency tree and relationships
- Usage statistics and last access
- Configuration file associations
- Security vulnerability status

### 6. 🔒 Security and Compliance Features

**Required Enhancement:**
```hcl
resource "pkg_package" "secure_package" {
  name  = "gnupg"
  state = "present"
  
  # NEW: Security features
  security = {
    verify_signatures    = true
    check_vulnerabilities = true
    audit_installation   = true
    compliance_tags     = ["sox", "hipaa"]
  }
  
  # NEW: Source validation
  trusted_sources_only = true
  allowed_repositories = ["homebrew/core"]
}
```

**Security Features:**
- Package signature verification
- Vulnerability scanning integration
- Audit logging for compliance
- Source repository restrictions
- Security policy enforcement

### 7. 🌐 Cross-Platform Enhancements

**Current Limitation:**
Basic cross-platform support with limited platform-specific features.

**Required Enhancement:**
```hcl
resource "pkg_package" "cross_platform_tool" {
  name = "terraform"
  state = "present"
  
  # NEW: Platform-specific configuration
  platform_config = {
    macos = {
      manager = "brew"
      options = ["--with-completion"]
    }
    linux = {
      manager = "apt"
      repository = "hashicorp"
    }
    windows = {
      manager = "chocolatey"
      source = "official"
    }
  }
  
  # NEW: Platform detection
  auto_detect_manager = true
  fallback_managers = ["snap", "flatpak"]
}
```

### 8. 📈 Performance and Optimization

**Required Enhancement:**
```hcl
provider "pkg" {
  # NEW: Performance settings
  performance = {
    parallel_installs    = 5
    cache_packages      = true
    cache_duration      = "24h"
    compress_cache      = true
    background_updates  = false
  }
  
  # NEW: Batch operations
  batch_operations = true
  batch_size      = 10
}
```

## Enhancement Requests by Priority

### 🔴 **Critical (Blocks Current Usage)**

1. **Cask Support** - Eliminates need for local-exec cask installations
2. **Error Recovery** - Fixes zip file corruption and download issues
3. **Enhanced State Tracking** - Better drift detection and remediation

### 🟡 **High Value (Improves User Experience)**

4. **Dependency Management** - Automatic dependency resolution
5. **Security Features** - Signature verification and vulnerability scanning
6. **Cross-Platform Enhancements** - Better platform detection and configuration

### 🟢 **Nice to Have (Future Enhancements)**

7. **Performance Optimization** - Faster operations and caching
8. **Advanced Configuration** - Platform-specific options and behaviors

## Implementation Suggestions

### Phase 1: Core Package Types (4-6 weeks)
- Add cask support with `package_type = "cask"`
- Implement basic dependency resolution
- Enhanced error handling and retry logic

### Phase 2: State and Security (3-4 weeks)  
- Advanced state tracking and drift detection
- Security features and signature verification
- Audit logging and compliance support

### Phase 3: Performance and Platform (2-3 weeks)
- Performance optimizations and caching
- Enhanced cross-platform support
- Batch operations and parallel installs

## Testing Requirements

```hcl
# Test cases needed for new features
resource "pkg_package" "test_cask" {
  name         = "cursor"
  package_type = "cask"
  state        = "present"
}

resource "pkg_package" "test_dependencies" {
  name = "docker"
  dependencies = ["colima"]
  dependency_strategy = "install_missing"
}

resource "pkg_package" "test_security" {
  name = "gnupg"
  security = {
    verify_signatures = true
    check_vulnerabilities = true
  }
}
```

## Documentation Needs

1. **Cask Management Guide** - How to manage GUI applications
2. **Dependency Resolution** - Best practices for package dependencies
3. **Security Configuration** - Security features and compliance
4. **Migration Guide** - Moving from local-exec to provider resources
5. **Troubleshooting** - Common issues and solutions

## Success Metrics

**Adoption Metrics:**
- 90% reduction in local-exec usage for package management
- Support for 100+ cask packages without shell commands
- Zero package installation failures due to download issues

**Performance Metrics:**
- <30s installation time for typical package sets
- 95% cache hit rate for repeated installations
- <5% resource drift in production environments

**Security Metrics:**
- 100% signature verification for security-critical packages
- Zero security vulnerabilities from package management
- Complete audit trail for compliance requirements

## Conclusion

These enhancements would transform the `jamesainslie/package` provider from a basic package installer into a comprehensive package management platform suitable for enterprise Infrastructure as Code implementations. The focus on cask support and error handling addresses immediate blockers, while the security and performance features enable advanced use cases.

**Priority Implementation Order:**
1. 🔴 **Cask Support** (immediate blocker)
2. 🔴 **Error Recovery** (reliability)  
3. 🟡 **Dependency Management** (user experience)
4. 🟡 **Security Features** (enterprise readiness)
5. 🟢 **Performance** (scale optimization)

---

*This document should be used as input for the package provider roadmap and development prioritization.*
