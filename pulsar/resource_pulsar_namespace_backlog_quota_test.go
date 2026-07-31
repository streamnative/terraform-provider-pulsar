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

func TestAccPulsarNamespace_BacklogQuotaTypeRemoval(t *testing.T) {
	cluster := acctest.RandString(10)
	tenant := acctest.RandString(10)
	namespace := acctest.RandString(10)
	fullNamespace := tenant + "/" + namespace

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		CheckDestroy:      testPulsarNamespaceDestroy,
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testPulsarNamespaceBacklogQuotas(testWebServiceURL, cluster, tenant, namespace, true, true),
				Check:  testNamespaceBacklogQuotaTypes(fullNamespace, utils.DestinationStorage, utils.MessageAge),
			},
			{
				Config: testPulsarNamespaceBacklogQuotas(testWebServiceURL, cluster, tenant, namespace, true, false),
				Check:  testNamespaceBacklogQuotaTypes(fullNamespace, utils.DestinationStorage),
			},
			{
				Config:             testPulsarNamespaceBacklogQuotas(testWebServiceURL, cluster, tenant, namespace, true, false),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				Config: testPulsarNamespaceBacklogQuotas(testWebServiceURL, cluster, tenant, namespace, true, true),
				Check:  testNamespaceBacklogQuotaTypes(fullNamespace, utils.DestinationStorage, utils.MessageAge),
			},
			{
				Config: testPulsarNamespaceBacklogQuotas(testWebServiceURL, cluster, tenant, namespace, false, true),
				Check:  testNamespaceBacklogQuotaTypes(fullNamespace, utils.MessageAge),
			},
			{
				Config:             testPulsarNamespaceBacklogQuotas(testWebServiceURL, cluster, tenant, namespace, false, true),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func testNamespaceBacklogQuotaTypes(
	namespace string,
	expected ...utils.BacklogQuotaType,
) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		quotas, err := getClientFromMeta(testAccProvider.Meta()).Namespaces().GetBacklogQuotaMap(namespace)
		if err != nil {
			return fmt.Errorf("get backlog quota map for %q: %w", namespace, err)
		}

		if len(quotas) != len(expected) {
			return fmt.Errorf("backlog quota types for %q: got %#v, want %v", namespace, quotas, expected)
		}
		for _, quotaType := range expected {
			if _, ok := quotas[quotaType]; !ok {
				return fmt.Errorf("backlog quota type %q missing from %q: %#v", quotaType, namespace, quotas)
			}
		}
		return nil
	}
}

func testPulsarNamespaceBacklogQuotas(
	wsURL, cluster, tenant, namespace string,
	destinationStorage, messageAge bool,
) string {
	quotaBlocks := ""
	if destinationStorage {
		quotaBlocks += `
  backlog_quota {
    limit_bytes   = "10000000000"
    limit_seconds = "-1"
    policy        = "producer_request_hold"
    type          = "destination_storage"
  }
`
	}
	if messageAge {
		quotaBlocks += `
  backlog_quota {
    limit_bytes   = "-1"
    limit_seconds = "3600"
    policy        = "consumer_backlog_eviction"
    type          = "message_age"
  }
`
	}

	return testPulsarNamespaceBase(wsURL, cluster, tenant) + fmt.Sprintf(`
resource "pulsar_namespace" "test" {
  tenant    = pulsar_tenant.test_tenant.tenant
  namespace = %q
%s
}
`, namespace, quotaBlocks)
}
