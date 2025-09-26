# Colima Service Management Test

This example demonstrates the fixed service management for Colima, which was previously affected by the "Health Check Strategy Mismatch Bug".

## Bug Description

Previously, when managing Colima:
1. The provider would use `DirectCommandStrategy` to start Colima (`colima start`)
2. But health checks would use the default detector logic (trying `brew services` first)
3. Since Colima doesn't use `brew services`, health checks would fail
4. This led to false negatives and infinite health check loops

## Fix

The refactored service management now uses **strategy-aware health checks**:
1. The same `DirectCommandStrategy` is used for both management AND health checks
2. Health checks use `colima status` (matching the management strategy)
3. This ensures consistency and eliminates the mismatch bug

## Usage

```bash
# Build the provider
cd /Volumes/Development/terraform-provider-package
go build -o terraform-provider-package .

# Initialize Terraform
cd examples/colima-test
terraform init

# Plan and apply (with debug logging)
export TF_LOG=DEBUG
terraform plan
terraform apply

# Check the output
terraform output colima_status
```

## Expected Behavior

- **Before the fix**: Health checks would fail even if Colima was running, causing timeouts or false negatives
- **After the fix**: Health checks use `colima status` and correctly detect when Colima is running

## Debug Logging

The configuration enables debug logging, which will show:
- Strategy selection (direct_command for Colima)
- Command execution details
- Health check results with strategy information
- Metadata showing which strategy was used

Look for log entries like:
```
DirectCommandLifecycleStrategy.HealthCheck starting
Executing status command for health check
Health check completed with strategy: direct_command
```
