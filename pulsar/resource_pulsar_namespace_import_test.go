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

// TestAccPulsarNamespace_hydratesPolicyBlocks is a regression test for issues #206 /
// support #4724: importing a namespace must populate the Optional+Computed policy blocks into state
// so the first plan after the import is empty and the user never has to run a mutating apply just to
// finish the import.
//
// Before the fix, resourcePulsarNamespaceRead gated every block behind d.GetOk, so an import — which
// starts from empty prior state — hydrated none of them and each one showed up as a pending change.
func TestAccPulsarNamespace_hydratesPolicyBlocks(t *testing.T) {
	tName := acctest.RandString(10)
	nsName := acctest.RandString(10)
	resourceName := "pulsar_namespace.test"
	fullNamespace := tName + "/" + nsName

	cfg := testPulsarNamespaceFullPolicyBlocksImportConfig(testWebServiceURL, tName, nsName)
	t.Cleanup(func() { deleteNamespaceImportFixture(tName, fullNamespace) })

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if err := createNamespacePartialBacklogQuotaImportFixture(tName, fullNamespace); err != nil {
				t.Fatalf("create namespace import fixture: %v", err)
			}
		},
		CheckDestroy:      testPulsarNamespaceDestroy,
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             cfg,
				ResourceName:       resourceName,
				ImportState:        true,
				ImportStatePersist: true,
				ImportStateId:      fullNamespace,
				ImportStateCheck:   testNamespacePolicyBlocksImported(),
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

