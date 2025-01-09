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
	"regexp"
	"strings"
	"testing"

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/admin"
	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func init() {
	initTestWebServiceURL()
}

func TestTenant(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t) },
		ProviderFactories:         testAccProviderFactories,
		PreventPostDestroyRefresh: false,
		CheckDestroy:              testPulsarTenantDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarTenant,
				Check:  resource.ComposeTestCheckFunc(testPulsarTenantExists()),
			},
		},
	})
}

func TestHandleExistingTenant(t *testing.T) {
	tName := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			createTenant(t, tName)
			t.Cleanup(func() {
				if err := getClientFromMeta(testAccProvider.Meta()).Tenants().Delete(tName); err != nil {
					t.Fatalf("ERROR_DELETING_TEST_TENANT: %v", err)
				}
			})
		},
		ProviderFactories:         testAccProviderFactories,
		PreventPostDestroyRefresh: false,
		CheckDestroy:              testPulsarTenantDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testPulsarExistingTenantConfig(testWebServiceURL, tName),
				ExpectError: regexp.MustCompile("Tenant already exist"),
			},
		},
	})
}

func TestImportExistingTenant(t *testing.T) {
	tName := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			createTenant(t, tName)
			t.Cleanup(func() {
				if err := getClientFromMeta(testAccProvider.Meta()).Tenants().Delete(tName); err != nil {
					t.Fatalf("ERROR_DELETING_TEST_TENANT: %v", err)
				}
			})
		},
		CheckDestroy:      testPulsarTenantDestroy,
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				ResourceName:     "pulsar_tenant.test",
				ImportState:      true,
				Config:           testPulsarExistingTenantConfig(testWebServiceURL, tName),
				ImportStateId:    tName,
				ImportStateCheck: testTenantImported(),
			},
		},
	})
}

func createTenant(t *testing.T, tname string) {
	client, err := sharedClient(testWebServiceURL)
	if err != nil {
		t.Fatalf("ERROR_GETTING_PULSAR_CLIENT: %v", err)
	}

	conn := client.(admin.Client)
	if err = conn.Tenants().Create(utils.TenantData{
		Name:            tname,
		AllowedClusters: []string{"standalone"},
	}); err != nil {
		t.Fatalf("ERROR_CREATING_TEST_TENANT: %v", err)
	}
}

func testTenantImported() resource.ImportStateCheckFunc {
	return func(s []*terraform.InstanceState) error {
		if len(s) != 1 {
			return fmt.Errorf("expected %d states, got %d: %#v", 1, len(s), s)
		}

		if len(s[0].Attributes) != 6 {
			return fmt.Errorf("expected %d attrs, got %d: %#v", 6, len(s[0].Attributes), s[0].Attributes)
		}

		return nil
	}
}

func testPulsarTenantExists() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		tenant := "pulsar_tenant.test"
		rs, ok := s.RootModule().Resources[tenant]
		if !ok {
			return fmt.Errorf("NOT_FOUND: %s", tenant)
		}

		client := getClientFromMeta(testAccProvider.Meta()).Tenants()

		_, err := client.Get(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("ERROR_READ_TENANT_DATA: %w", err)
		}

		return nil
	}
}

func testPulsarTenantDestroy(s *terraform.State) error {
	client := getClientFromMeta(testAccProvider.Meta()).Tenants()

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
			return fmt.Errorf("ERROR_RESOURCE_TENANT_STILL_EXISTS: %w", err)
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
	allowed_clusters = ["standalone"]
}`, testWebServiceURL)
)

func testPulsarExistingTenantConfig(url, tname string) string {
	return fmt.Sprintf(`
provider "pulsar" {
	web_service_url = "%s"
}

