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

// Package provider contains Linux-specific acceptance tests for the Terraform package provider.
package provider

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPackageResource_AptLinux(t *testing.T) {
	// Only run on Linux since we're testing APT functionality
	if runtime.GOOS != "linux" {
		t.Skip("Skipping APT test on non-Linux platform")
	}

	// Use a lightweight package that's commonly available and safe to install/remove
	packageName := "curl"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing - Install package
			{
				Config: testAccPackageResourceConfigLinux(packageName, "present"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pkg_package.test", "name", packageName),
					resource.TestCheckResourceAttr("pkg_package.test", "state", "present"),
					resource.TestCheckResourceAttr("pkg_package.test", "manager", "apt"),
					resource.TestCheckResourceAttrSet("pkg_package.test", "id"),
					resource.TestCheckResourceAttrSet("pkg_package.test", "version_actual"),
				),
			},
			// Update testing - Pin package
			{
				Config: testAccPackageResourceConfigLinuxWithPin(packageName, "present", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pkg_package.test", "name", packageName),
					resource.TestCheckResourceAttr("pkg_package.test", "state", "present"),
					resource.TestCheckResourceAttr("pkg_package.test", "pin", "true"),
					resource.TestCheckResourceAttr("pkg_package.test", "manager", "apt"),
				),
			},
			// Unpin package
			{
				Config: testAccPackageResourceConfigLinux(packageName, "present"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pkg_package.test", "name", packageName),
					resource.TestCheckResourceAttr("pkg_package.test", "state", "present"),
					resource.TestCheckResourceAttr("pkg_package.test", "pin", "false"),
				),
			},
			// Delete testing - Remove package
			{
				Config: testAccPackageResourceConfigLinux(packageName, "absent"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pkg_package.test", "name", packageName),
					resource.TestCheckResourceAttr("pkg_package.test", "state", "absent"),
					resource.TestCheckResourceAttr("pkg_package.test", "manager", "apt"),
				),
			},
		},
	})
}

func TestAccManagerInfoDataSource_AptLinux(t *testing.T) {
	// Only run on Linux since we're testing APT functionality
	if runtime.GOOS != "linux" {
		t.Skip("Skipping APT test on non-Linux platform")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccManagerInfoDataSourceConfigLinux(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.pkg_manager_info.test", "detected_manager", "apt"),
					resource.TestCheckResourceAttr("data.pkg_manager_info.test", "platform", "linux"),
					resource.TestCheckResourceAttrSet("data.pkg_manager_info.test", "available"),
					resource.TestCheckResourceAttrSet("data.pkg_manager_info.test", "version"),
					resource.TestCheckResourceAttrSet("data.pkg_manager_info.test", "path"),
				),
			},
		},
	})
}

