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
	"regexp"
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

var pulsarTestConfig = os.Getenv("PULSAR_TEST_CONFIG")

func TestSimpleTopic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testPulsarTopicDestroy,
		Steps: []resource.TestStep{
			{
				Config: testSimpleTopic,
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists("pulsar_topic.sample-topic-1", t),
					testPulsarTopicExists("pulsar_topic.sample-topic-2", t),
					testPulsarTopicExists("pulsar_topic.sample-topic-3", t),
					testPulsarTopicExists("pulsar_topic.sample-topic-4", t),
				),
			},
			{
				Config:             testSimpleTopic,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestTopic(t *testing.T) {
	skipIfNoTopicPolicies(t)
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testPulsarTopicDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarPartitionTopic,
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists("pulsar_topic.sample-topic-1", t),
					testPulsarTopicExists("pulsar_topic.sample-topic-2", t),
				),
			},
			{
				Config: testPulsarNonPartitionTopic,
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists("pulsar_topic.sample-topic-3", t),
					testPulsarTopicExists("pulsar_topic.sample-topic-4", t),
				),
			},
		},
	})
}

func TestTopicInNoTopicPoliciesEnvironment(t *testing.T) {
	if pulsarTestConfig != "no_topic_policies" {
		t.Skip("Skipping test: not running in 'topic_policies' environment")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testPulsarTopicDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testPulsarPartitionTopic,
				ExpectError: regexp.MustCompile("405"),
			},
		},
	})
}

