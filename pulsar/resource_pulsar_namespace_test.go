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
	"strings"
	"testing"

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/admin"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func init() {
	initTestWebServiceURL()

	resource.AddTestSweepers("pulsar_namespace", &resource.Sweeper{
		Name: "pulsar_namespace",
		F:    testSweepNS,
		Dependencies: []string{
			"pulsar_cluster",
			"pulsar_tenant",
		},
	})
}

func testSweepNS(url string) error {

	client, err := sharedClient(url)
	if err != nil {
		return fmt.Errorf("ERROR_GETTING_PULSAR_CLIENT: %w", err)
	}

	conn := client.(admin.Client)

	tenants, err := conn.Tenants().List()
	if err != nil {
		return fmt.Errorf("ERROR_GETTING_TENANTS: %w", err)
	}

	for _, t := range tenants {
		nsList, err := conn.Namespaces().GetNamespaces(t)
		if err != nil {
			return fmt.Errorf("ERROR_GETTING_NAMESPACE_LIST: %w", err)
		}

		for _, ns := range nsList {
			return conn.Namespaces().DeleteNamespace(ns)
		}
	}

	return nil
}

func TestNamespace(t *testing.T) {

	resourceName := "pulsar_namespace.test"
	cName := acctest.RandString(10)
	tName := acctest.RandString(10)
	nsName := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		IDRefreshName:     resourceName,
		CheckDestroy:      testPulsarNamespaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarNamespace(testWebServiceURL, cName, tName, nsName),
				Check: resource.ComposeTestCheckFunc(
					testPulsarNamespaceExists(resourceName),
				),
			},
			{
				Config:             testPulsarNamespace(testWebServiceURL, cName, tName, nsName),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestNamespaceWithUpdate(t *testing.T) {

	resourceName := "pulsar_namespace.test"
	cName := acctest.RandString(10)
	tName := acctest.RandString(10)
	nsName := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		IDRefreshName:     resourceName,
		CheckDestroy:      testPulsarNamespaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarNamespaceWithoutOptionals(testWebServiceURL, cName, tName, nsName),
				Check: resource.ComposeTestCheckFunc(
					testPulsarNamespaceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "dispatch_rate.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "subscription_dispatch_rate.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "retention_policies.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "namespace_config.#", "0"),
					resource.TestCheckNoResourceAttr(resourceName, "enable_deduplication"),
					resource.TestCheckNoResourceAttr(resourceName, "permission_grant.#"),
				),
			},
			{
				Config: testPulsarNamespace(testWebServiceURL, cName, tName, nsName),
				Check: resource.ComposeTestCheckFunc(
					testPulsarNamespaceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "dispatch_rate.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "subscription_dispatch_rate.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "retention_policies.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "namespace_config.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "enable_deduplication", "true"),
					resource.TestCheckResourceAttr(resourceName, "permission_grant.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "permission_grant.0.role", "some-role-1"),
					resource.TestCheckResourceAttr(resourceName, "permission_grant.0.actions.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "permission_grant.0.actions.0", "consume"),
					resource.TestCheckResourceAttr(resourceName, "permission_grant.0.actions.1", "functions"),
					resource.TestCheckResourceAttr(resourceName, "permission_grant.0.actions.2", "produce"),
					resource.TestCheckResourceAttr(resourceName, "permission_grant.1.role", "some-role-2"),
					resource.TestCheckResourceAttr(resourceName, "permission_grant.1.actions.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "permission_grant.1.actions.0", "consume"),
					resource.TestCheckResourceAttr(resourceName, "permission_grant.1.actions.1", "produce"),
					resource.TestCheckResourceAttr(resourceName, "topic_auto_creation.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "topic_auto_creation.0.enable", "true"),
					resource.TestCheckResourceAttr(resourceName, "topic_auto_creation.0.type", "partitioned"),
					resource.TestCheckResourceAttr(resourceName, "topic_auto_creation.0.partitions", "3"),
				),
			},
		},
	})
}

