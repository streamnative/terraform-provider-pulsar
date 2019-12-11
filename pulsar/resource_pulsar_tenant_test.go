// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package pulsar

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/streamnative/pulsarctl/pkg/pulsar"
)

var webServiceURL string

func init() {
	url, ok := os.LookupEnv("WEB_SERVICE_URL")
	if !ok {
		webServiceURL = "http://localhost:8080"
	}

	webServiceURL = url
}

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
	testPulsarTenant = fmt.Sprintf(`
provider "pulsar" {
  web_service_url = "%s"
}

resource "pulsar_tenant" "test" {
  tenant = "thanos"
}`, webServiceURL)
)
