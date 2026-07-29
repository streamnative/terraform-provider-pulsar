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

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// TestAccPulsarNamespace_importHydratesPolicyBlocks is a regression test for issues #206 /
// support #4724: importing a namespace must populate the Optional+Computed policy blocks into state
// so the first plan after the import is empty and the user never has to run a mutating apply just to
// finish the import.
//
// Before the fix, resourcePulsarNamespaceRead gated every block behind d.GetOk, so an import — which
// starts from empty prior state — hydrated none of them and each one showed up as a pending change.
func TestAccPulsarNamespace_importHydratesPolicyBlocks(t *testing.T) {
	cName := acctest.RandString(10)
	tName := acctest.RandString(10)
	nsName := acctest.RandString(10)
	resourceName := "pulsar_namespace.test"

	cfg := testPulsarNamespacePolicyBlocks(testWebServiceURL, cName, tName, nsName)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		CheckDestroy:      testPulsarNamespaceDestroy,
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeTestCheckFunc(
					testPulsarNamespaceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "dispatch_rate.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "subscription_dispatch_rate.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "persistence_policies.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "backlog_quota.#", "1"),
				),
			},
			{
				Config:           cfg,
				ResourceName:     resourceName,
				ImportState:      true,
				ImportStateId:    tName + "/" + nsName,
				ImportStateCheck: testNamespacePolicyBlocksImported(),
			},
			{
				// The reporter's actual complaint: the plan right after an import must be empty.
				Config:             cfg,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// TestAccPulsarNamespace_omittedPolicyBlocksAreNotRemoved is the safety counterpart of the test
// above, and the reason the hydrated blocks are Optional+Computed rather than plain Optional.
//
// A config that omits a policy block means "not managed here", not "delete it from the broker". If
// the blocks were plain Optional, hydrating them on import would make the very first apply propose —
// and for some blocks actually perform — a removal of policies Terraform never owned. This test
// asserts the opposite: after the block disappears from the config, the plan stays empty and the
// policy is still present on the broker.
func TestAccPulsarNamespace_omittedPolicyBlocksAreNotRemoved(t *testing.T) {
	cName := acctest.RandString(10)
	tName := acctest.RandString(10)
	nsName := acctest.RandString(10)
	resourceName := "pulsar_namespace.test"
	fullNamespace := tName + "/" + nsName

	withBlocks := testPulsarNamespacePolicyBlocks(testWebServiceURL, cName, tName, nsName)
	withoutBlocks := testPulsarNamespaceNoPolicyBlocks(testWebServiceURL, cName, tName, nsName)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		CheckDestroy:      testPulsarNamespaceDestroy,
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: withBlocks,
				Check: resource.ComposeTestCheckFunc(
					testPulsarNamespaceExists(resourceName),
					testNamespaceDispatchRateExists(fullNamespace, true),
				),
			},
			{
				// Dropping the blocks from the config must not produce a diff at all.
				Config:             withoutBlocks,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				// The dangerous shape of the reporter's workflow: import a namespace that carries
				// policies the config never mentions. The blocks are still hydrated...
				Config:        withoutBlocks,
				ResourceName:  resourceName,
				ImportState:   true,
				ImportStateId: fullNamespace,
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("expected 1 imported state, got %d", len(states))
					}
					if got := states[0].Attributes["dispatch_rate.#"]; got != "1" {
						return fmt.Errorf("import did not hydrate dispatch_rate: got %q, want %q", got, "1")
					}
					return nil
				},
			},
			{
				// ...and the plan that follows must still be empty, not a removal.
				Config:             withoutBlocks,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				// ...and applying that config must leave the broker-side policy untouched.
				Config: withoutBlocks,
				Check: resource.ComposeTestCheckFunc(
					testPulsarNamespaceExists(resourceName),
					testNamespaceDispatchRateExists(fullNamespace, true),
					resource.TestCheckResourceAttr(resourceName, "dispatch_rate.#", "1"),
				),
			},
		},
	})
}

