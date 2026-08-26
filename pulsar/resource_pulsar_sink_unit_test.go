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
	"context"
	"testing"

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sinkInputSpec(topic string, overrides map[string]interface{}) map[string]interface{} {
	spec := map[string]interface{}{
		resourceSinkInputSpecsSubsetTopicKey:             topic,
		resourceSinkInputSpecsSubsetSchemaTypeKey:        "",
		resourceSinkInputSpecsSubsetSerdeClassNameKey:    "",
		resourceSinkInputSpecsSubsetIsRegexPatternKey:    false,
		resourceSinkInputSpecsSubsetReceiverQueueSizeKey: defaultSinkReceiverQueueSize,
		resourceSinkInputSpecsSubsetPoolMessagesKey:      false,
	}
	for key, value := range overrides {
		spec[key] = value
	}

	return spec
}

func sinkResourceData(t *testing.T, values map[string]interface{}) *schema.ResourceData {
	t.Helper()

	d := schema.TestResourceDataRaw(t, resourcePulsarSink().Schema, map[string]interface{}{})
	for key, value := range values {
		require.NoError(t, d.Set(key, value))
	}

	return d
}

// The point of #218: tuning the queue size must not force the user to also name a schema type and
// a serde class, which Pulsar rejects together anyway.
func TestMarshalSinkInputSpecsQueueSizeOnly(t *testing.T) {
	d := sinkResourceData(t, map[string]interface{}{
		resourceSinkTenantKey:    "public",
		resourceSinkNamespaceKey: "default",
		resourceSinkNameKey:      "sink-1",
		resourceSinkInputSpecsKey: []interface{}{
			sinkInputSpec("persistent://public/default/in-1", map[string]interface{}{
				resourceSinkInputSpecsSubsetReceiverQueueSizeKey: 100,
			}),
		},
	})

	sinkConfig, err := marshalSinkConfig(d)
	require.NoError(t, err)

	spec, ok := sinkConfig.InputSpecs["persistent://public/default/in-1"]
	require.True(t, ok)
	assert.Equal(t, 100, spec.ReceiverQueueSize)
	assert.True(t, spec.HasReceiverQueueSize())
	assert.Empty(t, spec.SchemaType)
	assert.Empty(t, spec.SerdeClassName)
	assert.Nil(t, spec.ConsumerProperties)
}

func TestMarshalSinkInputSpecsSupportedFields(t *testing.T) {
	d := sinkResourceData(t, map[string]interface{}{
		resourceSinkTenantKey:    "public",
		resourceSinkNamespaceKey: "default",
		resourceSinkNameKey:      "sink-1",
		resourceSinkInputSpecsKey: []interface{}{
			sinkInputSpec("persistent://public/default/in-1", map[string]interface{}{
				resourceSinkInputSpecsSubsetPoolMessagesKey: true,
				resourceSinkInputSpecsSubsetConsumerPropertiesKey: map[string]interface{}{
					"application": "billing",
				},
			}),
		},
	})

	sinkConfig, err := marshalSinkConfig(d)
	require.NoError(t, err)

	spec := sinkConfig.InputSpecs["persistent://public/default/in-1"]
	assert.True(t, spec.PoolMessages)
	assert.Equal(t, map[string]string{"application": "billing"}, spec.ConsumerProperties)
}

func TestMarshalSinkInputSpecsExplicitZeroQueueSize(t *testing.T) {
	d := sinkResourceData(t, map[string]interface{}{
		resourceSinkTenantKey:    "public",
		resourceSinkNamespaceKey: "default",
		resourceSinkNameKey:      "sink-1",
		resourceSinkInputSpecsKey: []interface{}{
			sinkInputSpec("persistent://public/default/in-1", map[string]interface{}{
				resourceSinkInputSpecsSubsetReceiverQueueSizeKey: 0,
			}),
		},
	})

	sinkConfig, err := marshalSinkConfig(d)
	require.NoError(t, err)
	spec := sinkConfig.InputSpecs["persistent://public/default/in-1"]
	assert.True(t, spec.HasReceiverQueueSize())
	assert.Zero(t, spec.ReceiverQueueSize)
}

