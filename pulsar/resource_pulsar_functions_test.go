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
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func init() {
	url, ok := os.LookupEnv("WEB_SERVICE_URL")
	if !ok {
		webServiceURL = "http://localhost:8080"
	}

	webServiceURL = url
}

func TestFunction(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testPulsarFunctionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarFunction,
				Check:  resource.ComposeTestCheckFunc(testPulsarFunctionExists("pulsar_function.test")),
			},
		},
	})

}

func testPulsarFunctionExists(fn string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[fn]
		if !ok {
			return fmt.Errorf("NOT_FOUND: %s", fn)
		}

		client := testAccProvider.Meta().(tfPulsarClient).Functions()

		tenant, namespace, name, err := parseFQFN(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("ERROR_PARSE_FQFN: %w", err)
		}
		_, err = client.GetFunction(tenant, namespace, name)
		if err != nil {
			return fmt.Errorf("ERROR_READ_PULSAR_FUNCTION_DATA: %w", err)
		}

		return nil
	}
}

func testPulsarFunctionDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(tfPulsarClient).Functions()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "pulsar_function" {
			continue
		}

		// function's FQFN is set as ID for the resource
		tenant, namespace, name, err := parseFQFN(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("ERROR_PARSE_FQFN: %s", err)
		}

		resp, err := client.GetFunction(tenant, namespace, name)
		if err != nil {
			return nil
		}

		if resp.Name != "" {
			return fmt.Errorf("ERROR_RESOURCE_FUNCTION_STILL_EXISITS: %s", resp.Name)
		}

	}

	return nil
}

var (
	testPulsarFunction = fmt.Sprintf(`
provider "pulsar" {
  web_service_url = "%s"
}

resource "pulsar_cluster" "my_cluster" {
  cluster = "test-cluster"

  cluster_data {
    web_service_url    = "http://localhost:8080"
    broker_service_url = "http://localhost:6050"
    peer_clusters      = ["standalone"]
  }

}

resource "pulsar_tenant" "my_tenant" {
  tenant           = "test-tenant"
  allowed_clusters = [pulsar_cluster.my_cluster.cluster, "standalone"]
}

resource "pulsar_namespace" "my_ns" {
  tenant    = pulsar_tenant.my_tenant.tenant
  namespace = "test-namespace"

  namespace_config {
    anti_affinity                  = "anti-aff"
    max_consumers_per_subscription = "50"
    max_consumers_per_topic        = "50"
    max_producers_per_topic        = "50"
    replication_clusters           = ["standalone"]
  }

  dispatch_rate {
    dispatch_msg_throttling_rate  = 50
    rate_period_seconds           = 50
    dispatch_byte_throttling_rate = 2048
  }

  retention_policies {
    retention_minutes    = "1600"
    retention_size_in_mb = "10000"
  }

}

resource "pulsar_function" "test" {
  tenant     = pulsar_tenant.my_tenant.tenant
  namespace  = pulsar_namespace.my_ns.namespace
  name       = "exclamit"
  class_name = "org.apache.pulsar.functions.api.examples.JavaNativeExclamationFunction"
  jar_file   = abspath("../bin/exclam.jar")
  inputs     = ["test_input"]
  output     = "test_output"
}
`, webServiceURL)
)

func parseFQFN(fqfn string) (string, string, string, error) {
	parts := strings.Split(fqfn, "/")

	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("ERROR_INVALID_FQFN: %s", fqfn)
	}

	// fqfn syntax -> tenant/namespace/name
	return parts[0], parts[1], parts[2], nil
}