func TestAccPackageInfoDataSource_AptLinux(t *testing.T) {
	// Only run on Linux since we're testing APT functionality
	if runtime.GOOS != "linux" {
		t.Skip("Skipping APT test on non-Linux platform")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPackageInfoDataSourceConfigLinux(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.pkg_package_info.test", "name", "curl"),
					resource.TestCheckResourceAttr("data.pkg_package_info.test", "manager", "apt"),
					resource.TestCheckResourceAttrSet("data.pkg_package_info.test", "installed"),
					resource.TestCheckResourceAttrSet("data.pkg_package_info.test", "available_versions.#"),
					resource.TestCheckResourceAttrSet("data.pkg_package_info.test", "repository"),
					// Ensure we actually got valid version information
					resource.TestCheckResourceAttrWith("data.pkg_package_info.test", "available_versions.#", func(value string) error {
						if value == "0" {
							return fmt.Errorf("expected at least one available version for curl, got 0")
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestAccPackageSearchDataSource_AptLinux(t *testing.T) {
	// Only run on Linux since we're testing APT functionality
	if runtime.GOOS != "linux" {
		t.Skip("Skipping APT test on non-Linux platform")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPackageSearchDataSourceConfigLinux(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.pkg_package_search.test", "query", "curl"),
					resource.TestCheckResourceAttr("data.pkg_package_search.test", "manager", "apt"),
					resource.TestCheckResourceAttrSet("data.pkg_package_search.test", "results.#"),
					// Ensure search actually found the curl package
					resource.TestCheckResourceAttrWith("data.pkg_package_search.test", "results.#", func(value string) error {
						if value == "0" {
							return fmt.Errorf("expected search for 'curl' to return at least one result, got 0")
						}
						return nil
					}),
					// Verify first result has valid structure
					resource.TestCheckResourceAttrSet("data.pkg_package_search.test", "results.0.name"),
				),
			},
		},
	})
}

func TestAccInstalledPackagesDataSource_AptLinux(t *testing.T) {
	// Only run on Linux since we're testing APT functionality
	if runtime.GOOS != "linux" {
		t.Skip("Skipping APT test on non-Linux platform")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstalledPackagesDataSourceConfigLinux(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.pkg_installed_packages.test", "manager", "apt"),
					resource.TestCheckResourceAttrSet("data.pkg_installed_packages.test", "packages.#"),
					// Ensure we can successfully get info for at least some packages
					resource.TestCheckResourceAttrWith("data.pkg_installed_packages.test", "packages.#", func(value string) error {
						if value == "0" {
							return fmt.Errorf("expected at least one installed package, got 0")
						}
						return nil
					}),
					// Check that first package has valid structure
					resource.TestCheckResourceAttrSet("data.pkg_installed_packages.test", "packages.0.name"),
					resource.TestCheckResourceAttrSet("data.pkg_installed_packages.test", "packages.0.version"),
				),
			},
		},
	})
}

func TestAccOutdatedPackagesDataSource_AptLinux(t *testing.T) {
	// Only run on Linux since we're testing APT functionality
	if runtime.GOOS != "linux" {
		t.Skip("Skipping APT test on non-Linux platform")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOutdatedPackagesDataSourceConfigLinux(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.pkg_outdated_packages.test", "manager", "apt"),
					resource.TestCheckResourceAttrSet("data.pkg_outdated_packages.test", "packages.#"),
				),
			},
		},
	})
}

// Test configuration functions for Linux/APT

func testAccPackageResourceConfigLinux(name, state string) string {
	return fmt.Sprintf(`
provider "pkg" {
  default_manager = "apt"
  assume_yes      = true
  update_cache    = "never"
}

resource "pkg_package" "test" {
  name    = %[1]q
  state   = %[2]q
  manager = "apt"
}
`, name, state)
}

func testAccPackageResourceConfigLinuxWithPin(name, state string, pin bool) string {
	return fmt.Sprintf(`
provider "pkg" {
  default_manager = "apt"
  assume_yes      = true
  update_cache    = "never"
}

resource "pkg_package" "test" {
  name    = %[1]q
  state   = %[2]q
  manager = "apt"
  pin     = %[3]t
}
`, name, state, pin)
}

func testAccManagerInfoDataSourceConfigLinux() string {
	return `
provider "pkg" {
  default_manager = "apt"
  update_cache    = "never"
}

data "pkg_manager_info" "test" {
  manager = "auto"
}
`
}

func testAccPackageInfoDataSourceConfigLinux() string {
	return `
provider "pkg" {
  default_manager = "apt"
  update_cache    = "never"
}

data "pkg_package_info" "test" {
  name    = "curl"
  manager = "apt"
}
`
}

func testAccPackageSearchDataSourceConfigLinux() string {
	return `
provider "pkg" {
  default_manager = "apt"
  update_cache    = "never"
}

data "pkg_package_search" "test" {
  query   = "curl"
  manager = "apt"
}
`
}

func testAccInstalledPackagesDataSourceConfigLinux() string {
	return `
provider "pkg" {
  default_manager = "apt"
  update_cache    = "never"
}

data "pkg_installed_packages" "test" {
  manager = "apt"
  filter  = "base-files"  # Filter to only one common system package for faster test execution
}
`
}

func testAccOutdatedPackagesDataSourceConfigLinux() string {
	return `
provider "pkg" {
  default_manager = "apt"
  update_cache    = "never"
}

data "pkg_outdated_packages" "test" {
  manager = "apt"
}
`
}