func TestNamespaceWithUndefinedOptionalsUpdate(t *testing.T) {

	resourceName := "pulsar_namespace.test"
	cName := acctest.RandString(10)
	tName := acctest.RandString(10)
	nsName := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		IDRefreshName:     resourceName,
		CheckDestroy:      testPulsarNamespaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarNamespaceWithoutOptionals(testWebServiceURL, cName, tName, nsName),
				Check: resource.ComposeTestCheckFunc(
					testPulsarNamespaceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "dispatch_rate.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "subscription_dispatch_rate.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "retention_policies.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "backlog_quota.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "namespace_config.#", "0"),
					resource.TestCheckNoResourceAttr(resourceName, "enable_deduplication"),
					resource.TestCheckNoResourceAttr(resourceName, "permission_grant.#"),
				),
			},
			{
				Config: testPulsarNamespaceWithUndefinedOptionalsInNsConf(testWebServiceURL, cName, tName, nsName),
				Check: resource.ComposeTestCheckFunc(
					testPulsarNamespaceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "dispatch_rate.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "subscription_dispatch_rate.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "retention_policies.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "backlog_quota.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "namespace_config.#", "1"),
					resource.TestCheckNoResourceAttr(resourceName, "enable_deduplication"),
					resource.TestCheckNoResourceAttr(resourceName, "permission_grant.#"),
				),
			},
			{
				Config:             testPulsarNamespaceWithUndefinedOptionalsInNsConf(testWebServiceURL, cName, tName, nsName),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				Config:             testPulsarNamespaceWithoutOptionals(testWebServiceURL, cName, tName, nsName),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestNamespaceWithPermissionGrantUpdate(t *testing.T) {

	resourceName := "pulsar_namespace.test"
	cName := acctest.RandString(10)
	tName := acctest.RandString(10)
	nsName := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		IDRefreshName:     resourceName,
		CheckDestroy:      testPulsarNamespaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarNamespaceWithoutOptionals(testWebServiceURL, cName, tName, nsName),
				Check: resource.ComposeTestCheckFunc(
					testPulsarNamespaceExists(resourceName),
					resource.TestCheckNoResourceAttr(resourceName, "permission_grant.#"),
				),
			},
			{
				Config: testPulsarNamespaceWithPermissionGrants(testWebServiceURL, cName, tName, nsName,
					`permission_grant {
						role 		= "some-role-1"
						actions = ["produce", "consume", "functions"]
					}

					permission_grant {
						role 		= "some-role-2"
						actions = ["produce", "consume"]
					}`),
				Check: resource.ComposeTestCheckFunc(
					testPulsarNamespaceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "permission_grant.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "permission_grant.0.role", "some-role-1"),
					resource.TestCheckResourceAttr(resourceName, "permission_grant.0.actions.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "permission_grant.0.actions.0", "consume"),
					resource.TestCheckResourceAttr(resourceName, "permission_grant.0.actions.1", "functions"),
					resource.TestCheckResourceAttr(resourceName, "permission_grant.0.actions.2", "produce"),
					resource.TestCheckResourceAttr(resourceName, "permission_grant.1.role", "some-role-2"),
					resource.TestCheckResourceAttr(resourceName, "permission_grant.1.actions.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "permission_grant.1.actions.0", "consume"),
					resource.TestCheckResourceAttr(resourceName, "permission_grant.1.actions.1", "produce"),
				),
			},
			{
				Config: testPulsarNamespaceWithPermissionGrants(testWebServiceURL, cName, tName, nsName,
					`permission_grant {
						role 		= "some-role-2"
						actions = ["produce"]
					}`),
				Check: resource.ComposeTestCheckFunc(
					testPulsarNamespaceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "permission_grant.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "permission_grant.0.role", "some-role-2"),
					resource.TestCheckResourceAttr(resourceName, "permission_grant.0.actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "permission_grant.0.actions.0", "produce"),
				),
			},
		},
	})
}

