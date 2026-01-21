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
				),
			},
		},
	})
}

// TestNamespaceConfigRemoval verifies subscription_expiration_time_minutes can be removed
// This is the only removal supported by the API at this time
func TestNamespaceConfigRemoval(t *testing.T) {
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
					// Verify -1 values are actually set via API
					testNamespaceConfigValue(resourceName, "subscription expiration time", -1, func(c, ns interface{}) (int, error) {
						return c.(interface {
							GetSubscriptionExpirationTime(utils.NameSpaceName) (int, error)
						}).GetSubscriptionExpirationTime(*ns.(*utils.NameSpaceName))
					}),
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
					testNamespaceConfigValue(resourceName, "max consumers per subscription", 0, func(c, ns interface{}) (int, error) {
						return c.(interface {
							GetMaxConsumersPerSubscription(utils.NameSpaceName) (int, error)
						}).GetMaxConsumersPerSubscription(*ns.(*utils.NameSpaceName))
					}),
					testNamespaceConfigValue(resourceName, "max consumers per topic", 0, func(c, ns interface{}) (int, error) {
						return c.(interface {
							GetMaxConsumersPerTopic(utils.NameSpaceName) (int, error)
						}).GetMaxConsumersPerTopic(*ns.(*utils.NameSpaceName))
					}),
					testNamespaceConfigValue(resourceName, "max producers per topic", 0, func(c, ns interface{}) (int, error) {
						return c.(interface {
							GetMaxProducersPerTopic(utils.NameSpaceName) (int, error)
						}).GetMaxProducersPerTopic(*ns.(*utils.NameSpaceName))
					}),
					testNamespaceConfigValue(resourceName, "message TTL", 0, func(c, ns interface{}) (int, error) {
						return c.(interface {
							GetNamespaceMessageTTL(string) (int, error)
						}).GetNamespaceMessageTTL(ns.(*utils.NameSpaceName).String())
					}),
					testNamespaceConfigValue(resourceName, "subscription expiration time", 0, func(c, ns interface{}) (int, error) {
						return c.(interface {
							GetSubscriptionExpirationTime(utils.NameSpaceName) (int, error)
						}).GetSubscriptionExpirationTime(*ns.(*utils.NameSpaceName))
					}),
				),
			},
		},
	})
}

// Helper function to verify namespace has a specific config value
// The getConfig parameter is a function that calls the specific API method on the namespace client
//
//nolint:unparam // resourceName is intentionally a parameter for reusability across different tests
func testNamespaceConfigValue(
	resourceName string,
	configName string,
	expectedValue int,
	getConfig func(client interface{}, ns interface{}) (int, error),
) resource.TestCheckFunc {
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
		actualValue, err := getConfig(client, ns)
		if err != nil {
			return fmt.Errorf("failed to get %s: %w", configName, err)
		}

		if actualValue != expectedValue {
			return fmt.Errorf("expected %s %d, got %d", configName, expectedValue, actualValue)
		}

		return nil
	}
}
