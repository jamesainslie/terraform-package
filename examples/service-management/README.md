# Service Management Example

This example demonstrates how to use the `pkg_service` resource to manage service lifecycle across different platforms.

## Overview

The `pkg_service` resource allows you to:
- Start, stop, and restart services
- Enable or disable services for automatic startup
- Configure health checks for services
- Validate package dependencies
- Wait for services to become healthy

## Features Demonstrated

### 1. Docker Service Management
- Ensures Docker is running and enabled for startup
- Uses HTTP health check to verify Docker daemon is responding
- Validates that the Docker package is installed
- Waits up to 2 minutes for the service to become healthy

### 2. PostgreSQL Service Management
- Ensures PostgreSQL is running and enabled for startup
- Uses TCP health check on port 5432
- Validates that the PostgreSQL package is installed
- Waits up to 1 minute for the service to become healthy

### 3. Redis Service Management
- Ensures Redis is running but NOT enabled for startup
- Uses TCP health check on port 6379
- Validates that the Redis package is installed
- Waits up to 30 seconds for the service to become healthy

### 4. Nginx Service Management
- Ensures Nginx is running and enabled for startup
- Uses command-based health check with curl
- Validates that the Nginx package is installed
- Waits up to 45 seconds for the service to become healthy

## Health Check Types

### HTTP Health Check
```hcl
health_check = {
  type          = "http"
  url          = "http://localhost:2375/version"
  timeout      = "30s"
  expected_code = 200
}
```

### TCP Health Check
```hcl
health_check = {
  type     = "tcp"
  port     = 5432
  timeout  = "10s"
}
```

### Command Health Check
```hcl
health_check = {
  type     = "command"
  command  = "curl -f http://localhost:80/health || exit 1"
  timeout  = "15s"
}
```

## Service States

### Running State
- `"running"` - Service should be started and running
- `"stopped"` - Service should be stopped

### Startup Configuration
- `"enabled"` - Service will start automatically on system boot
- `"disabled"` - Service will not start automatically on system boot

## Platform Support

This example works across all supported platforms:

- **macOS**: Uses `launchd` and `brew services`
- **Linux**: Uses `systemd`
- **Windows**: Uses Windows Services and PowerShell

## Usage

1. **Initialize Terraform**:
   ```bash
   terraform init
   ```

2. **Plan the deployment**:
   ```bash
   terraform plan
   ```

3. **Apply the configuration**:
   ```bash
   terraform apply
   ```

4. **Check service status**:
   ```bash
   terraform output
   ```

## Outputs

The example provides detailed information about each managed service:
- Running status
- Health status
- Enabled status
- Version information
- Process ID
- Service manager type

## Advanced Usage

### Conditional Service Management
```hcl
# Only manage Docker if it's not already running
data "pkg_service_status" "docker_status" {
  service_name = "docker"
}

resource "pkg_service" "docker" {
  count = data.pkg_service_status.docker_status.running ? 0 : 1
  
  service_name = "docker"
  state       = "running"
  startup     = "enabled"
}
```

### Service Dependencies
```hcl
# Ensure Docker is running before starting dependent services
resource "pkg_service" "docker" {
  service_name = "docker"
  state       = "running"
  startup     = "enabled"
}

resource "pkg_service" "docker_compose" {
  depends_on = [pkg_service.docker]
  
  service_name = "docker-compose"
  state       = "running"
  startup     = "enabled"
}
```

## Troubleshooting

### Common Issues

1. **Service not found**: Ensure the service name matches the actual service name on your platform
2. **Permission denied**: Some operations may require elevated privileges
3. **Health check timeout**: Adjust timeout values based on your system's performance
4. **Package validation failed**: Ensure the required package is installed

### Debug Information

Use Terraform's debug output to troubleshoot issues:
```bash
export TF_LOG=DEBUG
terraform apply
```

## Best Practices

1. **Use appropriate health checks**: Choose the health check type that best fits your service
2. **Set reasonable timeouts**: Don't set timeouts too low or too high
3. **Validate packages**: Use `validate_package = true` to ensure dependencies are met
4. **Wait for health**: Use `wait_for_healthy = true` for critical services
5. **Monitor outputs**: Use the output values to monitor service status

## Integration with Other Resources

This example can be combined with other Terraform resources:

```hcl
# Install package first, then manage service
resource "pkg_package" "docker" {
  name = "docker"
  type = "formula"
}

resource "pkg_service" "docker" {
  depends_on = [pkg_package.docker]
  
  service_name = "docker"
  state       = "running"
  startup     = "enabled"
}
```
