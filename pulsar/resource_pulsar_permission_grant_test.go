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
	"testing"

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/admin"
	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestPermissionGrant(t *testing.T) {
	resourceName := "pulsar_permission_grant.test"
	cName := acctest.RandString(10)
	tName := acctest.RandString(10)
	nsName := acctest.RandString(10)
	roleName := acctest.RandString(10)

	config := testPulsarPermissionGrant(testWebServiceURL, cName, tName, nsName, roleName, `["produce", "consume"]`)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		IDRefreshName:     resourceName,
		CheckDestroy:      testPulsarPermissionGrantDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testPulsarPermissionGrantExists(),
					resource.TestCheckResourceAttr(resourceName, "namespace", tName+"/"+nsName),
					resource.TestCheckResourceAttr(resourceName, "role", roleName),
					resource.TestCheckResourceAttr(resourceName, "actions.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "actions.*", "produce"),
					resource.TestCheckTypeSetElemAttr(resourceName, "actions.*", "consume"),
				),
			},
			{
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestPermissionGrantUpdate(t *testing.T) {
	resourceName := "pulsar_permission_grant.test"
	cName := acctest.RandString(10)
	tName := acctest.RandString(10)
	nsName := acctest.RandString(10)
	roleName := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		IDRefreshName:     resourceName,
		CheckDestroy:      testPulsarPermissionGrantDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarPermissionGrant(testWebServiceURL, cName, tName, nsName, roleName, `["produce", "consume"]`),
				Check: resource.ComposeTestCheckFunc(
					testPulsarPermissionGrantExists(),
					resource.TestCheckResourceAttr(resourceName, "actions.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "actions.*", "produce"),
					resource.TestCheckTypeSetElemAttr(resourceName, "actions.*", "consume"),
				),
			},
			{
				Config: testPulsarPermissionGrant(testWebServiceURL, cName, tName, nsName, roleName,
					`["produce", "consume", "functions"]`),
				Check: resource.ComposeTestCheckFunc(
					testPulsarPermissionGrantExists(),
					resource.TestCheckResourceAttr(resourceName, "actions.#", "3"),
					resource.TestCheckTypeSetElemAttr(resourceName, "actions.*", "produce"),
					resource.TestCheckTypeSetElemAttr(resourceName, "actions.*", "consume"),
					resource.TestCheckTypeSetElemAttr(resourceName, "actions.*", "functions"),
				),
			},
			{
				Config: testPulsarPermissionGrant(testWebServiceURL, cName, tName, nsName, roleName,
					`["produce"]`),
				Check: resource.ComposeTestCheckFunc(
					testPulsarPermissionGrantExists(),
					resource.TestCheckResourceAttr(resourceName, "actions.#", "1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "actions.*", "produce"),
				),
			},
		},
	})
}

func TestPermissionGrantExternallyRemoved(t *testing.T) {
	resourceName := "pulsar_permission_grant.test"
	cName := acctest.RandString(10)
	tName := acctest.RandString(10)
	nsName := acctest.RandString(10)
	roleName := acctest.RandString(10)

	config := testPulsarPermissionGrant(testWebServiceURL, cName, tName, nsName, roleName, `["produce", "consume"]`)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		IDRefreshName:     resourceName,
		CheckDestroy:      testPulsarPermissionGrantDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testPulsarPermissionGrantExists(),
				),
			},
			{
				PreConfig: func() {
					client, err := sharedClient(testWebServiceURL)
					if err != nil {
						t.Fatalf("ERROR_GETTING_PULSAR_CLIENT: %v", err)
					}

					conn := client.(admin.Client)
					nsFullName := tName + "/" + nsName

					nsName, err := utils.GetNamespaceName(nsFullName)
					if err != nil {
						t.Fatalf("ERROR_PARSING_NAMESPACE: %v", err)
					}

					if err = conn.Namespaces().RevokeNamespacePermission(*nsName, roleName); err != nil {
						t.Fatalf("ERROR_REVOKING_PERMISSION: %v", err)
					}
				},
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testPulsarPermissionGrantExists() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		const resourceName = "pulsar_permission_grant.test"
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("NOT_FOUND: %s", resourceName)
		}

		namespace := rs.Primary.Attributes["namespace"]
		role := rs.Primary.Attributes["role"]

		if namespace == "" || role == "" {
			return fmt.Errorf("namespace or role is empty")
		}

		client := getClientFromMeta(testAccProvider.Meta()).Namespaces()

		nsName, err := utils.GetNamespaceName(namespace)
		if err != nil {
			return fmt.Errorf("ERROR_PARSING_NAMESPACE: %w", err)
		}

		permissions, err := client.GetNamespacePermissions(*nsName)
		if err != nil {
			return fmt.Errorf("ERROR_READ_NAMESPACE_PERMISSIONS: %w", err)
		}

		if _, exists := permissions[role]; !exists {
			return fmt.Errorf("permission grant for role %s on namespace %s does not exist", role, namespace)
		}

		return nil
	}
}

func testPulsarPermissionGrantDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "pulsar_permission_grant" {
			continue
		}

		namespace := rs.Primary.Attributes["namespace"]
		role := rs.Primary.Attributes["role"]

		client := getClientFromMeta(testAccProvider.Meta()).Namespaces()

		nsName, err := utils.GetNamespaceName(namespace)
		if err != nil {
			continue
		}

		permissions, err := client.GetNamespacePermissions(*nsName)
		if err != nil {
			continue
		}

		if _, exists := permissions[role]; exists {
			return fmt.Errorf("permission grant still exists for role %s on namespace %s", role, namespace)
		}
	}

	return nil
}

func testPulsarPermissionGrant(wsURL, cluster, tenant, namespace, role, actions string) string {
	return fmt.Sprintf(`
provider "pulsar" {
  web_service_url = "%s"
}

resource "pulsar_cluster" "test_cluster" {
  cluster = "%s"

  cluster_data {
    web_service_url    = "http://localhost:8080"
    broker_service_url = "pulsar://localhost:6050"
    peer_clusters      = ["standalone"]
  }
}

resource "pulsar_tenant" "test_tenant" {
  tenant           = "%s"
  allowed_clusters = [pulsar_cluster.test_cluster.cluster, "standalone"]
}

resource "pulsar_namespace" "test_namespace" {
  tenant    = pulsar_tenant.test_tenant.tenant
  namespace = "%s"
}

resource "pulsar_permission_grant" "test" {
  namespace = "${pulsar_tenant.test_tenant.tenant}/${pulsar_namespace.test_namespace.namespace}"
  role      = "%s"
  actions   = %s
}
`, wsURL, cluster, tenant, namespace, role, actions)
}
