# Note: This provider does not currently implement ephemeral resources
# This file is kept for documentation structure compatibility

terraform {
  required_providers {
    package = {
      source = "jamesainslie/package"
    }
  }
}

provider "pkg" {
  default_manager = "auto"
}

# Instead of ephemeral resources, use our data sources for temporary queries
# Example: Temporary package search
data "pkg_package_search" "temporary_search" {
  query   = "temp-package"
  manager = "auto"
}

# Example: Temporary package information lookup
data "pkg_package_info" "temporary_info" {
  name    = "git"
  manager = "auto"
}

# Output for temporary use
output "search_results" {
  value = data.pkg_package_search.temporary_search.results
}

output "package_details" {
  value = {
    installed = data.pkg_package_info.temporary_info.installed
    version   = data.pkg_package_info.temporary_info.version
  }
}