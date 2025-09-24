# Service Management Example
# This example demonstrates how to use the pkg_service resource to manage service lifecycle

terraform {
  required_providers {
    pkg = {
      source = "jamesainslie/package"
    }
  }
}

# Configure the provider
provider "pkg" {
  # Provider configuration is optional - defaults will be used
}

# Manage Docker service - ensure it's running and enabled for startup
resource "pkg_service" "docker" {
  service_name     = "docker"
  state           = "running"
  startup         = "enabled"
  validate_package = true
  package_name    = "docker"
  
  health_check = {
    type          = "http"
    url          = "http://localhost:2375/version"
    timeout      = "30s"
    expected_code = 200
  }
  
  wait_for_healthy = true
  wait_timeout    = "120s"
}

# Manage PostgreSQL service - ensure it's running and enabled for startup
resource "pkg_service" "postgresql" {
  service_name     = "postgresql"
  state           = "running"
  startup         = "enabled"
  validate_package = true
  package_name    = "postgresql"
  
  health_check = {
    type     = "tcp"
    port     = 5432
    timeout  = "10s"
  }
  
  wait_for_healthy = true
  wait_timeout    = "60s"
}

# Manage Redis service - ensure it's running but not enabled for startup
resource "pkg_service" "redis" {
  service_name     = "redis"
  state           = "running"
  startup         = "disabled"
  validate_package = true
  package_name    = "redis"
  
  health_check = {
    type     = "tcp"
    port     = 6379
    timeout  = "10s"
  }
  
  wait_for_healthy = true
  wait_timeout    = "30s"
}

# Manage Nginx service with command-based health check
resource "pkg_service" "nginx" {
  service_name     = "nginx"
  state           = "running"
  startup         = "enabled"
  validate_package = true
  package_name    = "nginx"
  
  health_check = {
    type     = "command"
    command  = "curl -f http://localhost:80/health || exit 1"
    timeout  = "15s"
  }
  
  wait_for_healthy = true
  wait_timeout    = "45s"
}

# Output service information
output "docker_service_info" {
  description = "Information about the Docker service"
  value = {
    running     = pkg_service.docker.running
    healthy     = pkg_service.docker.healthy
    enabled     = pkg_service.docker.enabled
    version     = pkg_service.docker.version
    process_id  = pkg_service.docker.process_id
    manager_type = pkg_service.docker.manager_type
  }
}

output "postgresql_service_info" {
  description = "Information about the PostgreSQL service"
  value = {
    running     = pkg_service.postgresql.running
    healthy     = pkg_service.postgresql.healthy
    enabled     = pkg_service.postgresql.enabled
    version     = pkg_service.postgresql.version
    process_id  = pkg_service.postgresql.process_id
    manager_type = pkg_service.postgresql.manager_type
  }
}

output "redis_service_info" {
  description = "Information about the Redis service"
  value = {
    running     = pkg_service.redis.running
    healthy     = pkg_service.redis.healthy
    enabled     = pkg_service.redis.enabled
    version     = pkg_service.redis.version
    process_id  = pkg_service.redis.process_id
    manager_type = pkg_service.redis.manager_type
  }
}

output "nginx_service_info" {
  description = "Information about the Nginx service"
  value = {
    running     = pkg_service.nginx.running
    healthy     = pkg_service.nginx.healthy
    enabled     = pkg_service.nginx.enabled
    version     = pkg_service.nginx.version
    process_id  = pkg_service.nginx.process_id
    manager_type = pkg_service.nginx.manager_type
  }
}
