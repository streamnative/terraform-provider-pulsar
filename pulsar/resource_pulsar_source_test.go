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
	"io/ioutil"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/pkg/errors"
	"github.com/streamnative/pulsar-admin-go/pkg/admin/config"
	"github.com/streamnative/pulsar-admin-go/pkg/rest"
	"github.com/streamnative/pulsar-admin-go/pkg/utils"
	"github.com/stretchr/testify/assert"

	"github.com/streamnative/terraform-provider-pulsar/bytesize"
)

func init() {
	initTestWebServiceURL()
}

var testdataSourceArchive = "https://www.apache.org/dyn/mirrors/mirrors.cgi" +
	"?action=download&filename=pulsar/pulsar-2.8.1/connectors/pulsar-io-file-2.8.1.nar"

func TestSource(t *testing.T) {
	configBytes, err := ioutil.ReadFile("testdata/source/main.tf")
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheckWithAPIVersion(t, config.V3) },
		ProviderFactories:         testAccProviderFactories,
		PreventPostDestroyRefresh: false,
		CheckDestroy:              testPulsarSourceDestroy,
		Steps: []resource.TestStep{
			{
				Config: string(configBytes),
				Check: resource.ComposeTestCheckFunc(func(s *terraform.State) error {
					name := "pulsar_source.source-1"
					rs, ok := s.RootModule().Resources[name]
					if !ok {
						return fmt.Errorf("%s not be found", name)
					}

					config, err := getPulsarSourceByResourceID(rs.Primary.ID)
					if err != nil {
						return err
					}

					if config == nil {
						return fmt.Errorf("failed to create %s source", rs.Primary.ID)
					}

					assert.Equal(t, "source-1", config.Name)
					assert.Equal(t, "public", config.Tenant)
					assert.Equal(t, "default", config.Namespace)
					// It always empty when config.Archive not built-in URL
					assert.Equal(t, "", config.Archive)
					assert.Equal(t, "source-1-topic", config.TopicName)
					assert.Equal(t, ProcessingGuaranteesEffectivelyOnce, config.ProcessingGuarantees)
					assert.Equal(t, 1, config.Parallelism)
					assert.NotNil(t, config.Configs)
					assert.NotNil(t, config.Resources)
					assert.Equal(t, 2, int(config.Resources.CPU))
					// 20GB
					assert.Equal(t, 20*1024*1024*1024, int(config.Resources.Disk))
					// 2GB
					assert.Equal(t, 2*1024*1024*1024, int(config.Resources.RAM))

					return nil
				}),
			},
		},
	})
}

func testPulsarSourceDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "pulsar_sink" {
			continue
		}
		config, err := getPulsarSourceByResourceID(rs.Primary.ID)
		if err != nil {
			return err
		}
		if config != nil {
			return errors.Errorf("%s still exists", rs.Primary.ID)
		}
	}

	return nil
}

func getPulsarSourceByResourceID(id string) (*utils.SourceConfig, error) {
	client := getClientFromMeta(testAccProvider.Meta()).Sources()

	parts := strings.Split(id, "/")
	if len(parts) != 3 {
		return nil, errors.New("Primary ID should be tenant/namespace/name format")
	}

	resp, err := client.GetSource(parts[0], parts[1], parts[2])
	if err != nil {
		if cliErr, ok := err.(rest.Error); ok && cliErr.Code == 404 {
			return nil, nil
		}
	}

	return &resp, err
}

func TestImportExistingSource(t *testing.T) {
	sourceName := acctest.RandString(6)
	err := createSampleSource(sourceName)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testPulsarSourceDestroy,
		Steps: []resource.TestStep{
			{
				ResourceName:     "pulsar_source.test",
				ImportState:      true,
				Config:           testSampleSource(sourceName),
				ImportStateId:    fmt.Sprintf("public/default/%s", sourceName),
				ImportStateCheck: testSourceImported(),
			},
		},
	})
}

func testSourceImported() resource.ImportStateCheckFunc {
	return func(s []*terraform.InstanceState) error {
		if len(s) != 1 {
			return fmt.Errorf("expected %d states, got %d: %#v", 1, len(s), s)
		}

		count := 16
		if len(s[0].Attributes) != count {
			return fmt.Errorf("expected %d attrs, got %d: %#v", count, len(s[0].Attributes), s[0].Attributes)
		}

		return nil
	}
}

func createSampleSource(name string) error {
	client, err := sharedClientWithVersion(testWebServiceURL, config.V3)
	if err != nil {
		return err
	}

	configsJSON := "{\"inputDirectory\":\"opt\"}"
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

	runtimeOptionsJSON := "{\"maxMessageRetries\": 10}"

	config := &utils.SourceConfig{
		Tenant:               "public",
		Namespace:            "default",
		Name:                 name,
		TopicName:            "source-1-topic",
		Parallelism:          1,
		Archive:              testdataSourceArchive,
		ProcessingGuarantees: ProcessingGuaranteesEffectivelyOnce,
		Configs:              configs,
		Resources: &utils.Resources{
			CPU:  2,
			Disk: int64(bytesize.FormMegaBytes(20480).ToBytes()),
			RAM:  int64(bytesize.FormMegaBytes(2048).ToBytes()),
		},
		Secrets:              secret,
		SchemaType:           "JSON",
		CustomRuntimeOptions: runtimeOptionsJSON,
	}

	return client.Sources().CreateSourceWithURL(config, config.Archive)
}

func testSampleSource(name string) string {
	//nolint
	return fmt.Sprintf(`
provider "pulsar" {
  web_service_url = "%s"
  api_version = "3"
}

resource "pulsar_source" "test" {
  provider = pulsar

  name = "%s"
  tenant = "public"
  namespace = "default"

  archive = "%s"

  destination_topic_name = "source-1-topic"

  processing_guarantees = "EFFECTIVELY_ONCE"

  configs = "{\"inputDirectory\":\"opt\"}"

  secrets ="{\"SECRET1\": {\"path\": \"sectest\", \"key\": \"hello\"}}"
  schema_type = "JSON"
  custom_runtime_options = "{\"maxMessageRetries\": 10}"

  cpu = 2
  disk_mb = 20480
  ram_mb = 2048
}
`, testWebServiceURL, name, testdataSourceArchive)
}
