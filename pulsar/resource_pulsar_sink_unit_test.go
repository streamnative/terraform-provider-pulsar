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
	"strings"
	"testing"

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
	"github.com/hashicorp/go-cty/cty"
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

// sinkV013InputSpecsSchema freezes the v0.13 state shape. Its input_specs block was
// Optional+Computed and had only these five required fields.
func sinkV013InputSpecsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Computed: true,
		Elem: &schema.Resource{Schema: map[string]*schema.Schema{
			resourceSinkInputSpecsSubsetTopicKey:             {Type: schema.TypeString, Required: true},
			resourceSinkInputSpecsSubsetSchemaTypeKey:        {Type: schema.TypeString, Required: true},
			resourceSinkInputSpecsSubsetSerdeClassNameKey:    {Type: schema.TypeString, Required: true},
			resourceSinkInputSpecsSubsetIsRegexPatternKey:    {Type: schema.TypeBool, Required: true},
			resourceSinkInputSpecsSubsetReceiverQueueSizeKey: {Type: schema.TypeInt, Required: true},
		}},
	}
}

func sinkV013InputSpec(topic string) map[string]interface{} {
	return map[string]interface{}{
		resourceSinkInputSpecsSubsetTopicKey:             topic,
		resourceSinkInputSpecsSubsetSchemaTypeKey:        "",
		resourceSinkInputSpecsSubsetSerdeClassNameKey:    "",
		resourceSinkInputSpecsSubsetIsRegexPatternKey:    false,
		resourceSinkInputSpecsSubsetReceiverQueueSizeKey: defaultSinkReceiverQueueSize,
	}
}

func sinkV013StateWithComputedInputSpecs(
	t *testing.T, config map[string]interface{}, inputSpecs []interface{},
) *terraform.InstanceState {
	t.Helper()

	legacyResource := resourcePulsarSink()
	legacyResource.Schema[resourceSinkInputSpecsKey] = sinkV013InputSpecsSchema()
	legacyData := schema.TestResourceDataRaw(t, legacyResource.Schema, config)
	require.NoError(t, legacyData.Set(resourceSinkInputSpecsKey, inputSpecs))
	legacyData.SetId("public/default/sink-1")

	return legacyData.State()
}

func sinkCurrentDataFromState(t *testing.T, state *terraform.InstanceState) *schema.ResourceData {
	t.Helper()

	d, err := schema.InternalMap(resourcePulsarSink().Schema).Data(state, nil)
	require.NoError(t, err)
	return d
}

func sinkLegacyInputConfig(topic string) map[string]interface{} {
	return sinkConfigWithBase(map[string]interface{}{
		resourceSinkInputsKey:  []interface{}{topic},
		resourceSinkAutoACKKey: true,
	})
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

func TestMarshalSinkInputSpecsMergesLegacyTypes(t *testing.T) {
	topic := "persistent://public/default/in-1"
	tests := []struct {
		name       string
		serde      map[string]interface{}
		schema     map[string]interface{}
		wantSerde  string
		wantSchema string
	}{
		{
			name:      "serde",
			serde:     map[string]interface{}{topic: "com.acme.Serde"},
			wantSerde: "com.acme.Serde",
		},
		{
			name:       "schema",
			schema:     map[string]interface{}{topic: "AVRO"},
			wantSchema: "AVRO",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			values := map[string]interface{}{
				resourceSinkInputSpecsKey: []interface{}{
					sinkInputSpec(topic, map[string]interface{}{
						resourceSinkInputSpecsSubsetReceiverQueueSizeKey: 100,
					}),
				},
			}
			if test.serde != nil {
				values[resourceSinkCustomSerdeInputsKey] = test.serde
			}
			if test.schema != nil {
				values[resourceSinkCustomSchemaInputsKey] = test.schema
			}

			sinkConfig, err := marshalSinkConfig(sinkResourceData(t, values))
			require.NoError(t, err)

			spec := sinkConfig.InputSpecs[topic]
			assert.Equal(t, 100, spec.ReceiverQueueSize)
			assert.Equal(t, test.wantSerde, spec.SerdeClassName)
			assert.Equal(t, test.wantSchema, spec.SchemaType)
			assert.Nil(t, sinkConfig.TopicToSerdeClassName)
			assert.Nil(t, sinkConfig.TopicToSchemaType)
		})
	}
}

