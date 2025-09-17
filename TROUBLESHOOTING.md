# Troubleshooting Guide

This guide helps resolve common issues with the Terraform Package Provider.

## Common Issues

### Provider Installation Issues

#### Issue: Provider not found
```
Error: Failed to query available provider packages
```

**Solution:**
1. Check provider source in `required_providers`:
   ```hcl
   terraform {
     required_providers {
       pkg = {
         source = "geico-private/pkg"  # Correct namespace
         version = "~> 0.1"
       }
     }
   }
   ```

2. For local development, use dev overrides:
   ```hcl
   # ~/.terraformrc
   provider_installation {
     dev_overrides {
       "geico-private/pkg" = "/path/to/terraform-provider-pkg"
     }
     direct {}
   }
   ```

#### Issue: Version constraints not satisfied
```
Error: Incompatible provider version
```

**Solution:**
- Check available versions: `terraform providers`
- Update version constraint: `version = ">= 0.1.0"`
- Clear provider cache: `rm -rf .terraform/`

### Package Manager Issues

#### Expected Error Messages (Homebrew)

When using the Homebrew adapter, you may see error messages like these in logs or test output:

```
Error: Cask 'jq' is unavailable: No Cask with this name exists.
Error: No available formula with the name 'firefox'
```

**This is NORMAL behavior** - these are not actual errors! The Homebrew adapter implements dual-type detection because packages can be either:
- **Formulae**: Command-line tools (like `jq`, `git`, `curl`)  
- **Casks**: GUI applications (like `firefox`, `chrome`, `vscode`)

The adapter automatically tries both types to determine which one applies:
1. First tries as a cask → May get "cask unavailable" error for formulae
2. Then tries as a formula → May get "formula unavailable" error for casks
3. Uses whichever succeeds

These stderr messages indicate the detection logic is working correctly and should be ignored.

#### Issue: Package manager not available
```
Error: package manager brew is not available on this system
```

**Solution:**
1. **macOS**: Install Homebrew:
   ```bash
   /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
   ```

2. **Linux**: Install APT (usually pre-installed):
   ```bash
   sudo apt-get update
   ```

3. **Windows**: Install winget (Windows 10 1709+) or Chocolatey

#### Issue: Privilege errors
```
Error: sudo required but not available
```

**Solution:**
1. **Configure sudo NOPASSWD** (recommended):
   ```bash
   echo "$USER ALL=(ALL) NOPASSWD: /usr/bin/apt-get" | sudo tee /etc/sudoers.d/terraform-pkg
   ```

2. **Disable sudo in provider**:
   ```hcl
   provider "pkg" {
     sudo_enabled = false
   }
   ```

3. **Run as privileged user** (not recommended for production)

### Package Installation Issues

#### Issue: Package not found
```
Error: package 'nonexistent' not found as formula or cask
```

**Solution:**
1. **Search for the package**:
   ```hcl
   data "pkg_package_search" "search" {
     query = "nonexistent"
   }
   ```

2. **Check package registry**:
   ```hcl
   data "pkg_registry_lookup" "lookup" {
     logical_name = "nonexistent"
   }
   ```

3. **Use platform-specific names**:
   ```hcl
   resource "pkg_package" "example" {
     name = "git"
     aliases = {
       darwin  = "git"
       windows = "Git.Git"
     }
   }
   ```

#### Issue: Version not available
```
Error: version '999.0.0' not available
```

**Solution:**
1. **Check available versions**:
   ```hcl
   data "pkg_version_history" "versions" {
     name = "git"
   }
   ```

2. **Use version ranges**:
   ```hcl
   resource "pkg_package" "example" {
     version = "2.*"  # Any 2.x version
   }
   ```

3. **Remove version constraint** for latest:
   ```hcl
   resource "pkg_package" "example" {
     version = ""  # Latest version
   }
   ```

### State Management Issues

#### Issue: State drift detected
```
Warning: Package version drift detected
```

**Solution:**
1. **Enable automatic reinstall**:
   ```hcl
   resource "pkg_package" "example" {
     reinstall_on_drift = true
   }
   ```

2. **Pin package version**:
   ```hcl
   resource "pkg_package" "example" {
     pin = true
   }
   ```

3. **Manual state refresh**:
   ```bash
   terraform refresh
   ```

#### Issue: Import state failed
```
Error: Package Manager Resolution Failed
```

**Solution:**
Use correct import format:
```bash
# Format: manager:package_name
terraform import pkg_package.example brew:git

# Or just package name (auto-detect manager)
terraform import pkg_package.example git
```

### Performance Issues

#### Issue: Operations timeout
```
Error: context deadline exceeded
```

