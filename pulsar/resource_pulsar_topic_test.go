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
	"time"

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/admin"
	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func init() {
	initTestWebServiceURL()
}

func TestTopic(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testPulsarTopicDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarPartitionTopic,
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists("pulsar_topic.sample-topic-1"),
					testPulsarTopicExists("pulsar_topic.sample-topic-2"),
				),
			},
			{
				Config: testPulsarNonPartitionTopic,
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists("pulsar_topic.sample-topic-3"),
					testPulsarTopicExists("pulsar_topic.sample-topic-4"),
				),
			},
		},
	})
}

func TestImportExistingTopic(t *testing.T) {
	tname := acctest.RandString(10)
	ttype := "persistent"
	pnum := 10

	fullID := strings.Join([]string{ttype + ":/", "public", "default", tname}, "/")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			createTopic(t, fullID, pnum)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testPulsarTopicDestroy,
		Steps: []resource.TestStep{
			{
				ResourceName:     "pulsar_topic.test",
				ImportState:      true,
				Config:           testPulsarTopic(testWebServiceURL, tname, ttype, pnum, ""),
				ImportStateId:    fullID,
				ImportStateCheck: testTopicImported(),
			},
		},
	})
}

func TestNonPartionedTopicWithPermissionGrantUpdate(t *testing.T) {
	testTopicWithPermissionGrantUpdate(t, 0)
}

func TestPartionedTopicWithPermissionGrantUpdate(t *testing.T) {
	testTopicWithPermissionGrantUpdate(t, 10)
}

func TestTopicNamespaceExternallyRemoved(t *testing.T) {

	resourceName := "pulsar_topic.test"
	cName := acctest.RandString(10)
	tName := acctest.RandString(10)
	nsName := acctest.RandString(10)
	topicName := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		IDRefreshName:     resourceName,
		CheckDestroy:      testPulsarTopicDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarNamespaceWithTopic(testWebServiceURL, cName, tName, nsName, topicName),
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists(resourceName),
				),
			},
			{
				PreConfig: func() {
					client, err := sharedClient(testWebServiceURL)
					if err != nil {
						t.Fatalf("ERROR_GETTING_PULSAR_CLIENT: %v", err)
					}

					conn := client.(admin.Client)
					topicName, err := utils.GetTopicName(fmt.Sprintf("persistent://%s/%s/%s", tName, nsName, topicName))
					if err != nil {
						t.Fatalf("ERROR_GETTING_TOPIC_NAME: %v", err)
					}
					if err = conn.Topics().Delete(*topicName, false, false); err != nil {
						t.Fatalf("ERROR_DELETING_TEST_TOPIC: %v", err)
					}
					if err = conn.Namespaces().DeleteNamespace(tName + "/" + nsName); err != nil {
						t.Fatalf("ERROR_DELETING_TEST_NS: %v", err)
					}
				},
				Config:             testPulsarNamespaceWithTopic(testWebServiceURL, cName, tName, nsName, topicName),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				ExpectError:        nil,
			},
		},
	})
}

func testTopicWithPermissionGrantUpdate(t *testing.T, pnum int) {
	resourceName := "pulsar_topic.test"
	tname := acctest.RandString(10)
	ttype := "persistent"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testPulsarTopicDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarTopic(testWebServiceURL, tname, ttype, pnum,
					`permission_grant {
						role 		= "some-role-1"
						actions = ["produce", "consume", "functions"]
					}
					permission_grant {
						role 		= "some-role-2"
						actions = ["produce", "consume"]
					}`),
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists(resourceName),
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
				Config: testPulsarTopic(testWebServiceURL, tname, ttype, pnum,
					`permission_grant {
						role 		= "some-role-2"
						actions = ["produce"]
					}`),
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "permission_grant.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "permission_grant.0.role", "some-role-2"),
					resource.TestCheckResourceAttr(resourceName, "permission_grant.0.actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "permission_grant.0.actions.0", "produce"),
				),
			},
		},
	})
}