func TestMarshalSinkInputSpecsRejectsAmbiguousLegacyTypes(t *testing.T) {
	topic := "persistent://public/default/in-1"
	tests := []struct {
		name    string
		spec    map[string]interface{}
		serde   map[string]interface{}
		schema  map[string]interface{}
		wantErr string
	}{
		{
			name:    "both legacy maps overlap queue-only spec",
			spec:    sinkInputSpec(topic, nil),
			serde:   map[string]interface{}{topic: "com.acme.Serde"},
			schema:  map[string]interface{}{topic: "AVRO"},
			wantErr: "overlaps both",
		},
		{
			name: "conflicting serde values",
			spec: sinkInputSpec(topic, map[string]interface{}{
				resourceSinkInputSpecsSubsetSerdeClassNameKey: "com.acme.NewSerde",
			}),
			serde:   map[string]interface{}{topic: "com.acme.OldSerde"},
			wantErr: "conflicting serde_class_name",
		},
		{
			name: "conflicting schema values",
			spec: sinkInputSpec(topic, map[string]interface{}{
				resourceSinkInputSpecsSubsetSchemaTypeKey: "AVRO",
			}),
			schema:  map[string]interface{}{topic: "JSON"},
			wantErr: "conflicting schema_type",
		},
		{
			name: "schema and legacy serde conflict",
			spec: sinkInputSpec(topic, map[string]interface{}{
				resourceSinkInputSpecsSubsetSchemaTypeKey: "AVRO",
			}),
			serde:   map[string]interface{}{topic: "com.acme.Serde"},
			wantErr: "cannot combine schema_type",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			d := sinkResourceData(t, map[string]interface{}{
				resourceSinkInputSpecsKey:         []interface{}{test.spec},
				resourceSinkCustomSerdeInputsKey:  test.serde,
				resourceSinkCustomSchemaInputsKey: test.schema,
			})

			_, err := marshalSinkConfig(d)
			require.ErrorContains(t, err, test.wantErr)
		})
	}
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

