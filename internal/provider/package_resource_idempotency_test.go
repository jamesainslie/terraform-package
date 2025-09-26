// MIT License
//
// Copyright (c) 2025 Terraform Package Provider Contributors
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jamesainslie/terraform-package/internal/adapters"
)

// TestIdempotencyLogic tests the core idempotency logic without mocking the full resource lifecycle
func TestIdempotencyLogic_AlreadyInstalled(t *testing.T) {
	// Test that when a package is already installed with the correct version,
	// no installation should be attempted

	// Simulate package already installed with correct version
	installedInfo := &adapters.PackageInfo{
		Name:      "test-package",
		Installed: true,
		Version:   "1.0.0",
		Type:      adapters.PackageTypeFormula,
	}

	// Test case 1: No version specified (any version is acceptable)
	desiredVersion := ""
	shouldInstall := shouldInstallPackage(installedInfo, desiredVersion)
	assert.False(t, shouldInstall, "Should not install when package is already installed and no specific version required")

	// Test case 2: Exact version match
	desiredVersion = "1.0.0"
	shouldInstall = shouldInstallPackage(installedInfo, desiredVersion)
	assert.False(t, shouldInstall, "Should not install when package is already installed with exact version match")
}

func TestIdempotencyLogic_NotInstalled(t *testing.T) {
	// Test that when a package is not installed, installation should proceed

	// Simulate package not installed
	notInstalledInfo := &adapters.PackageInfo{
		Name:      "test-package",
		Installed: false,
	}

	desiredVersion := "1.0.0"
	shouldInstall := shouldInstallPackage(notInstalledInfo, desiredVersion)
	assert.True(t, shouldInstall, "Should install when package is not installed")
}

func TestIdempotencyLogic_VersionMismatch(t *testing.T) {
	// Test that when a package is installed with wrong version, installation should proceed

	// Simulate package installed with different version
	installedInfo := &adapters.PackageInfo{
		Name:      "test-package",
		Installed: true,
		Version:   "1.0.0", // Different from desired
		Type:      adapters.PackageTypeFormula,
	}

	desiredVersion := "2.0.0"
	shouldInstall := shouldInstallPackage(installedInfo, desiredVersion)
	assert.True(t, shouldInstall, "Should install when package is installed with different version")
}

// Helper function that encapsulates the idempotency logic
func shouldInstallPackage(info *adapters.PackageInfo, desiredVersion string) bool {
	if info.Installed {
		// Package is already installed - check version compatibility
		if desiredVersion == "" || desiredVersion == info.Version {
			// Already installed with correct version, no action needed
			return false
		}
		// Different version requested - should install (upgrade/downgrade)
		return true
	}
	// Package not installed - should install
	return true
}