func testPulsarTopicDestroy(s *terraform.State) error {
	client := getClientFromMeta(testAccProvider.Meta()).Topics()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "pulsar_topic" {
			continue
		}

		topicName, err := utils.GetTopicName(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("ERROR_READ_TOPIC: %w", err)
		}
		namespace, err := utils.GetNameSpaceName(topicName.GetTenant(), topicName.GetNamespace())
		if err != nil {
			return fmt.Errorf("ERROR_READ_NAMESPACE: %w", err)
		}

		partitionedTopics, nonPartitionedTopics, err := client.List(*namespace)
		if err != nil {
			return fmt.Errorf("ERROR_READ_TOPIC_DATA: %w", err)
		}

		for _, topic := range append(partitionedTopics, nonPartitionedTopics...) {
			if rs.Primary.ID == topic {
				return fmt.Errorf("ERROR_RESOURCE_TOPIC_STILL_EXISTS: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testPulsarTopicExists(topic string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[topic]
		if !ok {
			return fmt.Errorf("NOT_FOUND: %s", topic)
		}

		topicName, err := utils.GetTopicName(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("ERROR_READ_TOPIC: %w", err)
		}
		namespace, err := utils.GetNameSpaceName(topicName.GetTenant(), topicName.GetNamespace())
		if err != nil {
			return fmt.Errorf("ERROR_READ_NAMESPACE: %w", err)
		}

		client := getClientFromMeta(testAccProvider.Meta()).Topics()

		_, retentionPoliciesFound := rs.Primary.Attributes["retention_policies.0.%"]
		if retentionPoliciesFound && topicName.IsPersistent() {
			<-time.After(3 * time.Second)
			retention, err := client.GetRetention(*topicName, true)
			if err != nil {
				return fmt.Errorf("ERROR_READ_RETENTION: %w", err)
			}

			retentionSizeInMB := int64(20000)
			retentionTimeInMinutes := 1600
			if retention.RetentionSizeInMB != retentionSizeInMB {
				return fmt.Errorf("%s retentionSizeInMB should be %d, but got %d",
					topicName, retentionSizeInMB, retention.RetentionSizeInMB)
			}
			if retention.RetentionTimeInMinutes != retentionTimeInMinutes {
				return fmt.Errorf("%s retentionTimeInMinutes should be %d, but got %d",
					topicName, retentionTimeInMinutes, retention.RetentionTimeInMinutes)
			}
		}

		partitionedTopics, nonPartitionedTopics, err := client.List(*namespace)
		if err != nil {
			return fmt.Errorf("ERROR_READ_TOPIC_DATA: %w", err)
		}

		for _, topic := range append(partitionedTopics, nonPartitionedTopics...) {
			if rs.Primary.ID == topic {
				return nil
			}
		}

		return fmt.Errorf("ERROR_RESOURCE_TOPIC_DOES_NOT_EXISTS")
	}
}

func testTopicImported() resource.ImportStateCheckFunc {
	return func(s []*terraform.InstanceState) error {
		if len(s) != 1 {
			return fmt.Errorf("expected %d states, got %d: %#v", 1, len(s), s)
		}

		if len(s[0].Attributes) != 9 {
			return fmt.Errorf("expected %d attrs, got %d: %#v", 9, len(s[0].Attributes), s[0].Attributes)
		}

		return nil
	}
}

func createTopic(t *testing.T, fullID string, pnum int) {
	client, err := sharedClient(testWebServiceURL)
	if err != nil {
		t.Fatalf("ERROR_GETTING_PULSAR_CLIENT: %v", err)
	}

	conn := client.(admin.Client)
	tname, _ := utils.GetTopicName(fullID)

	if err = conn.Topics().Create(*tname, pnum); err != nil {
		t.Fatalf("ERROR_CREATING_TEST_TOPIC: %v", err)
	}
}

var (
	testPulsarPartitionTopic = fmt.Sprintf(`
provider "pulsar" {
  web_service_url = "%s"
}

resource "pulsar_topic" "sample-topic-1" {
  tenant     = "public"
  namespace  = "default"
  topic_type = "persistent"
  topic_name = "partitioned-persistent-topic"
  partitions = 4

  retention_policies {
    retention_time_minutes = 1600
    retention_size_mb = 20000
  }
}

resource "pulsar_topic" "sample-topic-2" {
  tenant     = "public"
  namespace  = "default"
  topic_type = "non-persistent"
  topic_name = "partitioned-non-persistent-topic"
  partitions = 4
}
`, testWebServiceURL)

	testPulsarNonPartitionTopic = fmt.Sprintf(`
provider "pulsar" {
  web_service_url = "%s"
}

resource "pulsar_topic" "sample-topic-3" {
  tenant     = "public"
  namespace  = "default"
  topic_type = "persistent"
  topic_name = "non-partitioned-persistent-topic"
  partitions = 0

  retention_policies {
    retention_time_minutes = 1600
    retention_size_mb = 20000
  }
}

resource "pulsar_topic" "sample-topic-4" {
  tenant     = "public"
  namespace  = "default"
  topic_type = "non-persistent"
  topic_name = "non-partitioned-non-persistent-topic"
  partitions = 0
}


`, testWebServiceURL)
)

func testPulsarTopic(url, tname, ttype string, pnum int, permissionGrants string) string {
	return fmt.Sprintf(`
provider "pulsar" {
	web_service_url = "%s"
}

resource "pulsar_topic" "test" {
  tenant     = "public"
  namespace  = "default"
  topic_type = "%s"
  topic_name = "%s"
	partitions = %d

	%s

  retention_policies {
    retention_time_minutes = 1600
    retention_size_mb = 20000
  }
}
`, url, ttype, tname, pnum, permissionGrants)
}

func testPulsarNamespaceWithTopic(wsURL, cluster, tenant, ns, topicName string) string {
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
    replication_clusters           = ["standalone"]
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
		enable = false
	}

	depends_on = [
    pulsar_tenant.test_tenant
  ]
}

resource "pulsar_topic" "test" {
  tenant     = "%s"
  namespace  = "%s"
  topic_type = "persistent"
  topic_name = "%s"
	partitions = 0
	depends_on = [
    pulsar_namespace.test,
		pulsar_tenant.test_tenant
  ]
}
`, wsURL, cluster, tenant, ns, tenant, ns, topicName)
}
