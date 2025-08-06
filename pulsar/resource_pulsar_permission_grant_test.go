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

// TestPermissionGrant tests the basic lifecycle of a standalone permission grant resource
func TestPermissionGrant(t *testing.T) {
	// Generate random names to avoid conflicts
	resourceName := "pulsar_permission_grant.test"
	cName := acctest.RandString(10)    // cluster name
	tName := acctest.RandString(10)    // tenant name
	nsName := acctest.RandString(10)   // namespace name
	roleName := acctest.RandString(10) // role name

	// Define the config once - used in both create and idempotency test
	config := testPulsarPermissionGrant(testWebServiceURL, cName, tName, nsName, roleName, `["produce", "consume"]`)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },    // Check environment is ready
		ProviderFactories: testAccProviderFactories,         // Use test provider
		IDRefreshName:     resourceName,                     // Resource to check ID refresh
		CheckDestroy:      testPulsarPermissionGrantDestroy, // Verify cleanup after test
		Steps: []resource.TestStep{
			{
				// Step 1: Create the permission grant
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					// Verify the resource exists in Terraform state
					testPulsarPermissionGrantExists(resourceName),
					// Check that all fields are set correctly
					resource.TestCheckResourceAttr(resourceName, "namespace", tName+"/"+nsName),
					resource.TestCheckResourceAttr(resourceName, "role", roleName),
					resource.TestCheckResourceAttr(resourceName, "actions.#", "2"), // 2 actions
					resource.TestCheckTypeSetElemAttr(resourceName, "actions.*", "produce"),
					resource.TestCheckTypeSetElemAttr(resourceName, "actions.*", "consume"),
				),
			},
			{
				// Step 2: Test that no changes are detected (idempotency test)
				Config:             config, // Same exact config - should detect no changes
				PlanOnly:           true,   // Only run terraform plan, don't apply
				ExpectNonEmptyPlan: false,  // We expect no changes needed
			},
		},
	})
}

// TestPermissionGrantUpdate tests updating the actions on an existing permission grant
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
				// Step 1: Create with 2 actions
				Config: testPulsarPermissionGrant(testWebServiceURL, cName, tName, nsName, roleName, `["produce", "consume"]`),
				Check: resource.ComposeTestCheckFunc(
					testPulsarPermissionGrantExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "actions.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "actions.*", "produce"),
					resource.TestCheckTypeSetElemAttr(resourceName, "actions.*", "consume"),
				),
			},
			{
				// Step 2: Update to 3 actions
				Config: testPulsarPermissionGrant(testWebServiceURL, cName, tName, nsName, roleName,
					`["produce", "consume", "functions"]`),
				Check: resource.ComposeTestCheckFunc(
					testPulsarPermissionGrantExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "actions.#", "3"),
					resource.TestCheckTypeSetElemAttr(resourceName, "actions.*", "produce"),
					resource.TestCheckTypeSetElemAttr(resourceName, "actions.*", "consume"),
					resource.TestCheckTypeSetElemAttr(resourceName, "actions.*", "functions"),
				),
			},
			{
				// Step 3: Update to 1 action
				Config: testPulsarPermissionGrant(testWebServiceURL, cName, tName, nsName, roleName,
					`["produce"]`),
				Check: resource.ComposeTestCheckFunc(
					testPulsarPermissionGrantExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "actions.#", "1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "actions.*", "produce"),
				),
			},
		},
	})
}

// TestPermissionGrantExternallyRemoved tests drift detection when permission is removed outside Terraform
func TestPermissionGrantExternallyRemoved(t *testing.T) {
	resourceName := "pulsar_permission_grant.test"
	cName := acctest.RandString(10)
	tName := acctest.RandString(10)
	nsName := acctest.RandString(10)
	roleName := acctest.RandString(10)

	// Define the config once - used in both create and drift detection test
	config := testPulsarPermissionGrant(testWebServiceURL, cName, tName, nsName, roleName, `["produce", "consume"]`)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		IDRefreshName:     resourceName,
		CheckDestroy:      testPulsarPermissionGrantDestroy,
		Steps: []resource.TestStep{
			{
				// Step 1: Create the permission grant
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testPulsarPermissionGrantExists(resourceName),
				),
			},
			{
				// Step 2: Remove the permission externally (simulating drift)
				PreConfig: func() {
					// Get the Pulsar client and manually remove the permission
					client, err := sharedClient(testWebServiceURL)
					if err != nil {
						t.Fatalf("ERROR_GETTING_PULSAR_CLIENT: %v", err)
					}

					conn := client.(admin.Client)
					nsFullName := tName + "/" + nsName

					// Convert to proper namespace type and revoke permission
					nsName, err := utils.GetNamespaceName(nsFullName)
					if err != nil {
						t.Fatalf("ERROR_PARSING_NAMESPACE: %v", err)
					}

					if err = conn.Namespaces().RevokeNamespacePermission(*nsName, roleName); err != nil {
						t.Fatalf("ERROR_REVOKING_PERMISSION: %v", err)
					}
				},
				// Step 3: Plan should detect the resource needs to be recreated
				Config:             config, // Same config - should detect drift and plan recreation
				PlanOnly:           true,
				ExpectNonEmptyPlan: true, // Should detect drift - specifically that actions need to be re-added
			},
		},
	})
}

// Helper function to check if the permission grant exists in Pulsar
func testPulsarPermissionGrantExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Get the resource from Terraform state
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("NOT_FOUND: %s", resourceName)
		}

		// Get the namespace and role from the resource state
		namespace := rs.Primary.Attributes["namespace"]
		role := rs.Primary.Attributes["role"]

		if namespace == "" || role == "" {
			return fmt.Errorf("namespace or role is empty")
		}

		// Get Pulsar client and check if the permission exists
		client := getClientFromMeta(testAccProvider.Meta()).Namespaces()

		// Convert namespace string to proper type
		nsName, err := utils.GetNamespaceName(namespace)
		if err != nil {
			return fmt.Errorf("ERROR_PARSING_NAMESPACE: %w", err)
		}

		permissions, err := client.GetNamespacePermissions(*nsName)
		if err != nil {
			return fmt.Errorf("ERROR_READ_NAMESPACE_PERMISSIONS: %w", err)
		}

		// Check if our role has permissions
		if _, exists := permissions[role]; !exists {
			return fmt.Errorf("permission grant for role %s on namespace %s does not exist", role, namespace)
		}

		return nil
	}
}

// Helper function to verify all permission grants are cleaned up
func testPulsarPermissionGrantDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "pulsar_permission_grant" {
			continue
		}

		namespace := rs.Primary.Attributes["namespace"]
		role := rs.Primary.Attributes["role"]

		// Get Pulsar client and check that the permission no longer exists
		client := getClientFromMeta(testAccProvider.Meta()).Namespaces()

		// Convert namespace string to proper type
		nsName, err := utils.GetNamespaceName(namespace)
		if err != nil {
			// If we can't parse the namespace, that's fine - it might be gone
			continue
		}

		permissions, err := client.GetNamespacePermissions(*nsName)
		if err != nil {
			// If we can't read permissions, that's fine - the namespace might be gone
			continue
		}

		// If the role still has permissions, that's an error
		if _, exists := permissions[role]; exists {
			return fmt.Errorf("permission grant still exists for role %s on namespace %s", role, namespace)
		}
	}

	return nil
}

// Terraform config helper: creates cluster, tenant, namespace, and permission grant
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
