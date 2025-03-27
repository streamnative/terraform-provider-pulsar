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

func TestSubscription(t *testing.T) {
	resourceName := "pulsar_subscription.test"
	topic := "test-subscription-" + acctest.RandString(10)
	topicName := "persistent://public/default/" + topic
	subscriptionName := "sub-" + acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testPulsarSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarSubscription(testWebServiceURL, topic, topicName, subscriptionName, "latest"),
				Check: resource.ComposeTestCheckFunc(
					testPulsarSubscriptionExists(resourceName, t),
					resource.TestCheckResourceAttr(resourceName, "topic_name", topicName),
					resource.TestCheckResourceAttr(resourceName, "subscription_name", subscriptionName),
					resource.TestCheckResourceAttr(resourceName, "position", "latest"),
				),
			},
			{
				Config:             testPulsarSubscription(testWebServiceURL, topic, topicName, subscriptionName, "latest"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestSubscriptionWithDifferentPositions(t *testing.T) {
	topic := "test-subscription-pos-" + acctest.RandString(10)
	topicName := "persistent://public/default/" + topic

	// Test with the earliest position
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testPulsarSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarSubscription(testWebServiceURL, topic, topicName, "sub-earliest", "earliest"),
				Check: resource.ComposeTestCheckFunc(
					testPulsarSubscriptionExists("pulsar_subscription.test", t),
					resource.TestCheckResourceAttr("pulsar_subscription.test", "position", "earliest"),
				),
			},
		},
	})

	// Test with the latest position
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testPulsarSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarSubscription(testWebServiceURL, topic, topicName, "sub-latest", "latest"),
				Check: resource.ComposeTestCheckFunc(
					testPulsarSubscriptionExists("pulsar_subscription.test", t),
					resource.TestCheckResourceAttr("pulsar_subscription.test", "position", "latest"),
				),
			},
		},
	})
}

func TestImportExistingSubscription(t *testing.T) {
	topic := "test-subscription-import-" + acctest.RandString(10)
	topicName := "persistent://public/default/" + topic
	subscriptionName := "sub-import-" + acctest.RandString(10)
	fullID := fmt.Sprintf("%s:%s", topicName, subscriptionName)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			createSubscriptionForTest(t, topic, subscriptionName)
			t.Cleanup(func() {
				topicName, _ := utils.GetTopicName(topic)
				if err := getClientFromMeta(testAccProvider.Meta()).Subscriptions().Delete(*topicName,
					subscriptionName); err != nil {
					t.Logf("ERROR_DELETING_TEST_SUBSCRIPTION: %v", err)
				}
			})
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testPulsarSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ResourceName:     "pulsar_subscription.test",
				ImportState:      true,
				Config:           testPulsarSubscription(testWebServiceURL, topic, topicName, subscriptionName, "latest"),
				ImportStateId:    fullID,
				ImportStateCheck: testSubscriptionImported(),
			},
		},
	})
}

