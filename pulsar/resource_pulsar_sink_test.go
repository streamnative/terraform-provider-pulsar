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
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/admin/config"
	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/rest"
	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/pkg/errors"

	"github.com/streamnative/terraform-provider-pulsar/bytesize"
)

var testdataArchive = "https://www.apache.org/dyn/mirrors/mirrors.cgi?" +
	"action=download&filename=pulsar/pulsar-2.10.4/connectors/pulsar-io-jdbc-postgres-2.10.4.nar"

func init() {
	initTestWebServiceURL()
}

func TestSink(t *testing.T) {
	configBytes, err := os.ReadFile("testdata/sink/main.tf")
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t) },
		ProviderFactories:         testAccProviderFactories,
		PreventPostDestroyRefresh: false,
		CheckDestroy:              testPulsarSinkDestroy,
		Steps: []resource.TestStep{
			{
				Config: string(configBytes),
				Check: resource.ComposeTestCheckFunc(func(s *terraform.State) error {
					name := "pulsar_sink.sink-1"
					rs, ok := s.RootModule().Resources[name]
					if !ok {
						return fmt.Errorf("%s not be found", name)
					}

					client := getV3ClientFromMeta(testAccProvider.Meta()).Sinks()

					parts := strings.Split(rs.Primary.ID, "/")
					if len(parts) != 3 {
						return errors.New("resource id should be tenant/namespace/name format")
					}

					_, err := client.GetSink(parts[0], parts[1], parts[2])
					if err != nil {
						return err
					}

					return nil
				}),
			},
		},
	})
}

func testPulsarSinkDestroy(s *terraform.State) error {
	client := getV3ClientFromMeta(testAccProvider.Meta()).Sinks()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "pulsar_sink" {
			continue
		}

		id := rs.Primary.ID
		parts := strings.Split(id, "/")
		if len(parts) != 3 {
			return errors.New("id should be tenant/namespace/name format")
		}

		resp, err := client.GetSink(parts[0], parts[1], parts[2])
		if err != nil {
			if cliErr, ok := err.(rest.Error); ok && cliErr.Code == 404 {
				return nil
			}

			return err
		}

		if resp.Name != "" {
			return fmt.Errorf("%s still exist", id)
		}
	}

	return nil
}

func TestImportExistingSink(t *testing.T) {
	sinkName := acctest.RandString(6)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			createSampleSink(sinkName)
			t.Cleanup(func() {
				if err := getClientFromMeta(testAccProvider.Meta()).Sinks().DeleteSink(
					"public",
					"default",
					sinkName,
				); err != nil {
					if cliErr, ok := err.(rest.Error); ok && cliErr.Code == 404 {
						return
					}
					t.Fatalf("ERROR_DELETING_TEST_SINK: %v", err)
				}
			})
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testPulsarSinkDestroy,
		Steps: []resource.TestStep{
			{
				ResourceName:     "pulsar_sink.test",
				ImportState:      true,
				Config:           testSampleSink(sinkName),
				ImportStateId:    fmt.Sprintf("public/default/%s", sinkName),
				ImportStateCheck: testSinkImported(),
			},
		},
	})
}

func testSinkImported() resource.ImportStateCheckFunc {
	return func(s []*terraform.InstanceState) error {
		if len(s) != 1 {
			return fmt.Errorf("expected %d states, got %d: %#v", 1, len(s), s)
		}

		if len(s[0].Attributes) != 30 {
			return fmt.Errorf("expected %d attrs, got %d: %#v", 30, len(s[0].Attributes), s[0].Attributes)
		}

		return nil
	}
}