func TestImportExistingTopic(t *testing.T) {
	tname := acctest.RandString(10)
	ttype := "persistent"
	pnum := 10

	fullID := strings.Join([]string{ttype + ":/", "public", "default", tname}, "/")
	topicName, err := utils.GetTopicName(fullID)
	if err != nil {
		t.Fatalf("ERROR_GETTING_TOPIC_NAME: %v", err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			createTopic(t, fullID, pnum)
			t.Cleanup(func() {
				if err := getClientFromMeta(testAccProvider.Meta()).Topics().Delete(*topicName, true, pnum == 0); err != nil {
					t.Fatalf("ERROR_DELETING_TEST_TOPIC: %v", err)
				}
			})
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

func TestImportTopicWithConfig(t *testing.T) {
	skipIfNoTopicPolicies(t)
	tname := acctest.RandString(10)
	ttype := "persistent"
	pnum := 0

	fullID := strings.Join([]string{ttype + ":/", "public", "default", tname}, "/")
	topicName, err := utils.GetTopicName(fullID)
	if err != nil {
		t.Fatalf("ERROR_GETTING_TOPIC_NAME: %v", err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			client := getClientFromMeta(testAccProvider.Meta()).Topics()
			if err := client.Create(*topicName, pnum); err != nil {
				t.Fatalf("ERROR_CREATING_TEST_TOPIC: %v", err)
			}
			time.Sleep(5 * time.Second)
			if err := client.SetMaxConsumers(*topicName, 15); err != nil {
				t.Fatalf("ERROR_SETTING_MAX_CONSUMERS: %v", err)
			}
			time.Sleep(5 * time.Second)
			if err := client.SetMessageTTL(*topicName, 7200); err != nil {
				t.Fatalf("ERROR_SETTING_MESSAGE_TTL: %v", err)
			}
			time.Sleep(20 * time.Second)
			t.Cleanup(func() {
				if err := getClientFromMeta(testAccProvider.Meta()).Topics().Delete(*topicName, true, pnum == 0); err != nil {
					t.Fatalf("ERROR_DELETING_TEST_TOPIC: %v", err)
				}
			})
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testPulsarTopicDestroy,
		Steps: []resource.TestStep{
			{
				ResourceName:  "pulsar_topic.test",
				ImportState:   true,
				Config:        testPulsarTopicWithTopicConfig(testWebServiceURL, tname, ttype, pnum, ""),
				ImportStateId: fullID,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					if len(s) != 1 {
						return fmt.Errorf("expected %d states, got %d: %#v", 1, len(s), s)
					}

					if s[0].Attributes["topic_config.#"] != "1" {
						return fmt.Errorf("topic_config not found in imported state %v", s[0].Attributes)
					}

					maxConsumers := s[0].Attributes["topic_config.0.max_consumers"]
					if maxConsumers != "15" {
						return fmt.Errorf("expected max_consumers to be 15, got %s", maxConsumers)
					}

					messageTTL := s[0].Attributes["topic_config.0.message_ttl_seconds"]
					if messageTTL != "7200" {
						return fmt.Errorf("expected message_ttl_seconds to be 7200, got %s", messageTTL)
					}

					return nil
				},
			},
		},
	})
}

func TestNonPartionedTopicWithPermissionGrantUpdate(t *testing.T) {
	skipIfNoTopicPolicies(t)
	testTopicWithPermissionGrantUpdate(t, 0)
}

func TestPartionedTopicWithPermissionGrantUpdate(t *testing.T) {
	skipIfNoTopicPolicies(t)
	testTopicWithPermissionGrantUpdate(t, 10)
}

func TestTopicWithTopicConfigUpdate(t *testing.T) {
	skipIfNoTopicPolicies(t)
	resourceName := "pulsar_topic.test"
	tname := acctest.RandString(10)
	ttype := "persistent"
	pnum := 0

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testPulsarTopicDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarTopic(testWebServiceURL, tname, ttype, pnum, ""),
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists(resourceName, t),
					resource.TestCheckResourceAttr(resourceName, "topic_config.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.max_consumers", "0"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.max_producers", "0"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.message_ttl_seconds", "0"),
				),
			},
			{
				Config: testPulsarTopicWithTopicConfig(testWebServiceURL, tname, ttype, pnum, `
					topic_config {
						max_consumers = 10
						max_producers = 5
						message_ttl_seconds = 3600
					}
				`),
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists(resourceName, t),
					resource.TestCheckResourceAttr(resourceName, "topic_config.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.max_consumers", "10"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.max_producers", "5"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.message_ttl_seconds", "3600"),
				),
			},
			{
				Config: testPulsarTopicWithTopicConfig(testWebServiceURL, tname, ttype, pnum, `
					topic_config {
						max_consumers = 20
						max_producers = 10
						message_ttl_seconds = 7200
						max_unacked_messages_per_consumer = 1000
						max_unacked_messages_per_subscription = 2000
					}
				`),
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists(resourceName, t),
					resource.TestCheckResourceAttr(resourceName, "topic_config.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.max_consumers", "20"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.max_producers", "10"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.message_ttl_seconds", "7200"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.max_unacked_messages_per_consumer", "1000"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.max_unacked_messages_per_subscription", "2000"),
				),
			},
			{
				Config: testPulsarTopicWithTopicConfig(testWebServiceURL, tname, ttype, pnum, `
					topic_config {
						msg_publish_rate = 100
						byte_publish_rate = 1024
					}
				`),
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists(resourceName, t),
					resource.TestCheckResourceAttr(resourceName, "topic_config.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.msg_publish_rate", "100"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.byte_publish_rate", "1024"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.max_consumers", "0"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.max_producers", "0"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.message_ttl_seconds", "0"),
				),
			},
		},
	})
}

func TestTopicWithDelayedDeliveryUpdate(t *testing.T) {
	skipIfNoTopicPolicies(t)
	resourceName := "pulsar_topic.test"
	tname := acctest.RandString(10)
	ttype := "persistent"
	pnum := 0

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testPulsarTopicDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarTopic(testWebServiceURL, tname, ttype, pnum, ""),
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists(resourceName, t),
				),
			},
			{
				Config: testPulsarTopicWithTopicConfig(testWebServiceURL, tname, ttype, pnum, `
					topic_config {
						delayed_delivery {
							enabled = true
							time = 2.0
						}
					}
				`),
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists(resourceName, t),
					resource.TestCheckResourceAttr(resourceName, "topic_config.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.delayed_delivery.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.delayed_delivery.0.enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.delayed_delivery.0.time", "2"),
				),
			},
			{
				Config: testPulsarTopicWithTopicConfig(testWebServiceURL, tname, ttype, pnum, `
					topic_config {
						delayed_delivery {
							enabled = false
							time = 1.0
						}
					}
				`),
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists(resourceName, t),
					resource.TestCheckResourceAttr(resourceName, "topic_config.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.delayed_delivery.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.delayed_delivery.0.enabled", "false"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.delayed_delivery.0.time", "1"),
				),
			},
		},
	})
}

