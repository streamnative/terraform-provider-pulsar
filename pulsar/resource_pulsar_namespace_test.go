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

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/streamnative/pulsarctl/pkg/pulsar"
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

	conn := client.(pulsar.Client)

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
		PreCheck:      func() { testAccPreCheck(t) },
		Providers:     testAccProviders,
		IDRefreshName: resourceName,
		CheckDestroy:  testPulsarNamespaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarNamespace(testWebServiceURL, cName, tName, nsName),
				Check: resource.ComposeTestCheckFunc(
					testPulsarNamespaceExists(resourceName),
				),
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
		PreCheck:      func() { testAccPreCheck(t) },
		Providers:     testAccProviders,
		IDRefreshName: resourceName,
		CheckDestroy:  testPulsarNamespaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarNamespaceWithoutOptionals(testWebServiceURL, cName, tName, nsName),
				Check: resource.ComposeTestCheckFunc(
					testPulsarNamespaceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "dispatch_rate.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "retention_policies.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "namespace_config.#", "0"),
					resource.TestCheckNoResourceAttr(resourceName, "enable_deduplication"),
				),
			},
			{
				Config: testPulsarNamespace(testWebServiceURL, cName, tName, nsName),
				Check: resource.ComposeTestCheckFunc(
					testPulsarNamespaceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "dispatch_rate.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "retention_policies.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "namespace_config.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "enable_deduplication", "true"),
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
		PreCheck:      func() { testAccPreCheck(t) },
		Providers:     testAccProviders,
		IDRefreshName: resourceName,
		CheckDestroy:  testPulsarNamespaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarNamespaceWithoutOptionals(testWebServiceURL, cName, tName, nsName),
				Check: resource.ComposeTestCheckFunc(
					testPulsarNamespaceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "dispatch_rate.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "retention_policies.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "namespace_config.#", "0"),
					resource.TestCheckNoResourceAttr(resourceName, "enable_deduplication"),
				),
			},
			{
				Config: testPulsarNamespaceWithUndefinedOptionalsInNsConf(testWebServiceURL, cName, tName, nsName),
				Check: resource.ComposeTestCheckFunc(
					testPulsarNamespaceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "dispatch_rate.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "retention_policies.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "namespace_config.#", "1"),
					resource.TestCheckNoResourceAttr(resourceName, "enable_deduplication"),
				),
				ExpectNonEmptyPlan: true,
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
		},
		CheckDestroy: testPulsarNamespaceDestroy,
		Providers:    testAccProviders,
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

func createNamespace(t *testing.T, id string) {
	client, err := sharedClient(testWebServiceURL)
	if err != nil {
		t.Fatalf("ERROR_GETTING_PULSAR_CLIENT: %v", err)
	}

	conn := client.(pulsar.Client)
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

		client := testAccProvider.Meta().(pulsar.Client).Namespaces()

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

		if len(s[0].Attributes) != 7 {
			return fmt.Errorf("expected %d attrs, got %d: %#v", 7, len(s[0].Attributes), s[0].Attributes)
		}

		return nil
	}
}

func testPulsarNamespaceDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(pulsar.Client).Namespaces()

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
    anti_affinity                  = "anti-aff"
    max_consumers_per_subscription = "50"
    max_consumers_per_topic        = "50"
    max_producers_per_topic        = "50"
    replication_clusters           = ["standalone"]
  }

  dispatch_rate {
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
