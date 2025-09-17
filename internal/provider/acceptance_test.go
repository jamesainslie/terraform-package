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

// Package provider contains acceptance tests for the Terraform package provider.
package provider

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPackageResource_Hello(t *testing.T) {
	// Only run on macOS since we only support Homebrew in Phase 2
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping Homebrew test on non-macOS platform")
	}

	// NOTE: During this test, you will see expected stderr messages like:
	// "Error: Cask 'jq' is unavailable: No Cask with this name exists."
	// This is NORMAL behavior - the brew adapter tries both cask and formula
	// detection to determine the correct package type. These error messages
	// indicate the detection logic is working correctly.

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccPackageResourceConfig("jq", "present"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pkg_package.test", "name", "jq"),
					resource.TestCheckResourceAttr("pkg_package.test", "state", "present"),
					resource.TestCheckResourceAttrSet("pkg_package.test", "id"),
					resource.TestCheckResourceAttrSet("pkg_package.test", "version_actual"),
				),
			},
			// Update testing (change pin status)
			{
				Config: testAccPackageResourceConfigWithPin("jq", "present", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pkg_package.test", "name", "jq"),
					resource.TestCheckResourceAttr("pkg_package.test", "state", "present"),
					resource.TestCheckResourceAttr("pkg_package.test", "pin", "true"),
				),
			},
			// Delete testing
			{
				Config: testAccPackageResourceConfig("jq", "absent"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pkg_package.test", "name", "jq"),
					resource.TestCheckResourceAttr("pkg_package.test", "state", "absent"),
				),
			},
		},
	})
}

func TestAccManagerInfoDataSource(t *testing.T) {
	// Only run on macOS since we only support Homebrew in Phase 2
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping Homebrew test on non-macOS platform")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccManagerInfoDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.pkg_manager_info.test", "detected_manager", "brew"),
					resource.TestCheckResourceAttr("data.pkg_manager_info.test", "platform", "darwin"),
					resource.TestCheckResourceAttrSet("data.pkg_manager_info.test", "available"),
					resource.TestCheckResourceAttrSet("data.pkg_manager_info.test", "version"),
					resource.TestCheckResourceAttrSet("data.pkg_manager_info.test", "path"),
				),
			},
		},
	})
}

func TestAccRegistryLookupDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRegistryLookupDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.pkg_registry_lookup.test", "logical_name", "jq"),
					resource.TestCheckResourceAttr("data.pkg_registry_lookup.test", "found", "true"),
					resource.TestCheckResourceAttr("data.pkg_registry_lookup.test", "darwin", "jq"),
					resource.TestCheckResourceAttr("data.pkg_registry_lookup.test", "linux", "jq"),
					resource.TestCheckResourceAttr("data.pkg_registry_lookup.test", "windows", "jqlang.jq"),
				),
			},
		},
	})
}

func TestAccPackageInfoDataSource(t *testing.T) {
	// Only run on macOS since we only support Homebrew in Phase 2
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping Homebrew test on non-macOS platform")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPackageInfoDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.pkg_package_info.test", "name", "jq"),
					resource.TestCheckResourceAttr("data.pkg_package_info.test", "manager", "brew"),
					resource.TestCheckResourceAttrSet("data.pkg_package_info.test", "installed"),
					resource.TestCheckResourceAttrSet("data.pkg_package_info.test", "available_versions.#"),
					resource.TestCheckResourceAttrSet("data.pkg_package_info.test", "repository"),
					// Ensure we actually got valid version information
					resource.TestCheckResourceAttrWith("data.pkg_package_info.test", "available_versions.#", func(value string) error {
						if value == "0" {
							return fmt.Errorf("expected at least one available version for jq, got 0")
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestAccPackageSearchDataSource(t *testing.T) {
	// Only run on macOS since we only support Homebrew in Phase 2
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping Homebrew test on non-macOS platform")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPackageSearchDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.pkg_package_search.test", "query", "jq"),
					resource.TestCheckResourceAttr("data.pkg_package_search.test", "manager", "brew"),
					resource.TestCheckResourceAttrSet("data.pkg_package_search.test", "results.#"),
					// Ensure search actually found the jq package
					resource.TestCheckResourceAttrWith("data.pkg_package_search.test", "results.#", func(value string) error {
						if value == "0" {
							return fmt.Errorf("expected search for 'jq' to return at least one result, got 0")
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

func TestAccInstalledPackagesDataSource(t *testing.T) {
	// Only run on macOS since we only support Homebrew in Phase 2
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping Homebrew test on non-macOS platform")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstalledPackagesDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.pkg_installed_packages.test", "manager", "brew"),
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
					// Ensure the repository is not "unknown" (which indicates brew info failure)
					resource.TestCheckResourceAttrWith("data.pkg_installed_packages.test", "packages.0.repository", func(value string) error {
						if value == "unknown" {
							return fmt.Errorf("first package has 'unknown' repository, indicating brew info command failure")
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestAccOutdatedPackagesDataSource(t *testing.T) {
	// Only run on macOS since we only support Homebrew in Phase 2
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping Homebrew test on non-macOS platform")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOutdatedPackagesDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.pkg_outdated_packages.test", "manager", "brew"),
					resource.TestCheckResourceAttrSet("data.pkg_outdated_packages.test", "packages.#"),
				),
			},
		},
	})
}

func TestAccVersionHistoryDataSource(t *testing.T) {
	// Only run on macOS since we only support Homebrew in Phase 2
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping Homebrew test on non-macOS platform")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVersionHistoryDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.pkg_version_history.test", "name", "jq"),
					resource.TestCheckResourceAttr("data.pkg_version_history.test", "manager", "brew"),
					resource.TestCheckResourceAttrSet("data.pkg_version_history.test", "versions.#"),
				),
			},
		},
	})
}

