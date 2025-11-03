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

// TestNamespaceParameterDefaults verifies that omitted parameters default to -1 and are not set
func TestNamespaceParameterDefaults(t *testing.T) {
	resourceName := "pulsar_namespace.test"
	nsname := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testPulsarNamespaceDestroy,
		Steps: []resource.TestStep{
			{
				// Create namespace without specifying the parameters that now default to -1
				Config: fmt.Sprintf(`
					provider "pulsar" {
						web_service_url = "%s"
					}
					resource "pulsar_namespace" "test" {
						tenant    = "public"
						namespace = "test-ns-defaults-%s"

						namespace_config {
							# Parameters with -1 defaults are omitted
							# Only set required or different parameters
							is_allow_auto_update_schema = true
						}
					}`, testWebServiceURL, nsname),
				Check: resource.ComposeTestCheckFunc(
					testPulsarNamespaceExists(resourceName),
					// Verify internal defaults are -1 (state check)
					resource.TestCheckResourceAttr(resourceName, "namespace_config.0.max_consumers_per_subscription", "-1"),
					resource.TestCheckResourceAttr(resourceName, "namespace_config.0.max_consumers_per_topic", "-1"),
					resource.TestCheckResourceAttr(resourceName, "namespace_config.0.max_producers_per_topic", "-1"),
					resource.TestCheckResourceAttr(resourceName, "namespace_config.0.message_ttl_seconds", "-1"),
					resource.TestCheckResourceAttr(resourceName, "namespace_config.0.subscription_expiration_time_minutes", "-1"),
					// Verify these are using broker defaults (not explicitly set to 0)
					testNamespaceUsesDefaults(resourceName),
				),
			},
		},
	})
}

// TestNamespaceSubscriptionExpirationRemoval verifies subscription_expiration_time_minutes can be removed
func TestNamespaceSubscriptionExpirationRemoval(t *testing.T) {
	resourceName := "pulsar_namespace.test"
	nsname := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testPulsarNamespaceDestroy,
		Steps: []resource.TestStep{
			{
				// First, create namespace with explicit subscription expiration
				Config: fmt.Sprintf(`
					provider "pulsar" {
						web_service_url = "%s"
					}
					resource "pulsar_namespace" "test" {
						tenant    = "public"
						namespace = "test-ns-removal-%s"

						namespace_config {
							subscription_expiration_time_minutes = 60
						}
					}`, testWebServiceURL, nsname),
				Check: resource.ComposeTestCheckFunc(
					testPulsarNamespaceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "namespace_config.0.subscription_expiration_time_minutes", "60"),
					testNamespaceHasSubscriptionExpiration(resourceName, 60),
				),
			},
			{
				// Then update to omit the parameter (should default to -1 and be removed)
				Config: fmt.Sprintf(`
					provider "pulsar" {
						web_service_url = "%s"
					}
					resource "pulsar_namespace" "test" {
						tenant    = "public"
						namespace = "test-ns-removal-%s"

						namespace_config {
							# subscription_expiration_time_minutes omitted - should trigger removal
						}
					}`, testWebServiceURL, nsname),
				Check: resource.ComposeTestCheckFunc(
					testPulsarNamespaceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "namespace_config.0.subscription_expiration_time_minutes", "-1"),
					// Verify it was removed (should return 0 which is the broker default)
					testNamespaceHasSubscriptionExpiration(resourceName, 0),
				),
			},
		},
	})
}