func TestNamespaceWithTopicAutoCreationUpdate(t *testing.T) {

	resourceName := "pulsar_namespace.test"
	cName := acctest.RandString(10)
	tName := acctest.RandString(10)
	nsName := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		IDRefreshName:     resourceName,
		CheckDestroy:      testPulsarNamespaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarNamespaceWithoutOptionals(testWebServiceURL, cName, tName, nsName),
				Check: resource.ComposeTestCheckFunc(
					testPulsarNamespaceExists(resourceName),
					resource.TestCheckNoResourceAttr(resourceName, "topic_auto_creation.#"),
				),
			},
			{
				Config: testPulsarNamespaceWithTopicAutoCreation(testWebServiceURL, cName, tName, nsName,
					`topic_auto_creation {
						enable = "false"
					}`),
				Check: resource.ComposeTestCheckFunc(
					testPulsarNamespaceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "topic_auto_creation.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "topic_auto_creation.0.enable", "false"),
				),
			},
			{
				Config: testPulsarNamespaceWithTopicAutoCreation(testWebServiceURL, cName, tName, nsName,
					`topic_auto_creation {
						enable = "true"
					}`),
				Check: resource.ComposeTestCheckFunc(
					testPulsarNamespaceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "topic_auto_creation.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "topic_auto_creation.0.enable", "true"),
					resource.TestCheckResourceAttr(resourceName, "topic_auto_creation.0.type", "non-partitioned"),
				),
			},
			{
				Config: testPulsarNamespaceWithTopicAutoCreation(testWebServiceURL, cName, tName, nsName,
					`topic_auto_creation {
						enable = "true"
						type = "partitioned"
						partitions = 3
					}`),
				Check: resource.ComposeTestCheckFunc(
					testPulsarNamespaceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "topic_auto_creation.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "topic_auto_creation.0.enable", "true"),
					resource.TestCheckResourceAttr(resourceName, "topic_auto_creation.0.type", "partitioned"),
					resource.TestCheckResourceAttr(resourceName, "topic_auto_creation.0.partitions", "3"),
				),
			},
			{
				Config: testPulsarNamespaceWithoutOptionals(testWebServiceURL, cName, tName, nsName),
				Check: resource.ComposeTestCheckFunc(
					testPulsarNamespaceExists(resourceName),
					resource.TestCheckNoResourceAttr(resourceName, "topic_auto_creation.#"),
				),
			},
		},
	})
}

func TestImportExistingNamespace(t *testing.T) {
	tname := "public"
	ns := acctest.RandString(10)

	id := tname + "/" + ns

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			createNamespace(t, id)
			t.Cleanup(func() {
				if err := getClientFromMeta(testAccProvider.Meta()).Namespaces().DeleteNamespace(id); err != nil {
					t.Fatalf("ERROR_DELETING_TEST_NS: %v", err)
				}
			})
		},
		CheckDestroy:      testPulsarNamespaceDestroy,
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				ResourceName:     "pulsar_namespace.test",
				ImportState:      true,
				Config:           testPulsarExistingNamespaceWithoutOptionals(testWebServiceURL, ns),
				ImportStateId:    id,
				ImportStateCheck: testNamespaceImported(),
			},
		},
	})
}

func TestNamespaceExternallyRemoved(t *testing.T) {

	resourceName := "pulsar_namespace.test"
	cName := acctest.RandString(10)
	tName := acctest.RandString(10)
	nsName := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		IDRefreshName:     resourceName,
		CheckDestroy:      testPulsarNamespaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarNamespace(testWebServiceURL, cName, tName, nsName),
				Check: resource.ComposeTestCheckFunc(
					testPulsarNamespaceExists(resourceName),
				),
			},
			{
				PreConfig: func() {
					client, err := sharedClient(testWebServiceURL)
					if err != nil {
						t.Fatalf("ERROR_GETTING_PULSAR_CLIENT: %v", err)
					}

					conn := client.(admin.Client)
					if err = conn.Namespaces().DeleteNamespace(tName + "/" + nsName); err != nil {
						t.Fatalf("ERROR_DELETING_TEST_NS: %v", err)
					}
				},
				Config:             testPulsarNamespace(testWebServiceURL, cName, tName, nsName),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestNamespaceWithUndefinedOptionalsDrift(t *testing.T) {

	resourceName := "pulsar_namespace.test"
	cName := acctest.RandString(10)
	tName := acctest.RandString(10)
	nsName := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		IDRefreshName:     resourceName,
		CheckDestroy:      testPulsarNamespaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarNamespaceWithUndefinedOptionalsInNsConf(testWebServiceURL, cName, tName, nsName),
				Check: resource.ComposeTestCheckFunc(
					testPulsarNamespaceExists(resourceName),
				),
			},
			{
				Config: testPulsarNamespaceWithUndefinedOptionalsInNsConf(testWebServiceURL, cName, tName, nsName),
				Check: resource.ComposeTestCheckFunc(
					testPulsarNamespaceExists(resourceName),
				),
				ExpectNonEmptyPlan: false,
			},
			{
				Config:             testPulsarNamespaceWithUndefinedOptionalsInNsConf(testWebServiceURL, cName, tName, nsName),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestNamespaceReplicationClustersDrift(t *testing.T) {

	resourceName := "pulsar_namespace.test"
	cName := acctest.RandString(10)
	tName := acctest.RandString(10)
	nsName := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		IDRefreshName:     resourceName,
		CheckDestroy:      testPulsarNamespaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarNamespaceWithMultipleReplicationClusters(testWebServiceURL, cName, tName, nsName),
				Check: resource.ComposeTestCheckFunc(
					testPulsarNamespaceExists(resourceName),
				),
			},
			{
				Config:             testPulsarNamespaceWithMultipleReplicationClusters(testWebServiceURL, cName, tName, nsName),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				Config: testPulsarNamespace(testWebServiceURL, cName, tName, nsName),
				Check: resource.ComposeTestCheckFunc(
					testPulsarNamespaceExists(resourceName),
				),
			},
		},
	})
}

