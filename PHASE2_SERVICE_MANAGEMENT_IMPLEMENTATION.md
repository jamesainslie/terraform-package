# Phase 2: Service Management Resource Implementation

## Overview

This document describes the complete implementation of Phase 2 of the Package Provider Service Status Extension, which adds comprehensive service management capabilities through the `pkg_service` Terraform resource.

## Implementation Summary

Phase 2 extends the provider with full service lifecycle management, allowing users to manage service states (running/stopped) and startup configuration (enabled/disabled) across all supported platforms.

## Core Components

### 1. ServiceManager Interface

The `ServiceManager` interface extends `ServiceDetector` with lifecycle management methods:

```go
type ServiceManager interface {
    ServiceDetector

    // StartService starts a service
    StartService(ctx context.Context, serviceName string) error

    // StopService stops a service
    StopService(ctx context.Context, serviceName string) error

    // RestartService restarts a service
    RestartService(ctx context.Context, serviceName string) error

    // DisableService disables a service from starting automatically on system startup
    DisableService(ctx context.Context, serviceName string) error

    // IsServiceEnabled checks if a service is enabled for automatic startup
    IsServiceEnabled(ctx context.Context, serviceName string) (bool, error)

    // SetServiceStartup sets whether a service should start on system startup
    SetServiceStartup(ctx context.Context, serviceName string, enabled bool) error
}
```

### 2. Enhanced ServiceInfo Structure

Added `Enabled` field to track service startup configuration:

```go
type ServiceInfo struct {
    Name        string            `json:"name"`
    Running     bool              `json:"running"`
    Healthy     bool              `json:"healthy"`
    Enabled     bool              `json:"enabled"` // New field for startup status
    Version     string            `json:"version,omitempty"`
    ProcessID   string            `json:"process_id,omitempty"`
    StartTime   *time.Time        `json:"start_time,omitempty"`
    ManagerType ServiceManagerType `json:"manager_type"`
    Package     *PackageInfo      `json:"package,omitempty"`
    Ports       []int             `json:"ports,omitempty"`
    Metadata    map[string]string `json:"metadata,omitempty"`
    Error       string            `json:"error,omitempty"`
}
```

## Platform-Specific Implementations

### macOS Implementation (`macos_detector.go`)

**Service Management Methods:**
- `StartService`: Uses `brew services start` or `launchctl start`
- `StopService`: Uses `brew services stop` or `launchctl stop`
- `RestartService`: Stops then starts the service
- `DisableService`: Uses `brew services stop` or `launchctl unload`
- `IsServiceEnabled`: Checks brew services status or launchd loaded state
- `SetServiceStartup`: Unified method for enabling/disabling startup

**Key Features:**
- Dual support for Homebrew services and launchd
- Automatic fallback between service managers
- Proper error handling and status reporting

### Linux Implementation (`linux_detector.go`)

**Service Management Methods:**
- `StartService`: Uses `systemctl start`
- `StopService`: Uses `systemctl stop`
- `RestartService`: Uses `systemctl restart`
- `DisableService`: Uses `systemctl disable`
- `IsServiceEnabled`: Uses `systemctl is-enabled`
- `SetServiceStartup`: Uses `systemctl enable/disable`

**Key Features:**
- Full systemd integration
- Process name fallback for non-systemd services
- Comprehensive error handling

### Windows Implementation (`windows_detector.go`)

**Service Management Methods:**
- `StartService`: Uses PowerShell `Start-Service`
- `StopService`: Uses PowerShell `Stop-Service`
- `RestartService`: Uses PowerShell `Restart-Service`
- `DisableService`: Uses PowerShell `Set-Service -StartupType Disabled`
- `IsServiceEnabled`: Uses PowerShell to check startup type
- `SetServiceStartup`: Uses PowerShell `Set-Service -StartupType`

**Key Features:**
- PowerShell-based service management
- Windows Services integration
- Process name fallback detection

### Generic Implementation (`generic_detector.go`)

**Service Management Methods:**
- All methods return appropriate error messages indicating limited functionality
- Provides fallback for unsupported platforms
- Maintains interface compliance

## Terraform Resource Implementation

### pkg_service Resource Schema

```hcl
resource "pkg_service" "example" {
  service_name     = "docker"
  state           = "running"        # "running" or "stopped"
  startup         = "enabled"        # "enabled" or "disabled"
  validate_package = true
  package_name    = "docker"
  
  health_check {
    type    = "tcp"
    port    = 2376
    timeout = "30s"
  }
  
  wait_for_healthy = true
  wait_timeout     = "60s"
}
```

