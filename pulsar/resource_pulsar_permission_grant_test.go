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

	config := testPulsarPermissionGrantNamespace(testWebServiceURL, cName, tName, nsName, roleName,
		`["produce", "consume"]`)

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

func TestPermissionGrantNamespaceUpdate(t *testing.T) {
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
				Config: testPulsarPermissionGrantNamespace(testWebServiceURL, cName, tName, nsName, roleName,
					`["produce", "consume"]`),
				Check: resource.ComposeTestCheckFunc(
					testPulsarPermissionGrantExists(),
					resource.TestCheckResourceAttr(resourceName, "actions.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "actions.*", "produce"),
					resource.TestCheckTypeSetElemAttr(resourceName, "actions.*", "consume"),
				),
			},
			{
				Config: testPulsarPermissionGrantNamespace(testWebServiceURL, cName, tName, nsName, roleName,
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
				Config: testPulsarPermissionGrantNamespace(testWebServiceURL, cName, tName, nsName, roleName,
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

func TestPermissionGrantNamespaceExternallyRemoved(t *testing.T) {
	resourceName := "pulsar_permission_grant.test"
	cName := acctest.RandString(10)
	tName := acctest.RandString(10)
	nsName := acctest.RandString(10)
	roleName := acctest.RandString(10)

	config := testPulsarPermissionGrantNamespace(testWebServiceURL, cName, tName, nsName, roleName,
		`["produce", "consume"]`)

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

		role := rs.Primary.Attributes["role"]
		if role == "" {
			return fmt.Errorf("role is empty")
		}

		client := getClientFromMeta(testAccProvider.Meta())

		// Check if it's a namespace or topic permission
		if namespace := rs.Primary.Attributes["namespace"]; namespace != "" {
			nsName, err := utils.GetNamespaceName(namespace)
			if err != nil {
				return fmt.Errorf("ERROR_PARSING_NAMESPACE: %w", err)
			}

			permissions, err := client.Namespaces().GetNamespacePermissions(*nsName)
			if err != nil {
				return fmt.Errorf("ERROR_READ_NAMESPACE_PERMISSIONS: %w", err)
			}

			if _, exists := permissions[role]; !exists {
				return fmt.Errorf("permission grant for role %s on namespace %s does not exist", role, namespace)
			}
		} else if topic := rs.Primary.Attributes["topic"]; topic != "" {
			topicName, err := utils.GetTopicName(topic)
			if err != nil {
				return fmt.Errorf("ERROR_PARSING_TOPIC: %w", err)
			}

			permissions, err := client.Topics().GetPermissions(*topicName)
			if err != nil {
				return fmt.Errorf("ERROR_READ_TOPIC_PERMISSIONS: %w", err)
			}

			if _, exists := permissions[role]; !exists {
				return fmt.Errorf("permission grant for role %s on topic %s does not exist", role, topic)
			}
		} else {
			return fmt.Errorf("neither namespace nor topic is set")
		}

		return nil
	}
}

func testPulsarPermissionGrantDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "pulsar_permission_grant" {
			continue
		}

		role := rs.Primary.Attributes["role"]
		client := getClientFromMeta(testAccProvider.Meta())

		// Check namespace permissions
		if namespace := rs.Primary.Attributes["namespace"]; namespace != "" {
			nsName, err := utils.GetNamespaceName(namespace)
			if err != nil {
				continue
			}

			permissions, err := client.Namespaces().GetNamespacePermissions(*nsName)
			if err != nil {
				continue
			}

			if _, exists := permissions[role]; exists {
				return fmt.Errorf("permission grant still exists for role %s on namespace %s", role, namespace)
			}
		}

		// Check topic permissions
		if topic := rs.Primary.Attributes["topic"]; topic != "" {
			topicName, err := utils.GetTopicName(topic)
			if err != nil {
				continue
			}

			permissions, err := client.Topics().GetPermissions(*topicName)
			if err != nil {
				continue
			}

			if _, exists := permissions[role]; exists {
				return fmt.Errorf("permission grant still exists for role %s on topic %s", role, topic)
			}
		}
	}

	return nil
}

func testPulsarPermissionGrantNamespace(wsURL, cluster, tenant, namespace, role, actions string) string {
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

func TestPermissionGrantTopic(t *testing.T) {
	resourceName := "pulsar_permission_grant.test"
	cName := acctest.RandString(10)
	tName := acctest.RandString(10)
	nsName := acctest.RandString(10)
	topicName := acctest.RandString(10)
	roleName := acctest.RandString(10)

	config := testPulsarPermissionGrantTopic(testWebServiceURL, cName, tName, nsName, topicName, roleName,
		`["produce", "consume"]`)

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
					resource.TestCheckResourceAttr(resourceName, "topic",
						fmt.Sprintf("persistent://%s/%s/%s", tName, nsName, topicName)),
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

func TestPermissionGrantTopicUpdate(t *testing.T) {
	resourceName := "pulsar_permission_grant.test"
	cName := acctest.RandString(10)
	tName := acctest.RandString(10)
	nsName := acctest.RandString(10)
	topicName := acctest.RandString(10)
	roleName := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		IDRefreshName:     resourceName,
		CheckDestroy:      testPulsarPermissionGrantDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarPermissionGrantTopic(testWebServiceURL, cName, tName, nsName, topicName, roleName,
					`["produce"]`),
				Check: resource.ComposeTestCheckFunc(
					testPulsarPermissionGrantExists(),
					resource.TestCheckResourceAttr(resourceName, "actions.#", "1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "actions.*", "produce"),
				),
			},
			{
				Config: testPulsarPermissionGrantTopic(testWebServiceURL, cName, tName, nsName, topicName, roleName,
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
				Config: testPulsarPermissionGrantTopic(testWebServiceURL, cName, tName, nsName, topicName, roleName,
					`["consume"]`),
				Check: resource.ComposeTestCheckFunc(
					testPulsarPermissionGrantExists(),
					resource.TestCheckResourceAttr(resourceName, "actions.#", "1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "actions.*", "consume"),
				),
			},
		},
	})
}

func testPulsarPermissionGrantTopic(wsURL, cluster, tenant, namespace, topic, role, actions string) string {
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

resource "pulsar_topic" "test_topic" {
    tenant     = pulsar_tenant.test_tenant.tenant
    namespace  = pulsar_namespace.test_namespace.namespace
    topic_type = "persistent"
    topic_name = "%s"
    partitions = 0
}

resource "pulsar_permission_grant" "test" {
    topic   = pulsar_topic.test_topic.id
    role    = "%s"
    actions = %s
}
`, wsURL, cluster, tenant, namespace, topic, role, actions)
}

func TestPermissionGrantBothNamespaceAndTopic(t *testing.T) {
	cName := acctest.RandString(10)
	tName := acctest.RandString(10)
	nsName := acctest.RandString(10)
	topicName := acctest.RandString(10)
	roleName := acctest.RandString(10)

	config := testPulsarPermissionGrantBothNamespaceAndTopic(testWebServiceURL, cName, tName, nsName, topicName, roleName)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`only one of .*namespace,topic.* can be specified`),
			},
		},
	})
}

func testPulsarPermissionGrantBothNamespaceAndTopic(wsURL, cluster, tenant, namespace, topic, role string) string {
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

resource "pulsar_topic" "test_topic" {
    tenant     = pulsar_tenant.test_tenant.tenant
    namespace  = pulsar_namespace.test_namespace.namespace
    topic_type = "persistent"
    topic_name = "%s"
    partitions = 0
}

resource "pulsar_permission_grant" "test" {
    namespace = "${pulsar_tenant.test_tenant.tenant}/${pulsar_namespace.test_namespace.namespace}"
    topic     = pulsar_topic.test_topic.id
    role      = "%s"
    actions   = ["produce", "consume"]
}
`, wsURL, cluster, tenant, namespace, topic, role)
}

func TestPermissionGrantNeitherNamespaceNorTopic(t *testing.T) {
	roleName := acctest.RandString(10)

	config := testPulsarPermissionGrantNeitherNamespaceNorTopic(testWebServiceURL, roleName)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`one of .*namespace,topic.* must be specified`),
			},
		},
	})
}

func testPulsarPermissionGrantNeitherNamespaceNorTopic(wsURL, role string) string {
	return fmt.Sprintf(`
provider "pulsar" {
    web_service_url = "%s"
}

resource "pulsar_permission_grant" "test" {
    role    = "%s"
    actions = ["produce", "consume"]
}
`, wsURL, role)
}
