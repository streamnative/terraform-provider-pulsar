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

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/streamnative/pulsarctl/pkg/pulsar"
	"github.com/streamnative/pulsarctl/pkg/pulsar/utils"
)

func init() {
	initTestWebServiceURL()
}

func TestTopic(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testPulsarTopicDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPulsarPartitionTopic,
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists("pulsar_topic.sample-topic-1"),
					testPulsarTopicExists("pulsar_topic.sample-topic-2"),
				),
			},
			{
				Config: testPulsarNonPartitionTopic,
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists("pulsar_topic.sample-topic-3"),
					testPulsarTopicExists("pulsar_topic.sample-topic-4"),
				),
			},
		},
	})
}

func TestImportExistingTopic(t *testing.T) {
	tname := acctest.RandString(10)
	ttype := "persistent"
	pnum := 10

	fullID := strings.Join([]string{ttype + ":/", "public", "default", tname}, "/")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			createTopic(t, fullID, pnum)
		},
		Providers:    testAccProviders,
		CheckDestroy: testPulsarTopicDestroy,
		Steps: []resource.TestStep{
			{
				ResourceName:     "pulsar_topic.test",
				ImportState:      true,
				Config:           testPulsarExistingTopicConfig(testWebServiceURL, tname, ttype, pnum),
				ImportStateId:    fullID,
				ImportStateCheck: testTopicImported(),
			},
		},
	})
}

func testPulsarTopicDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(pulsar.Client).Topics()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "pulsar_topic" {
			continue
		}

		topicName, err := utils.GetTopicName(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("ERROR_READ_TOPIC: %w", err)
		}
		namespace, err := utils.GetNameSpaceName(topicName.GetTenant(), topicName.GetNamespace())
		if err != nil {
			return fmt.Errorf("ERROR_READ_NAMESPACE: %w", err)
		}

		partitionedTopics, nonPartitionedTopics, err := client.List(*namespace)
		if err != nil {
			return fmt.Errorf("ERROR_READ_TOPIC_DATA: %w", err)
		}

		for _, topic := range append(partitionedTopics, nonPartitionedTopics...) {
			if rs.Primary.ID == topic {
				return fmt.Errorf("ERROR_RESOURCE_TOPIC_STILL_EXISTS: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testPulsarTopicExists(topic string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[topic]
		if !ok {
			return fmt.Errorf("NOT_FOUND: %s", topic)
		}

		topicName, err := utils.GetTopicName(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("ERROR_READ_TOPIC: %w", err)
		}
		namespace, err := utils.GetNameSpaceName(topicName.GetTenant(), topicName.GetNamespace())
		if err != nil {
			return fmt.Errorf("ERROR_READ_NAMESPACE: %w", err)
		}

		client := testAccProvider.Meta().(pulsar.Client).Topics()
		partitionedTopics, nonPartitionedTopics, err := client.List(*namespace)
		if err != nil {
			return fmt.Errorf("ERROR_READ_TOPIC_DATA: %w", err)
		}

		for _, topic := range append(partitionedTopics, nonPartitionedTopics...) {
			if rs.Primary.ID == topic {
				return nil
			}
		}

		return fmt.Errorf("ERROR_RESOURCE_TOPIC_DOES_NOT_EXISTS")
	}
}

func testTopicImported() resource.ImportStateCheckFunc {
	return func(s []*terraform.InstanceState) error {
		if len(s) != 1 {
			return fmt.Errorf("expected %d states, got %d: %#v", 1, len(s), s)
		}

		if len(s[0].Attributes) != 6 {
			return fmt.Errorf("expected %d attrs, got %d: %#v", 6, len(s[0].Attributes), s[0].Attributes)
		}

		return nil
	}
}

func createTopic(t *testing.T, fullID string, pnum int) {
	client, err := sharedClient(testWebServiceURL)
	if err != nil {
		t.Fatalf("ERROR_GETTING_PULSAR_CLIENT: %v", err)
	}

	conn := client.(pulsar.Client)
	tname, _ := utils.GetTopicName(fullID)

	if err = conn.Topics().Create(*tname, pnum); err != nil {
		t.Fatalf("ERROR_CREATING_TEST_TOPIC: %v", err)
	}
}

var (
	testPulsarPartitionTopic = fmt.Sprintf(`
provider "pulsar" {
  web_service_url = "%s"
}

resource "pulsar_topic" "sample-topic-1" {
  tenant     = "public"
  namespace  = "default"
  topic_type = "persistent"
  topic_name = "partitioned-persistent-topic"
  partitions = 4
}

resource "pulsar_topic" "sample-topic-2" {
  tenant     = "public"
  namespace  = "default"
  topic_type = "non-persistent"
  topic_name = "partitioned-non-persistent-topic"
  partitions = 4
}
`, testWebServiceURL)

	testPulsarNonPartitionTopic = fmt.Sprintf(`
provider "pulsar" {
  web_service_url = "%s"
}

resource "pulsar_topic" "sample-topic-3" {
  tenant     = "public"
  namespace  = "default"
  topic_type = "persistent"
  topic_name = "non-partitioned-persistent-topic"
  partitions = 0
}

resource "pulsar_topic" "sample-topic-4" {
  tenant     = "public"
  namespace  = "default"
  topic_type = "non-persistent"
  topic_name = "non-partitioned-non-persistent-topic"
  partitions = 0
}


`, testWebServiceURL)
)

func testPulsarExistingTopicConfig(url, tname, ttype string, pnum int) string {
	return fmt.Sprintf(`
provider "pulsar" {
	web_service_url = "%s"
}

resource "pulsar_topic" "test" {
  tenant     = "public"
  namespace  = "default"
  topic_type = "%s"
  topic_name = "%s"
  partitions = %d
}
`, url, ttype, tname, pnum)
}