func TestTopicWithInactiveTopicUpdate(t *testing.T) {
	skipIfNoTopicPolicies(t)
	resourceName := "pulsar_topic.test"
	tname := acctest.RandString(10)
	ttype := "persistent"
	pnum := 0

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testPulsarTopicDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarTopic(testWebServiceURL, tname, ttype, pnum, ""),
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists(resourceName, t),
				),
			},
			{
				Config: testPulsarTopicWithTopicConfig(testWebServiceURL, tname, ttype, pnum, `
					topic_config {
						inactive_topic {
							enable_delete_while_inactive = true
							max_inactive_duration = "60s"
							delete_mode = "delete_when_no_subscriptions"
						}
					}
				`),
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists(resourceName, t),
					resource.TestCheckResourceAttr(resourceName, "topic_config.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.inactive_topic.#", "1"),
					resource.TestCheckResourceAttr(resourceName,
						"topic_config.0.inactive_topic.0.enable_delete_while_inactive", "true"),
					resource.TestCheckResourceAttr(resourceName,
						"topic_config.0.inactive_topic.0.max_inactive_duration", "60s"),
					resource.TestCheckResourceAttr(resourceName,
						"topic_config.0.inactive_topic.0.delete_mode", "delete_when_no_subscriptions"),
				),
			},
			{
				Config: testPulsarTopicWithTopicConfig(testWebServiceURL, tname, ttype, pnum, `
					topic_config {
						inactive_topic {
							enable_delete_while_inactive = false
							max_inactive_duration = "120s"
							delete_mode = "delete_when_no_subscriptions"
						}
					}
				`),
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists(resourceName, t),
					resource.TestCheckResourceAttr(resourceName, "topic_config.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.inactive_topic.#", "1"),
					resource.TestCheckResourceAttr(resourceName,
						"topic_config.0.inactive_topic.0.enable_delete_while_inactive", "false"),
					resource.TestCheckResourceAttr(resourceName,
						"topic_config.0.inactive_topic.0.max_inactive_duration", "120s"),
					resource.TestCheckResourceAttr(resourceName,
						"topic_config.0.inactive_topic.0.delete_mode", "delete_when_no_subscriptions"),
				),
			},
		},
	})
}

func TestTopicWithCompactionThresholdUpdate(t *testing.T) {
	skipIfNoTopicPolicies(t)
	resourceName := "pulsar_topic.test"
	tname := acctest.RandString(10)
	ttype := "persistent"
	pnum := 0

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testPulsarTopicDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarTopic(testWebServiceURL, tname, ttype, pnum, ""),
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists(resourceName, t),
				),
			},
			{
				Config: testPulsarTopicWithTopicConfig(testWebServiceURL, tname, ttype, pnum, `
					topic_config {
						compaction_threshold = 1073741824
					}
				`),
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists(resourceName, t),
					resource.TestCheckResourceAttr(resourceName, "topic_config.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.compaction_threshold", "1073741824"),
				),
			},
			{
				Config: testPulsarTopicWithTopicConfig(testWebServiceURL, tname, ttype, pnum, `
					topic_config {
						compaction_threshold = 2147483648
					}
				`),
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists(resourceName, t),
					resource.TestCheckResourceAttr(resourceName, "topic_config.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.compaction_threshold", "2147483648"),
				),
			},
		},
	})
}