func TestSinkInputSpecsLegacyOverlapValidationDuringPlan(t *testing.T) {
	topic := "persistent://public/default/in-1"
	res := resourcePulsarSink()
	d := schema.TestResourceDataRaw(t, res.Schema, map[string]interface{}{})
	config := sinkConfigWithBase(map[string]interface{}{
		resourceSinkInputSpecsKey: []interface{}{sinkInputSpec(topic, nil)},
		resourceSinkCustomSerdeInputsKey: map[string]interface{}{
			topic: "com.acme.Serde",
		},
		resourceSinkCustomSchemaInputsKey: map[string]interface{}{
			topic: "AVRO",
		},
	})

	_, err := res.Diff(context.Background(), d.State(), terraform.NewResourceConfigRaw(config), nil)
	require.ErrorContains(t, err, "overlaps both")
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

func TestUnmarshalSinkInputSpecsRefreshesLegacyRepresentation(t *testing.T) {
	topics := map[string]string{
		"plain":          "persistent://public/default/plain",
		"removed":        "persistent://public/default/removed",
		"became_pattern": "persistent://public/default/became-pattern",
		"pattern":        "persistent://public/default/pattern-.*",
		"serde":          "persistent://public/default/serde",
		"schema":         "persistent://public/default/schema",
	}
	d := sinkResourceData(t, map[string]interface{}{
		resourceSinkInputsKey: []interface{}{
			topics["plain"], topics["removed"], topics["became_pattern"],
		},
		resourceSinkTopicsPatternKey: topics["pattern"],
		resourceSinkCustomSerdeInputsKey: map[string]interface{}{
			topics["serde"]:   "com.acme.OldSerde",
			topics["removed"]: "com.acme.RemovedSerde",
		},
		resourceSinkCustomSchemaInputsKey: map[string]interface{}{
			topics["schema"]:  "STRING",
			topics["removed"]: "BYTES",
		},
	})

	err := unmarshalSinkInputSpecs(utils.SinkConfig{InputSpecs: map[string]utils.ConsumerConfig{
		topics["plain"]: {},
		topics["became_pattern"]: {
			RegexPattern: true,
		},
		topics["pattern"]: {
			RegexPattern: true,
		},
		topics["serde"]: {
			SerdeClassName: "com.acme.NewSerde",
		},
		topics["schema"]: {
			SchemaType: "AVRO",
		},
	}}, d)
	require.NoError(t, err)

	assert.Equal(t, []interface{}{topics["plain"]}, d.Get(resourceSinkInputsKey).(*schema.Set).List())
	assert.Equal(t, topics["pattern"], d.Get(resourceSinkTopicsPatternKey))
	assert.Equal(t, map[string]interface{}{topics["serde"]: "com.acme.NewSerde"},
		d.Get(resourceSinkCustomSerdeInputsKey))
	assert.Equal(t, map[string]interface{}{topics["schema"]: "AVRO"},
		d.Get(resourceSinkCustomSchemaInputsKey))

	specs := d.Get(resourceSinkInputSpecsKey).(*schema.Set).List()
	require.Len(t, specs, 1)
	assert.Equal(t, topics["became_pattern"],
		specs[0].(map[string]interface{})[resourceSinkInputSpecsSubsetTopicKey])
}

func TestUnmarshalImportedSinkInputsUsesLegacyFieldsWhenLossless(t *testing.T) {
	topics := map[string]string{
		"plain":    "persistent://public/default/plain",
		"pattern":  "persistent://public/default/pattern-.*",
		"serde":    "persistent://public/default/serde",
		"schema":   "persistent://public/default/schema",
		"advanced": "persistent://public/default/advanced",
	}
	advanced := utils.ConsumerConfig{
		PoolMessages:       true,
		ConsumerProperties: map[string]string{"application": "billing"},
	}
	advanced.SetReceiverQueueSize(0)
	d := sinkResourceData(t, nil)

	err := unmarshalSinkInputSpecs(utils.SinkConfig{InputSpecs: map[string]utils.ConsumerConfig{
		topics["plain"]: {},
		topics["pattern"]: {
			RegexPattern: true,
		},
		topics["serde"]: {
			SerdeClassName: "com.acme.Serde",
		},
		topics["schema"]: {
			SchemaType: "AVRO",
		},
		topics["advanced"]: advanced,
	}}, d)
	require.NoError(t, err)

	assert.Equal(t, []interface{}{topics["plain"]}, d.Get(resourceSinkInputsKey).(*schema.Set).List())
	assert.Equal(t, topics["pattern"], d.Get(resourceSinkTopicsPatternKey))
	assert.Equal(t, map[string]interface{}{topics["serde"]: "com.acme.Serde"},
		d.Get(resourceSinkCustomSerdeInputsKey))
	assert.Equal(t, map[string]interface{}{topics["schema"]: "AVRO"},
		d.Get(resourceSinkCustomSchemaInputsKey))

	specs := d.Get(resourceSinkInputSpecsKey).(*schema.Set).List()
	require.Len(t, specs, 1)
	spec := specs[0].(map[string]interface{})
	assert.Equal(t, topics["advanced"], spec[resourceSinkInputSpecsSubsetTopicKey])
	assert.Zero(t, spec[resourceSinkInputSpecsSubsetReceiverQueueSizeKey])
	assert.True(t, spec[resourceSinkInputSpecsSubsetPoolMessagesKey].(bool))
	assert.Equal(t, map[string]interface{}{"application": "billing"},
		spec[resourceSinkInputSpecsSubsetConsumerPropertiesKey])
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

func sinkInputSpecsDiffContains(diff *terraform.InstanceDiff, want string) bool {
	for key, attr := range diff.Attributes {
		if strings.HasPrefix(key, resourceSinkInputSpecsKey+".") && attr != nil && attr.New == want {
			return true
		}
	}

	return false
}

func TestSinkV013ComputedInputSpecsPlanWithoutRefresh(t *testing.T) {
	topic := "persistent://public/default/in-1"
	legacyConfig := sinkLegacyInputConfig(topic)
	legacyState := sinkV013StateWithComputedInputSpecs(t, legacyConfig, []interface{}{
		sinkV013InputSpec(topic),
	})

	res := resourcePulsarSink()
	assert.True(t, res.Schema[resourceSinkInputSpecsKey].Computed)

	diff, err := res.Diff(context.Background(), legacyState,
		terraform.NewResourceConfigRaw(legacyConfig), nil)
	require.NoError(t, err)
	if diff != nil {
		assert.True(t, diff.Empty(), "legacy state re-planned: %#v", diff.Attributes)
	}
}

func TestSinkV013ComputedInputSpecsRefreshPlansCleanly(t *testing.T) {
	topic := "persistent://public/default/in-1"
	legacyConfig := sinkLegacyInputConfig(topic)
	legacyState := sinkV013StateWithComputedInputSpecs(t, legacyConfig, []interface{}{
		sinkV013InputSpec(topic),
	})

	// ReadResource sends CurrentState but no configuration. Decode the frozen v0.13 flatmap through
	// the current schema, then exercise the same input refresh helper that resourcePulsarSinkRead uses.
	d := sinkCurrentDataFromState(t, legacyState)
	require.NoError(t, unmarshalSinkInputSpecs(utils.SinkConfig{
		InputSpecs: map[string]utils.ConsumerConfig{topic: {}},
	}, d))
	require.Len(t, d.Get(resourceSinkInputSpecsKey).(*schema.Set).List(), 1)

	diff, err := resourcePulsarSink().Diff(context.Background(), d.State(),
		terraform.NewResourceConfigRaw(legacyConfig), nil)
	require.NoError(t, err)
	if diff != nil {
		assert.True(t, diff.Empty(), "refreshed legacy state re-planned: %#v", diff.Attributes)
	}
}

func TestUnmarshalSinkInputSpecsUsesRawConfigOwnership(t *testing.T) {
	topic := "persistent://public/default/in-1"
	tests := []struct {
		name      string
		rawConfig cty.Value
		wantSpecs int
	}{
		{
			name: "legacy HCL omits input_specs",
			rawConfig: cty.ObjectVal(map[string]cty.Value{
				resourceSinkInputSpecsKey: cty.NullVal(cty.Set(cty.EmptyObject)),
			}),
		},
		{
			name: "explicit input_specs remains owned",
			rawConfig: cty.ObjectVal(map[string]cty.Value{
				resourceSinkInputSpecsKey: cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						resourceSinkInputSpecsSubsetTopicKey: cty.StringVal(topic),
					}),
				}),
			}),
			wantSpecs: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			legacyState := sinkV013StateWithComputedInputSpecs(t, sinkLegacyInputConfig(topic), []interface{}{
				sinkV013InputSpec(topic),
			})
			legacyState.RawConfig = test.rawConfig
			d := sinkCurrentDataFromState(t, legacyState)

			require.NoError(t, unmarshalSinkInputSpecs(utils.SinkConfig{
				InputSpecs: map[string]utils.ConsumerConfig{topic: {}},
			}, d))
			assert.Len(t, d.Get(resourceSinkInputSpecsKey).(*schema.Set).List(), test.wantSpecs)
		})
	}
}