// testNamespacePolicyBlocksImported asserts that each Optional+Computed policy block was hydrated
// into the imported state. Under the pre-fix behavior every count below is "0"/absent.
func testNamespacePolicyBlocksImported() resource.ImportStateCheckFunc {
	return func(states []*terraform.InstanceState) error {
		if len(states) != 1 {
			return fmt.Errorf("expected 1 imported state, got %d", len(states))
		}
		attrs := states[0].Attributes

		expectedCounts := map[string]string{
			"dispatch_rate.#":              "1",
			"subscription_dispatch_rate.#": "1",
			"persistence_policies.#":       "1",
			"backlog_quota.#":              "1",
		}
		for key, want := range expectedCounts {
			if got := attrs[key]; got != want {
				return fmt.Errorf("import did not hydrate %q: got %q, want %q", key, got, want)
			}
		}

		// TypeSet element keys are hashed, so scan for a representative nested value round-trip.
		if !importedAttrHasValue(attrs, "dispatch_msg_throttling_rate", "50") {
			return fmt.Errorf("expected dispatch_rate to hydrate dispatch_msg_throttling_rate=50; attrs=%#v", attrs)
		}
		return nil
	}
}

func importedAttrHasValue(attrs map[string]string, suffix, want string) bool {
	for k, v := range attrs {
		if strings.HasSuffix(k, suffix) && v == want {
			return true
		}
	}
	return false
}

// testNamespaceDispatchRateExists checks the broker directly, so an assertion about "the policy was
// not removed" cannot be satisfied by Terraform state alone.
func testNamespaceDispatchRateExists(namespace string, shouldExist bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := getClientFromMeta(testAccProvider.Meta()).Namespaces()

		nsName, err := utils.GetNamespaceName(namespace)
		if err != nil {
			return fmt.Errorf("ERROR_PARSING_NAMESPACE: %w", err)
		}

		rate, err := client.GetDispatchRate(*nsName)
		if err != nil {
			if isIgnorableNotFoundError(err) {
				if shouldExist {
					return fmt.Errorf("expected namespace %q to still have a dispatch rate, got 404", namespace)
				}
				return nil
			}
			return fmt.Errorf("ERROR_GETTING_NAMESPACE_DISPATCH_RATE: %w", err)
		}

		if got := isDispatchRateConfigured(rate); got != shouldExist {
			return fmt.Errorf(
				"namespace %q dispatch rate configured=%v, want %v (rate=%+v)",
				namespace, got, shouldExist, rate,
			)
		}
		return nil
	}
}

// testPulsarNamespacePolicyBlocks configures a namespace with exactly the Optional+Computed policy
// blocks that are hydrated on import, so a post-import plan can assert zero drift. The blocks that
// keep removal semantics (retention_policies, inactive_topic, topic_auto_creation, permission_grant)
// and namespace_config are deliberately absent — they are not hydrated on import, by design.
func testPulsarNamespacePolicyBlocks(wsURL, cluster, tenant, ns string) string {
	return testPulsarNamespaceBase(wsURL, cluster, tenant) + fmt.Sprintf(`
resource "pulsar_namespace" "test" {
  tenant    = pulsar_tenant.test_tenant.tenant
  namespace = "%s"

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

  persistence_policies {
    bookkeeper_ensemble                 = 2
    bookkeeper_write_quorum             = 2
    bookkeeper_ack_quorum               = 2
    managed_ledger_max_mark_delete_rate = 0.0
  }

  backlog_quota {
    limit_bytes   = "10000000000"
    limit_seconds = "-1"
    policy        = "producer_request_hold"
    type          = "destination_storage"
  }
}
`, ns)
}

// testPulsarNamespaceNoPolicyBlocks is the same namespace with every policy block omitted.
func testPulsarNamespaceNoPolicyBlocks(wsURL, cluster, tenant, ns string) string {
	return testPulsarNamespaceBase(wsURL, cluster, tenant) + fmt.Sprintf(`
resource "pulsar_namespace" "test" {
  tenant    = pulsar_tenant.test_tenant.tenant
  namespace = "%s"
}
`, ns)
}

func testPulsarNamespaceBase(wsURL, cluster, tenant string) string {
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
`, wsURL, cluster, tenant)
}
