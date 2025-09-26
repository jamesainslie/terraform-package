package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestServiceStatusDataSource_Schema tests the schema of the ServiceStatusDataSource.
func TestServiceStatusDataSource_Schema(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"pkg": providerserver.NewProtocol6WithError(New("test")()),
		},
		Steps: []resource.TestStep{
			{
				Config: `
data "pkg_service_status" "test" {
  name = "test-service"
}
				`,
				Check: resource.ComposeTestCheckFunc(
					// Test required attribute
					resource.TestCheckResourceAttr("data.pkg_service_status.test", "name", "test-service"),
					// Test defaults and optionals
					resource.TestCheckResourceAttr("data.pkg_service_status.test", "timeout", "10s"),
					// Computed attributes
					resource.TestCheckResourceAttrSet("data.pkg_service_status.test", "id"),
					resource.TestCheckResourceAttrSet("data.pkg_service_status.test", "running"),
					resource.TestCheckResourceAttrSet("data.pkg_service_status.test", "healthy"),
					resource.TestCheckResourceAttrSet("data.pkg_service_status.test", "version"),
					resource.TestCheckResourceAttrSet("data.pkg_service_status.test", "process_id"),
					resource.TestCheckResourceAttrSet("data.pkg_service_status.test", "start_time"),
					resource.TestCheckResourceAttrSet("data.pkg_service_status.test", "manager_type"),
					resource.TestCheckResourceAttrSet("data.pkg_service_status.test", "ports"),
				),
			},
			// Test with optional configuration
			{
				Config: `
data "pkg_service_status" "full" {
  name = "full-service"
  required_package = "test-package"
  package_manager = "brew"
  timeout = "30s"
  
  health_check {
    command = "health-check-cmd"
    http_endpoint = "http://localhost:8080/health"
    expected_status = 200
    timeout = "10s"
  }
}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.pkg_service_status.full", "name", "full-service"),
					resource.TestCheckResourceAttr("data.pkg_service_status.full", "required_package", "test-package"),
					resource.TestCheckResourceAttr("data.pkg_service_status.full", "package_manager", "brew"),
					resource.TestCheckResourceAttr("data.pkg_service_status.full", "timeout", "30s"),
					// Health check
					resource.TestCheckResourceAttr("data.pkg_service_status.full", "health_check.command", "health-check-cmd"),
					resource.TestCheckResourceAttr("data.pkg_service_status.full", "health_check.http_endpoint", "http://localhost:8080/health"),
					resource.TestCheckResourceAttr("data.pkg_service_status.full", "health_check.expected_status", "200"),
					resource.TestCheckResourceAttr("data.pkg_service_status.full", "health_check.timeout", "10s"),
					// Computed
					resource.TestCheckResourceAttrSet("data.pkg_service_status.full", "id"),
					resource.TestCheckResourceAttrSet("data.pkg_service_status.full", "running"),
					resource.TestCheckResourceAttrSet("data.pkg_service_status.full", "healthy"),
					resource.TestCheckResourceAttrSet("data.pkg_service_status.full", "version"),
					resource.TestCheckResourceAttrSet("data.pkg_service_status.full", "process_id"),
					resource.TestCheckResourceAttrSet("data.pkg_service_status.full", "start_time"),
					resource.TestCheckResourceAttrSet("data.pkg_service_status.full", "manager_type"),
					resource.TestCheckResourceAttrSet("data.pkg_service_status.full", "ports"),
				),
			},
			// Test validation for package_manager
			{
				Config: `
data "pkg_service_status" "invalid" {
  name = "invalid-service"
  package_manager = "invalid_manager"
}
				`,
				ExpectError: regexp.MustCompile(`expected package_manager to be one of \["brew" "apt" "winget" "choco"\]; got invalid_manager`),
			},
		},
	})
}

// TestServiceStatusDataSource_Configure tests the Configure method.
func TestServiceStatusDataSource_Configure(t *testing.T) {
	// Verify detector injection
	t.Skip("Configure test to be implemented after mocking")
}

// TestServiceStatusDataSource_Read tests the Read method with mocked data.
func TestServiceStatusDataSource_Read(t *testing.T) {
	// This would require mocking the service detector
	// For now, test basic structure
	t.Skip("Read test to be implemented with mocks")
}