func TestTopicWithConfigAndOtherFields(t *testing.T) {
	skipIfNoTopicPolicies(t)
	resourceName := "pulsar_topic.test"
	tname := acctest.RandString(10)
	ttype := "persistent"
	pnum := 0

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testPulsarTopicDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarTopicWithTopicConfigAndOtherFields(testWebServiceURL, tname, ttype, pnum, `
					topic_config {
						max_consumers = 15
						max_producers = 8
						message_ttl_seconds = 7200
					}

					permission_grant {
						role = "admin"
						actions = ["produce", "consume"]
					}

					persistence_policies {
						bookkeeper_ensemble = 2
						bookkeeper_write_quorum = 2
						bookkeeper_ack_quorum = 2
						managed_ledger_max_mark_delete_rate = 0.5
					}
				`),
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists(resourceName, t),
					resource.TestCheckResourceAttr(resourceName, "topic_config.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.max_consumers", "15"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.max_producers", "8"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.message_ttl_seconds", "7200"),
					resource.TestCheckResourceAttr(resourceName, "permission_grant.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "permission_grant.0.role", "admin"),
					resource.TestCheckResourceAttr(resourceName, "permission_grant.0.actions.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "persistence_policies.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "persistence_policies.0.bookkeeper_ensemble", "2"),
					resource.TestCheckResourceAttr(resourceName, "persistence_policies.0.bookkeeper_write_quorum", "2"),
					resource.TestCheckResourceAttr(resourceName, "persistence_policies.0.bookkeeper_ack_quorum", "2"),
					resource.TestCheckResourceAttr(resourceName, "persistence_policies.0.managed_ledger_max_mark_delete_rate", "0.5"),
				),
			},
			{
				Config: testPulsarTopicWithTopicConfigAndOtherFields(testWebServiceURL, tname, ttype, pnum, `
					topic_config {
						max_consumers = 20
						message_ttl_seconds = 3600
						max_unacked_messages_per_consumer = 1000
						delayed_delivery {
							enabled = true
							time = 1500
						}
					}

					permission_grant {
						role = "admin"
						actions = ["produce", "consume", "functions"]
					}

					permission_grant {
						role = "user"
						actions = ["consume"]
					}

					persistence_policies {
						bookkeeper_ensemble = 3
						bookkeeper_write_quorum = 2
						bookkeeper_ack_quorum = 2
						managed_ledger_max_mark_delete_rate = 1.0
					}
				`),
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists(resourceName, t),
					resource.TestCheckResourceAttr(resourceName, "topic_config.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.max_consumers", "20"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.message_ttl_seconds", "3600"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.max_unacked_messages_per_consumer", "1000"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.delayed_delivery.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.delayed_delivery.0.enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.delayed_delivery.0.time", "1500"),
					resource.TestCheckResourceAttr(resourceName, "permission_grant.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "permission_grant.0.actions.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "permission_grant.1.role", "user"),
					resource.TestCheckResourceAttr(resourceName, "permission_grant.1.actions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "permission_grant.1.actions.0", "consume"),
					resource.TestCheckResourceAttr(resourceName, "persistence_policies.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "persistence_policies.0.bookkeeper_ensemble", "3"),
				),
			},
		},
	})
}

func TestTopicConfigRemoval(t *testing.T) {
	skipIfNoTopicPolicies(t)
	resourceName := "pulsar_topic.test"
	tname := acctest.RandString(10)
	ttype := "persistent"
	pnum := 0

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testPulsarTopicDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarTopicWithTopicConfig(testWebServiceURL, tname, ttype, pnum, `
					topic_config {
						max_consumers = 10
						max_producers = 5
						message_ttl_seconds = 3600
						delayed_delivery {
							enabled = true
							time = 2.0
						}
					}
				`),
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists(resourceName, t),
					resource.TestCheckResourceAttr(resourceName, "topic_config.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.max_consumers", "10"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.max_producers", "5"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.message_ttl_seconds", "3600"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.delayed_delivery.#", "1"),
				),
			},
			{
				Config: testPulsarTopic(testWebServiceURL, tname, ttype, pnum, ""),
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists(resourceName, t),
				),
			},
			{
				// Confirm that topic_config can be added back after removal
				Config: testPulsarTopicWithTopicConfig(testWebServiceURL, tname, ttype, pnum, `
					topic_config {
						max_consumers = 20
						inactive_topic {
							enable_delete_while_inactive = true
							max_inactive_duration = "60s"
							delete_mode = "delete_when_no_subscriptions"
						}
					}
				`),
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists(resourceName, t),
					resource.TestCheckResourceAttr(resourceName, "topic_config.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.max_consumers", "20"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.inactive_topic.#", "1"),
				),
			},
		},
	})
}