**Solution:**
1. **Increase timeouts**:
   ```hcl
   resource "pkg_package" "example" {
     timeouts {
       create = "30m"  # Increase from default 15m
       delete = "20m"  # Increase from default 10m
     }
   }
   ```

2. **Check network connectivity**:
   ```bash
   # Test package manager connectivity
   brew update  # macOS
   apt-get update  # Linux
   ```

3. **Disable cache updates**:
   ```hcl
   provider "pkg" {
     update_cache = "never"
   }
   ```

#### Issue: Slow package operations
```
Package installation taking too long
```

**Solution:**
1. **Use bottle/binary packages** (Homebrew):
   ```bash
   brew install --force-bottle package_name
   ```

2. **Parallel operations** (where safe):
   ```hcl
   # Install multiple packages
   resource "pkg_package" "package1" { ... }
   resource "pkg_package" "package2" { ... }
   # Terraform handles parallelization
   ```

## Platform-Specific Issues

### macOS (Homebrew)

#### Issue: Xcode Command Line Tools required
```
Error: invalid active developer path
```

**Solution:**
```bash
xcode-select --install
```

#### Issue: Homebrew permissions
```
Error: Permission denied
```

**Solution:**
```bash
# Fix Homebrew permissions
sudo chown -R $(whoami) $(brew --prefix)/*
```

### Linux (APT) - Phase 3

#### Issue: Package locks
```
Error: Could not get lock /var/lib/dpkg/lock-frontend
```

**Solution:**
1. **Wait for other package operations** to complete
2. **Increase lock timeout**:
   ```hcl
   provider "pkg" {
     lock_timeout = "20m"
   }
   ```

### Windows (winget) - Phase 4

#### Issue: Elevation required
```
Error: This operation requires elevation
```

**Solution:**
1. **Run PowerShell as Administrator**
2. **Configure UAC settings** for automation
3. **Use Chocolatey** as alternative

## Debugging

### Enable Debug Logging

```bash
# Terraform debug logging
export TF_LOG=DEBUG
terraform apply

# Provider-specific logging
export TF_LOG_PROVIDER=DEBUG
terraform apply
```

### Common Debug Commands

```bash
# Check provider version
terraform version

# List providers
terraform providers

# Validate configuration
terraform validate

# Show plan details
terraform plan -out=plan.out
terraform show plan.out
```

### Debug Data Sources

```hcl
# Check manager availability
data "pkg_manager_info" "debug" {
  manager = "auto"
}

output "debug_manager" {
  value = data.pkg_manager_info.debug
}

# Check package registry
data "pkg_registry_lookup" "debug" {
  logical_name = "your-package"
}

output "debug_registry" {
  value = data.pkg_registry_lookup.debug
}
```

## Getting Support

### Before Asking for Help

1. **Check this troubleshooting guide**
2. **Search existing issues** on GitHub
3. **Enable debug logging** and collect logs
4. **Try with minimal configuration**

### When Asking for Help

Include:
- **Terraform version**: `terraform version`
- **Provider version**: From `terraform providers`
- **Operating system**: `uname -a` (Unix) or `systeminfo` (Windows)
- **Package manager version**: `brew --version`, `apt-get --version`, etc.
- **Configuration**: Minimal reproduction case
- **Error logs**: With debug logging enabled
- **Steps to reproduce**: Clear reproduction steps

### Support Channels

- **GitHub Issues**: Bug reports and feature requests
- **GitHub Discussions**: Questions and community help
- **Documentation**: Check provider and Terraform docs

## Known Limitations

### Phase 2 (Current)
- **macOS only**: Linux and Windows support planned
- **Homebrew only**: Other package managers in future phases
- **No parallel installs**: Sequential package operations

### General Limitations
- **Network required**: Package managers need internet access
- **Privileges required**: Some operations need sudo/elevation
- **Platform-specific**: Package names vary across platforms
- **Version constraints**: Limited by package manager capabilities

## Performance Tips

### Optimization Strategies

1. **Pin stable packages**:
   ```hcl
   resource "pkg_package" "stable" {
     pin = true  # Prevents unnecessary updates
   }
   ```

2. **Batch operations**:
   ```hcl
   # Group related packages
   resource "pkg_package" "dev_tools" {
     for_each = toset(["git", "jq", "curl"])
     name     = each.value
   }
   ```

3. **Cache management**:
   ```hcl
   provider "pkg" {
     update_cache = "on_change"  # Only when needed
   }
   ```

4. **Timeout tuning**:
   ```hcl
   resource "pkg_package" "large_package" {
     timeouts {
       create = "60m"  # For large packages
     }
   }
   ```

Still having issues? Check our [GitHub Issues](https://github.com/geico-private/terraform-provider-pkg/issues) or create a new one!