func TestAccDependenciesDataSource(t *testing.T) {
	// Only run on macOS since we only support Homebrew in Phase 2
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping Homebrew test on non-macOS platform")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDependenciesDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.pkg_dependencies.test", "name", "git"),
					resource.TestCheckResourceAttr("data.pkg_dependencies.test", "manager", "brew"),
					resource.TestCheckResourceAttr("data.pkg_dependencies.test", "type", "runtime"),
					resource.TestCheckResourceAttrSet("data.pkg_dependencies.test", "dependencies.#"),
				),
			},
		},
	})
}

func TestAccRepositoryResource_Tap(t *testing.T) {
	// Only run on macOS since we only support Homebrew in Phase 2
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping Homebrew test on non-macOS platform")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing - Test with a third-party tap
			{
				Config: testAccRepositoryResourceConfig("jamesainslie/antimoji"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pkg_repo.test", "manager", "brew"),
					resource.TestCheckResourceAttr("pkg_repo.test", "name", "jamesainslie/antimoji"),
					resource.TestCheckResourceAttr("pkg_repo.test", "uri", "jamesainslie/antimoji"),
					resource.TestCheckResourceAttrSet("pkg_repo.test", "id"),
					resource.TestCheckResourceAttr("pkg_repo.test", "enabled", "true"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "pkg_repo.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     "brew:jamesainslie/antimoji",
			},
		},
	})
}

func testAccPackageResourceConfig(name, state string) string {
	return fmt.Sprintf(`
provider "pkg" {
  default_manager = "brew"
  assume_yes      = true
  update_cache    = "never"
}

resource "pkg_package" "test" {
  name  = %[1]q
  state = %[2]q
}
`, name, state)
}

func testAccPackageResourceConfigWithPin(name, state string, pin bool) string {
	return fmt.Sprintf(`
provider "pkg" {
  default_manager = "brew"
  assume_yes      = true
  update_cache    = "never"
}

resource "pkg_package" "test" {
  name  = %[1]q
  state = %[2]q
  pin   = %[3]t
}
`, name, state, pin)
}

func testAccManagerInfoDataSourceConfig() string {
	return `
provider "pkg" {
  default_manager = "brew"
  update_cache    = "never"
}

data "pkg_manager_info" "test" {
  manager = "auto"
}
`
}

func testAccRegistryLookupDataSourceConfig() string {
	return `
provider "pkg" {
  default_manager = "brew"
  update_cache    = "never"
}

data "pkg_registry_lookup" "test" {
  logical_name = "jq"
}
`
}

func testAccPackageInfoDataSourceConfig() string {
	return `
provider "pkg" {
  default_manager = "brew"
  update_cache    = "never"
}

data "pkg_package_info" "test" {
  name    = "jq"
  manager = "brew"
}
`
}

func testAccPackageSearchDataSourceConfig() string {
	return `
provider "pkg" {
  default_manager = "brew"
  update_cache    = "never"
}

data "pkg_package_search" "test" {
  query   = "jq"
  manager = "brew"
}
`
}

func testAccInstalledPackagesDataSourceConfig() string {
	return `
provider "pkg" {
  default_manager = "brew"
  update_cache    = "never"
}

data "pkg_installed_packages" "test" {
  manager = "brew"
  filter  = "git"  # Filter to only one common package for faster test execution
}
`
}

func testAccOutdatedPackagesDataSourceConfig() string {
	return `
provider "pkg" {
  default_manager = "brew"
  update_cache    = "never"
}

data "pkg_outdated_packages" "test" {
  manager = "brew"
}
`
}

func testAccVersionHistoryDataSourceConfig() string {
	return `
provider "pkg" {
  default_manager = "brew"
  update_cache    = "never"
}

data "pkg_version_history" "test" {
  name    = "jq"
  manager = "brew"
}
`
}

func testAccDependenciesDataSourceConfig() string {
	return `
provider "pkg" {
  default_manager = "brew"
  update_cache    = "never"
}

data "pkg_dependencies" "test" {
  name    = "git"
  manager = "brew"
  type    = "runtime"
}
`
}

func testAccRepositoryResourceConfig(tapName string) string {
	return fmt.Sprintf(`
provider "pkg" {
  default_manager = "brew"
  update_cache    = "never"
}

resource "pkg_repo" "test" {
  manager = "brew"
  name    = %[1]q
  uri     = %[1]q
}
`, tapName)
}