func TestTopicNamespaceExternallyRemoved(t *testing.T) {
	skipIfNoTopicPolicies(t)
	resourceName := "pulsar_topic.test"
	tName := acctest.RandString(10)
	nsName := acctest.RandString(10)
	topicName := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testPulsarTopicDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarNamespaceWithTopic(testWebServiceURL, tName, nsName, topicName),
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists(resourceName, t),
				),
				ExpectError: nil,
			},
			{
				PreConfig: func() {
					client := getClientFromMeta(testAccProvider.Meta())
					topicName, err := utils.GetTopicName(fmt.Sprintf("persistent://%s/%s/%s", tName, nsName, topicName))
					if err != nil {
						t.Fatalf("ERROR_GETTING_TOPIC_NAME: %v", err)
					}
					namespace, err := utils.GetNameSpaceName(topicName.GetTenant(), topicName.GetNamespace())
					if err != nil {
						t.Fatalf("ERROR_READ_NAMESPACE: %v", err)
					}
					partitionedTopics, nonPartitionedTopics, err := client.Topics().List(*namespace)
					if err != nil {
						t.Fatalf("ERROR_READ_TOPIC_DATA: %v", err)
					}

					for _, topic := range append(partitionedTopics, nonPartitionedTopics...) {
						if topicName.String() == topic {
							if err = client.Topics().Delete(*topicName, true, true); err != nil {
								t.Fatalf("ERROR_DELETING_TEST_TOPIC: %v", err)
							}
						}
					}
					if err = client.Namespaces().DeleteNamespace(tName + "/" + nsName); err != nil {
						t.Fatalf("ERROR_DELETING_TEST_NS: %v", err)
					}
				},
				Config:             testPulsarNamespaceWithTopic(testWebServiceURL, tName, nsName, topicName),
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
					testPulsarTopicExists(resourceName, t),
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
					testPulsarTopicExists(resourceName, t),
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

func testPulsarTopicExists(topic string, t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[topic]
		if !ok {
			return fmt.Errorf("NOT_FOUND: %s", topic)
		}

		topicName, err := utils.GetTopicName(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("ERROR_READ_TOPIC: %w", err)
		}
		t.Logf("topicName: %v", topicName)
		t.Logf("rs.Primary.Attributes: %v", rs.Primary.Attributes)
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

		if len(s[0].Attributes) != 32 {
			return fmt.Errorf("expected %d attrs, got %d: %#v", 32, len(s[0].Attributes), s[0].Attributes)
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
	testSimpleTopic = fmt.Sprintf(`
provider "pulsar" {
  web_service_url = "%s"
}

resource "pulsar_topic" "sample-topic-1" {
  tenant     = "public"
  namespace  = "default"
  topic_type = "persistent"
  topic_name = "simple-persistent-topic"
	partitions = 0
}

resource "pulsar_topic" "sample-topic-2" {
  tenant     = "public"
  namespace  = "default"
  topic_type = "non-persistent"
  topic_name = "simple-non-persistent-topic"
	partitions = 0
}

resource "pulsar_topic" "sample-topic-3" {
  tenant     = "public"
  namespace  = "default"
  topic_type = "persistent"
  topic_name = "simple-persistent-partitioned-topic"
  partitions = 4
}

resource "pulsar_topic" "sample-topic-4" {
  tenant     = "public"
  namespace  = "default"
  topic_type = "non-persistent"
  topic_name = "simple-non-persistent-partitioned-topic"
  partitions = 4
}
`, testWebServiceURL)

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

	testPulsarPartitionTopicWithConfig = fmt.Sprintf(`
provider "pulsar" {
  web_service_url = "%s"
}

resource "pulsar_topic" "sample-topic-config-1" {
  tenant     = "public"
  namespace  = "default"
  topic_type = "persistent"
  topic_name = "partitioned-persistent-topic-with-config"
  partitions = 4

  retention_policies {
    retention_time_minutes = 1600
    retention_size_mb = 20000
  }

  topic_config {
    message_ttl_seconds = 3600
    max_consumers = 50
  }
}

resource "pulsar_topic" "sample-topic-config-2" {
  tenant     = "public"
  namespace  = "default"
  topic_type = "non-persistent"
  topic_name = "partitioned-non-persistent-topic-with-config"
  partitions = 4
}
`, testWebServiceURL)

	testPulsarNonPartitionTopicWithConfig = fmt.Sprintf(`
provider "pulsar" {
  web_service_url = "%s"
}

resource "pulsar_topic" "sample-topic-config-3" {
  tenant     = "public"
  namespace  = "default"
  topic_type = "persistent"
  topic_name = "non-partitioned-persistent-topic-with-config"
  partitions = 0

  retention_policies {
    retention_time_minutes = 1600
    retention_size_mb = 20000
  }

  topic_config {
    delayed_delivery {
      enabled = true
      time = 1500
    }
  }
}

resource "pulsar_topic" "sample-topic-config-4" {
  tenant     = "public"
  namespace  = "default"
  topic_type = "non-persistent"
  topic_name = "non-partitioned-non-persistent-topic-with-config"
  partitions = 0
}
`, testWebServiceURL)
)

//nolint:unparam
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

//nolint:unparam
func testPulsarNamespaceWithTopic(wsURL, tenant, ns, topicName string) string {
	return fmt.Sprintf(`
provider "pulsar" {
  web_service_url = "%s"
}

resource "pulsar_tenant" "test_tenant" {
  tenant           = "%s"
  allowed_clusters = ["standalone"]
}

resource "pulsar_namespace" "test" {
  tenant    = pulsar_tenant.test_tenant.tenant
  namespace = "%s"

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
		pulsar_namespace.test
	]
}
`, wsURL, tenant, ns, tenant, ns, topicName)
}

//nolint:unparam
func testPulsarTopicWithTopicConfig(url, tname, ttype string, pnum int, topicConfig string) string {
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

  retention_policies {
    retention_time_minutes = 1600
    retention_size_mb = 20000
  }

  %s
}
`, url, ttype, tname, pnum, topicConfig)
}

func testPulsarTopicWithProperties(url, tname, ttype string, pnum int, propertiesHCL string) string {
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

  retention_policies {
    retention_time_minutes = 1600
    retention_size_mb      = 20000
  }

  %s
}
`, url, ttype, tname, pnum, propertiesHCL)
}