func createSampleSink(name string) error {
	client, err := sharedClientWithVersion(testWebServiceURL, config.V3)
	if err != nil {
		return err
	}

	configsJSON := "{\"jdbcUrl\":\"jdbc:postgresql://localhost:5432/pulsar_postgres_jdbc_sink\"," +
		"\"password\":\"password\",\"tableName\":\"pulsar_postgres_jdbc_sink\",\"userName\":\"postgres\"}"
	configs := make(map[string]interface{})
	err = json.Unmarshal([]byte(configsJSON), &configs)
	if err != nil {
		return err
	}

	secretJSON := "{\"secret1\": {\"path\":\"sectest\",\"key\":\"hello\"}}"
	secret := make(map[string]interface{})
	err = json.Unmarshal([]byte(secretJSON), &secret)
	if err != nil {
		return err
	}

	config := &utils.SinkConfig{
		CleanupSubscription:        false,
		RetainOrdering:             true,
		AutoAck:                    true,
		Parallelism:                1,
		Tenant:                     "public",
		Namespace:                  "default",
		Name:                       name,
		Archive:                    testdataArchive,
		ProcessingGuarantees:       "EFFECTIVELY_ONCE",
		SourceSubscriptionPosition: "Latest",
		Inputs:                     []string{"sink-1-topic"},
		Configs:                    configs,
		Resources: &utils.Resources{
			CPU:  1,
			Disk: int64(bytesize.FormMegaBytes(102400).ToBytes()),
			RAM:  int64(bytesize.FormMegaBytes(2048).ToBytes()),
		},
		Secrets:                      secret,
		DeadLetterTopic:              "dl-topic",
		MaxMessageRetries:            5,
		NegativeAckRedeliveryDelayMs: 3000,
		RetainKeyOrdering:            false,
	}

	return client.Sinks().CreateSinkWithURL(config, config.Archive)
}

func testSampleSink(name string) string {
	//nolint
	return fmt.Sprintf(`
provider "pulsar" {
  web_service_url = "http://localhost:8080"
}

resource "pulsar_sink" "test" {
  provider = "pulsar"

  name = "%s"
  tenant = "public"
  namespace = "default"
  inputs = ["sink-1-topic"]
  subscription_position = "Latest"
  cleanup_subscription = false
  parallelism = 1
  auto_ack = true

  dead_letter_topic = "dl-topic"
  max_redeliver_count = 5
  negative_ack_redelivery_delay_ms = 3000
  retain_key_ordering = false 
	retain_ordering = true
  secrets ="{\"SECRET1\": {\"path\": \"sectest\", \"key\": \"hello\"}}"

  processing_guarantees = "EFFECTIVELY_ONCE"

  cpu = 1
  ram_mb = 2048
  disk_mb = 102400

  archive = "%s"
  configs = "{\"jdbcUrl\":\"jdbc:postgresql://localhost:5432/pulsar_postgres_jdbc_sink\",\"password\":\"password\",\"tableName\":\"pulsar_postgres_jdbc_sink\",\"userName\":\"postgres\"}"
}
`, name, testdataArchive)
}

func TestSinkUpdate(t *testing.T) {
	configBytes, err := os.ReadFile("testdata/sink/main.tf")
	if err != nil {
		t.Fatal(err)
	}
	configString := string(configBytes)
	newName := "sink" + acctest.RandString(10)
	configString = strings.ReplaceAll(configString, "sink-1", newName)

	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t) },
		ProviderFactories:         testAccProviderFactories,
		PreventPostDestroyRefresh: false,
		CheckDestroy:              testPulsarSinkDestroy,
		Steps: []resource.TestStep{
			{
				Config: configString,
				Check: resource.ComposeTestCheckFunc(func(s *terraform.State) error {
					name := "pulsar_sink." + newName
					rs, ok := s.RootModule().Resources[name]
					if !ok {
						return fmt.Errorf("%s not be found", name)
					}

					client := getV3ClientFromMeta(testAccProvider.Meta()).Sinks()

					parts := strings.Split(rs.Primary.ID, "/")
					if len(parts) != 3 {
						return errors.New("resource id should be tenant/namespace/name format")
					}

					_, err := client.GetSink(parts[0], parts[1], parts[2])
					if err != nil {
						return err
					}

					return nil
				}),
			},
			{
				Config:             configString,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}