### Resource Attributes

**Input Attributes:**
- `service_name` (required): Name of the service to manage
- `state` (optional): Desired service state ("running" or "stopped")
- `startup` (optional): Startup configuration ("enabled" or "disabled")
- `validate_package` (optional): Whether to validate package installation
- `package_name` (optional): Package name for validation
- `health_check` (optional): Health check configuration
- `wait_for_healthy` (optional): Wait for service to be healthy
- `wait_timeout` (optional): Timeout for health check

**Computed Attributes:**
- `running`: Current running status
- `healthy`: Current health status
- `enabled`: Current startup configuration
- `version`: Service version
- `process_id`: Process ID if running
- `start_time`: Service start time
- `manager_type`: Service manager type
- `package`: Package information
- `ports`: Listening ports
- `metadata`: Additional metadata
- `last_updated`: Last update timestamp

### Health Check Configuration

```hcl
health_check {
  type          = "http"           # "command", "http", or "tcp"
  command       = "docker ps"      # For command type
  url           = "http://localhost:8080/health"  # For http type
  port          = 2376             # For tcp type
  timeout       = "30s"
  expected_code = 200              # For http type
  interval      = "5s"
}
```

## Factory Pattern Implementation

### Platform Detection

The factory pattern automatically selects the appropriate service manager:

```go
func NewServiceManager() ServiceManager {
    return newPlatformServiceDetector()
}
```

**Build Tags:**
- `//go:build darwin` - macOS implementation
- `//go:build linux` - Linux implementation  
- `//go:build windows` - Windows implementation
- Default - Generic fallback implementation

### Factory Files

- `factory_darwin.go`: macOS service manager creation
- `factory_linux.go`: Linux service manager creation
- `factory_windows.go`: Windows service manager creation
- `factory_default.go`: Generic fallback creation

## Resource Lifecycle Management

### Create Operation

1. **Validation**: Validates service name and configuration
2. **Package Check**: Optionally validates package installation
3. **State Management**: Sets desired service state (running/stopped)
4. **Startup Configuration**: Sets startup configuration (enabled/disabled)
5. **Health Check**: Performs health check if configured
6. **Wait Logic**: Waits for service to be healthy if requested

### Read Operation

1. **Service Detection**: Detects current service status
2. **State Reading**: Reads running and enabled status
3. **Health Check**: Performs health check if configured
4. **Metadata Collection**: Collects version, PID, start time, etc.

### Update Operation

1. **State Changes**: Handles state transitions (running ↔ stopped)
2. **Startup Changes**: Handles startup configuration changes
3. **Health Check Updates**: Updates health check configuration
4. **Validation**: Re-validates package if needed

### Delete Operation

1. **State Preservation**: Optionally stops service
2. **Startup Preservation**: Optionally disables startup
3. **Cleanup**: Removes resource from state

### Import Operation

1. **Service Detection**: Detects existing service
2. **State Reading**: Reads current configuration
3. **Resource Creation**: Creates resource in state

## Error Handling

### Service Management Errors

- **Service Not Found**: Clear error messages for missing services
- **Permission Denied**: Guidance for privilege escalation
- **Command Execution**: Detailed error reporting for failed commands
- **Timeout Handling**: Proper timeout management for all operations

### Validation Errors

- **Invalid Service Name**: Validation for service name format
- **Invalid State**: Validation for state values
- **Invalid Startup**: Validation for startup values
- **Health Check Errors**: Validation for health check configuration

## Testing Strategy

### Unit Tests

- **Service Manager Tests**: Test all service management methods
- **Platform-Specific Tests**: Test platform-specific implementations
- **Error Handling Tests**: Test error conditions and edge cases
- **Health Check Tests**: Test health check functionality

### Integration Tests

- **Resource Lifecycle**: Test complete resource lifecycle
- **State Management**: Test state transitions
- **Startup Configuration**: Test startup configuration changes
- **Health Check Integration**: Test health check integration

### Acceptance Tests

- **Real Service Management**: Test with actual services
- **Cross-Platform**: Test across different platforms
- **Error Scenarios**: Test error handling in real environments

## Usage Examples

### Basic Service Management