func createNamespace(t *testing.T, id string) {
	client, err := sharedClient(testWebServiceURL)
	if err != nil {
		t.Fatalf("ERROR_GETTING_PULSAR_CLIENT: %v", err)
	}

	conn := client.(admin.Client)
	if err = conn.Namespaces().CreateNamespace(id); err != nil {
		t.Fatalf("ERROR_CREATING_TEST_NS: %v", err)
	}
}

//nolint:unparam
func testPulsarNamespaceExists(ns string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[ns]
		if !ok {
			return fmt.Errorf("NOT_FOUND: %s", ns)
		}

		client := getClientFromMeta(testAccProvider.Meta()).Namespaces()

		if rs.Primary.ID == "" || !strings.Contains(rs.Primary.ID, "/") {
			return fmt.Errorf(`ERROR_NAMESPACE_ID_INVALID: "%s"`, rs.Primary.ID)
		}

		// id is the full path of the namespace, tenant-name/namespace-name
		// split would give us [tenant-name, namespace-name]
		nsParts := strings.Split(rs.Primary.ID, "/")

		nsList, err := client.GetNamespaces(nsParts[0])
		if err != nil {
			return fmt.Errorf("ERROR_READ_NAMESPACE_DATA: %w\n input data: %s", err, nsParts[0])
		}

		for _, ns := range nsList {

			if ns == rs.Primary.ID {
				return nil
			}
		}

		return fmt.Errorf(`ERROR_RESOURCE_NAMESPACE_DOES_NOT_EXISTS: "%s"`, ns)
	}
}

func testNamespaceImported() resource.ImportStateCheckFunc {
	return func(s []*terraform.InstanceState) error {
		if len(s) != 1 {
			return fmt.Errorf("expected %d states, got %d: %#v", 1, len(s), s)
		}

		if len(s[0].Attributes) != 12 {
			return fmt.Errorf("expected %d attrs, got %d: %#v", 12, len(s[0].Attributes), s[0].Attributes)
		}

		return nil
	}
}

func testPulsarNamespaceDestroy(s *terraform.State) error {
	client := getClientFromMeta(testAccProvider.Meta()).Namespaces()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "pulsar_namespace" {
			continue
		}

		// id is the full path of the namespace, in the format of tenant-name/namespace-name
		// split would give us [tenant-name, namespace-name]
		if rs.Primary.ID == "" || !strings.Contains(rs.Primary.ID, "/") {
			return fmt.Errorf(`ERROR_INVALID_RESOURCE_ID: "%s"`, rs.Primary.ID)
		}

		nsParts := strings.Split(rs.Primary.ID, "/")

		nsList, err := client.GetNamespaces(nsParts[0])
		if err != nil {
			return nil
		}

		for _, ns := range nsList {
			if ns == rs.Primary.ID {
				return fmt.Errorf("ERROR_RESOURCE_NAMESPACE_STILL_EXISTS: %s", ns)
			}
		}
	}

	return nil
}