func TestValidateSinkInputSpecs(t *testing.T) {
	tests := []struct {
		name    string
		specs   []interface{}
		wantErr string
	}{
		{
			name: "duplicate topic keys are rejected",
			specs: []interface{}{
				sinkInputSpec("persistent://public/default/in-1", map[string]interface{}{
					resourceSinkInputSpecsSubsetReceiverQueueSizeKey: 100,
				}),
				sinkInputSpec("persistent://public/default/in-1", map[string]interface{}{
					resourceSinkInputSpecsSubsetReceiverQueueSizeKey: 999,
				}),
			},
			wantErr: "duplicate",
		},
		{
			// SinkConfigUtils rejects this server-side; catching it at plan time is a better error.
			name: "schema_type and serde_class_name are mutually exclusive",
			specs: []interface{}{
				sinkInputSpec("persistent://public/default/in-1", map[string]interface{}{
					resourceSinkInputSpecsSubsetSchemaTypeKey:     "avro",
					resourceSinkInputSpecsSubsetSerdeClassNameKey: "com.acme.MySerde",
				}),
			},
			wantErr: "cannot set both",
		},
		{
			name: "queue size alone is valid",
			specs: []interface{}{
				sinkInputSpec("persistent://public/default/in-1", map[string]interface{}{
					resourceSinkInputSpecsSubsetReceiverQueueSizeKey: 100,
				}),
			},
		},
		{
			name: "distinct topics are valid",
			specs: []interface{}{
				sinkInputSpec("persistent://public/default/in-1", map[string]interface{}{
					resourceSinkInputSpecsSubsetReceiverQueueSizeKey: 100,
				}),
				sinkInputSpec("persistent://public/default/in-2", map[string]interface{}{
					resourceSinkInputSpecsSubsetSchemaTypeKey: "avro",
				}),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			d := sinkResourceData(t, map[string]interface{}{
				resourceSinkInputSpecsKey: test.specs,
			})

			err := validateSinkInputSpecs(d.Get(resourceSinkInputSpecsKey))
			if test.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), test.wantErr)
		})
	}
}

func TestMarshalSinkInputSpecsFilterLegacyOverlaps(t *testing.T) {
	topics := []string{
		"persistent://public/default/plain",
		"persistent://public/default/pattern-.*",
		"persistent://public/default/serde",
		"persistent://public/default/schema",
	}
	d := sinkResourceData(t, map[string]interface{}{
		resourceSinkTenantKey:            "public",
		resourceSinkNamespaceKey:         "default",
		resourceSinkNameKey:              "sink-1",
		resourceSinkInputsKey:            []interface{}{topics[0]},
		resourceSinkTopicsPatternKey:     topics[1],
		resourceSinkCustomSerdeInputsKey: map[string]interface{}{topics[2]: "com.acme.Serde"},
		resourceSinkCustomSchemaInputsKey: map[string]interface{}{
			topics[3]: "STRING",
		},
		resourceSinkInputSpecsKey: []interface{}{
			sinkInputSpec(topics[0], nil),
			sinkInputSpec(topics[1], map[string]interface{}{
				resourceSinkInputSpecsSubsetIsRegexPatternKey: true,
			}),
			sinkInputSpec(topics[2], map[string]interface{}{
				resourceSinkInputSpecsSubsetSerdeClassNameKey: "com.acme.Serde",
			}),
			sinkInputSpec(topics[3], map[string]interface{}{
				resourceSinkInputSpecsSubsetSchemaTypeKey: "STRING",
			}),
		},
	})

	sinkConfig, err := marshalSinkConfig(d)
	require.NoError(t, err)
	assert.Nil(t, sinkConfig.Inputs)
	assert.Nil(t, sinkConfig.TopicsPattern)
	assert.Nil(t, sinkConfig.TopicToSerdeClassName)
	assert.Nil(t, sinkConfig.TopicToSchemaType)
	assert.Len(t, sinkConfig.InputSpecs, len(topics))
}

func TestSinkInputSpecsValidationDuringPlan(t *testing.T) {
	res := resourcePulsarSink()
	d := schema.TestResourceDataRaw(t, res.Schema, map[string]interface{}{})
	config := sinkConfigWithBase(map[string]interface{}{
		resourceSinkInputSpecsKey: []interface{}{
			sinkInputSpec("persistent://public/default/in-1", map[string]interface{}{
				resourceSinkInputSpecsSubsetReceiverQueueSizeKey: 100,
			}),
			sinkInputSpec("persistent://public/default/in-1", map[string]interface{}{
				resourceSinkInputSpecsSubsetReceiverQueueSizeKey: 200,
			}),
		},
	})

	_, err := res.Diff(context.Background(), d.State(), terraform.NewResourceConfigRaw(config), nil)
	require.ErrorContains(t, err, "duplicate")
}

func TestUnmarshalSinkInputSpecsPreservesLegacyRepresentation(t *testing.T) {
	topic := "persistent://public/default/in-1"
	d := sinkResourceData(t, map[string]interface{}{
		resourceSinkInputsKey: []interface{}{topic},
	})
	consumerConfig := utils.ConsumerConfig{}
	consumerConfig.SetReceiverQueueSize(100)

	err := unmarshalSinkInputSpecs(utils.SinkConfig{
		InputSpecs: map[string]utils.ConsumerConfig{topic: consumerConfig},
	}, d)
	require.NoError(t, err)
	assert.Empty(t, d.Get(resourceSinkInputSpecsKey).(*schema.Set).List())
	assert.Equal(t, []interface{}{topic}, d.Get(resourceSinkInputsKey).(*schema.Set).List())
}

