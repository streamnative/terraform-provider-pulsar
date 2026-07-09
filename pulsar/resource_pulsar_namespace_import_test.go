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

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// TestImportNamespaceHydratesOptionalBlocks is a regression test for issues #206 / support #4724:
// importing a namespace must populate its optional leaf sub-blocks into state so a subsequent plan
// shows no spurious drift. Before the fix, resourcePulsarNamespaceRead gated every block behind
// d.GetOk, so an import (which starts from empty prior state) hydrated none of them and every block
// showed up as a pending change on the next plan.
//
// The final PlanOnly step is the real assertion of the reporter's complaint: after import the plan
// must be empty. namespace_config is intentionally excluded from this config because it is not
// force-hydrated on import (see resourcePulsarNamespaceReadWithHydration).
func TestImportNamespaceHydratesOptionalBlocks(t *testing.T) {
	cName := acctest.RandString(10)
	tName := acctest.RandString(10)
	nsName := acctest.RandString(10)
	resourceName := "pulsar_namespace.test"
	id := tName + "/" + nsName

	cfg := testPulsarNamespaceLeafBlocks(testWebServiceURL, cName, tName, nsName)

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
					resource.TestCheckResourceAttr(resourceName, "permission_grant.#", "2"),
				),
			},
			{
				ResourceName:     resourceName,
				ImportState:      true,
				ImportStateId:    id,
				Config:           cfg,
				ImportStateCheck: testNamespaceOptionalBlocksImported(),
			},
			{
				// After import the state must fully match config: no spurious drift.
				Config:             cfg,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// testNamespaceOptionalBlocksImported asserts that each leaf sub-block was hydrated into the
// imported state. Under the pre-fix behavior every count below is "0"/absent.
func testNamespaceOptionalBlocksImported() resource.ImportStateCheckFunc {
	return func(states []*terraform.InstanceState) error {
		if len(states) != 1 {
			return fmt.Errorf("expected 1 imported state, got %d", len(states))
		}
		attrs := states[0].Attributes

		expectedCounts := map[string]string{
			"dispatch_rate.#":              "1",
			"subscription_dispatch_rate.#": "1",
			"retention_policies.#":         "1",
			"persistence_policies.#":       "1",
			"backlog_quota.#":              "1",
			"topic_auto_creation.#":        "1",
			"permission_grant.#":           "2",
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

// testPulsarNamespaceLeafBlocks configures a namespace with only the optional leaf blocks that the
// import fix hydrates (no namespace_config), so a post-import plan can assert zero drift.
func testPulsarNamespaceLeafBlocks(wsURL, cluster, tenant, ns string) string {
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

  retention_policies {
    retention_minutes    = "1600"
    retention_size_in_mb = "10000"
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

  permission_grant {
    role    = "some-role-1"
    actions = ["produce", "consume", "functions"]
  }

  permission_grant {
    role    = "some-role-2"
    actions = ["produce", "consume"]
  }

  topic_auto_creation {
    enable     = true
    type       = "partitioned"
    partitions = 3
  }
}
`, wsURL, cluster, tenant, ns)
}
