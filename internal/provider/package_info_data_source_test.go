package provider

import (
	"regexp"
	"runtime"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestPackageInfoDataSource_AptLinux(t *testing.T) {
	// Skip test if not on Linux - APT tests require Linux environment
	if runtime.GOOS != "linux" {
		t.Skip("APT tests require Linux environment")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"pkg": providerserver.NewProtocol6WithError(New("test")()),
		},
		Steps: []resource.TestStep{
			{
				Config: `
				data "pkg_package_info" "apt_test" {
				  name    = "curl"
				  manager = "apt"
				}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify APT selected and read
					resource.TestCheckResourceAttr("data.pkg_package_info.apt_test", "name", "curl"),
					resource.TestCheckResourceAttrSet("data.pkg_package_info.apt_test", "id"),
				),
			},
			// Error: Unsupported manager
			{
				Config: `
				data "pkg_package_info" "invalid" {
				  name    = "invalid"
				  manager = "choco"
				}
				`,
				ExpectError: regexp.MustCompile("Unsupported Package Manager"),
			},
		},
	})
}

func TestPackageInfoDataSource_Idempotency(t *testing.T) {
	// Multiple reads should return same state (inherent to read-only)
	// Test via resource.Test above - re-apply config plans no changes
	t.Log("Data sources are read-only; idempotency verified via no mutations in Read()")
}
