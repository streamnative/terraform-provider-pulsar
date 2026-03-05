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

// TestAccPulsarNamespace_retentionPolicyRemoval verifies that removing retention_policies
// from namespace configuration removes namespace-level retention policy in Pulsar.
func TestAccPulsarNamespace_retentionPolicyRemoval(t *testing.T) {
	resourceName := "pulsar_namespace.test"
	clusterName := acctest.RandString(10)
	tenantName := acctest.RandString(10)
	namespaceName := acctest.RandString(10)
	fullNamespace := tenantName + "/" + namespaceName

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testPulsarNamespaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testNamespaceWithRetention(testWebServiceURL, clusterName, tenantName, namespaceName),
				Check: resource.ComposeTestCheckFunc(
					testPulsarNamespaceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "retention_policies.#", "1"),
					testNamespaceRetentionPolicy(fullNamespace, true, 1600, 10000),
				),
			},
			{
				Config: testNamespaceWithoutRetention(testWebServiceURL, clusterName, tenantName, namespaceName),
				Check: resource.ComposeTestCheckFunc(
					testPulsarNamespaceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "retention_policies.#", "0"),
					testNamespaceRetentionPolicy(fullNamespace, false, 0, 0),
				),
			},
			{
				Config:             testNamespaceWithoutRetention(testWebServiceURL, clusterName, tenantName, namespaceName),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func testNamespaceWithRetention(wsURL, cluster, tenant, namespace string) string {
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

  retention_policies {
    retention_minutes    = "1600"
    retention_size_in_mb = "10000"
  }
}
`, wsURL, cluster, tenant, namespace)
}

func testNamespaceWithoutRetention(wsURL, cluster, tenant, namespace string) string {
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
}
`, wsURL, cluster, tenant, namespace)
}

func testNamespaceRetentionPolicy(
	namespace string,
	shouldExist bool,
	expectedMinutes int,
	expectedSizeMB int64,
) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := getClientFromMeta(testAccProvider.Meta()).Namespaces()

		nsName, err := utils.GetNamespaceName(namespace)
		if err != nil {
			return fmt.Errorf("ERROR_PARSING_NAMESPACE: %w", err)
		}

		ret, err := client.GetRetention(nsName.String())
		if !shouldExist {
			if err != nil {
				if isIgnorableNotFoundError(err) {
					return nil
				}
				return fmt.Errorf("ERROR_GETTING_NAMESPACE_RETENTION: %w", err)
			}

			if ret == nil || (ret.RetentionTimeInMinutes == 0 && ret.RetentionSizeInMB == 0) {
				return nil
			}

			return fmt.Errorf(
				"expected namespace retention to be removed, but got time=%d size=%d",
				ret.RetentionTimeInMinutes,
				ret.RetentionSizeInMB,
			)
		}

		if err != nil {
			return fmt.Errorf("ERROR_GETTING_NAMESPACE_RETENTION: %w", err)
		}
		if ret == nil {
			return fmt.Errorf("expected namespace retention policy to exist, but got nil")
		}
		if ret.RetentionTimeInMinutes != expectedMinutes {
			return fmt.Errorf("expected retention_minutes=%d, got=%d", expectedMinutes, ret.RetentionTimeInMinutes)
		}
		if ret.RetentionSizeInMB != expectedSizeMB {
			return fmt.Errorf("expected retention_size_in_mb=%d, got=%d", expectedSizeMB, ret.RetentionSizeInMB)
		}

		return nil
	}
}
