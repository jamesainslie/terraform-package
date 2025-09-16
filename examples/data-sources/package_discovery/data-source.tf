terraform {
  required_providers {
    pkg = {
      source = "geico-private/pkg"
    }
  }
}

provider "pkg" {
  default_manager = "auto"
}

# Example: Get package manager information
data "pkg_manager_info" "current" {
  manager = "auto"
}

# Example: Look up cross-platform package names
data "pkg_registry_lookup" "docker" {
  logical_name = "docker"
}

# Example: Get detailed package information
data "pkg_package_info" "git" {
  name    = "git"
  manager = "brew"
}

# Example: Search for packages
data "pkg_package_search" "development_tools" {
  query   = "git"
  manager = "brew"
}

# Example: List installed packages
data "pkg_installed_packages" "all" {
  manager = "brew"
  filter  = "*" # all packages
}

# Example: Find outdated packages
data "pkg_outdated_packages" "updates_needed" {
  manager = "brew"
}

# Example: Get package dependencies
data "pkg_dependencies" "node_deps" {
  name    = "node"
  manager = "brew"
  type    = "runtime"
}

# Example: Check available versions
data "pkg_version_history" "python_versions" {
  name    = "python"
  manager = "brew"
}

# Outputs to demonstrate data source results
output "manager_info" {
  description = "Information about the detected package manager"
  value = {
    detected_manager = data.pkg_manager_info.current.detected_manager
    available        = data.pkg_manager_info.current.available
    version          = data.pkg_manager_info.current.version
    platform         = data.pkg_manager_info.current.platform
  }
}

output "cross_platform_names" {
  description = "Cross-platform package names for docker"
  value = {
    found   = data.pkg_registry_lookup.docker.found
    darwin  = data.pkg_registry_lookup.docker.darwin
    linux   = data.pkg_registry_lookup.docker.linux
    windows = data.pkg_registry_lookup.docker.windows
  }
}

output "git_package_info" {
  description = "Detailed information about git package"
  value = {
    installed          = data.pkg_package_info.git.installed
    version            = data.pkg_package_info.git.version
    available_versions = data.pkg_package_info.git.available_versions
    repository         = data.pkg_package_info.git.repository
  }
}