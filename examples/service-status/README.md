# Service Status Example

This example demonstrates the service status checking capabilities of the jamesainslie/package provider. The example shows how to:

1. **Check individual service status** with the `pkg_service_status` data source
2. **Check multiple services at once** with the `pkg_services_overview` data source  
3. **Create resources conditionally** based on service status
4. **Validate service health** with custom health checks

## Features Demonstrated

### Individual Service Status Checking

```hcl
data "pkg_service_status" "colima" {
  name             = "colima"
  required_package = "colima"
  package_manager  = "brew"
  
  health_check {
    command = "colima status"
    timeout = "5s"
  }
}
```

### Bulk Service Overview

```hcl
data "pkg_services_overview" "development_stack" {
  services = ["colima", "docker", "postgres"]
  
  package_manager         = "brew"
  installed_packages_only = true
  health_checks           = true
}
```

### Conditional Resource Creation

```hcl
resource "local_file" "docker_config" {
  count = data.pkg_service_status.docker.running && data.pkg_service_status.docker.healthy ? 1 : 0
  
  content  = jsonencode({...})
  filename = "${path.module}/docker-daemon.json"
}
```

### Service Health Checks

The provider supports multiple types of health checks:

- **Command-based**: Execute shell commands to verify service health
- **HTTP-based**: Check HTTP endpoints for expected status codes
- **TCP-based**: Verify network connectivity to service ports

## Prerequisites

1. **macOS** (this example uses Homebrew)
2. **Terraform** >= 1.0
3. **jamesainslie/package provider** with service status support

## Services Used

- **Colima**: Container runtime for macOS
- **Docker**: Container engine  
- **PostgreSQL**: Database server

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

## Expected Outputs

### Service Status
Individual service information including:
- Running status (true/false)
- Health status (true/false)  
- Service version (if detectable)
- Process ID
- Service manager type (launchd, systemd, etc.)

### Services Overview
Aggregate statistics:
- Total services checked
- Number of running services
- Number of healthy services
- Detailed summary with service names

### Conditional Resources
Status of resources created based on service availability:
- Docker configuration file (only if Docker is running and healthy)
- Environment readiness flags

## Service Dependencies

The example demonstrates proper service dependency management:

1. **Packages must be installed** before checking service status
2. **Docker depends on Colima** being running on macOS
3. **Configuration files are only created** when services are healthy

## Advanced Usage

### Custom Health Checks

```hcl
health_check {
  http_endpoint   = "http://localhost:9200/_health"
  expected_status = 200
  timeout         = "10s"
}
```

### Service Dependency Validation

```hcl
locals {
  stack_ready = alltrue([
    data.pkg_service_status.colima.running,
    data.pkg_service_status.docker.healthy,
    data.pkg_service_status.postgres.healthy
  ])
}
```

### Integration with Other Providers

The service status can be used with other Terraform providers:

```hcl
# Only create Kubernetes resources if Docker is ready
resource "kubernetes_deployment" "app" {
  count = local.container_services_ready ? 1 : 0
  # ...
}
```

## Troubleshooting

### Service Not Detected

If a service is not detected, check:
1. **Package is installed**: Verify with `brew list`
2. **Service is running**: Check with `brew services list`
3. **Service name mapping**: See provider documentation for supported services

### Health Check Failures

Common health check issues:
1. **Timeout too short**: Increase timeout for slow-starting services
2. **Wrong endpoint**: Verify the HTTP endpoint or command
3. **Service not ready**: Some services need time to become fully operational

### Platform Differences

The provider handles platform differences automatically:
- **macOS**: Uses `launchd` and `brew services`
- **Linux**: Uses `systemd` and process checking
- **Windows**: Uses Windows Services API

## Clean Up

To remove all created resources:

```bash
terraform destroy
```

This will remove:
- Conditionally created configuration files
- Service status data (refresh only)
- Package installations (if desired)

