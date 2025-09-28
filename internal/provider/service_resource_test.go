package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestServiceResource_Schema tests the schema of the ServiceResource.
func TestServiceResource_Schema(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping service resource test in short mode")
	}
	
	t.Skip("Service tests temporarily disabled - requires service management refactoring")
	
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"pkg": providerserver.NewProtocol6WithError(New("test")()),
		},
		Steps: []resource.TestStep{
			{
				Config: `
resource "pkg_service" "test" {
  service_name = "test-service"
}
				`,
				Check: resource.ComposeTestCheckFunc(
					// Test that required attributes are accepted
					resource.TestCheckResourceAttr("pkg_service.test", "service_name", "test-service"),
					// Test defaults
					resource.TestCheckResourceAttr("pkg_service.test", "state", "running"),
					resource.TestCheckResourceAttr("pkg_service.test", "startup", "enabled"),
					resource.TestCheckResourceAttr("pkg_service.test", "validate_package", "true"),
					resource.TestCheckResourceAttr("pkg_service.test", "management_strategy", "auto"),
					resource.TestCheckResourceAttr("pkg_service.test", "wait_for_healthy", "true"),
				),
			},
			// Test with full configuration including custom_commands and health_check
			{
				Config: `
resource "pkg_service" "full" {
  service_name = "full-service"
  state = "running"
  startup = "enabled"
  validate_package = true
  package_name = "test-package"
  management_strategy = "direct_command"
  wait_for_healthy = true
  wait_timeout = "120s"
  
  health_check {
    type = "command"
    command = "health-check-cmd"
    timeout = "30s"
    expected_code = 0
  }
  
  custom_commands {
    start = ["start-cmd", "arg1"]
    stop = ["stop-cmd"]
    restart = ["restart-cmd"]
    status = ["status-cmd"]
  }
}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("pkg_service.full", "service_name", "full-service"),
					resource.TestCheckResourceAttr("pkg_service.full", "state", "running"),
					resource.TestCheckResourceAttr("pkg_service.full", "startup", "enabled"),
					resource.TestCheckResourceAttr("pkg_service.full", "validate_package", "true"),
					resource.TestCheckResourceAttr("pkg_service.full", "package_name", "test-package"),
					resource.TestCheckResourceAttr("pkg_service.full", "management_strategy", "direct_command"),
					resource.TestCheckResourceAttr("pkg_service.full", "wait_for_healthy", "true"),
					resource.TestCheckResourceAttr("pkg_service.full", "wait_timeout", "120s"),
					// Verify health_check block
					resource.TestCheckResourceAttr("pkg_service.full", "health_check.type", "command"),
					resource.TestCheckResourceAttr("pkg_service.full", "health_check.command", "health-check-cmd"),
					resource.TestCheckResourceAttr("pkg_service.full", "health_check.timeout", "30s"),
					resource.TestCheckResourceAttr("pkg_service.full", "health_check.expected_code", "0"),
					// Verify custom_commands block
					resource.TestCheckResourceAttr("pkg_service.full", "custom_commands.start.0", "start-cmd"),
					resource.TestCheckResourceAttr("pkg_service.full", "custom_commands.start.1", "arg1"),
					resource.TestCheckResourceAttr("pkg_service.full", "custom_commands.stop.0", "stop-cmd"),
					resource.TestCheckResourceAttr("pkg_service.full", "custom_commands.restart.0", "restart-cmd"),
					resource.TestCheckResourceAttr("pkg_service.full", "custom_commands.status.0", "status-cmd"),
					// Computed attributes should be set
					resource.TestCheckResourceAttrSet("pkg_service.full", "id"),
					resource.TestCheckResourceAttrSet("pkg_service.full", "running"),
					resource.TestCheckResourceAttrSet("pkg_service.full", "healthy"),
					resource.TestCheckResourceAttrSet("pkg_service.full", "enabled"),
					resource.TestCheckResourceAttrSet("pkg_service.full", "version"),
					resource.TestCheckResourceAttrSet("pkg_service.full", "process_id"),
					resource.TestCheckResourceAttrSet("pkg_service.full", "start_time"),
					resource.TestCheckResourceAttrSet("pkg_service.full", "manager_type"),
					resource.TestCheckResourceAttrSet("pkg_service.full", "package"),
					resource.TestCheckResourceAttrSet("pkg_service.full", "ports"),
					resource.TestCheckResourceAttrSet("pkg_service.full", "metadata"),
					resource.TestCheckResourceAttrSet("pkg_service.full", "last_updated"),
				),
			},
			// Test validation errors
			{
				Config: `
resource "pkg_service" "invalid" {
  service_name = "invalid-service"
  state = "invalid_state"  # Invalid value
}
				`,
				ExpectError: regexp.MustCompile(`expected state to be one of \["running" "stopped"\]; got invalid_state`),
			},
			{
				Config: `
resource "pkg_service" "invalid_strategy" {
  service_name = "strategy-service"
  management_strategy = "invalid_strategy"  # Invalid value
}
				`,
				ExpectError: regexp.MustCompile(`expected management_strategy to be one of \["auto" "brew_services" "direct_command" "launchd" "process_only"\]; got invalid_strategy`),
			},
		},
	})
}

// TestServiceResource_Configure tests the Configure method.
func TestServiceResource_Configure(t *testing.T) {
	// This test would verify provider data injection
	// Implementation depends on ProviderData structure
	t.Skip("Configure test to be implemented after ProviderData is mocked")
}