func TestSinkImportedLegacyInputPlansCleanly(t *testing.T) {
	topic := "persistent://public/default/in-1"
	res := resourcePulsarSink()
	assert.True(t, res.Schema[resourceSinkInputSpecsKey].Computed)
	d := schema.TestResourceDataRaw(t, res.Schema, sinkConfigWithBase(nil))
	d.SetId("public/default/sink-1")

	require.NoError(t, unmarshalSinkInputSpecs(utils.SinkConfig{
		InputSpecs: map[string]utils.ConsumerConfig{topic: {}},
	}, d))
	assert.Empty(t, d.Get(resourceSinkInputSpecsKey).(*schema.Set).List())

	config := sinkConfigWithBase(map[string]interface{}{
		resourceSinkInputsKey: []interface{}{topic},
	})
	diff, err := res.Diff(context.Background(), d.State(), terraform.NewResourceConfigRaw(config), nil)
	require.NoError(t, err)
	if diff != nil {
		assert.True(t, diff.Empty(), "unexpected import follow-up diff: %#v", diff.Attributes)
	}
}

func TestSinkExistingDefaultQueueStatePlansCleanly(t *testing.T) {
	topic := "persistent://public/default/in-1"
	res := resourcePulsarSink()
	d := schema.TestResourceDataRaw(t, res.Schema, map[string]interface{}{})
	for key, value := range sinkConfigWithBase(map[string]interface{}{
		resourceSinkInputSpecsKey: []interface{}{sinkInputSpec(topic, nil)},
	}) {
		require.NoError(t, d.Set(key, value))
	}
	d.SetId("public/default/sink-1")

	config := sinkConfigWithBase(map[string]interface{}{
		resourceSinkInputSpecsKey: []interface{}{
			map[string]interface{}{resourceSinkInputSpecsSubsetTopicKey: topic},
		},
	})
	diff, err := res.Diff(context.Background(), d.State(), terraform.NewResourceConfigRaw(config), nil)
	require.NoError(t, err)
	if diff != nil {
		assert.True(t, diff.Empty(), "existing queue state re-planned: %#v", diff.Attributes)
	}
}

func TestSinkInputSpecsForceNew(t *testing.T) {
	topic := "persistent://public/default/in-1"
	tests := []struct {
		name        string
		state       map[string]interface{}
		config      map[string]interface{}
		requiresNew bool
		plans       string
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
			plans: "250",
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
			plans: "250",
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
			if test.plans != "" {
				assert.True(t, sinkInputSpecsDiffContains(diff, test.plans),
					"input_specs diff does not contain %q: %#v", test.plans, diff.Attributes)
			}
		})
	}
}
