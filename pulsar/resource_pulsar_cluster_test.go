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

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/streamnative/pulsarctl/pkg/pulsar"
)

func TestCluster(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t) },
		Providers:                 testAccProviders,
		PreventPostDestroyRefresh: false,
		CheckDestroy:              testPulsarClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarCluster,
				Check:  resource.ComposeTestCheckFunc(testPulsarClusterExists("pulsar_cluster.test")),
			},
		},
	})

}

func testPulsarClusterExists(cluster string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[cluster]
		if !ok {
			return fmt.Errorf("NOT_FOUND: %s", cluster)
		}

		client := testAccProvider.Meta().(pulsar.Client).Clusters()

		_, err := client.Get(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("ERROR_READ_CLUSTER_DATA: %w", err)
		}

		return nil
	}
}

func testPulsarClusterDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(pulsar.Client).Clusters()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "pulsar_cluster" {
			continue
		}

		// cluster name is set as ID for the resource
		cluster := rs.Primary.ID

		_, err := client.Get(cluster)
		if err != nil {
			return nil
		}

		return fmt.Errorf("ERROR_RESOURCE_CLUSTER_STILL_EXISITS: %w", err)
	}

	return nil
}

var (
	testPulsarCluster = fmt.Sprintf(`
provider "pulsar" {
  web_service_url = "%s"
}

resource "pulsar_cluster" "test" {
  cluster = "eternals"

  cluster_data {
    web_service_url    = "http://localhost:8080"
    broker_service_url = "http://localhost:6050"
    peer_clusters      = ["skrulls", "krees"]
  }
}`, webServiceURL)
)
