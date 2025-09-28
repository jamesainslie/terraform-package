# Service Management with Multiple Strategies

This example demonstrates how to use the `pkg_service` resource with different management strategies to handle various types of services.

## Overview

The `pkg_service` resource now supports multiple strategies for managing services:

- **`auto`** (default): Automatically tries multiple strategies in order of preference
- **`brew_services`**: Uses Homebrew services management
- **`direct_command`**: Uses custom commands for service management
- **`launchd`**: Uses macOS launchd for system services
- **`process_only`**: Only checks if a process is running (read-only)

## Examples

### 1. Auto Strategy (Default)
```hcl
resource "pkg_service" "postgresql" {
  service_name = "postgresql"
  state        = "running"
  startup      = "enabled"
  
  # Uses auto strategy by default
}
```

### 2. Brew Services Strategy
```hcl
resource "pkg_service" "redis" {
  service_name        = "redis"
  state              = "running"
  startup            = "enabled"
  management_strategy = "brew_services"
}
```

### 3. Direct Command Strategy
```hcl
resource "pkg_service" "colima" {
  service_name        = "colima"
  state              = "running"
  startup            = "enabled"
  management_strategy = "direct_command"
  
  custom_commands = {
    start   = ["colima", "start"]
    stop    = ["colima", "stop"]
    restart = ["colima", "restart"]
    status  = ["colima", "status"]
  }
}
```

### 4. Process-Only Strategy
```hcl
resource "pkg_service" "existing_service" {
  service_name        = "some-existing-service"
  state              = "running"
  management_strategy = "process_only"
  
  # Only checks if process is running, cannot manage it
}
```

## Strategy Selection

### Auto Strategy Behavior
The `auto` strategy tries multiple approaches in order:

1. **Brew Services**: For services that support `brew services`
2. **Direct Commands**: For services with known command patterns
3. **Launchd**: For system services with plist files
4. **Process Check**: Final fallback to check if process is running

### Service-Specific Defaults
The provider includes built-in knowledge of common services:

- **Brew Services**: postgresql, mysql, redis, nginx, apache, mongodb, etc.
- **Direct Commands**: colima, docker-desktop, lima, podman, etc.

### Custom Commands
When using `direct_command` strategy, you can specify custom commands:

```hcl
custom_commands = {
  start   = ["service", "start"]
  stop    = ["service", "stop"]
  restart = ["service", "restart"]
  status  = ["service", "status"]
}
```

## Health Checks

The resource supports multiple health check types:

### Command Health Check
```hcl
health_check = {
  type    = "command"
  command = "colima status"
  timeout = "30s"
}
```

### HTTP Health Check
```hcl
health_check = {
  type          = "http"
  url           = "http://localhost:8080/health"
  expected_code = 200
  timeout       = "30s"
}
```

### TCP Health Check
```hcl
health_check = {
  type    = "tcp"
  port    = 5432
  timeout = "30s"
}
```

## Usage

1. **Choose the appropriate strategy** for your service
2. **Configure custom commands** if using `direct_command` strategy
3. **Set up health checks** to ensure services are working properly
4. **Use `wait_for_healthy`** to wait for services to be ready

## Benefits

- **Flexibility**: Support for different service management approaches
- **Reliability**: Multiple fallback strategies increase success rate
- **Explicit Control**: Users can specify exactly how services should be managed
- **Backward Compatibility**: Default `auto` strategy maintains existing behavior
- **Extensibility**: Easy to add new strategies for future services

## Troubleshooting

### Service Won't Start
1. Check if the service supports the chosen strategy
2. Verify custom commands are correct
3. Try the `auto` strategy to let the provider choose
4. Check service logs for specific error messages

### Strategy Not Working
1. Verify the service name is correct
2. Check if the service is installed
3. Try a different strategy
4. Use `process_only` to just monitor the service

### Health Check Failing
1. Verify the health check configuration
2. Check if the service is actually running
3. Adjust timeout values if needed
4. Test the health check command manually
