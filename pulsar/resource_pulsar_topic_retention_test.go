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

// TestAccPulsarTopic_retentionPolicyRemoval verifies that removing the retention_policies block
// from a topic configuration actually removes the retention policy from Pulsar.
func TestAccPulsarTopic_retentionPolicyRemoval(t *testing.T) {
	skipIfNoTopicPolicies(t)
	resourceName := "pulsar_topic.test"
	tname := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testPulsarTopicDestroy,
		Steps: []resource.TestStep{
			{
				// Step 1: Create topic with retention_policies
				Config: testTopicWithRetention(testWebServiceURL, tname, 1600, 20000),
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists(resourceName, t),
					resource.TestCheckResourceAttr(resourceName, "retention_policies.#", "1"),
					testTopicRetentionPolicy(resourceName, true, 1600, 20000),
				),
			},
			{
				// Step 2: Remove retention_policies block
				Config: testTopicWithoutRetention(testWebServiceURL, tname),
				Check: resource.ComposeTestCheckFunc(
					testPulsarTopicExists(resourceName, t),
					resource.TestCheckResourceAttr(resourceName, "retention_policies.#", "0"),
					testTopicRetentionPolicy(resourceName, false, 0, 0),
				),
			},
		},
	})
}

func testTopicWithRetention(url, tname string, retentionMinutes, retentionSizeMB int) string {
	return fmt.Sprintf(`
provider "pulsar" {
	web_service_url = "%s"
}

resource "pulsar_topic" "test" {
  tenant     = "public"
  namespace  = "default"
  topic_type = "persistent"
  topic_name = "%s"
  partitions = 0

  retention_policies {
    retention_time_minutes = %d
    retention_size_mb      = %d
  }
}
`, url, tname, retentionMinutes, retentionSizeMB)
}

func testTopicWithoutRetention(url, tname string) string {
	return fmt.Sprintf(`
provider "pulsar" {
	web_service_url = "%s"
}

resource "pulsar_topic" "test" {
  tenant     = "public"
  namespace  = "default"
  topic_type = "persistent"
  topic_name = "%s"
  partitions = 0
}
`, url, tname)
}

// testTopicRetentionPolicy verifies the retention policy on a topic via the Pulsar admin API.
func testTopicRetentionPolicy(
	resourceName string,
	shouldExist bool,
	expectedMinutes int,
	expectedSizeMB int64,
) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found", resourceName)
		}

		topicName, err := utils.GetTopicName(rs.Primary.ID)
		if err != nil {
			return err
		}

		client := getClientFromMeta(testAccProvider.Meta()).Topics()
		ret, err := client.GetRetention(*topicName, false)

		if !shouldExist {
			if err != nil {
				// Error getting retention means it's not set — that's expected
				return nil
			}
			// If no error, check that values are at defaults (0/0)
			if ret.RetentionTimeInMinutes == 0 && ret.RetentionSizeInMB == 0 {
				return nil
			}
			return fmt.Errorf("expected retention to be removed, but got: time=%d, size=%d",
				ret.RetentionTimeInMinutes, ret.RetentionSizeInMB)
		}

		if err != nil {
			return fmt.Errorf("failed to get retention: %w", err)
		}

		if ret.RetentionTimeInMinutes != expectedMinutes {
			return fmt.Errorf("expected retention_time_minutes=%d, got=%d",
				expectedMinutes, ret.RetentionTimeInMinutes)
		}
		if ret.RetentionSizeInMB != expectedSizeMB {
			return fmt.Errorf("expected retention_size_mb=%d, got=%d",
				expectedSizeMB, ret.RetentionSizeInMB)
		}
		return nil
	}
}
