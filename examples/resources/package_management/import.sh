#!/bin/bash

# Example: Import an existing package into Terraform state
# This allows you to manage packages that were installed outside of Terraform

# Import with auto-detected manager
terraform import pkg_package.git git

# Import with specific manager
terraform import pkg_package.git brew:git

# Import a repository/tap
terraform import pkg_repo.fonts brew:homebrew/cask-fonts
