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

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func init() {
	initTestWebServiceURL()
}

func TestPulsarSchema(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t) },
		ProviderFactories:         testAccProviderFactories,
		PreventPostDestroyRefresh: false,
		CheckDestroy:              testPulsarSchemaDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarSchemaString,
				Check: resource.ComposeTestCheckFunc(
					testPulsarSchemaExists("pulsar_schema.test"),
					resource.TestCheckResourceAttr("pulsar_schema.test", "tenant", "public"),
					resource.TestCheckResourceAttr("pulsar_schema.test", "namespace", "default"),
					resource.TestCheckResourceAttr("pulsar_schema.test", "topic", "test-string-schema"),
					resource.TestCheckResourceAttr("pulsar_schema.test", "type", "STRING"),
					resource.TestCheckResourceAttr("pulsar_schema.test", "schema_data", ""),
					resource.TestCheckResourceAttrSet("pulsar_schema.test", "version"),
					resource.TestCheckResourceAttrSet("pulsar_schema.test", "timestamp"),
				),
			},
		},
	})
}

func TestPulsarSchemaAvro(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t) },
		ProviderFactories:         testAccProviderFactories,
		PreventPostDestroyRefresh: false,
		CheckDestroy:              testPulsarSchemaDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarSchemaAvro,
				Check: resource.ComposeTestCheckFunc(
					testPulsarSchemaExists("pulsar_schema.test_avro"),
					resource.TestCheckResourceAttr("pulsar_schema.test_avro", "tenant", "public"),
					resource.TestCheckResourceAttr("pulsar_schema.test_avro", "namespace", "default"),
					resource.TestCheckResourceAttr("pulsar_schema.test_avro", "topic", "test-avro-schema"),
					resource.TestCheckResourceAttr("pulsar_schema.test_avro", "type", "AVRO"),
					resource.TestCheckResourceAttrSet("pulsar_schema.test_avro", "schema_data"),
					resource.TestCheckResourceAttr("pulsar_schema.test_avro", "properties.author", "terraform-test"),
					resource.TestCheckResourceAttrSet("pulsar_schema.test_avro", "version"),
					resource.TestCheckResourceAttrSet("pulsar_schema.test_avro", "timestamp"),
				),
			},
		},
	})
}

func TestPulsarSchemaJSON(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t) },
		ProviderFactories:         testAccProviderFactories,
		PreventPostDestroyRefresh: false,
		CheckDestroy:              testPulsarSchemaDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarSchemaJSON,
				Check: resource.ComposeTestCheckFunc(
					testPulsarSchemaExists("pulsar_schema.test_json"),
					resource.TestCheckResourceAttr("pulsar_schema.test_json", "tenant", "public"),
					resource.TestCheckResourceAttr("pulsar_schema.test_json", "namespace", "default"),
					resource.TestCheckResourceAttr("pulsar_schema.test_json", "topic", "test-json-schema"),
					resource.TestCheckResourceAttr("pulsar_schema.test_json", "type", "JSON"),
					resource.TestCheckResourceAttrSet("pulsar_schema.test_json", "schema_data"),
					resource.TestCheckResourceAttrSet("pulsar_schema.test_json", "version"),
					resource.TestCheckResourceAttrSet("pulsar_schema.test_json", "timestamp"),
				),
			},
		},
	})
}

func TestPulsarSchemaUpdate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t) },
		ProviderFactories:         testAccProviderFactories,
		PreventPostDestroyRefresh: false,
		CheckDestroy:              testPulsarSchemaDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarSchemaAvro,
				Check: resource.ComposeTestCheckFunc(
					testPulsarSchemaExists("pulsar_schema.test_avro"),
					resource.TestCheckResourceAttr("pulsar_schema.test_avro", "type", "AVRO"),
					resource.TestCheckResourceAttr("pulsar_schema.test_avro", "properties.author", "terraform-test"),
				),
			},
			{
				Config: testPulsarSchemaAvroUpdated,
				Check: resource.ComposeTestCheckFunc(
					testPulsarSchemaExists("pulsar_schema.test_avro"),
					resource.TestCheckResourceAttr("pulsar_schema.test_avro", "type", "AVRO"),
					resource.TestCheckResourceAttr("pulsar_schema.test_avro", "properties.author", "terraform-test-updated"),
					resource.TestCheckResourceAttr("pulsar_schema.test_avro", "properties.version", "2.0"),
				),
			},
		},
	})
}