func testPulsarNamespaceWithoutOptionals(wsURL, cluster, tenant, ns string) string {
	return fmt.Sprintf(`
provider "pulsar" {
  web_service_url = "%s"
}

resource "pulsar_cluster" "test_cluster" {
  cluster = "%s"

  cluster_data {
    web_service_url    = "http://localhost:8080"
    broker_service_url = "http://localhost:6050"
    peer_clusters      = ["standalone"]
  }

}

resource "pulsar_tenant" "test_tenant" {
  tenant           = "%s"
  allowed_clusters = [pulsar_cluster.test_cluster.cluster, "standalone"]
}

resource "pulsar_namespace" "test" {
  tenant    = pulsar_tenant.test_tenant.tenant
  namespace = "%s"
}
`, wsURL, cluster, tenant, ns)
}

func testPulsarNamespace(wsURL, cluster, tenant, ns string) string {
	return fmt.Sprintf(`
provider "pulsar" {
  web_service_url = "%s"
}

resource "pulsar_cluster" "test_cluster" {
  cluster = "%s"

  cluster_data {
    web_service_url    = "http://localhost:8080"
    broker_service_url = "http://localhost:6050"
    peer_clusters      = ["standalone"]
  }

}

resource "pulsar_tenant" "test_tenant" {
  tenant           = "%s"
  allowed_clusters = [pulsar_cluster.test_cluster.cluster, "standalone"]
}

resource "pulsar_namespace" "test" {
  tenant    = pulsar_tenant.test_tenant.tenant
  namespace = "%s"

  enable_deduplication = true

  namespace_config {
    anti_affinity                  			= "anti-aff"
    is_allow_auto_update_schema    			= false
	max_consumers_per_subscription 			= "50"
    max_consumers_per_topic        			= "50"
    max_producers_per_topic        			= "50"
    message_ttl_seconds            			= "86400"
    offload_threshold_size_in_mb   			= "100"
	replication_clusters           		 	= ["standalone"]
	subscription_expiration_time_minutes 	= 90
  }

  dispatch_rate {
    dispatch_msg_throttling_rate  = 50
    rate_period_seconds           = 50
    dispatch_byte_throttling_rate = 2048
  }

  subscription_dispatch_rate {
    dispatch_msg_throttling_rate  = 50
    rate_period_seconds           = 50
    dispatch_byte_throttling_rate = 2048
  }

  retention_policies {
    retention_minutes    = "1600"
    retention_size_in_mb = "10000"
  }

  persistence_policies {
    bookkeeper_ensemble                   = 2
    bookkeeper_write_quorum               = 2
    bookkeeper_ack_quorum                 = 2
    managed_ledger_max_mark_delete_rate   = 0.0
  }

  backlog_quota {
    limit_bytes  = "10000000000"
    limit_seconds = "-1"
    policy = "producer_request_hold"
    type = "destination_storage"
	}

	permission_grant {
		role 		= "some-role-1"
		actions = ["produce", "consume", "functions"]
	}

	permission_grant {
		role 		= "some-role-2"
		actions = ["produce", "consume"]
	}

	topic_auto_creation {
		enable = true
		type = "partitioned"
		partitions = 3
	}
}
`, wsURL, cluster, tenant, ns)
}

func testPulsarNamespaceWithUndefinedOptionalsInNsConf(wsURL, cluster, tenant, ns string) string {
	return fmt.Sprintf(`
provider "pulsar" {
  web_service_url = "%s"
}

resource "pulsar_cluster" "test_cluster" {
  cluster = "%s"

  cluster_data {
    web_service_url    = "http://localhost:8080"
    broker_service_url = "http://localhost:6050"
    peer_clusters      = ["standalone"]
  }

}

resource "pulsar_tenant" "test_tenant" {
  tenant           = "%s"
  allowed_clusters = [pulsar_cluster.test_cluster.cluster, "standalone"]
}

resource "pulsar_namespace" "test" {
  tenant    = pulsar_tenant.test_tenant.tenant
  namespace = "%s"

  namespace_config {
    anti_affinity                  = "anti-aff"
    max_producers_per_topic        = "50"
  }

}
`, wsURL, cluster, tenant, ns)
}

func testPulsarExistingNamespaceWithoutOptionals(wsURL, ns string) string {
	return fmt.Sprintf(`
provider "pulsar" {
  web_service_url = "%s"
}

resource "pulsar_namespace" "test" {
  tenant    = "public"
  namespace = "%s"
}
`, wsURL, ns)
}

