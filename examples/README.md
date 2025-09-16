# Terraform Package Provider Examples

This directory contains practical examples demonstrating how to use the Terraform Package Provider for cross-platform package management.

## Available Examples

### Provider Configuration
- **[provider/](./provider/)** - Basic and advanced provider configuration examples

### Package Management  
- **[test-module/](./test-module/)** - Complete test module with real package installation
- **[resources/package_management/](./resources/package_management/)** - Package and repository resource examples

### Data Sources
- **[data-sources/package_discovery/](./data-sources/package_discovery/)** - Package discovery and information examples

## Running Examples

### Prerequisites
- Terraform 1.5+
- Platform-specific package manager:
  - **macOS**: Homebrew (`brew`)
  - **Linux**: APT (`apt-get`) - Phase 3
  - **Windows**: winget or Chocolatey - Phase 4

### Basic Usage

1. **Navigate to an example directory**:
   ```bash
   cd examples/test-module/
   ```

2. **Initialize Terraform**:
   ```bash
   terraform init
   ```

3. **Plan the changes**:
   ```bash
   terraform plan
   ```

4. **Apply the configuration**:
   ```bash
   terraform apply
   ```

### Local Development

For testing with a locally built provider:

1. **Build the provider**:
   ```bash
   make build
   ```

2. **Set up dev override**:
   ```bash
   export TF_CLI_CONFIG_FILE="$(pwd)/.terraformrc"
   ```

3. **Run the example**:
   ```bash
   cd examples/test-module/
   terraform init
   terraform apply
   ```

## Example Descriptions

### test-module/
Complete demonstration of provider capabilities:
- Package installation and management
- Data source usage for discovery
- Cross-platform name resolution
- Real package operations with GNU Hello

### provider/
Provider configuration examples:
- Basic setup with auto-detection
- Advanced configuration with custom paths
- Platform-specific configurations
- Security and timeout settings

Each example includes detailed comments explaining the configuration and expected behavior.
