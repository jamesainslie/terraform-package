# DEBUG Logging Example

This example demonstrates the comprehensive DEBUG logging capabilities of the Terraform Package Provider.

## Usage

1. **Enable DEBUG logging**:
   ```bash
   export TF_LOG=DEBUG
   export TF_LOG_PROVIDER=DEBUG
   ```

2. **Optional: Save logs to file**:
   ```bash
   export TF_LOG_PATH=./terraform-debug.log
   ```

3. **Run the example**:
   ```bash
   terraform init
   terraform plan   # See planning phase logs
   terraform apply  # See creation phase logs
   ```

## What You'll See in the Logs

### Package Installation Logs
- Package manager resolution (`brew` vs `apt` detection)
- Package type detection (formula vs cask for Homebrew)
- Idempotency checks
- Command construction and execution
- Installation results

### Service Management Logs  
- Service strategy selection
- Command configuration
- Service state checks
- Start/stop operations
- Error handling (e.g., "already running" detection)

### Data Source Logs
- Manager detection and validation
- System information gathering

## Example Log Flow

```
DEBUG: PackageResource.Create starting
DEBUG: Package configuration from plan
DEBUG: Package manager resolved
DEBUG: Starting idempotency check
DEBUG: BrewAdapter.InstallWithType starting
DEBUG: Auto-detecting package type
DEBUG: Executing brew install command
DEBUG: SystemExecutor.Run starting
DEBUG: Command execution completed
DEBUG: Installation completed successfully
```

## Filtering Logs

```bash
# View only package-related logs
terraform apply 2>&1 | grep "PackageResource\|BrewAdapter"

# View only service-related logs  
terraform apply 2>&1 | grep "ServiceResource\|DirectCommandStrategy"

# View only command execution logs
terraform apply 2>&1 | grep "SystemExecutor"
```

## Troubleshooting

If you don't see DEBUG logs:
1. Verify environment variables: `echo $TF_LOG $TF_LOG_PROVIDER`
2. Check Terraform version compatibility
3. Ensure provider is properly configured

The logs provide detailed insight into every operation, making it easy to understand what the provider is doing and troubleshoot any issues.
