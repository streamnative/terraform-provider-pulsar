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
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/rest"
	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	initTestWebServiceURL()
}

func TestFunction(t *testing.T) {
	configBytes, err := os.ReadFile("testdata/function/main.tf")
	if err != nil {
		t.Fatal(err)
	}

	updatedConfigBytes, err := os.ReadFile("testdata/function/main_updated.tf")
	if err != nil {
		t.Fatal(err)
	}

	zeroQueueConfigBytes, err := os.ReadFile("testdata/function/main_zero_queue.tf")
	if err != nil {
		t.Fatal(err)
	}

	// Captured after create so the update step can prove the function was updated in place rather
	// than destroyed and recreated.
	var createdID string

	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t) },
		ProviderFactories:         testAccProviderFactories,
		PreventPostDestroyRefresh: false,
		CheckDestroy:              testPulsarFunctionDestroy,
		Steps: []resource.TestStep{
			{
				Config: string(configBytes),
				Check: resource.ComposeTestCheckFunc(func(s *terraform.State) error {
					name := "pulsar_function.function-1"
					rs, ok := s.RootModule().Resources[name]
					if !ok {
						return fmt.Errorf("%s not be found", name)
					}

					config, err := getPulsarFunctionByResourceID(rs.Primary.ID)
					if err != nil {
						return err
					}

					if config == nil {
						return fmt.Errorf("failed to create %s function", rs.Primary.ID)
					}
					fmt.Printf("config: %v\n", config)

					createdID = rs.Primary.ID

					assert.Equal(t, "function-1", config.Name)
					assert.Equal(t, "public", config.Tenant)
					assert.Equal(t, "default", config.Namespace)
					assert.Equal(t, ProcessingGuaranteesAtLeastOnce, config.ProcessingGuarantees)
					assert.NotNil(t, config.TimeoutMs)
					assert.Equal(t, int64(6666), *config.TimeoutMs)
					assert.NotNil(t, config.Resources)

					// input1 carries an input_specs block; input2 is a plain input. Pulsar returns a
					// spec for both regardless.
					assert.Equal(t, 100, config.InputSpecs["public/default/input1"].ReceiverQueueSize)
					assert.Equal(t, "avro", config.InputSpecs["public/default/input1"].SchemaType)
					assert.Contains(t, config.InputSpecs, "public/default/input2")
					assert.Equal(t, 101,
						config.InputSpecs["public/default/pattern-.*"].ReceiverQueueSize)
					assert.True(t, config.InputSpecs["public/default/pattern-.*"].RegexPattern)
					assert.Equal(t, 102,
						config.InputSpecs["public/default/serde-input"].ReceiverQueueSize)
					assert.Equal(t, "org.apache.pulsar.functions.api.utils.DefaultSerDe",
						config.InputSpecs["public/default/serde-input"].SerdeClassName)
					assert.Equal(t, 103,
						config.InputSpecs["public/default/schema-input"].ReceiverQueueSize)
					assert.Equal(t, "STRING", config.InputSpecs["public/default/schema-input"].SchemaType)

					// #220 part A: the output producer configuration must round-trip.
					require.NotNil(t, config.ProducerConfig)
					assert.Equal(t, "ZSTD", config.ProducerConfig.CompressionType)
					assert.Equal(t, "KEY_BASED", config.ProducerConfig.BatchBuilder)
					assert.Equal(t, 1000, config.ProducerConfig.MaxPendingMessages)
					assert.Equal(t, 50000, config.ProducerConfig.MaxPendingMessagesAcrossPartitions)
					assert.True(t, config.ProducerConfig.UseThreadLocalProducers)

					return nil
				}),
			},
			{
				// Pulsar accepts a receiver queue size change on an existing topic, so this must be
				// an in-place update.
				Config: string(updatedConfigBytes),
				Check: resource.ComposeTestCheckFunc(func(s *terraform.State) error {
					name := "pulsar_function.function-1"
					rs, ok := s.RootModule().Resources[name]
					if !ok {
						return fmt.Errorf("%s not be found", name)
					}

					if rs.Primary.ID != createdID {
						return fmt.Errorf("function was replaced: id changed from %s to %s",
							createdID, rs.Primary.ID)
					}

					config, err := getPulsarFunctionByResourceID(rs.Primary.ID)
					if err != nil {
						return err
					}

					if config == nil {
						return fmt.Errorf("failed to update %s function", rs.Primary.ID)
					}

					assert.Equal(t, 250, config.InputSpecs["public/default/input1"].ReceiverQueueSize)
					assert.Equal(t, 251,
						config.InputSpecs["public/default/pattern-.*"].ReceiverQueueSize)
					assert.Equal(t, 252,
						config.InputSpecs["public/default/serde-input"].ReceiverQueueSize)
					assert.Equal(t, 253,
						config.InputSpecs["public/default/schema-input"].ReceiverQueueSize)

					// The topic listed in both inputs and input_specs must keep its consumer config
					// across the update: if the provider left it in inputs, validateUpdate() would
					// have reset it to a default ConsumerConfig here.
					assert.Equal(t, "avro", config.InputSpecs["public/default/input1"].SchemaType)

					require.NotNil(t, config.ProducerConfig)
					assert.Equal(t, "LZ4", config.ProducerConfig.CompressionType)
					assert.Equal(t, "DEFAULT", config.ProducerConfig.BatchBuilder)
					assert.Zero(t, config.ProducerConfig.MaxPendingMessages)
					assert.Zero(t, config.ProducerConfig.MaxPendingMessagesAcrossPartitions)
					assert.False(t, config.ProducerConfig.UseThreadLocalProducers)

					return nil
				}),
			},
			{
				// Zero is a valid explicit value, distinct from an omitted queue size. Verify it
				// reaches Pulsar, survives GET, and remains an in-place update.
				Config: string(zeroQueueConfigBytes),
				Check: resource.ComposeTestCheckFunc(func(s *terraform.State) error {
					name := "pulsar_function.function-1"
					rs, ok := s.RootModule().Resources[name]
					if !ok {
						return fmt.Errorf("%s not be found", name)
					}

					if rs.Primary.ID != createdID {
						return fmt.Errorf("function was replaced: id changed from %s to %s",
							createdID, rs.Primary.ID)
					}

					config, err := getPulsarFunctionByResourceID(rs.Primary.ID)
					if err != nil {
						return err
					}
					if config == nil {
						return fmt.Errorf("failed to update %s function", rs.Primary.ID)
					}

					consumerConfig := config.InputSpecs["public/default/input1"]
					assert.True(t, consumerConfig.HasReceiverQueueSize())
					assert.Zero(t, consumerConfig.ReceiverQueueSize)

					return nil
				}),
			},
		},
	})
}