func TestSubscriptionExternallyRemoved(t *testing.T) {
	resourceName := "pulsar_subscription.test"
	topic := "test-subscription-external-" + acctest.RandString(10)
	topicName := "persistent://public/default/" + topic
	subscriptionName := "sub-external-" + acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testPulsarSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarSubscription(testWebServiceURL, topic, topicName, subscriptionName, "latest"),
				Check: resource.ComposeTestCheckFunc(
					testPulsarSubscriptionExists(resourceName, t),
				),
			},
			{
				PreConfig: func() {
					client := getClientFromMeta(testAccProvider.Meta())
					topicName, _ := utils.GetTopicName(topic)
					if err := client.Subscriptions().Delete(*topicName, subscriptionName); err != nil {
						t.Fatalf("ERROR_DELETING_TEST_SUBSCRIPTION: %v", err)
					}
				},
				Config:             testPulsarSubscription(testWebServiceURL, topic, topicName, subscriptionName, "latest"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testPulsarSubscriptionDestroy(s *terraform.State) error {
	client := getClientFromMeta(testAccProvider.Meta())

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "pulsar_subscription" {
			continue
		}

		idParts := strings.Split(rs.Primary.ID, ":")
		if len(idParts) != 2 {
			return fmt.Errorf("ERROR_INVALID_RESOURCE_ID: %s", rs.Primary.ID)
		}

		topic := idParts[0]
		subscriptionName := idParts[1]

		topicName, err := utils.GetTopicName(topic)
		if err != nil {
			return fmt.Errorf("ERROR_PARSE_TOPIC_NAME: %w", err)
		}

		subscriptions, err := client.Subscriptions().List(*topicName)
		if err != nil {
			return nil // Topic might not exist anymore, which is fine
		}

		for _, sub := range subscriptions {
			if sub == subscriptionName {
				return fmt.Errorf("ERROR_RESOURCE_SUBSCRIPTION_STILL_EXISTS: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

//nolint:unparam
func testPulsarSubscriptionExists(subscription string, t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[subscription]
		if !ok {
			return fmt.Errorf("NOT_FOUND: %s", subscription)
		}

		idParts := strings.Split(rs.Primary.ID, ":")
		if len(idParts) != 2 {
			return fmt.Errorf("ERROR_INVALID_RESOURCE_ID: %s", rs.Primary.ID)
		}

		topic := idParts[0]
		subscriptionName := idParts[1]

		topicName, err := utils.GetTopicName(topic)
		if err != nil {
			return fmt.Errorf("ERROR_PARSE_TOPIC_NAME: %w", err)
		}

		client := getClientFromMeta(testAccProvider.Meta())

		// Check if the topic exists first
		namespaceName, err := utils.GetNameSpaceName(topicName.GetTenant(), topicName.GetNamespace())
		if err != nil {
			return fmt.Errorf("ERROR_READ_NAMESPACE: %w", err)
		}

		topicsClient := client.Topics()
		partitionedTopics, nonPartitionedTopics, err := topicsClient.List(*namespaceName)
		if err != nil {
			return fmt.Errorf("ERROR_READ_TOPIC_DATA: %w", err)
		}

		topicExists := false
		for _, t := range append(partitionedTopics, nonPartitionedTopics...) {
			if topicName.String() == t {
				topicExists = true
				break
			}
		}

		if !topicExists {
			return fmt.Errorf("ERROR_TOPIC_DOES_NOT_EXIST: %s", topic)
		}

		// Now check if the subscription exists
		subscriptions, err := client.Subscriptions().List(*topicName)
		if err != nil {
			return fmt.Errorf("ERROR_READ_SUBSCRIPTION_DATA: %w", err)
		}

		for _, sub := range subscriptions {
			if sub == subscriptionName {
				return nil
			}
		}

		return fmt.Errorf("ERROR_RESOURCE_SUBSCRIPTION_DOES_NOT_EXIST: %s", rs.Primary.ID)
	}
}

func testSubscriptionImported() resource.ImportStateCheckFunc {
	return func(s []*terraform.InstanceState) error {
		if len(s) != 1 {
			return fmt.Errorf("expected %d states, got %d: %#v", 1, len(s), s)
		}

		// Check that we have the required attributes
		requiredAttrs := []string{"topic_name", "subscription_name", "position"}
		for _, attr := range requiredAttrs {
			if _, ok := s[0].Attributes[attr]; !ok {
				return fmt.Errorf("expected attribute %s to be set", attr)
			}
		}

		return nil
	}
}

func createSubscriptionForTest(t *testing.T, topic, subscriptionName string) {
	client, err := sharedClient(testWebServiceURL)
	if err != nil {
		t.Fatalf("ERROR_GETTING_PULSAR_CLIENT: %v", err)
	}

	// First create a topic if it doesn't exist
	topicName, err := utils.GetTopicName(topic)
	if err != nil {
		t.Fatalf("ERROR_PARSING_TOPIC_NAME: %v", err)
	}

	conn := client.(admin.Client)
	// Create topic
	err = conn.Topics().Create(*topicName, 0)
	if err != nil {
		// Topic might already exist
		t.Logf("NOTE_TOPIC_CREATE: %v", err)
	}

	// Wait for the topic to be fully initialized
	time.Sleep(1 * time.Second)

	// Then create a subscription
	messageID := utils.Latest
	err = conn.Subscriptions().Create(*topicName, subscriptionName, messageID)
	if err != nil {
		t.Fatalf("ERROR_CREATING_TEST_SUBSCRIPTION: %v", err)
	}
}

func testPulsarSubscription(url, topic, topicName, subscriptionName, position string) string {
	return fmt.Sprintf(`
provider "pulsar" {
	web_service_url = "%s"
}

resource "pulsar_topic" "sample-topic-1" {
  tenant     = "public"
  namespace  = "default"
  topic_type = "persistent"
  topic_name = "%s"
  partitions = 0
}

resource "pulsar_subscription" "test" {
  topic_name        = "%s"
  subscription_name = "%s"
  position          = "%s"

	depends_on = [pulsar_topic.sample-topic-1]
}
`, url, topic, topicName, subscriptionName, position)
}