func TestPulsarSchemaImport(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t) },
		ProviderFactories:         testAccProviderFactories,
		PreventPostDestroyRefresh: false,
		CheckDestroy:              testPulsarSchemaDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarSchemaString,
			},
			{
				ResourceName:      "pulsar_schema.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     "public/default/test-string-schema",
			},
		},
	})
}

func testPulsarSchemaExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("NOT_FOUND: %s", resourceName)
		}

		// Extract tenant, namespace, topic from ID
		parts := strings.Split(rs.Primary.ID, "/")
		if len(parts) != 3 {
			return fmt.Errorf("invalid resource ID format: %s", rs.Primary.ID)
		}

		client := getClientFromMeta(testAccProvider.Meta()).Schemas()
		topicName, err := utils.GetTopicName(fmt.Sprintf("persistent://%s/%s/%s", parts[0], parts[1], parts[2]))
		if err != nil {
			return fmt.Errorf("ERROR_PARSE_TOPIC_NAME: %w", err)
		}

		_, err = client.GetSchemaInfo(topicName.String())
		if err != nil {
			return fmt.Errorf("ERROR_READ_SCHEMA: %w", err)
		}

		return nil
	}
}

func testPulsarSchemaDestroy(s *terraform.State) error {
	client := getClientFromMeta(testAccProvider.Meta()).Schemas()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "pulsar_schema" {
			continue
		}

		// Extract tenant, namespace, topic from ID
		parts := strings.Split(rs.Primary.ID, "/")
		if len(parts) != 3 {
			continue
		}

		topicName, err := utils.GetTopicName(fmt.Sprintf("persistent://%s/%s/%s", parts[0], parts[1], parts[2]))
		if err != nil {
			continue
		}

		_, err = client.GetSchemaInfo(topicName.String())
		if err == nil {
			return fmt.Errorf("ERROR_RESOURCE_STILL_EXISTS: schema %s still exists", rs.Primary.ID)
		}
	}

	return nil
}

// Test configurations

var testPulsarSchemaString = fmt.Sprintf(`
provider "pulsar" {
  web_service_url = "%s"
}

resource "pulsar_schema" "test" {
  tenant    = "public"
  namespace = "default"
  topic     = "test-string-schema"
  type      = "STRING"
}
`, testWebServiceURL)

var testPulsarSchemaAvro = fmt.Sprintf(`
provider "pulsar" {
  web_service_url = "%s"
}

resource "pulsar_schema" "test_avro" {
  tenant      = "public"
  namespace   = "default"
  topic       = "test-avro-schema"
  type        = "AVRO"
  schema_data = jsonencode({
    "type": "record",
    "name": "User",
    "fields": [
      {
        "name": "name",
        "type": "string"
      },
      {
        "name": "age", 
        "type": "int"
      }
    ]
  })
  
  properties = {
    "author" = "terraform-test"
  }
}
`, testWebServiceURL)

var testPulsarSchemaAvroUpdated = fmt.Sprintf(`
provider "pulsar" {
  web_service_url = "%s"
}

resource "pulsar_schema" "test_avro" {
  tenant      = "public"
  namespace   = "default"
  topic       = "test-avro-schema"
  type        = "AVRO"
  schema_data = jsonencode({
    "type": "record",
    "name": "User",
    "fields": [
      {
        "name": "name",
        "type": "string"
      },
      {
        "name": "age", 
        "type": "int"
      },
      {
        "name": "email",
        "type": ["null", "string"],
        "default": null
      }
    ]
  })
  
  properties = {
    "author"  = "terraform-test-updated"
    "version" = "2.0"
  }
}
`, testWebServiceURL)

var testPulsarSchemaJSON = fmt.Sprintf(`
provider "pulsar" {
  web_service_url = "%s"
}

resource "pulsar_schema" "test_json" {
  tenant      = "public"
  namespace   = "default"
  topic       = "test-json-schema"
  type        = "JSON"
  schema_data = jsonencode({
    "type": "object",
    "properties": {
      "name": {
        "type": "string"
      },
      "age": {
        "type": "integer",
        "minimum": 0
      }
    },
    "required": ["name", "age"]
  })
}
`, testWebServiceURL)
