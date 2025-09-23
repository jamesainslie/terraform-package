terraform {
  required_providers {
    pkg = {
      source = "jamesainslie/package"
    }
  }
}

# Configure the package provider
provider "pkg" {
  default_manager = "brew"
}

# Install required packages first
resource "pkg_package" "colima" {
  name    = "colima"
  state   = "present"
  manager = "brew"
}

resource "pkg_package" "docker" {
  name    = "docker"
  state   = "present"
  manager = "brew"
}

resource "pkg_package" "postgresql" {
  name    = "postgresql"
  state   = "present"
  manager = "brew"
}

# Check individual service status
data "pkg_service_status" "colima" {
  name             = "colima"
  required_package = "colima"
  package_manager  = "brew"
  timeout          = "10s"
  
  health_check {
    command = "colima status"
    timeout = "5s"
  }
  
  depends_on = [pkg_package.colima]
}

data "pkg_service_status" "docker" {
  name             = "docker"
  required_package = "docker"
  package_manager  = "brew"
  
  health_check {
    http_endpoint   = "http://localhost:2375/_ping"
    expected_status = 200
    timeout         = "5s"
  }
  
  depends_on = [
    pkg_package.docker,
    data.pkg_service_status.colima  # Docker requires Colima on macOS
  ]
}

data "pkg_service_status" "postgres" {
  name             = "postgres"
  required_package = "postgresql"
  package_manager  = "brew"
  
  health_check {
    command = "pg_isready -h localhost -p 5432"
    timeout = "5s"
  }
  
  depends_on = [pkg_package.postgresql]
}

# Check multiple services at once
data "pkg_services_overview" "development_stack" {
  services = ["colima", "docker", "postgres"]
  
  package_manager         = "brew"
  installed_packages_only = true
  health_checks           = true
  timeout                 = "30s"
  
  depends_on = [
    pkg_package.colima,
    pkg_package.docker,
    pkg_package.postgresql
  ]
}

# Conditional resource creation based on service status
resource "local_file" "docker_config" {
  count = data.pkg_service_status.docker.running && data.pkg_service_status.docker.healthy ? 1 : 0
  
  content = jsonencode({
    registry-mirrors = ["https://mirror.gcr.io"]
    insecure-registries = ["localhost:5000"]
    log-driver = "json-file"
    log-opts = {
      max-size = "10m"
      max-file = "3"
    }
  })
  
  filename = "${path.module}/docker-daemon.json"
}

# Use service status in local variables
locals {
  container_services_ready = (
    data.pkg_service_status.colima.running &&
    data.pkg_service_status.docker.running &&
    data.pkg_service_status.docker.healthy
  )
  
  database_ready = data.pkg_service_status.postgres.running && data.pkg_service_status.postgres.healthy
  
  development_environment_ready = (
    local.container_services_ready &&
    local.database_ready
  )
}

# Output service information
output "service_status" {
  description = "Status of individual services"
  value = {
    colima = {
      running      = data.pkg_service_status.colima.running
      healthy      = data.pkg_service_status.colima.healthy
      version      = data.pkg_service_status.colima.version
      process_id   = data.pkg_service_status.colima.process_id
      manager_type = data.pkg_service_status.colima.manager_type
    }
    docker = {
      running      = data.pkg_service_status.docker.running
      healthy      = data.pkg_service_status.docker.healthy
      version      = data.pkg_service_status.docker.version
      manager_type = data.pkg_service_status.docker.manager_type
    }
    postgres = {
      running      = data.pkg_service_status.postgres.running
      healthy      = data.pkg_service_status.postgres.healthy
      version      = data.pkg_service_status.postgres.version
      manager_type = data.pkg_service_status.postgres.manager_type
    }
  }
}

output "services_overview" {
  description = "Overview of all development services"
  value = {
    total_services   = data.pkg_services_overview.development_stack.total_services
    running_services = data.pkg_services_overview.development_stack.running_services
    healthy_services = data.pkg_services_overview.development_stack.healthy_services
    summary         = data.pkg_services_overview.development_stack.summary
    all_ready       = local.development_environment_ready
  }
}

output "conditional_resources" {
  description = "Status of conditionally created resources"
  value = {
    docker_config_created = length(resource.local_file.docker_config) > 0
    container_stack_ready = local.container_services_ready
    database_ready        = local.database_ready
  }
}
