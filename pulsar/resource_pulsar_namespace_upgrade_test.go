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
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

var pulsarV011ExternalProvider = map[string]resource.ExternalProvider{
	"pulsar": {
		Source:            "registry.terraform.io/streamnative/pulsar",
		VersionConstraint: "= 0.11.0",
	},
}

func TestAccPulsarNamespace_UpgradeFromV011WithManagedPolicyBlocks(t *testing.T) {
	cluster := acctest.RandString(10)
	tenant := acctest.RandString(10)
	namespace := acctest.RandString(10)
	config := testPulsarNamespacePolicyBlocks(testWebServiceURL, cluster, tenant, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testPulsarNamespaceDestroy,
		Steps: []resource.TestStep{
			{
				Config:            config,
				ExternalProviders: pulsarV011ExternalProvider,
			},
			{
				Config:             config,
				ProviderFactories:  testAccProviderFactories,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccPulsarNamespace_UpgradeFromV011WithOmittedPolicyBlocks(t *testing.T) {
	cluster := acctest.RandString(10)
	tenant := acctest.RandString(10)
	namespace := acctest.RandString(10)
	config := testPulsarNamespaceNoPolicyBlocks(testWebServiceURL, cluster, tenant, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testPulsarNamespaceDestroy,
		Steps: []resource.TestStep{
			{
				Config:            config,
				ExternalProviders: pulsarV011ExternalProvider,
			},
			{
				Config:             config,
				ProviderFactories:  testAccProviderFactories,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}