func TestUnmarshalSinkInputSpecsPreservesExplicitZero(t *testing.T) {
	topic := "persistent://public/default/in-1"
	d := sinkResourceData(t, map[string]interface{}{
		resourceSinkInputSpecsKey: []interface{}{
			sinkInputSpec(topic, map[string]interface{}{
				resourceSinkInputSpecsSubsetReceiverQueueSizeKey: 0,
			}),
		},
	})
	consumerConfig := utils.ConsumerConfig{}
	consumerConfig.SetReceiverQueueSize(0)

	err := unmarshalSinkInputSpecs(utils.SinkConfig{
		InputSpecs: map[string]utils.ConsumerConfig{topic: consumerConfig},
	}, d)
	require.NoError(t, err)
	specs := d.Get(resourceSinkInputSpecsKey).(*schema.Set).List()
	require.Len(t, specs, 1)
	assert.Zero(t, specs[0].(map[string]interface{})[resourceSinkInputSpecsSubsetReceiverQueueSizeKey])
}

func sinkConfigWithBase(values map[string]interface{}) map[string]interface{} {
	config := map[string]interface{}{
		resourceSinkTenantKey:              "public",
		resourceSinkNamespaceKey:           "default",
		resourceSinkNameKey:                "sink-1",
		resourceSinkCleanupSubscriptionKey: false,
		resourceSinkArchiveKey:             "builtin://jdbc",
	}
	for key, value := range values {
		config[key] = value
	}
	return config
}

func sinkInputSpecsDiff(t *testing.T, state, config map[string]interface{}) *terraform.InstanceDiff {
	t.Helper()
	res := resourcePulsarSink()
	d := schema.TestResourceDataRaw(t, res.Schema, map[string]interface{}{})
	for key, value := range state {
		require.NoError(t, d.Set(key, value))
	}
	d.SetId("public/default/sink-1")

	diff, err := res.Diff(context.Background(), d.State(), terraform.NewResourceConfigRaw(config), nil)
	require.NoError(t, err)
	require.NotNil(t, diff)
	return diff
}

func TestSinkInputSpecsForceNew(t *testing.T) {
	topic := "persistent://public/default/in-1"
	tests := []struct {
		name        string
		state       map[string]interface{}
		config      map[string]interface{}
		requiresNew bool
	}{
		{
			name: "queue size updates in place",
			state: sinkConfigWithBase(map[string]interface{}{
				resourceSinkInputSpecsKey: []interface{}{sinkInputSpec(topic, map[string]interface{}{
					resourceSinkInputSpecsSubsetReceiverQueueSizeKey: 100,
				})},
			}),
			config: sinkConfigWithBase(map[string]interface{}{
				resourceSinkInputSpecsKey: []interface{}{sinkInputSpec(topic, map[string]interface{}{
					resourceSinkInputSpecsSubsetReceiverQueueSizeKey: 250,
				})},
			}),
		},
		{
			name: "adopting input_specs for an existing topic updates in place",
			state: sinkConfigWithBase(map[string]interface{}{
				resourceSinkInputsKey: []interface{}{topic},
			}),
			config: sinkConfigWithBase(map[string]interface{}{
				resourceSinkInputsKey: []interface{}{topic},
				resourceSinkInputSpecsKey: []interface{}{sinkInputSpec(topic, map[string]interface{}{
					resourceSinkInputSpecsSubsetReceiverQueueSizeKey: 250,
				})},
			}),
		},
		{
			name: "renaming an input_specs topic replaces the sink",
			state: sinkConfigWithBase(map[string]interface{}{
				resourceSinkInputSpecsKey: []interface{}{sinkInputSpec(topic, nil)},
			}),
			config: sinkConfigWithBase(map[string]interface{}{
				resourceSinkInputSpecsKey: []interface{}{
					sinkInputSpec("persistent://public/default/renamed", nil),
				},
			}),
			requiresNew: true,
		},
		{
			name: "flipping the regex flag replaces the sink",
			state: sinkConfigWithBase(map[string]interface{}{
				resourceSinkInputSpecsKey: []interface{}{sinkInputSpec(topic, nil)},
			}),
			config: sinkConfigWithBase(map[string]interface{}{
				resourceSinkInputSpecsKey: []interface{}{sinkInputSpec(topic, map[string]interface{}{
					resourceSinkInputSpecsSubsetIsRegexPatternKey: true,
				})},
			}),
			requiresNew: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			diff := sinkInputSpecsDiff(t, test.state, test.config)
			assert.Equal(t, test.requiresNew, diff.RequiresNew())
		})
	}
}
