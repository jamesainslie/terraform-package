# Comprehensive DEBUG Logging Guide for Terraform Package Provider

## Overview

The Terraform Package Provider now includes comprehensive DEBUG logging throughout all layers of the system. This guide explains how to enable and use the detailed logging for troubleshooting and development.

## Enabling DEBUG Logging

### Environment Variables

```bash
# Enable DEBUG level logging for the entire Terraform operation
export TF_LOG=DEBUG

# Enable DEBUG level logging specifically for the provider
export TF_LOG_PROVIDER=DEBUG

# Optional: Save logs to a file for analysis
export TF_LOG_PATH=./terraform-debug.log
```

### Provider-Specific Logging

The provider uses Terraform Plugin Framework's `tflog` package for structured logging. All log entries include:
- **Timestamp**: When the operation occurred
- **Context**: Which component/operation is logging
- **Structured Data**: Key-value pairs with relevant information

## Logging Layers

### 1. SystemExecutor Layer

**Location**: `internal/executor/system_executor.go`

**What it logs**:
- Command execution requests with full arguments
- Environment variables and working directory
- Command preparation (sudo handling)
- Execution timing and results
- stdout/stderr output (truncated for safety)

**Example log entry**:
```json
{
  "timestamp": "2025-01-27T10:30:15.123Z",
  "level": "DEBUG",
  "msg": "SystemExecutor.Run starting",
  "command": "brew",
  "args": ["install", "--cask", "firefox"],
  "timeout": "5m0s",
  "work_dir": "",
  "env_count": 0,
  "use_sudo": false,
  "non_interactive": false
}
```

### 2. Package Adapter Layer

**Location**: `internal/adapters/brew/adapter.go`, `internal/adapters/apt/adapter.go`

**What it logs**:
- Package installation requests
- Package type detection (cask vs formula for Homebrew)
- Idempotency checks and decisions
- Version compatibility checks
- Command construction and execution results

**Example log entry**:
```json
{
  "timestamp": "2025-01-27T10:30:15.456Z",
  "level": "DEBUG",
  "msg": "BrewAdapter.InstallWithType starting",
  "package_name": "firefox",
  "version": "",
  "package_type": "auto",
  "brew_path": "/opt/homebrew/bin/brew"
}
```

### 3. Service Strategy Layer

**Location**: `internal/services/strategies.go`

**What it logs**:
- Service management operations (start/stop/restart)
- Strategy selection and execution
- Idempotency checks for service states
- Command configuration and execution
- Error pattern recognition (e.g., "already running" detection)

**Example log entry**:
```json
{
  "timestamp": "2025-01-27T10:30:16.789Z",
  "level": "DEBUG",
  "msg": "DirectCommandStrategy.StartService starting",
  "service_name": "colima",
  "strategy": "direct_command"
}
```

### 4. Provider Resource Layer

**Location**: `internal/provider/package_resource.go`, `internal/provider/service_resource.go`

**What it logs**:
- Terraform CRUD operations (Create, Read, Update, Delete)
- Plan data and configuration
- Package manager resolution
- Idempotency checks and early returns
- State management operations

**Example log entry**:
```json
{
  "timestamp": "2025-01-27T10:30:17.012Z",
  "level": "DEBUG",
  "msg": "PackageResource.Create starting",
  "resource_type": "pkg_package"
}
```

## Common Debugging Scenarios

### 1. Package Installation Issues

**Problem**: Package installation fails
**Logs to check**:
1. `PackageResource.Create` - Check configuration and manager resolution
2. `BrewAdapter.InstallWithType` - Check package type detection and command construction
3. `SystemExecutor.Run` - Check actual command execution and output

**Example debugging session**:
```bash
export TF_LOG=DEBUG
terraform apply

# Look for these log sequences:
# 1. "PackageResource.Create starting"
# 2. "Package manager resolved" 
# 3. "BrewAdapter.InstallWithType starting"
# 4. "Executing brew install command"
# 5. "SystemExecutor.Run starting"
# 6. "Command execution completed"
```

### 2. Service Management Issues

**Problem**: Service fails to start (e.g., Colima)
**Logs to check**:
1. `DirectCommandStrategy.StartService` - Check service state and command execution
2. `SystemExecutor.Run` - Check actual command and results
3. Error pattern recognition logs

**Example debugging session**:
```bash
export TF_LOG=DEBUG
terraform apply

# Look for these log sequences:
# 1. "DirectCommandStrategy.StartService starting"
# 2. "Checking if service is already running"
# 3. "Executing start command"
# 4. "Start command execution completed"
# 5. "Detected 'already running' error pattern" (if applicable)
```