func testNonPersistentPulsarTopicWithProperties(url, tname, ttype string, pnum int, propertiesHCL string) string {
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
}
`, url, ttype, tname, pnum, propertiesHCL)
}

func testPulsarTopicWithTopicConfigAndOtherFields(url, tname, ttype string, pnum int, allConfigs string) string {
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

  retention_policies {
    retention_time_minutes = 1600
    retention_size_mb = 20000
  }

  %s
}
`, url, ttype, tname, pnum, allConfigs)
}

func TestTopicWithConfig(t *testing.T) {
	skipIfNoTopicPolicies(t)
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testPulsarTopicDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarPartitionTopicWithConfig,
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists("pulsar_topic.sample-topic-config-1", t),
					resource.TestCheckResourceAttr("pulsar_topic.sample-topic-config-1", "topic_config.#", "1"),
					resource.TestCheckResourceAttr("pulsar_topic.sample-topic-config-1", "topic_config.0.message_ttl_seconds", "3600"),
					resource.TestCheckResourceAttr("pulsar_topic.sample-topic-config-1", "topic_config.0.max_consumers", "50"),
					testPulsarTopicExists("pulsar_topic.sample-topic-config-2", t),
				),
			},
			{
				Config: testPulsarNonPartitionTopicWithConfig,
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists("pulsar_topic.sample-topic-config-3", t),
					resource.TestCheckResourceAttr("pulsar_topic.sample-topic-config-3", "topic_config.#", "1"),
					resource.TestCheckResourceAttr("pulsar_topic.sample-topic-config-3", "topic_config.0.delayed_delivery.#", "1"),
					resource.TestCheckResourceAttr("pulsar_topic.sample-topic-config-3",
						"topic_config.0.delayed_delivery.0.enabled", "true"),
					resource.TestCheckResourceAttr("pulsar_topic.sample-topic-config-3",
						"topic_config.0.delayed_delivery.0.time", "1500"),
					testPulsarTopicExists("pulsar_topic.sample-topic-config-4", t),
				),
			},
		},
	})
}

func TestPartitionedTopicWithTopicConfig(t *testing.T) {
	skipIfNoTopicPolicies(t)
	resourceName := "pulsar_topic.test"
	tname := acctest.RandString(10)
	ttype := "persistent"
	pnum := 4 // Number of partitions

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testPulsarTopicDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarTopicWithTopicConfig(testWebServiceURL, tname, ttype, pnum, `
					topic_config {
						max_consumers = 10
						max_producers = 5
						message_ttl_seconds = 3600
					}
				`),
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists(resourceName, t),
					resource.TestCheckResourceAttr(resourceName, "partitions", "4"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.max_consumers", "10"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.max_producers", "5"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.message_ttl_seconds", "3600"),
				),
			},
			{
				// Update topic_config for a partitioned topic
				Config: testPulsarTopicWithTopicConfig(testWebServiceURL, tname, ttype, pnum, `
					topic_config {
						max_consumers = 20
						message_ttl_seconds = 7200
						max_unacked_messages_per_consumer = 1000
						max_unacked_messages_per_subscription = 2000
					}
				`),
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists(resourceName, t),
					resource.TestCheckResourceAttr(resourceName, "partitions", "4"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.max_consumers", "20"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.message_ttl_seconds", "7200"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.max_unacked_messages_per_consumer", "1000"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.max_unacked_messages_per_subscription", "2000"),
				),
			},
		},
	})
}

