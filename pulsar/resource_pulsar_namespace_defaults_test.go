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

// TestNamespaceConfigRemoval verifies namespace config policies can be removed
// by omitting them from configuration and falling back to broker defaults.
func TestNamespaceConfigRemoval(t *testing.T) {
	resourceName := "pulsar_namespace.test"
	nsname := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testPulsarNamespaceDestroy,
		Steps: []resource.TestStep{
			{
				// First, set namespace policies with explicit values.
				Config: fmt.Sprintf(`
						provider "pulsar" {
							web_service_url = "%s"
						}
						resource "pulsar_namespace" "test" {
							tenant    = "public"
							namespace = "test-ns-removal-%s"

							namespace_config {
								max_consumers_per_subscription      = 60
								max_consumers_per_topic             = 50
								max_producers_per_topic             = 40
								message_ttl_seconds                 = 3600
								subscription_expiration_time_minutes = 30
							}
						}`, testWebServiceURL, nsname),
				Check: resource.ComposeTestCheckFunc(
					testPulsarNamespaceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "namespace_config.0.max_consumers_per_subscription", "60"),
					resource.TestCheckResourceAttr(resourceName, "namespace_config.0.max_consumers_per_topic", "50"),
					resource.TestCheckResourceAttr(resourceName, "namespace_config.0.max_producers_per_topic", "40"),
					resource.TestCheckResourceAttr(resourceName, "namespace_config.0.message_ttl_seconds", "3600"),
					resource.TestCheckResourceAttr(resourceName, "namespace_config.0.subscription_expiration_time_minutes", "30"),
				),
			},
			{
				// Then omit the policies to trigger remove and fallback behavior.
				Config: fmt.Sprintf(`
						provider "pulsar" {
							web_service_url = "%s"
					}
					resource "pulsar_namespace" "test" {
							tenant    = "public"
							namespace = "test-ns-removal-%s"

							namespace_config {
								# policies omitted - should trigger removal
							}
						}`, testWebServiceURL, nsname),
				Check: resource.ComposeTestCheckFunc(
					testPulsarNamespaceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "namespace_config.0.max_consumers_per_subscription", "-1"),
					resource.TestCheckResourceAttr(resourceName, "namespace_config.0.max_consumers_per_topic", "-1"),
					resource.TestCheckResourceAttr(resourceName, "namespace_config.0.max_producers_per_topic", "-1"),
					resource.TestCheckResourceAttr(resourceName, "namespace_config.0.message_ttl_seconds", "-1"),
					resource.TestCheckResourceAttr(resourceName, "namespace_config.0.subscription_expiration_time_minutes", "-1"),
					testNamespaceNullablePolicyValue("max consumers per subscription", false, func(p *utils.Policies) *int {
						return p.MaxConsumersPerSubscription
					}),
					testNamespaceNullablePolicyValue("max consumers per topic", false, func(p *utils.Policies) *int {
						return p.MaxConsumersPerTopic
					}),
					testNamespaceNullablePolicyValue("max producers per topic", false, func(p *utils.Policies) *int {
						return p.MaxProducersPerTopic
					}),
					testNamespaceNullablePolicyValue("message TTL", false, func(p *utils.Policies) *int {
						return p.MessageTTLInSeconds
					}),
					testNamespaceConfigValue(resourceName, "subscription expiration time", -1, func(c, ns interface{}) (int, error) {
						return c.(interface {
							GetSubscriptionExpirationTime(utils.NameSpaceName) (int, error)
						}).GetSubscriptionExpirationTime(*ns.(*utils.NameSpaceName))
					}),
				),
			},
			{
				// Repeated refresh/plan after unset should not drift.
				Config: fmt.Sprintf(`
						provider "pulsar" {
							web_service_url = "%s"
						}
						resource "pulsar_namespace" "test" {
							tenant    = "public"
							namespace = "test-ns-removal-%s"

							namespace_config {
								# policies omitted - should remain removed
							}
						}`, testWebServiceURL, nsname),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				// A second plan-only run validates idempotency after another refresh.
				Config: fmt.Sprintf(`
						provider "pulsar" {
							web_service_url = "%s"
						}
						resource "pulsar_namespace" "test" {
							tenant    = "public"
							namespace = "test-ns-removal-%s"

							namespace_config {
								# policies omitted - should remain removed
							}
						}`, testWebServiceURL, nsname),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
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
					testNamespaceNullablePolicyValue("max consumers per subscription", true, func(p *utils.Policies) *int {
						return p.MaxConsumersPerSubscription
					}),
					testNamespaceNullablePolicyValue("max consumers per topic", true, func(p *utils.Policies) *int {
						return p.MaxConsumersPerTopic
					}),
					testNamespaceNullablePolicyValue("max producers per topic", true, func(p *utils.Policies) *int {
						return p.MaxProducersPerTopic
					}),
					testNamespaceNullablePolicyValue("message TTL", true, func(p *utils.Policies) *int {
						return p.MessageTTLInSeconds
					}),
					testNamespaceConfigValue(resourceName, "subscription expiration time", 0, func(c, ns interface{}) (int, error) {
						return c.(interface {
							GetSubscriptionExpirationTime(utils.NameSpaceName) (int, error)
						}).GetSubscriptionExpirationTime(*ns.(*utils.NameSpaceName))
					}),
				),
			},
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
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
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

func testNamespaceNullablePolicyValue(
	policyName string,
	expectSet bool,
	getValue func(*utils.Policies) *int,
) resource.TestCheckFunc {
	const (
		resourceName  = "pulsar_namespace.test"
		expectedValue = 0
	)

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
		policies, err := client.GetPolicies(ns.String())
		if err != nil {
			return fmt.Errorf("failed to get namespace policies for %s: %w", policyName, err)
		}

		actual := getValue(policies)
		if !expectSet {
			if actual != nil {
				return fmt.Errorf("expected %s to be unset, got %d", policyName, *actual)
			}
			return nil
		}

		if actual == nil {
			return fmt.Errorf("expected %s to be set to %d, but value is unset", policyName, expectedValue)
		}

		if *actual != expectedValue {
			return fmt.Errorf("expected %s %d, got %d", policyName, expectedValue, *actual)
		}

		return nil
	}
}
