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

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		Providers:     testAccProviders,
		IDRefreshName: resourceName,
		CheckDestroy:  testPulsarNamespaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarNamespace,
				Check: resource.ComposeTestCheckFunc(
					testPulsarNamespaceExists(resourceName),
				),
			},
		},
	})
}

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

		return fmt.Errorf("ERROR_RESOURCE_NAMESPACE_DOES_NOT_EXISTS")
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

var (
	testPulsarNamespace = fmt.Sprintf(`
provider "pulsar" {
  web_service_url = "%s"
}

resource "pulsar_cluster" "test_cluster" {
  cluster = "skrulls"

  cluster_data {
    web_service_url    = "http://localhost:8080"
    broker_service_url = "http://localhost:6050"
    peer_clusters      = ["standalone"]
  }

}

resource "pulsar_tenant" "test_tenant" {
  tenant           = "thanos"
  allowed_clusters = [pulsar_cluster.test_cluster.cluster, "standalone"]
}

resource "pulsar_namespace" "test" {
  tenant    = pulsar_tenant.test_tenant.tenant
  namespace = "eternals"

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
}
`, testWebServiceURL)
)
