# Context Timeout Chain Reaction Bug Fix

## Overview

This document describes the fix for a critical context timeout bug in the `terraform-provider-package` service resource that was causing all system commands to fail immediately with "context deadline exceeded" errors.

## The Problem

### Root Cause
The bug occurred in the service resource's health check functionality where a timeout context was created and then reused for individual system commands. When the timeout expired, all subsequent commands would fail immediately because they inherited the expired context.

### Affected Code
- **Primary**: `internal/provider/service_resource.go` - `waitForHealthy()` method (line 771)
- **Secondary**: `internal/provider/service_resource.go` - `waitForDependencyHealthy()` method (line 874)

### Bug Pattern
```go
// BUGGY CODE - DO NOT USE
func (r *ServiceResource) waitForHealthy(ctx context.Context, model *ServiceResourceModel) error {
    // Create timeout context
    ctx, cancel := context.WithTimeout(ctx, timeout) //  BUG: Reuses variable name
    defer cancel()
    
    for {
        select {
        case <-ctx.Done():
            return fmt.Errorf("timeout")
        case <-ticker.C:
            // Uses the same timeout context for commands
            serviceInfo, err := r.serviceManager.GetServiceInfo(ctx, serviceName) //  BUG: Expired context
        }
    }
}
```

### Symptoms
- All system commands fail with `execution_time_ms=0` 
- Error messages: `"context deadline exceeded"`
- Commands never actually execute - they fail at the context level
- Service management completely broken (Colima, Docker, PostgreSQL, etc.)

## The Solution

### Fix Strategy
Implemented **Option 1: Separate Contexts** - Create separate contexts for the overall operation timeout vs. individual command executions.

### Key Changes

#### 1. Enhanced `waitForHealthy()` Method
```go
// FIXED CODE
func (r *ServiceResource) waitForHealthy(ctx context.Context, model *ServiceResourceModel) error {
    // Create context with timeout for the overall operation
    operationCtx, cancel := context.WithTimeout(ctx, timeout) //  Separate variable
    defer cancel()
    
    for {
        select {
        case <-operationCtx.Done(): //  Check operation timeout
            return fmt.Errorf("timeout waiting for service to be healthy")
        case <-ticker.C:
            // Create fresh context for each health check command
            checkCtx, checkCancel := context.WithTimeout(context.Background(), 30*time.Second) //  Fresh context
            
            serviceInfo, err := r.serviceManager.GetServiceInfo(checkCtx, serviceName) //  Uses fresh context
            checkCancel() // Always cancel the check context
            
            // Handle results...
        }
    }
}
```

#### 2. Enhanced `waitForDependencyHealthy()` Method
Applied the same pattern to dependency health checks:
```go
// Create fresh context for each health check command
checkCtx, checkCancel := context.WithTimeout(context.Background(), 15*time.Second)
healthy, err := r.performHealthCheck(checkCtx, dep.HealthCheck)
checkCancel()
```

#### 3. Added Context Validation to SystemExecutor
Enhanced the executor to catch expired contexts early:
```go
func (e *SystemExecutor) Run(ctx context.Context, command string, args []string, opts ExecOpts) (ExecResult, error) {
    // Check if context is already cancelled/expired before starting
    select {
    case <-ctx.Done():
        return ExecResult{ExitCode: -1}, fmt.Errorf("context already expired before execution: %w", ctx.Err())
    default:
        // Context is still valid, proceed
    }
    // ... rest of execution
}
```

#### 4. Added Comprehensive DEBUG Logging
Enhanced logging throughout the health check process:
```go
tflog.Debug(ctx, "Starting service health check", map[string]interface{}{
    "service_name": serviceName,
    "timeout":      timeout.String(),
})

tflog.Debug(ctx, "Performing health check", map[string]interface{}{
    "service_name":   serviceName,
    "check_timeout":  "30s",
})
```

## Technical Details

### Context Lifecycle Management

#### Before (Buggy)
```
1. waitForHealthy creates timeout context (e.g., 120s)
2. Health check loop starts
3. After 120s, context expires
4. All subsequent GetServiceInfo calls fail immediately
5. Commands never execute (execution_time_ms=0)
```