// Regression guard: a function configured only with `inputs` must not drift. The broker returns an
// inputSpecs entry for every one of those topics, so a read that mirrored them into state would
// plan a change on every run. resource.Test fails the step if the post-apply plan is non-empty.
func TestFunctionLegacyInputsOnly(t *testing.T) {
	configBytes, err := os.ReadFile("testdata/function/legacy_inputs_only.tf")
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t) },
		ProviderFactories:         testAccProviderFactories,
		PreventPostDestroyRefresh: false,
		CheckDestroy:              testPulsarFunctionDestroy,
		Steps: []resource.TestStep{
			{Config: string(configBytes)},
		},
	})
}

// Regression guard for the other legacy input forms. Pulsar returns both as inputSpecs and does not
// reconstruct either custom map on GET, so refresh must not invent input_specs blocks for them.
func TestFunctionLegacyCustomInputs(t *testing.T) {
	configBytes, err := os.ReadFile("testdata/function/legacy_custom_inputs.tf")
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                  func() { testAccPreCheck(t) },
		ProviderFactories:         testAccProviderFactories,
		PreventPostDestroyRefresh: false,
		CheckDestroy:              testPulsarFunctionDestroy,
		Steps: []resource.TestStep{
			{Config: string(configBytes)},
		},
	})
}

func testPulsarFunctionDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "pulsar_function" {
			continue
		}

		config, err := getPulsarFunctionByResourceID(rs.Primary.ID)
		if err != nil {
			return err
		}

		if config != nil {
			return fmt.Errorf("function still exists")
		}
	}

	return nil
}

func getPulsarFunctionByResourceID(id string) (*utils.FunctionConfig, error) {
	client := getV3ClientFromMeta(testAccProvider.Meta()).Functions()

	parts := strings.Split(id, "/")
	if len(parts) != 3 {
		return nil, errors.New("Primary ID should be tenant/namespace/name format")
	}

	resp, err := client.GetFunction(parts[0], parts[1], parts[2])
	if err != nil {
		if cliErr, ok := err.(rest.Error); ok && cliErr.Code == 404 {
			return nil, nil
		}
	}

	return &resp, nil
}