func testPulsarNamespaceWithPermissionGrants(wsURL, cluster, tenant, ns string, permissionGrants string) string {
	return fmt.Sprintf(`
provider "pulsar" {
  web_service_url = "%s"
}

resource "pulsar_cluster" "test_cluster" {
  cluster = "%s"

  cluster_data {
    web_service_url    = "http://localhost:8080"
    broker_service_url = "http://localhost:6050"
    peer_clusters      = ["standalone"]
  }

}

resource "pulsar_tenant" "test_tenant" {
  tenant           = "%s"
  allowed_clusters = [pulsar_cluster.test_cluster.cluster, "standalone"]
}

resource "pulsar_namespace" "test" {
  tenant    = pulsar_tenant.test_tenant.tenant
	namespace = "%s"

	%s
}
`, wsURL, cluster, tenant, ns, permissionGrants)
}

func testPulsarNamespaceWithTopicAutoCreation(wsURL, cluster, tenant, ns string, topicAutoCreation string) string {
	return fmt.Sprintf(`
provider "pulsar" {
  web_service_url = "%s"
}

resource "pulsar_cluster" "test_cluster" {
  cluster = "%s"

  cluster_data {
    web_service_url    = "http://localhost:8080"
    broker_service_url = "http://localhost:6050"
    peer_clusters      = ["standalone"]
  }

}

resource "pulsar_tenant" "test_tenant" {
  tenant           = "%s"
  allowed_clusters = [pulsar_cluster.test_cluster.cluster, "standalone"]
}

resource "pulsar_namespace" "test" {
  tenant    = pulsar_tenant.test_tenant.tenant
	namespace = "%s"

	%s
}
`, wsURL, cluster, tenant, ns, topicAutoCreation)
}

func testPulsarNamespaceWithMultipleReplicationClusters(wsURL, cluster, tenant, ns string) string {
	return fmt.Sprintf(`
provider "pulsar" {
  web_service_url = "%s"
}

resource "pulsar_cluster" "test_cluster" {
  cluster = "%s"

  cluster_data {
    web_service_url    = "http://localhost:8080"
    broker_service_url = "http://localhost:6050"
    peer_clusters      = ["standalone"]
  }

}

resource "pulsar_tenant" "test_tenant" {
  tenant           = "%s"
  allowed_clusters = [pulsar_cluster.test_cluster.cluster, "standalone"]
}

resource "pulsar_namespace" "test" {
  tenant    = pulsar_tenant.test_tenant.tenant
  namespace = "%s"

  enable_deduplication = true

  namespace_config {
    anti_affinity                  = "anti-aff"
    max_consumers_per_subscription = "50"
    max_consumers_per_topic        = "50"
    max_producers_per_topic        = "50"
    message_ttl_seconds            = "86400"
    replication_clusters           = [pulsar_cluster.test_cluster.cluster]
    is_allow_auto_update_schema    = false
	offload_threshold_size_in_mb   = "100"
  }

  dispatch_rate {
    dispatch_msg_throttling_rate  = 50
    rate_period_seconds           = 50
    dispatch_byte_throttling_rate = 2048
  }

  subscription_dispatch_rate {
    dispatch_msg_throttling_rate  = 50
    rate_period_seconds           = 50
    dispatch_byte_throttling_rate = 2048
  }

  retention_policies {
    retention_minutes    = "1600"
    retention_size_in_mb = "10000"
  }

  persistence_policies {
    bookkeeper_ensemble                   = 2
    bookkeeper_write_quorum               = 2
    bookkeeper_ack_quorum                 = 2
    managed_ledger_max_mark_delete_rate   = 0.0
  }

  backlog_quota {
    limit_bytes  = "10000000000"
    limit_seconds = "-1"
    policy = "producer_request_hold"
    type = "destination_storage"
	}

	permission_grant {
		role 		= "some-role-1"
		actions = ["produce", "consume", "functions"]
	}

	permission_grant {
		role 		= "some-role-2"
		actions = ["produce", "consume"]
	}

	topic_auto_creation {
		enable = true
		type = "partitioned"
		partitions = 3
	}
}
`, wsURL, cluster, tenant, ns)
}
