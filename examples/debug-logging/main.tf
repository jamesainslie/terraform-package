# Example configuration to demonstrate DEBUG logging
# Run with: TF_LOG=DEBUG terraform apply

terraform {
  required_providers {
    pkg = {
      source = "jamesainslie/package"
    }
  }
}

provider "pkg" {
  default_manager = "auto"
  assume_yes      = true
}

# This will demonstrate package installation logging
resource "pkg_package" "debug_example" {
  name  = "jq" # Simple package that exists on most systems
  state = "present"

  # This will show package manager resolution and installation logic
  managers = ["brew", "apt"]
}

# This will demonstrate service management logging (if Colima is available)
resource "pkg_service" "debug_service" {
  service_name = "colima"
  state        = "running"

  # This will show service strategy selection and execution
  management_strategy = "direct_command"

  custom_commands {
    start  = ["colima", "start", "--cpu", "2", "--memory", "4"]
    stop   = ["colima", "stop"]
    status = ["colima", "status"]
  }

  # Only create if colima package exists
  depends_on = [pkg_package.debug_example]
}

# Data source to demonstrate read operations logging
data "pkg_manager_info" "debug_info" {
  manager = "auto"
}

output "debug_manager_info" {
  value = data.pkg_manager_info.debug_info
}

output "debug_package_info" {
  value = {
    id      = pkg_package.debug_example.id
    version = pkg_package.debug_example.version_actual
  }
}