// TestNamespaceExplicitZeroValues verifies that explicit 0 values are set properly
func TestNamespaceExplicitZeroValues(t *testing.T) {
	resourceName := "pulsar_namespace.test"
	nsname := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testPulsarNamespaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					provider "pulsar" {
						web_service_url = "%s"
					}
					resource "pulsar_namespace" "test" {
						tenant    = "public"
						namespace = "test-ns-zero-%s"

						namespace_config {
							max_consumers_per_subscription = 0
							max_consumers_per_topic = 0
							max_producers_per_topic = 0
							message_ttl_seconds = 0
							subscription_expiration_time_minutes = 0
						}
					}`, testWebServiceURL, nsname),
				Check: resource.ComposeTestCheckFunc(
					testPulsarNamespaceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "namespace_config.0.max_consumers_per_subscription", "0"),
					resource.TestCheckResourceAttr(resourceName, "namespace_config.0.max_consumers_per_topic", "0"),
					resource.TestCheckResourceAttr(resourceName, "namespace_config.0.max_producers_per_topic", "0"),
					resource.TestCheckResourceAttr(resourceName, "namespace_config.0.message_ttl_seconds", "0"),
					resource.TestCheckResourceAttr(resourceName, "namespace_config.0.subscription_expiration_time_minutes", "0"),
					// Verify 0 values are actually set via API
					testNamespaceHasExplicitZeros(resourceName),
				),
			},
		},
	})
}

// Helper function to verify namespace is using broker defaults
func testNamespaceUsesDefaults(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found", resourceName)
		}

		tenant := rs.Primary.Attributes["tenant"]
		namespace := rs.Primary.Attributes["namespace"]
		ns, err := utils.GetNameSpaceName(tenant, namespace)
		if err != nil {
			return err
		}

		client := getClientFromMeta(testAccProvider.Meta()).Namespaces()

		// When using defaults, these should return broker default values
		// The actual default values depend on the broker configuration
		// We're mainly checking that our -1 default didn't cause Set operations

		// For parameters without Remove methods, they might return 0 if never set
		maxConsPerSub, err := client.GetMaxConsumersPerSubscription(*ns)
		if err != nil {
			return fmt.Errorf("failed to get max consumers per subscription: %w", err)
		}
		// If never set, broker typically returns 0
		if maxConsPerSub != 0 {
			return fmt.Errorf("expected max consumers per subscription to be broker default (0), got %d", maxConsPerSub)
		}

		return nil
	}
}

// Helper function to verify namespace has specific subscription expiration
func testNamespaceHasSubscriptionExpiration(resourceName string, expected int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found", resourceName)
		}

		tenant := rs.Primary.Attributes["tenant"]
		namespace := rs.Primary.Attributes["namespace"]
		ns, err := utils.GetNameSpaceName(tenant, namespace)
		if err != nil {
			return err
		}

		client := getClientFromMeta(testAccProvider.Meta()).Namespaces()

		actual, err := client.GetSubscriptionExpirationTime(*ns)
		if err != nil {
			return fmt.Errorf("failed to get subscription expiration time: %w", err)
		}

		if actual != expected {
			return fmt.Errorf("expected subscription expiration time %d, got %d", expected, actual)
		}

		return nil
	}
}

// Helper function to verify namespace has explicit zero values
func testNamespaceHasExplicitZeros(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found", resourceName)
		}

		tenant := rs.Primary.Attributes["tenant"]
		namespace := rs.Primary.Attributes["namespace"]
		ns, err := utils.GetNameSpaceName(tenant, namespace)
		if err != nil {
			return err
		}

		client := getClientFromMeta(testAccProvider.Meta()).Namespaces()

		// Verify all values are explicitly set to 0
		maxConsPerSub, err := client.GetMaxConsumersPerSubscription(*ns)
		if err != nil {
			return fmt.Errorf("failed to get max consumers per subscription: %w", err)
		}
		if maxConsPerSub != 0 {
			return fmt.Errorf("expected max consumers per subscription 0, got %d", maxConsPerSub)
		}

		maxConsPerTopic, err := client.GetMaxConsumersPerTopic(*ns)
		if err != nil {
			return fmt.Errorf("failed to get max consumers per topic: %w", err)
		}
		if maxConsPerTopic != 0 {
			return fmt.Errorf("expected max consumers per topic 0, got %d", maxConsPerTopic)
		}

		maxProdPerTopic, err := client.GetMaxProducersPerTopic(*ns)
		if err != nil {
			return fmt.Errorf("failed to get max producers per topic: %w", err)
		}
		if maxProdPerTopic != 0 {
			return fmt.Errorf("expected max producers per topic 0, got %d", maxProdPerTopic)
		}

		messageTTL, err := client.GetNamespaceMessageTTL(ns.String())
		if err != nil {
			return fmt.Errorf("failed to get message TTL: %w", err)
		}
		if messageTTL != 0 {
			return fmt.Errorf("expected message TTL 0, got %d", messageTTL)
		}

		subExpTime, err := client.GetSubscriptionExpirationTime(*ns)
		if err != nil {
			return fmt.Errorf("failed to get subscription expiration time: %w", err)
		}
		if subExpTime != 0 {
			return fmt.Errorf("expected subscription expiration time 0, got %d", subExpTime)
		}

		return nil
	}
}
