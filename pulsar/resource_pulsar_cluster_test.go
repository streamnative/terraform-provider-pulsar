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
	"regexp"
	"strings"
	"testing"

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/admin"
	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func init() {
	initTestWebServiceURL()
}

func TestCluster(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t) },
		ProviderFactories:         testAccProviderFactories,
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

func TestHandleExistingCluster(t *testing.T) {
	cName := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			createCluster(t, cName)
			t.Cleanup(func() {
				if err := getClientFromMeta(testAccProvider.Meta()).Clusters().Delete(cName); err != nil {
					t.Fatalf("ERROR_DELETING_TEST_CLUSTER: %v", err)
				}
			})
		},
		CheckDestroy:      testPulsarClusterDestroy,
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testPulsarExistingCluster(testWebServiceURL, cName),
				ExpectError: regexp.MustCompile("Cluster already exists"),
			},
		},
	})
}

func TestImportExistingCluster(t *testing.T) {
	cName := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			createCluster(t, cName)
			t.Cleanup(func() {
				if err := getClientFromMeta(testAccProvider.Meta()).Clusters().Delete(cName); err != nil {
					t.Fatalf("ERROR_DELETING_TEST_CLUSTER: %v", err)
				}
			})
		},
		CheckDestroy:      testPulsarClusterDestroy,
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				ResourceName:     "pulsar_cluster.test",
				ImportState:      true,
				Config:           testPulsarExistingCluster(testWebServiceURL, cName),
				ImportStateId:    cName,
				ImportStateCheck: testClusterImported(),
			},
		},
	})
}

func createCluster(t *testing.T, cname string) {
	client, err := sharedClient(testWebServiceURL)
	if err != nil {
		t.Fatalf("ERROR_GETTING_PULSAR_CLIENT: %v", err)
	}

	conn := client.(admin.Client)
	if err = conn.Clusters().Create(utils.ClusterData{
		Name:             cname,
		ServiceURL:       "http://localhost:8080",
		BrokerServiceURL: "http://localhost:6050",
		PeerClusterNames: []string{"standalone"},
	}); err != nil {
		t.Log("createCluster", err)
		if strings.Contains(err.Error(), "already exists") {
			return
		}
		t.Fatalf("ERROR_CREATING_TEST_CLUSTER: %v", err)
	}
}

func testClusterImported() resource.ImportStateCheckFunc {
	return func(s []*terraform.InstanceState) error {
		if len(s) != 1 {
			return fmt.Errorf("expected %d states, got %d: %#v", 1, len(s), s)
		}

		if len(s[0].Attributes) != 11 {
			return fmt.Errorf("expected %d attrs, got %d: %#v", 11, len(s[0].Attributes), s[0].Attributes)
		}

		return nil
	}
}

func testPulsarClusterExists(cluster string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[cluster]
		if !ok {
			return fmt.Errorf("NOT_FOUND: %s", cluster)
		}

		client := getClientFromMeta(testAccProvider.Meta()).Clusters()

		_, err := client.Get(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("ERROR_READ_CLUSTER_DATA: %w", err)
		}

		return nil
	}
}

func testPulsarClusterDestroy(s *terraform.State) error {
	client := getClientFromMeta(testAccProvider.Meta()).Clusters()

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

		return fmt.Errorf("ERROR_RESOURCE_CLUSTER_STILL_EXISTS: %w", err)
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
    web_service_url_tls    = "http://localhost:8443"
    broker_service_url = "http://localhost:6050"
    broker_service_url_tls = "pulsar+ssl://localhost:6051"
    peer_clusters      = ["skrulls", "krees"]
  }
}`, testWebServiceURL)
)

func testPulsarExistingCluster(url, cname string) string {
	return fmt.Sprintf(`
provider "pulsar" {
  web_service_url = "%s"
}

resource "pulsar_cluster" "test" {
  cluster = "%s"

  cluster_data {
    web_service_url    = "http://localhost:8080"
    broker_service_url = "http://localhost:6050"
	peer_clusters      = ["standalone"]
  }
}`, url, cname)
}
