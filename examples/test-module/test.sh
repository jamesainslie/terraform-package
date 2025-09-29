#!/bin/bash

# Test script for the Terraform Package Provider
set -e

echo " Testing Terraform Package Provider with wget installation"
echo

# Set up Terraform to use our local provider
export TF_CLI_CONFIG_FILE="$(pwd)/.terraformrc"

# Initialize Terraform
echo " Initializing Terraform..."
terraform init

echo
echo " Planning Terraform changes..."
terraform plan

echo
echo " Showing what data sources discovered..."
echo "Manager Info:"
terraform plan -out=plan.out >/dev/null 2>&1 || true
terraform show -json plan.out 2>/dev/null | jq -r '.planned_values.root_module.resources[] | select(.type == "pkg_manager_info") | .values' 2>/dev/null || echo "  (Data will be available after apply)"

echo
echo "Registry Lookup:"
terraform show -json plan.out 2>/dev/null | jq -r '.planned_values.root_module.resources[] | select(.type == "pkg_registry_lookup") | .values' 2>/dev/null || echo "  (Data will be available after apply)"

echo
echo " Applying Terraform configuration..."
echo "This will install wget using Homebrew if you're on macOS"
echo

# Prompt for confirmation
read -p "Continue with installation? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo " Installation cancelled"
    exit 0
fi

# Apply the configuration
terraform apply -auto-approve

echo
echo " Installation complete!"
echo
echo " Final state:"
terraform output -json

echo
echo " Verifying wget installation:"
which wget && wget --version | head -1 || echo "wget not found in PATH"

echo
echo " Cleanup (removing wget):"
read -p "Remove wget? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    # Update the configuration to remove wget
    sed -i '' 's/state   = "present"/state   = "absent"/' main.tf
    terraform apply -auto-approve
    echo " wget removed"
else
    echo "ℹ️  wget left installed"
fi

echo
echo " Test complete!"