```hcl
resource "pkg_service" "docker" {
  service_name = "docker"
  state        = "running"
  startup      = "enabled"
}
```

### Service with Health Check

```hcl
resource "pkg_service" "postgres" {
  service_name = "postgresql"
  state        = "running"
  startup      = "enabled"
  
  health_check {
    type    = "tcp"
    port    = 5432
    timeout = "30s"
  }
  
  wait_for_healthy = true
  wait_timeout     = "60s"
}
```

### Conditional Service Management

```hcl
data "pkg_service_status" "docker_status" {
  service_name = "docker"
}

resource "pkg_service" "docker" {
  count        = data.pkg_service_status.docker_status.running ? 0 : 1
  service_name = "docker"
  state        = "running"
  startup      = "enabled"
}
```

### Package Validation

```hcl
resource "pkg_service" "nginx" {
  service_name     = "nginx"
  state           = "running"
  startup         = "enabled"
  validate_package = true
  package_name    = "nginx"
}
```

## Performance Considerations

### Optimization Strategies

- **Bulk Operations**: Efficient bulk service management
- **Caching**: Cache service status information
- **Parallel Health Checks**: Concurrent health check execution
- **Timeout Management**: Proper timeout configuration

### Resource Management

- **Connection Pooling**: Efficient command execution
- **Memory Management**: Proper resource cleanup
- **Error Recovery**: Graceful error handling and recovery

## Security Considerations

### Command Execution

- **Input Validation**: Validate all service names and parameters
- **Command Sanitization**: Sanitize command arguments
- **Privilege Management**: Handle privilege escalation safely
- **Timeout Enforcement**: Enforce timeouts for all operations

### Error Handling

- **Information Disclosure**: Avoid exposing sensitive information
- **Error Messages**: Provide helpful but secure error messages
- **Logging**: Secure logging practices

## Integration with Existing Provider

### Provider Registration

```go
func (p *PackageProvider) Resources(_ context.Context) []func() resource.Resource {
    return []func() resource.Resource{
        NewPackageResource,
        NewRepositoryResource,
        NewServiceResource, // New service management resource
    }
}
```

### Data Source Integration

- **Service Status Data Source**: Integration with `pkg_service_status`
- **Services Overview Data Source**: Integration with `pkg_services_overview`
- **Package Integration**: Integration with `pkg_package` resource

## Migration and Compatibility

### Backward Compatibility

- **No Breaking Changes**: All existing functionality preserved
- **Optional Features**: New features are optional and backward compatible
- **Migration Path**: Clear migration path from existing workarounds

### Upgrade Path

- **Gradual Adoption**: Users can adopt new features gradually
- **Existing Resources**: Existing resources continue to work
- **New Capabilities**: New capabilities available immediately

## Success Metrics

### Technical Metrics

- **Test Coverage**: Comprehensive test coverage for all functionality
- **Performance**: Efficient service management operations
- **Reliability**: Robust error handling and recovery
- **Compatibility**: Cross-platform compatibility

### User Experience Metrics

- **Ease of Use**: Simple and intuitive resource configuration
- **Documentation**: Comprehensive documentation and examples
- **Error Messages**: Clear and helpful error messages
- **Integration**: Seamless integration with existing workflows

## Future Enhancements

### Potential Improvements

- **Advanced Health Checks**: More sophisticated health check types
- **Service Dependencies**: Service dependency management
- **Clustering Support**: Multi-node service management
- **Monitoring Integration**: Integration with monitoring systems

### Extension Points

- **Custom Health Checks**: Plugin system for custom health checks
- **Service Templates**: Predefined service configurations
- **Automated Recovery**: Automatic service recovery mechanisms
- **Performance Metrics**: Service performance monitoring

## Conclusion

Phase 2 successfully implements comprehensive service management capabilities through the `pkg_service` Terraform resource. The implementation provides:

- **Full Service Lifecycle Management**: Start, stop, restart, enable, disable services
- **Cross-Platform Support**: macOS, Linux, Windows, and generic fallback
- **Health Check Integration**: Built-in health checking capabilities
- **Terraform Integration**: Native Terraform resource with proper state management
- **Robust Error Handling**: Comprehensive error handling and user guidance
- **Performance Optimization**: Efficient operations with proper timeout management
- **Security**: Secure command execution and input validation

The implementation maintains backward compatibility while providing powerful new capabilities for service management in Infrastructure as Code workflows.