resource "pulsar_tenant" "test" {
	tenant = "%s"
	allowed_clusters = ["standalone"]
}
`, url, tname)
}

const (
	testPulsarTenantWithAdminRoles1 = "pulsar-tenant-admin-role-1"
	testPulsarTenantWithAdminRoles2 = "pulsar-oauth2-tenant-admin-role@testing.local"
)

func TestTenantWithAdminRoles(t *testing.T) {
	tName := acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProviderFactories:         testAccProviderFactories,
		PreventPostDestroyRefresh: false,
		CheckDestroy:              testPulsarTenantDestroy,
		Steps: []resource.TestStep{
			{
				Config: strings.Replace(testPulsarTenantWithAdminRoles, "thanos", tName, 1),
				Check: resource.ComposeTestCheckFunc(
					testPulsarTenantExists(),
					resource.TestCheckResourceAttr("pulsar_tenant.test", "admin_roles.#", "2"),
					resource.TestCheckTypeSetElemAttr("pulsar_tenant.test", "admin_roles.*", testPulsarTenantWithAdminRoles1),
					resource.TestCheckTypeSetElemAttr("pulsar_tenant.test", "admin_roles.*", testPulsarTenantWithAdminRoles2),
				),
			},
		},
	})
}

func TestTenantUpdateAdminRoles(t *testing.T) {
	tName := acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t) },
		ProviderFactories:         testAccProviderFactories,
		PreventPostDestroyRefresh: false,
		CheckDestroy:              testPulsarTenantDestroy,
		Steps: []resource.TestStep{
			{
				Config: strings.Replace(testPulsarTenantWithAdminRoles, "thanos", tName, 1),
				Check: resource.ComposeTestCheckFunc(
					testPulsarTenantExists(),
				),
			},
			{
				Config: strings.Replace(testPulsarTenantWithAdminRolesUpdated, "thanos", tName, 1),
				Check: resource.ComposeTestCheckFunc(
					testPulsarTenantExists(),
					resource.TestCheckResourceAttr("pulsar_tenant.test", "admin_roles.#", "2"),
					resource.TestCheckTypeSetElemAttr("pulsar_tenant.test", "admin_roles.*", testPulsarTenantWithAdminRoles1),
					resource.TestCheckTypeSetElemAttr("pulsar_tenant.test", "admin_roles.*", testPulsarTenantWithAdminRoles2),
				),
			},
		},
	})
}

func TestTenantAdminRolesDrift(t *testing.T) {
	tName := acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t) },
		ProviderFactories:         testAccProviderFactories,
		PreventPostDestroyRefresh: false,
		CheckDestroy:              testPulsarTenantDestroy,
		Steps: []resource.TestStep{
			{
				Config: strings.Replace(testPulsarTenantWithAdminRoles, "thanos", tName, 1),
				Check: resource.ComposeTestCheckFunc(
					testPulsarTenantExists(),
					resource.TestCheckResourceAttr("pulsar_tenant.test", "admin_roles.#", "2"),
					resource.TestCheckTypeSetElemAttr("pulsar_tenant.test", "admin_roles.*", testPulsarTenantWithAdminRoles1),
					resource.TestCheckTypeSetElemAttr("pulsar_tenant.test", "admin_roles.*", testPulsarTenantWithAdminRoles2),
				),
			},
			{
				Config:             strings.Replace(testPulsarTenantWithAdminRoles, "thanos", tName, 1),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

var testPulsarTenantWithAdminRoles = fmt.Sprintf(`
provider "pulsar" {
  web_service_url = "%s"
}

resource "pulsar_tenant" "test" {
  tenant = "thanos"
  allowed_clusters = ["standalone"]
  admin_roles = ["%s", "%s"]
}`, testWebServiceURL, testPulsarTenantWithAdminRoles1, testPulsarTenantWithAdminRoles2)

var testPulsarTenantWithAdminRolesUpdated = fmt.Sprintf(`
provider "pulsar" {
  web_service_url = "%s"
}

resource "pulsar_tenant" "test" {
  tenant = "thanos"
  allowed_clusters = ["standalone"]
  admin_roles = ["%s", "%s"]
}`, testWebServiceURL, testPulsarTenantWithAdminRoles2, testPulsarTenantWithAdminRoles1)