#### After (Fixed)
```
1. waitForHealthy creates operation timeout context (e.g., 120s)
2. Health check loop starts
3. Each health check creates fresh 30s context
4. Individual commands can succeed even if operation timeout approaches
5. Only the overall operation times out, not individual commands
```

### Context Hierarchy
```
Root Context (from Terraform)
├── Operation Context (120s timeout)
│   └── Health Check Loop
│       ├── Fresh Check Context #1 (30s timeout) → Command execution
│       ├── Fresh Check Context #2 (30s timeout) → Command execution  
│       └── Fresh Check Context #N (30s timeout) → Command execution
```

## Testing

### Unit Tests Added
- `TestServiceResource_WaitForHealthy_ContextTimeout_Fix` - Verifies timeout chain reaction is fixed
- `TestServiceResource_WaitForHealthy_FreshContext_Success` - Verifies normal operation works
- `TestServiceResource_WaitForDependencyHealthy_ContextTimeout_Fix` - Verifies dependency checks work
- `TestSystemExecutor_ContextValidation` - Verifies early context validation

### Manual Testing
```bash
# Before fix: All commands fail with "context deadline exceeded"
export TF_LOG=DEBUG
terraform apply

# After fix: Commands execute normally with proper timing
export TF_LOG=DEBUG  
terraform apply
# Look for: execution_time_ms > 0 and successful command completion
```

## Impact

### Before Fix
-  Service management completely broken
-  Health checks fail immediately  
-  Colima, Docker, PostgreSQL management unusable
-  False negatives for running services

### After Fix
-  Service management works correctly
-  Health checks execute with proper timeouts
-  All service management features functional
-  Accurate service status reporting

## Performance Characteristics

### Timeout Behavior
- **Operation Timeout**: Overall health check operation (configurable, default 120s)
- **Command Timeout**: Individual system commands (30s for health checks, 15s for dependencies)
- **Check Interval**: 5 seconds between health checks

### Resource Usage
- **Memory**: Minimal overhead from context creation/cleanup
- **CPU**: No significant impact
- **Network**: No change

## Debugging

### Log Indicators of Success
```json
{
  "msg": "Starting service health check",
  "service_name": "colima", 
  "timeout": "120s"
}

{
  "msg": "Performing health check",
  "service_name": "colima",
  "check_timeout": "30s" 
}

{
  "msg": "Command execution completed",
  "execution_time_ms": 1234,  // > 0 indicates actual execution
  "exit_code": 0
}
```

### Log Indicators of Problems
```json
{
  "msg": "Context already expired before command execution",
  "command": "brew",
  "error": "context deadline exceeded"
}

{
  "msg": "Command execution completed", 
  "execution_time_ms": 0,  // = 0 indicates immediate failure
  "exit_code": -1
}
```

## Files Modified

### Core Implementation
- `internal/provider/service_resource.go` - Fixed timeout context handling
- `internal/executor/system_executor.go` - Added context validation

### Testing
- `internal/provider/service_resource_context_test.go` - Comprehensive context timeout tests

### Documentation
- `CONTEXT_TIMEOUT_BUG_FIX.md` - This document
- `DEBUG_LOGGING_GUIDE.md` - Updated with context debugging info

## Backward Compatibility

-  **API Compatible**: No changes to Terraform configuration schema
-  **Behavior Compatible**: Health checks work the same from user perspective
-  **Performance Compatible**: No significant performance impact
-  **Configuration Compatible**: All existing timeout settings work as expected

## Future Considerations

### Prevention
- **Code Review**: Always check for context reuse patterns
- **Testing**: Include timeout scenarios in unit tests
- **Linting**: Consider adding custom linters for context anti-patterns

### Monitoring
- **Metrics**: Track command execution times to detect timeout issues
- **Alerts**: Monitor for high rates of context deadline exceeded errors
- **Logging**: Continue using structured logging for timeout debugging

## Conclusion

This fix resolves a critical bug that made service management functionality completely unusable. The solution properly separates operation-level timeouts from command-level timeouts, ensuring that individual system commands can execute successfully even when approaching overall operation timeouts.

The fix is backward compatible, well-tested, and includes comprehensive logging for future debugging. Service management features (Colima, Docker, PostgreSQL) should now work correctly without the timeout chain reaction issue.