### 3. Idempotency Issues

**Problem**: Provider attempts to install already-installed packages
**Logs to check**:
1. `Starting idempotency check`
2. `Package already installed, checking version compatibility`
3. `Package already in desired state, skipping installation`

**Example debugging session**:
```bash
export TF_LOG=DEBUG
terraform plan  # Should show no changes if idempotent

# Look for:
# 1. "Starting idempotency check"
# 2. "Package already in desired state, skipping installation"
# 3. "PackageResource.Create completed via idempotency check"
```

## Log Analysis Tips

### 1. Filtering Logs

```bash
# Filter for specific components
grep "BrewAdapter" terraform-debug.log
grep "SystemExecutor" terraform-debug.log
grep "DirectCommandStrategy" terraform-debug.log

# Filter for specific operations
grep "InstallWithType" terraform-debug.log
grep "StartService" terraform-debug.log

# Filter for errors and failures
grep -i "error\|fail" terraform-debug.log
```

### 2. Timing Analysis

```bash
# Look for execution timing information
grep "execution_time_ms\|total_duration_ms" terraform-debug.log

# Check for timeout issues
grep "timeout\|context" terraform-debug.log
```

### 3. Command Analysis

```bash
# See all commands being executed
grep "Executing.*command\|Command execution completed" terraform-debug.log

# Check command arguments and results
grep "args\|exit_code\|stdout\|stderr" terraform-debug.log
```

## Performance Monitoring

The DEBUG logs include timing information for performance analysis:

```json
{
  "msg": "Command execution completed",
  "execution_time_ms": 1234,
  "total_duration_ms": 1250
}
```

Use this to identify slow operations:
```bash
# Find slow operations (>5 seconds)
grep "execution_time_ms" terraform-debug.log | awk -F'"execution_time_ms":' '{print $2}' | awk -F',' '{if($1 > 5000) print}'
```

## Security Considerations

- **Sensitive Data**: The logging system automatically truncates output to prevent logging sensitive information
- **Command Arguments**: Full command arguments are logged but sensitive environment variables are not expanded
- **File Paths**: Working directories and file paths are logged for debugging but should not contain sensitive information

## Troubleshooting Common Issues

### 1. Missing Log Output

**Problem**: No DEBUG logs appearing
**Solution**:
```bash
# Ensure environment variables are set
echo $TF_LOG
echo $TF_LOG_PROVIDER

# Try with explicit provider logging
export TF_LOG_PROVIDER_TERRAFORM_PROVIDER_PACKAGE=DEBUG
```

### 2. Too Much Log Output

**Problem**: Logs are overwhelming
**Solution**:
```bash
# Use more specific log levels
export TF_LOG=INFO
export TF_LOG_PROVIDER=DEBUG

# Or filter specific components
terraform apply 2>&1 | grep "PackageResource\|BrewAdapter"
```

### 3. Log File Rotation

**Problem**: Log files getting too large
**Solution**:
```bash
# Use logrotate or similar tools
# Or split logs by operation:
export TF_LOG_PATH=./terraform-$(date +%Y%m%d-%H%M%S).log
```

## Integration with CI/CD

For automated environments:

```yaml
# GitHub Actions example
- name: Debug Terraform Apply
  run: |
    export TF_LOG=DEBUG
    export TF_LOG_PATH=./terraform-debug.log
    terraform apply -auto-approve
  env:
    TF_LOG: DEBUG
    TF_LOG_PROVIDER: DEBUG

- name: Upload Debug Logs
  if: failure()
  uses: actions/upload-artifact@v3
  with:
    name: terraform-debug-logs
    path: terraform-debug.log
```

## Contributing Debug Information

When reporting issues, please include:

1. **Environment Variables**: Your `TF_LOG*` settings
2. **Relevant Log Sections**: The specific log entries around the failure
3. **Command Context**: What Terraform command you were running
4. **System Information**: OS, Terraform version, provider version

**Example issue report**:
```
Environment:
- TF_LOG=DEBUG
- TF_LOG_PROVIDER=DEBUG
- OS: macOS 14.6.0
- Terraform: v1.6.0
- Provider: v0.2.0

Command: terraform apply

Relevant logs:
[Include the specific log entries showing the issue]
```

This comprehensive DEBUG logging system provides unprecedented visibility into the provider's operations, making it much easier to troubleshoot issues and understand the provider's behavior.