func TestAccPulsarNamespace_partialImportPreservesUnmanagedBacklogQuotaTypes(t *testing.T) {
	tName := acctest.RandString(10)
	nsName := acctest.RandString(10)
	resourceName := "pulsar_namespace.test"
	fullNamespace := tName + "/" + nsName

	cfg := testPulsarNamespacePolicyBlocksImportConfig(testWebServiceURL, tName, nsName)
	cfgWithUnrelatedUpdate := testPulsarNamespacePolicyBlocksImportConfigWithDeduplication(
		testWebServiceURL, tName, nsName, true,
	)
	t.Cleanup(func() { deleteNamespaceImportFixture(tName, fullNamespace) })

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if err := createNamespacePartialBacklogQuotaImportFixture(tName, fullNamespace); err != nil {
				t.Fatalf("create namespace partial backlog quota import fixture: %v", err)
			}
		},
		CheckDestroy:      testPulsarNamespaceDestroy,
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             cfg,
				ResourceName:       resourceName,
				ImportState:        true,
				ImportStatePersist: true,
				ImportStateId:      fullNamespace,
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("expected 1 imported state, got %d", len(states))
					}
					if got := states[0].Attributes["backlog_quota.#"]; got != "2" {
						return fmt.Errorf("import did not hydrate both backlog quota types: got %q, want %q", got, "2")
					}
					if got := states[0].Attributes[backlogQuotaManagedTypesStateAttr+".#"]; got != "0" {
						return fmt.Errorf("imported backlog quota ownership must start empty: got %q, want %q", got, "0")
					}
					return nil
				},
			},
			{
				Config:             cfg,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				Config: cfg,
				Check: resource.ComposeTestCheckFunc(
					testPulsarNamespaceExists(resourceName),
					testNamespaceBacklogQuotaTypes(
						fullNamespace,
						utils.DestinationStorage,
						utils.MessageAge,
					),
				),
			},
			{
				Config:             cfg,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				Config: cfgWithUnrelatedUpdate,
				Check: resource.ComposeTestCheckFunc(
					testNamespaceBacklogQuotaTypes(
						fullNamespace,
						utils.DestinationStorage,
						utils.MessageAge,
					),
					resource.TestCheckResourceAttr(
						resourceName, backlogQuotaManagedTypesStateAttr+".#", "1",
					),
					resource.TestCheckTypeSetElemAttr(
						resourceName, backlogQuotaManagedTypesStateAttr+".*", utils.DestinationStorage.String(),
					),
				),
			},
			{
				Config:             cfgWithUnrelatedUpdate,
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
// asserts the opposite: with the blocks absent from the importing config, the plan stays empty and
// the policies remain present on the broker.
func TestAccPulsarNamespace_omittedPolicyBlocksAreNotRemoved(t *testing.T) {
	tName := acctest.RandString(10)
	nsName := acctest.RandString(10)
	resourceName := "pulsar_namespace.test"
	fullNamespace := tName + "/" + nsName

	withoutBlocks := testPulsarNamespaceNoPolicyBlocksImportConfig(testWebServiceURL, tName, nsName)
	t.Cleanup(func() { deleteNamespaceImportFixture(tName, fullNamespace) })

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if err := createNamespacePolicyBlocksImportFixture(tName, fullNamespace); err != nil {
				t.Fatalf("create namespace import fixture: %v", err)
			}
		},
		CheckDestroy:      testPulsarNamespaceDestroy,
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				// The dangerous shape of the reporter's workflow: import a namespace that carries
				// policies the config never mentions. The blocks are still hydrated...
				Config:             withoutBlocks,
				ResourceName:       resourceName,
				ImportState:        true,
				ImportStatePersist: true,
				ImportStateId:      fullNamespace,
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

func createNamespacePolicyBlocksImportFixture(tenant, namespace string) error {
	adminClient := getClientFromMeta(testAccProvider.Meta())
	if err := adminClient.Tenants().Create(utils.TenantData{
		Name:            tenant,
		AllowedClusters: []string{"standalone"},
	}); err != nil {
		return fmt.Errorf("create tenant %q: %w", tenant, err)
	}

	client := adminClient.Namespaces()
	if err := client.CreateNamespace(namespace); err != nil {
		return fmt.Errorf("create namespace %q: %w", namespace, err)
	}

	nsName, err := utils.GetNamespaceName(namespace)
	if err != nil {
		return fmt.Errorf("parse namespace %q: %w", namespace, err)
	}
	rate := utils.DispatchRate{
		DispatchThrottlingRateInMsg:  50,
		DispatchThrottlingRateInByte: 2048,
		RatePeriodInSecond:           50,
	}
	if err := client.SetDispatchRate(*nsName, rate); err != nil {
		return fmt.Errorf("set dispatch rate for %q: %w", namespace, err)
	}
	if err := client.SetSubscriptionDispatchRate(*nsName, rate); err != nil {
		return fmt.Errorf("set subscription dispatch rate for %q: %w", namespace, err)
	}
	if err := client.SetPersistence(namespace, utils.PersistencePolicies{
		BookkeeperEnsemble:             2,
		BookkeeperWriteQuorum:          2,
		BookkeeperAckQuorum:            2,
		ManagedLedgerMaxMarkDeleteRate: 0,
	}); err != nil {
		return fmt.Errorf("set persistence for %q: %w", namespace, err)
	}
	if err := client.SetBacklogQuota(namespace, utils.BacklogQuota{
		LimitSize: 10_000_000_000,
		LimitTime: -1,
		Policy:    utils.ProducerRequestHold,
	}, utils.DestinationStorage); err != nil {
		return fmt.Errorf("set backlog quota for %q: %w", namespace, err)
	}
	return nil
}

func createNamespacePartialBacklogQuotaImportFixture(tenant, namespace string) error {
	if err := createNamespacePolicyBlocksImportFixture(tenant, namespace); err != nil {
		return err
	}

	if err := getClientFromMeta(testAccProvider.Meta()).Namespaces().SetBacklogQuota(
		namespace,
		utils.BacklogQuota{
			LimitSize: -1,
			LimitTime: 3600,
			Policy:    utils.ConsumerBacklogEviction,
		},
		utils.MessageAge,
	); err != nil {
		return fmt.Errorf("set message age backlog quota for %q: %w", namespace, err)
	}

	return nil
}

func deleteNamespaceImportFixture(tenant, namespace string) {
	meta := testAccProvider.Meta()
	if meta == nil {
		return
	}
	client := getClientFromMeta(meta)
	_ = client.Namespaces().DeleteNamespace(namespace)
	_ = client.Tenants().Delete(tenant)
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
			"dispatch_rate.#":                        "1",
			"subscription_dispatch_rate.#":           "1",
			"persistence_policies.#":                 "1",
			"backlog_quota.#":                        "2",
			backlogQuotaManagedTypesStateAttr + ".#": "0",
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

func testPulsarNamespacePolicyBlocksImportConfig(wsURL, tenant, namespace string) string {
	return testPulsarNamespacePolicyBlocksImportConfigWithOptions(wsURL, tenant, namespace, false, false)
}

func testPulsarNamespaceFullPolicyBlocksImportConfig(wsURL, tenant, namespace string) string {
	return testPulsarNamespacePolicyBlocksImportConfigWithOptions(wsURL, tenant, namespace, false, true)
}

func testPulsarNamespacePolicyBlocksImportConfigWithDeduplication(
	wsURL, tenant, namespace string,
	enableDeduplication bool,
) string {
	return testPulsarNamespacePolicyBlocksImportConfigWithOptions(
		wsURL, tenant, namespace, enableDeduplication, false,
	)
}

func testPulsarNamespacePolicyBlocksImportConfigWithOptions(
	wsURL, tenant, namespace string,
	enableDeduplication, includeMessageAge bool,
) string {
	deduplication := ""
	if enableDeduplication {
		deduplication = "  enable_deduplication = true\n"
	}
	messageAge := ""
	if includeMessageAge {
		messageAge = `
  backlog_quota {
    limit_bytes   = "-1"
    limit_seconds = "3600"
    policy        = "consumer_backlog_eviction"
    type          = "message_age"
  }
`
	}

	return fmt.Sprintf(`
provider "pulsar" {
  web_service_url = %q
}

resource "pulsar_namespace" "test" {
  tenant    = %q
  namespace = %q
%s

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
%s
}
`, wsURL, tenant, namespace, deduplication, messageAge)
}

func testPulsarNamespaceNoPolicyBlocksImportConfig(wsURL, tenant, namespace string) string {
	return fmt.Sprintf(`
provider "pulsar" {
  web_service_url = %q
}

resource "pulsar_namespace" "test" {
  tenant    = %q
  namespace = %q
}
`, wsURL, tenant, namespace)
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
