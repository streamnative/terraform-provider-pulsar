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

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// TestTopicParameterDefaults verifies that omitted parameters default to -1 and are not set
func TestTopicParameterDefaults(t *testing.T) {
	skipIfNoTopicPolicies(t)
	resourceName := "pulsar_topic.test"
	tname := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testPulsarTopicDestroy,
		Steps: []resource.TestStep{
			{
				// Create topic without specifying any topic_config parameters
				Config: fmt.Sprintf(`
					provider "pulsar" {
						web_service_url = "%s"
					}
					resource "pulsar_topic" "test" {
						tenant     = "public"
						namespace  = "default"
						topic_type = "persistent"
						topic_name = "test-topic-defaults-%s"
						partitions = 0

						topic_config {
							# All parameters omitted - should default to -1 internally
						}
					}`, testWebServiceURL, tname),
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists(resourceName, t),
					// Verify internal defaults are -1 (state check)
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.max_consumers", "-1"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.max_producers", "-1"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.message_ttl_seconds", "-1"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.max_unacked_messages_per_consumer", "-1"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.max_unacked_messages_per_subscription", "-1"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.msg_publish_rate", "-1"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.byte_publish_rate", "-1"),
					// Verify no topic-level policies are set via API
					testTopicHasNoExplicitConfig(resourceName),
				),
			},
		},
	})
}

// TestTopicParameterRemoval verifies that setting parameters to -1 removes them
func TestTopicParameterRemoval(t *testing.T) {
	skipIfNoTopicPolicies(t)
	resourceName := "pulsar_topic.test"
	tname := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testPulsarTopicDestroy,
		Steps: []resource.TestStep{
			{
				// First, create topic with explicit values
				Config: fmt.Sprintf(`
					provider "pulsar" {
						web_service_url = "%s"
					}
					resource "pulsar_topic" "test" {
						tenant     = "public"
						namespace  = "default"
						topic_type = "persistent"
						topic_name = "test-topic-removal-%s"
						partitions = 0

						topic_config {
							max_consumers = 100
							max_producers = 50
							message_ttl_seconds = 3600
							max_unacked_messages_per_consumer = 1000
							max_unacked_messages_per_subscription = 5000
							msg_publish_rate = 1000
							byte_publish_rate = 1048576
						}
					}`, testWebServiceURL, tname),
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists(resourceName, t),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.max_consumers", "100"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.max_producers", "50"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.message_ttl_seconds", "3600"),
					testTopicHasExplicitConfig(resourceName, 100, 50, 3600),
				),
			},
			{
				// Then update to omit parameters (they should default to -1 and be removed)
				Config: fmt.Sprintf(`
					provider "pulsar" {
						web_service_url = "%s"
					}
					resource "pulsar_topic" "test" {
						tenant     = "public"
						namespace  = "default"
						topic_type = "persistent"
						topic_name = "test-topic-removal-%s"
						partitions = 0

						topic_config {
							# All parameters omitted - should trigger removal
						}
					}`, testWebServiceURL, tname),
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists(resourceName, t),
					// Verify defaults are applied
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.max_consumers", "-1"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.max_producers", "-1"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.message_ttl_seconds", "-1"),
					// Verify topic-level policies were removed
					testTopicHasNoExplicitConfig(resourceName),
				),
			},
		},
	})
}

// TestTopicExplicitZeroValues verifies that explicit 0 values are set properly
func TestTopicExplicitZeroValues(t *testing.T) {
	skipIfNoTopicPolicies(t)
	resourceName := "pulsar_topic.test"
	tname := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testPulsarTopicDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					provider "pulsar" {
						web_service_url = "%s"
					}
					resource "pulsar_topic" "test" {
						tenant     = "public"
						namespace  = "default"
						topic_type = "persistent"
						topic_name = "test-topic-zero-%s"
						partitions = 0

						topic_config {
							max_consumers = 0
							max_producers = 0
							message_ttl_seconds = 0
							max_unacked_messages_per_consumer = 0
							max_unacked_messages_per_subscription = 0
							msg_publish_rate = 0
							byte_publish_rate = 0
						}
					}`, testWebServiceURL, tname),
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists(resourceName, t),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.max_consumers", "0"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.max_producers", "0"),
					resource.TestCheckResourceAttr(resourceName, "topic_config.0.message_ttl_seconds", "0"),
					// Verify 0 values are actually set via API
					testTopicHasExplicitConfig(resourceName, 0, 0, 0),
				),
			},
		},
	})
}

// Helper function to verify no topic-level configuration is set
func testTopicHasNoExplicitConfig(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found", resourceName)
		}

		topicName, err := utils.GetTopicName(rs.Primary.ID)
		if err != nil {
			return err
		}

		client := getClientFromMeta(testAccProvider.Meta()).Topics()

		// Check that getting these values returns errors (not found/not set)
		// The actual error type/message depends on whether topic policies are enabled
		_, err = client.GetMaxConsumers(*topicName)
		if err == nil {
			return fmt.Errorf("expected GetMaxConsumers to fail for unset value, but it succeeded")
		}

		_, err = client.GetMaxProducers(*topicName)
		if err == nil {
			return fmt.Errorf("expected GetMaxProducers to fail for unset value, but it succeeded")
		}

		_, err = client.GetMessageTTL(*topicName)
		if err == nil {
			return fmt.Errorf("expected GetMessageTTL to fail for unset value, but it succeeded")
		}

		return nil
	}
}

// Helper function to verify topic has explicit configuration
func testTopicHasExplicitConfig(resourceName string, maxConsumers, maxProducers, messageTTL int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found", resourceName)
		}

		topicName, err := utils.GetTopicName(rs.Primary.ID)
		if err != nil {
			return err
		}

		client := getClientFromMeta(testAccProvider.Meta()).Topics()

		// Verify values are set correctly
		actualMaxConsumers, err := client.GetMaxConsumers(*topicName)
		if err != nil {
			return fmt.Errorf("failed to get max consumers: %w", err)
		}
		if actualMaxConsumers != maxConsumers {
			return fmt.Errorf("expected max consumers %d, got %d", maxConsumers, actualMaxConsumers)
		}

		actualMaxProducers, err := client.GetMaxProducers(*topicName)
		if err != nil {
			return fmt.Errorf("failed to get max producers: %w", err)
		}
		if actualMaxProducers != maxProducers {
			return fmt.Errorf("expected max producers %d, got %d", maxProducers, actualMaxProducers)
		}

		actualMessageTTL, err := client.GetMessageTTL(*topicName)
		if err != nil {
			return fmt.Errorf("failed to get message TTL: %w", err)
		}
		if actualMessageTTL != messageTTL {
			return fmt.Errorf("expected message TTL %d, got %d", messageTTL, actualMessageTTL)
		}

		return nil
	}
}