func TestTopicWithPropertiesUpdate(t *testing.T) {
	skipIfNoTopicPolicies(t)
	resourceName := "pulsar_topic.test"
	tname := acctest.RandString(10)
	ttype := "persistent"
	pnum := 0

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testPulsarTopicDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarTopicWithProperties(testWebServiceURL, tname,
					ttype, pnum, `topic_properties = { k1 = "v1", k2 = "v2" }`),
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists(resourceName, t),
					resource.TestCheckResourceAttr(resourceName, "topic_properties.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "topic_properties.k1", "v1"),
					resource.TestCheckResourceAttr(resourceName, "topic_properties.k2", "v2"),
				),
			},
			{
				Config: testPulsarTopicWithProperties(testWebServiceURL, tname,
					ttype, pnum, `topic_properties = { k2 = "v2new", k3 = "v3" }`),
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists(resourceName, t),
					resource.TestCheckResourceAttr(resourceName, "topic_properties.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "topic_properties.k2", "v2new"),
					resource.TestCheckResourceAttr(resourceName, "topic_properties.k3", "v3"),
				),
			},
		},
	})
}

func TestNonPersistentTopicWithPropertiesFails(t *testing.T) {
	tname := acctest.RandString(10)
	ttype := "non-persistent"
	pnum := 0

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testPulsarTopicDestroy,
		Steps: []resource.TestStep{
			{
				Config: testNonPersistentPulsarTopicWithProperties(testWebServiceURL, tname,
					ttype, pnum, `topic_properties = { k1 = "v1" }`),
				ExpectError: regexp.MustCompile("persistent"),
			},
		},
	})
}

func TestImportTopicWithProperties(t *testing.T) {
	resourceName := "pulsar_topic.test"
	tname := acctest.RandString(10)
	pnum := 0
	ttype := "persistent"
	fullID := strings.Join([]string{ttype + ":/", "public", "default", tname}, "/")
	topicName, err := utils.GetTopicName(fullID)
	if err != nil {
		t.Fatalf("ERROR_GETTING_TOPIC_NAME: %v", err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			// create topic and set properties via admin client
			props := map[string]string{"importK1": "importV1", "importK2": "importV2"}
			client := getClientFromMeta(testAccProvider.Meta()).Topics()
			if err := client.CreateWithProperties(*topicName, pnum, props); err != nil {
				t.Fatalf("ERROR_CREATING_TEST_TOPIC: %v", err)
			}
			// cleanup
			t.Cleanup(func() {
				_ = client.Delete(*topicName, true, pnum == 0)
			})
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testPulsarTopicDestroy,
		Steps: []resource.TestStep{
			{
				ResourceName: resourceName,
				ImportState:  true,
				Config: testPulsarTopicWithProperties(testWebServiceURL, tname, ttype, pnum,
					`topic_properties = { importK1 = "importV1", importK2 = "importV2" }`),
				ImportStateId: fullID,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					if len(s) != 1 {
						return fmt.Errorf("expected 1 state, got %d", len(s))
					}
					if s[0].Attributes["topic_properties.%"] != "2" {
						return fmt.Errorf("expected 2 topic_properties, got %s", s[0].Attributes["topic_properties.%"])
					}
					if v := s[0].Attributes["topic_properties.importK1"]; v != "importV1" {
						return fmt.Errorf("unexpected importK1 value: %s", v)
					}
					if v := s[0].Attributes["topic_properties.importK2"]; v != "importV2" {
						return fmt.Errorf("unexpected importK2 value: %s", v)
					}
					return nil
				},
			},
		},
	})
}

func skipIfNoTopicPolicies(t *testing.T) {
	if pulsarTestConfig == "no_topic_policies" {
		t.Skip("Skipping test: not running in 'no_topic_policies' environment")
	}
}

