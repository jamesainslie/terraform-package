# Example: Service Management with Multiple Strategies
# This example demonstrates how to use the pkg_service resource with different management strategies

terraform {
  required_providers {
    pkg = {
      source  = "jamesainslie/package"
      version = "~> 0.2.0"
    }
  }
}

# Example 1: Auto strategy (default) - tries multiple strategies automatically
resource "pkg_service" "postgresql" {
  service_name = "postgresql"
  state        = "running"
  startup      = "enabled"
  
  # Uses auto strategy by default, which will try:
  # 1. brew services (if supported)
  # 2. direct commands (if available)
  # 3. launchd (if applicable)
}

# Example 2: Explicit brew services strategy
resource "pkg_service" "redis" {
  service_name        = "redis"
  state              = "running"
  startup            = "enabled"
  management_strategy = "brew_services"
  
  # Explicitly use brew services management
}

# Example 3: Direct command strategy with custom commands
resource "pkg_service" "colima" {
  service_name        = "colima"
  state              = "running"
  startup            = "enabled"
  management_strategy = "direct_command"
  
  # Custom commands for Colima
  custom_commands = {
    start   = ["colima", "start"]
    stop    = ["colima", "stop"]
    restart = ["colima", "restart"]
    status  = ["colima", "status"]
  }
  
  # Health check to ensure Colima is running properly
  health_check = {
    type    = "command"
    command = "colima status"
    timeout = "30s"
  }
  
  wait_for_healthy = true
  wait_timeout     = "120s"
}

# Example 4: Docker Desktop with direct command strategy
resource "pkg_service" "docker_desktop" {
  service_name        = "docker-desktop"
  state              = "running"
  startup            = "enabled"
  management_strategy = "direct_command"
  
  # Custom commands for Docker Desktop
  custom_commands = {
    start   = ["open", "-a", "Docker"]
    stop    = ["killall", "Docker"]
    restart = ["killall", "Docker", "&&", "open", "-a", "Docker"]
    status  = ["pgrep", "-f", "Docker"]
  }
  
  # Health check using Docker command
  health_check = {
    type    = "command"
    command = "docker ps"
    timeout = "30s"
  }
  
  wait_for_healthy = true
  wait_timeout     = "60s"
}

# Example 5: Lima with direct command strategy
resource "pkg_service" "lima" {
  service_name        = "lima"
  state              = "running"
  startup            = "enabled"
  management_strategy = "direct_command"
  
  # Custom commands for Lima
  custom_commands = {
    start   = ["limactl", "start", "default"]
    stop    = ["limactl", "stop", "default"]
    restart = ["limactl", "stop", "default", "&&", "limactl", "start", "default"]
    status  = ["limactl", "list"]
  }
  
  # Health check using Lima command
  health_check = {
    type    = "command"
    command = "limactl list"
    timeout = "30s"
  }
  
  wait_for_healthy = true
  wait_timeout     = "90s"
}

# Example 6: Process-only strategy (read-only)
resource "pkg_service" "existing_service" {
  service_name        = "some-existing-service"
  state              = "running"
  management_strategy = "process_only"
  
  # This strategy only checks if the process is running
  # It cannot start, stop, or restart the service
  # Useful for monitoring existing services
}

# Example 7: Launchd strategy for system services
resource "pkg_service" "system_service" {
  service_name        = "com.example.service"
  state              = "running"
  startup            = "enabled"
  management_strategy = "launchd"
  
  # Uses launchd for system service management
  # Requires proper plist files in /Library/LaunchDaemons/ or ~/Library/LaunchAgents/
}

# Example 8: Service with package validation
resource "pkg_service" "nginx" {
  service_name        = "nginx"
  state              = "running"
  startup            = "enabled"
  management_strategy = "brew_services"
  validate_package    = true
  package_name       = "nginx"
  
  # Validates that the nginx package is installed
  # before attempting to manage the service
}

# Example 9: Service with HTTP health check
resource "pkg_service" "web_service" {
  service_name        = "web-service"
  state              = "running"
  startup            = "enabled"
  management_strategy = "auto"
  
  # HTTP health check
  health_check = {
    type          = "http"
    url           = "http://localhost:8080/health"
    expected_code = 200
    timeout       = "30s"
  }
  
  wait_for_healthy = true
  wait_timeout     = "60s"
}

# Example 10: Service with TCP health check
resource "pkg_service" "database" {
  service_name        = "database"
  state              = "running"
  startup            = "enabled"
  management_strategy = "auto"
  
  # TCP health check
  health_check = {
    type    = "tcp"
    port    = 5432
    timeout = "30s"
  }
  
  wait_for_healthy = true
  wait_timeout     = "60s"
}

# Outputs to show the results
output "service_status" {
  description = "Status of managed services"
  value = {
    postgresql = {
      running = pkg_service.postgresql.running
      healthy = pkg_service.postgresql.healthy
      enabled = pkg_service.postgresql.enabled
    }
    redis = {
      running = pkg_service.redis.running
      healthy = pkg_service.redis.healthy
      enabled = pkg_service.redis.enabled
    }
    colima = {
      running = pkg_service.colima.running
      healthy = pkg_service.colima.healthy
      enabled = pkg_service.colima.enabled
    }
    docker_desktop = {
      running = pkg_service.docker_desktop.running
      healthy = pkg_service.docker_desktop.healthy
      enabled = pkg_service.docker_desktop.enabled
    }
  }
}
