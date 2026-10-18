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

func TestAccPulsarNamespace_DelayedDelivery(t *testing.T) {
	namespace := "delayed-" + acctest.RandString(10)
	fullNamespace := "public/" + namespace
	resourceName := "pulsar_namespace.test"
	base := fmt.Sprintf(`
provider "pulsar" {
  web_service_url = %q
}
resource "pulsar_namespace" "test" {
  tenant = "public"
  namespace = %q
  %%s
}
`, testWebServiceURL, namespace)
	config := func(block string) string { return fmt.Sprintf(base, block) }
	initial := config(`delayed_delivery {
  enabled = true
  tick_duration_ms = 100
}`)
	disabled := config(`delayed_delivery {
  enabled = false
  tick_duration_ms = 200
  max_delivery_delay_ms = 60000
}`)
	check := func(want *utils.DelayedDeliveryData) resource.TestCheckFunc {
		return func(s *terraform.State) error {
			if s.RootModule().Resources[resourceName].Primary.ID != fullNamespace {
				return fmt.Errorf("namespace ID changed")
			}
			got, err := getClientFromMeta(testAccProvider.Meta()).Namespaces().GetDelayedDelivery(fullNamespace)
			if err != nil {
				return err
			}
			if want == nil && got == nil {
				return nil
			}
			if want == nil || got == nil || *want != *got {
				return fmt.Errorf("delayed delivery: got %+v, want %+v", got, want)
			}
			return nil
		}
	}
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testPulsarNamespaceDestroy,
		Steps: []resource.TestStep{
			{Config: initial, Check: check(utils.NewDelayedDeliveryData(100, true))},
			{Config: initial, PlanOnly: true, ExpectNonEmptyPlan: false},
			{
				ResourceName: resourceName, ImportState: true, ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					backlogQuotaManagedTypesStateAttr, "persistence_policies", "dispatch_rate", "subscription_dispatch_rate",
				},
			},
			{Config: disabled, Check: check(utils.NewDelayedDeliveryDataWithMaxDelay(200, false, 60000))},
			{Config: disabled, PlanOnly: true, ExpectNonEmptyPlan: false},
			{
				Config: disabled,
				PreConfig: func() {
					if err := getClientFromMeta(testAccProvider.Meta()).Namespaces().RemoveDelayedDelivery(fullNamespace); err != nil {
						t.Fatal(err)
					}
				},
				Check: check(utils.NewDelayedDeliveryDataWithMaxDelay(200, false, 60000)),
			},
			{Config: initial, Check: check(utils.NewDelayedDeliveryData(100, true))},
			{Config: config(""), Check: check(nil)},
			{Config: config(""), PlanOnly: true, ExpectNonEmptyPlan: false},
		},
	})
}