func TestTopicDoesNotInterfereWithExternalPermissions(t *testing.T) {
	resourceName := "pulsar_topic.test-topic"
	topicName := acctest.RandString(10)
	externalRole := acctest.RandString(10) + "-external"
	topicRole := acctest.RandString(10) + "-topic-managed"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		IDRefreshName:     resourceName,
		CheckDestroy:      testPulsarTopicDestroy,
		Steps: []resource.TestStep{
			{
				// Step 1: Create basic topic, then manually add external permission via API
				Config: testPulsarBasicTopic(testWebServiceURL, topicName, "persistent"),
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists(resourceName, t),
					// After topic is created, add external permission via API
					func(s *terraform.State) error {
						client, err := sharedClient(testWebServiceURL)
						if err != nil {
							return fmt.Errorf("ERROR_GETTING_PULSAR_CLIENT: %v", err)
						}

						conn := client.(admin.Client)
						topicFullName := fmt.Sprintf("persistent://public/default/%s", topicName)

						topicName, err := utils.GetTopicName(topicFullName)
						if err != nil {
							return fmt.Errorf("ERROR_PARSING_TOPIC: %v", err)
						}

						// Add external permission (simulating standalone permission resource)
						externalActions := []utils.AuthAction{utils.AuthAction("produce")}
						if err = conn.Topics().GrantPermission(*topicName, externalRole, externalActions); err != nil {
							return fmt.Errorf("ERROR_GRANTING_EXTERNAL_PERMISSION: %v", err)
						}
						return nil
					},
					// Verify the external permission was created
					testTopicPermissionExists(fmt.Sprintf("persistent://public/default/%s", topicName), externalRole),
				),
			},
			{
				// Step 2: Add topic resource with its own permissions
				// External permission should NOT be removed
				Config: testPulsarTopicWithOwnPermissions(testWebServiceURL, topicName,
					"persistent", topicRole, `["produce", "consume"]`),
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists(resourceName, t),
					// Topic should only show its managed permission
					resource.TestCheckResourceAttr(resourceName, "permission_grant.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "permission_grant.0.role", topicRole),
					// External permission should still exist in Pulsar
					testTopicPermissionExists(fmt.Sprintf("persistent://public/default/%s", topicName), externalRole),
				),
			},
			{
				// Step 3: Update topic permission - external permission should remain untouched
				Config: testPulsarTopicWithOwnPermissions(testWebServiceURL, topicName,
					"persistent", topicRole, `["produce", "consume", "functions"]`),
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists(resourceName, t),
					// Topic permission should be updated
					resource.TestCheckResourceAttr(resourceName, "permission_grant.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "permission_grant.0.role", topicRole),
					resource.TestCheckResourceAttr(resourceName, "permission_grant.0.actions.#", "3"),
					// External permission should still exist and be unchanged
					testTopicPermissionExists(fmt.Sprintf("persistent://public/default/%s", topicName), externalRole),
				),
			},
		},
	})
}

func testPulsarBasicTopic(wsURL, topic, topicType string) string {
	return fmt.Sprintf(`
provider "pulsar" {
  web_service_url = "%s"
}

resource "pulsar_topic" "test-topic" {
  tenant     = "public"
  namespace  = "default"
  topic_type = "%s"
  topic_name = "%s"
  partitions = 0
}
`, wsURL, topicType, topic)
}

func testPulsarTopicWithOwnPermissions(wsURL, topic, topicType, role string,
	actions string) string {
	return fmt.Sprintf(`
provider "pulsar" {
  web_service_url = "%s"
}

resource "pulsar_topic" "test-topic" {
  tenant     = "public"
  namespace  = "default"
  topic_type = "%s"
  topic_name = "%s"
  partitions = 0

  permission_grant {
    role    = "%s"
    actions = %s
  }
}
`, wsURL, topicType, topic, role, actions)
}

func testTopicPermissionExists(topic, role string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := getClientFromMeta(testAccProvider.Meta()).Topics()

		topicName, err := utils.GetTopicName(topic)
		if err != nil {
			return fmt.Errorf("ERROR_PARSING_TOPIC: %w", err)
		}

		permissions, err := client.GetPermissions(*topicName)
		if err != nil {
			return fmt.Errorf("ERROR_READ_TOPIC_PERMISSIONS: %w", err)
		}

		if _, exists := permissions[role]; !exists {
			return fmt.Errorf("permission for role %s should exist", role)
		}

		return nil
	}
}
