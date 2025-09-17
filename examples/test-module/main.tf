terraform {
  required_providers {
    pkg = {
      source = "jamesainslie/pkg"
    }
  }
}

provider "pkg" {
  default_manager = "auto"
  assume_yes      = true
  sudo_enabled    = true
  update_cache    = "on_change"
}

# Test data sources first
data "pkg_manager_info" "current" {
  manager = "auto"
}

data "pkg_registry_lookup" "hello" {
  logical_name = "hello"
}

data "pkg_package_info" "hello_info" {
  name    = "hello"
  manager = "brew"
}

# Install hello package (GNU Hello - safe test package)
resource "pkg_package" "hello" {
  name  = "hello"
  state = "present"
  pin   = false

  timeouts {
    create = "10m"
    read   = "2m"
    update = "10m"
    delete = "5m"
  }
}

# Output information
output "manager_info" {
  description = "Information about the detected package manager"
  value = {
    detected_manager = data.pkg_manager_info.current.detected_manager
    available        = data.pkg_manager_info.current.available
    version          = data.pkg_manager_info.current.version
    path             = data.pkg_manager_info.current.path
    platform         = data.pkg_manager_info.current.platform
  }
}

output "registry_lookup" {
  description = "Cross-platform package name resolution for hello"
  value = {
    found   = data.pkg_registry_lookup.hello.found
    darwin  = data.pkg_registry_lookup.hello.darwin
    linux   = data.pkg_registry_lookup.hello.linux
    windows = data.pkg_registry_lookup.hello.windows
  }
}

output "package_info" {
  description = "Information about hello package"
  value = {
    installed          = data.pkg_package_info.hello_info.installed
    version            = data.pkg_package_info.hello_info.version
    available_versions = data.pkg_package_info.hello_info.available_versions
    pinned             = data.pkg_package_info.hello_info.pinned
    repository         = data.pkg_package_info.hello_info.repository
  }
}

output "hello_package" {
  description = "Information about the managed hello package"
  value = {
    id             = pkg_package.hello.id
    name           = pkg_package.hello.name
    state          = pkg_package.hello.state
    version_actual = pkg_package.hello.version_actual
    pin            = pkg_package.hello.pin
  }
}
