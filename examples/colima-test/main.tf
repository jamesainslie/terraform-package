terraform {
  required_providers {
    pkg = {
      source  = "jamesainslie/package"
      version = "~> 0.1"
    }
  }
}

provider "pkg" {
  debug = true
}

# Test Colima service management with direct command strategy
resource "pkg_service" "colima" {
  service_name        = "colima"
  state               = "running"
  startup             = "disabled"
  management_strategy = "direct_command"

  custom_commands = {
    start   = ["colima", "start"]
    stop    = ["colima", "stop"]
    restart = ["colima", "restart"]
    status  = ["colima", "status"]
  }

  wait_for_healthy = true
  wait_timeout     = "60s"

  timeout = "30s"
}

output "colima_status" {
  value = {
    running      = pkg_service.colima.running
    healthy      = pkg_service.colima.healthy
    enabled      = pkg_service.colima.enabled
    manager_type = pkg_service.colima.manager_type
    metadata     = pkg_service.colima.metadata
  }
}
