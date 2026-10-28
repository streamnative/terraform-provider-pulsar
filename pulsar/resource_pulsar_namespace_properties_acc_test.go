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
	"reflect"
	"testing"

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestNamespaceProperties(t *testing.T) {
	resourceName := "pulsar_namespace.test"
	cName := acctest.RandString(10)
	tName := acctest.RandString(10)
	nsName := acctest.RandString(10)

	setOutsideTerraform := func() {
		client := getClientFromMeta(testAccProvider.Meta()).Namespaces()
		ns, err := utils.GetNameSpaceName(tName, nsName)
		if err != nil {
			t.Fatal(err)
		}
		if err := client.UpdateProperties(*ns, map[string]string{"owner": "outside"}); err != nil {
			t.Fatal(err)
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		IDRefreshName:     resourceName,
		CheckDestroy:      testPulsarNamespaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarNamespaceProperties(testWebServiceURL, cName, tName, nsName,
					`{
    "standalone.feature.enabled"      = "true"
    "standalone.feature.name" = "value_a"
  }`),
				Check: resource.ComposeTestCheckFunc(
					testPulsarNamespaceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "namespace_properties.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "namespace_properties.standalone.feature.enabled", "true"),
					testRemoteNamespaceProperties(map[string]string{
						"standalone.feature.enabled": "true",
						"standalone.feature.name":    "value_a",
					}),
				),
			},
			{
				PreConfig: setOutsideTerraform,
				Config: testPulsarNamespaceProperties(testWebServiceURL, cName, tName, nsName,
					`{
    "standalone.feature.enabled"      = "true"
    "standalone.feature.name" = "value_b"
    "team"                        = "data"
  }`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "namespace_properties.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "namespace_properties.standalone.feature.name", "value_b"),
					testRemoteNamespaceProperties(map[string]string{
						"standalone.feature.enabled": "true",
						"standalone.feature.name":    "value_b",
						"team":                       "data",
						"owner":                      "outside",
					}),
				),
			},
			{
				// The undeclared "owner" key must not show as drift.
				Config: testPulsarNamespaceProperties(testWebServiceURL, cName, tName, nsName,
					`{
    "standalone.feature.enabled"      = "true"
    "standalone.feature.name" = "value_b"
    "team"                        = "data"
  }`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				// Dropping a declared key deletes only that key.
				Config: testPulsarNamespaceProperties(testWebServiceURL, cName, tName, nsName,
					`{
    "standalone.feature.enabled" = "false"
    "standalone.feature.name" = "value_b"
  }`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "namespace_properties.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "namespace_properties.standalone.feature.enabled", "false"),
					testRemoteNamespaceProperties(map[string]string{
						"standalone.feature.enabled": "false",
						"standalone.feature.name":    "value_b",
						"owner":                      "outside",
					}),
				),
			},
			{
				// Import does not read namespace_properties, so it is excluded from the comparison.
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"namespace_properties.%",
					"namespace_properties.standalone.feature.enabled",
					"namespace_properties.standalone.feature.name",
				},
			},
			{
				// Dropping the attribute removes the declared keys and leaves "owner" alone.
				Config: testPulsarNamespaceWithoutProperties(testWebServiceURL, cName, tName, nsName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(resourceName, "namespace_properties.%"),
					testRemoteNamespaceProperties(map[string]string{"owner": "outside"}),
				),
			},
		},
	})
}

// testRemoteNamespaceProperties compares the properties on the broker for pulsar_namespace.test.
func testRemoteNamespaceProperties(want map[string]string) resource.TestCheckFunc {
	const resourceName = "pulsar_namespace.test"
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("NOT_FOUND: %s", resourceName)
		}

		ns, err := utils.GetNamespaceName(rs.Primary.ID)
		if err != nil {
			return err
		}

		got, err := getClientFromMeta(testAccProvider.Meta()).Namespaces().GetProperties(*ns)
		if err != nil {
			return fmt.Errorf("ERROR_READ_NAMESPACE_PROPERTIES: %w", err)
		}
		if !reflect.DeepEqual(got, want) {
			return fmt.Errorf("remote namespace properties = %v, want %v", got, want)
		}
		return nil
	}
}

func testPulsarNamespaceProperties(wsURL, cluster, tenant, ns, properties string) string {
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

resource "pulsar_namespace" "test" {
  tenant    = pulsar_tenant.test_tenant.tenant
  namespace = "%s"

  namespace_properties = %s
}
`, wsURL, cluster, tenant, ns, properties)
}

func testPulsarNamespaceWithoutProperties(wsURL, cluster, tenant, ns string) string {
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

resource "pulsar_namespace" "test" {
  tenant    = pulsar_tenant.test_tenant.tenant
  namespace = "%s"
}
`, wsURL, cluster, tenant, ns)
}

// TestNamespaceAndTopicDottedPropertiesRoundTrip applies the same dotted keys to a namespace and to a
// topic in it, then checks that the state matches the broker and a second plan is empty.
func TestNamespaceAndTopicDottedPropertiesRoundTrip(t *testing.T) {
	skipIfNoTopicPolicies(t)
	nsResource := "pulsar_namespace.test"
	topicResource := "pulsar_topic.test"
	nsName := acctest.RandString(10)
	tName := acctest.RandString(10)
	config := testPulsarNamespaceAndTopicProperties(testWebServiceURL, nsName, tName)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testPulsarTopicDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(nsResource, "namespace_properties.standalone.feature.enabled", "true"),
					resource.TestCheckResourceAttr(topicResource, "topic_properties.standalone.feature.enabled", "true"),
					resource.TestCheckResourceAttr(topicResource, "topic_properties.standalone.feature.name", "example"),
					testRemoteNamespaceProperties(map[string]string{
						"standalone.feature.enabled": "true",
						"standalone.feature.name":    "example",
					}),
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

func testPulsarNamespaceAndTopicProperties(wsURL, ns, topic string) string {
	return fmt.Sprintf(`
provider "pulsar" {
  web_service_url = "%s"
}

resource "pulsar_namespace" "test" {
  tenant    = "public"
  namespace = "%s"

  namespace_properties = {
    "standalone.feature.enabled" = "true"
    "standalone.feature.name"    = "example"
  }
}

resource "pulsar_topic" "test" {
  tenant     = "public"
  namespace  = pulsar_namespace.test.namespace
  topic_type = "persistent"
  topic_name = "%s"
  partitions = 0

  topic_properties = {
    "standalone.feature.enabled" = "true"
    "standalone.feature.name"    = "example"
  }
}
`, wsURL, ns, topic)
}
