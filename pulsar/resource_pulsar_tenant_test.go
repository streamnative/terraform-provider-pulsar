package pulsar

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/streamnative/pulsarctl/pkg/pulsar"
	"testing"
)

func TestTenant(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t) },
		Providers:                 testAccProviders,
		PreventPostDestroyRefresh: false,
		CheckDestroy:              testPulsarTenantDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarTenant,
				Check:  resource.ComposeTestCheckFunc(testPulsarTenantExists("pulsar_tenant.test")),
			},
		},
	})

}

func testPulsarTenantExists(tenant string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[tenant]
		if !ok {
			return fmt.Errorf("NOT_FOUND: %s", tenant)
		}

		client := testAccProvider.Meta().(pulsar.Client).Tenants()

		_, err := client.Get(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("ERROR_READ_TENANT_DATA: %w", err)
		}

		return nil
	}
}

func testPulsarTenantDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(pulsar.Client).Tenants()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "pulsar_tenant" {
			continue
		}

		// tenant name is set as ID for the resource
		tenant := rs.Primary.ID

		resp, err := client.Get(tenant)
		if err != nil {
			return nil
		}

		// we don't need to check for allowed_clusters and admin_roles as they can be empty but still
		// tenant name could be present
		if resp.Name != "" {
			return fmt.Errorf("ERROR_RESOURCE_TENANT_STILL_EXISITS: %w", err)
		}

	}

	return nil
}

var (
	testPulsarTenant = `
resource "pulsar_tenant" "test" {
	tenant           = "test-my-tenant"
    allowed_clusters = ["pulsar-cluster-1"]
}
`
)
