package pulsar

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFunctionRuntimeConfigConfigsSensitive(t *testing.T) {
	resourceSchema := resourcePulsarFunction().Schema
	for _, key := range []string{resourceFunctionSinkConfigKey, resourceFunctionSourceConfigKey} {
		runtimeConfig, ok := resourceSchema[key].Elem.(*schema.Resource)
		require.True(t, ok)
		require.True(t, runtimeConfig.Schema[resourceFunctionRuntimeConfigConfigsKey].Sensitive)
	}
}

func TestUnmarshalFunctionProducerConfigClearsZeroValues(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourcePulsarFunction().Schema, map[string]interface{}{
		resourceFunctionPCMaxPendingMsgKey:                1000,
		resourceFunctionPCMaxPendingMsgAcrossPartitionKey: 50000,
		resourceFunctionPCUseThreadLocalProducersKey:      true,
		resourceFunctionPCBatchBuilderKey:                 "KEY_BASED",
		resourceFunctionPCCompressionTypeKey:              "ZSTD",
	})

	err := unmarshalFunctionProducerConfig(utils.FunctionConfig{
		ProducerConfig: &utils.ProducerConfig{},
	}, d)
	require.NoError(t, err)
	assert.Zero(t, d.Get(resourceFunctionPCMaxPendingMsgKey))
	assert.Zero(t, d.Get(resourceFunctionPCMaxPendingMsgAcrossPartitionKey))
	assert.False(t, d.Get(resourceFunctionPCUseThreadLocalProducersKey).(bool))
	assert.Empty(t, d.Get(resourceFunctionPCBatchBuilderKey))
	assert.Empty(t, d.Get(resourceFunctionPCCompressionTypeKey))
}

func TestFunctionProducerConfigSchemaIsOptionalComputed(t *testing.T) {
	resourceSchema := resourcePulsarFunction().Schema
	for _, key := range functionProducerConfigKeys {
		assert.True(t, resourceSchema[key].Optional, "expected %s to be optional", key)
		assert.True(t, resourceSchema[key].Computed, "expected %s to be computed", key)
	}
}

func TestFunctionProducerConfigUpgradeFromV013PlansCleanly(t *testing.T) {
	res := resourcePulsarFunction()
	legacyState := &terraform.InstanceState{
		ID: "public/default/function-1",
		Attributes: map[string]string{
			resourceFunctionTenantKey:              "public",
			resourceFunctionNamespaceKey:           "default",
			resourceFunctionNameKey:                "function-1",
			resourceFunctionSinkConfigKey + ".#":   "0",
			resourceFunctionSourceConfigKey + ".#": "0",
		},
	}

	// v0.13 state had no producer attributes. A v0.14 refresh must retain remote values as computed
	// state so unchanged v0.13 HCL does not plan a destructive producer update.
	d := res.Data(legacyState)
	remoteProducerConfig := utils.ProducerConfig{
		MaxPendingMessages:                 1000,
		MaxPendingMessagesAcrossPartitions: 50000,
		UseThreadLocalProducers:            true,
		BatchBuilder:                       "KEY_BASED",
		CompressionType:                    "ZSTD",
	}
	require.NoError(t, unmarshalFunctionProducerConfig(utils.FunctionConfig{
		ProducerConfig: &remoteProducerConfig,
	}, d))

	diff, err := res.Diff(
		context.Background(),
		d.State(),
		terraform.NewResourceConfigRaw(functionConfigWithBase(nil)),
		nil,
	)
	require.NoError(t, err)
	if diff != nil {
		assert.True(t, diff.Empty(), "v0.13 config re-planned producer values: %#v", diff.Attributes)
	}
}

func TestFunctionProducerConfigApplyOwnership(t *testing.T) {
	remoteProducerConfig := utils.ProducerConfig{
		MaxPendingMessages:                 1000,
		MaxPendingMessagesAcrossPartitions: 50000,
		UseThreadLocalProducers:            true,
		BatchBuilder:                       "KEY_BASED",
		CompressionType:                    "ZSTD",
	}

	tests := []struct {
		name   string
		state  *terraform.InstanceState
		config map[string]interface{}
		want   *utils.ProducerConfig
	}{
		{
			name: "explicit values are sent on create",
			config: functionConfigWithBase(map[string]interface{}{
				resourceFunctionPCMaxPendingMsgKey:                10,
				resourceFunctionPCMaxPendingMsgAcrossPartitionKey: 20,
				resourceFunctionPCUseThreadLocalProducersKey:      true,
				resourceFunctionPCBatchBuilderKey:                 "DEFAULT",
				resourceFunctionPCCompressionTypeKey:              "LZ4",
			}),
			want: &utils.ProducerConfig{
				MaxPendingMessages:                 10,
				MaxPendingMessagesAcrossPartitions: 20,
				UseThreadLocalProducers:            true,
				BatchBuilder:                       "DEFAULT",
				CompressionType:                    "LZ4",
			},
		},
		{
			name: "explicit zero uses function compression default on create",
			config: functionConfigWithBase(map[string]interface{}{
				resourceFunctionPCMaxPendingMsgKey: 0,
			}),
			want: &utils.ProducerConfig{
				CompressionType: "LZ4",
			},
		},
		{
			name:  "explicit zero preserves omitted remote peers on update",
			state: functionProducerState(t, remoteProducerConfig, "old-output"),
			config: functionConfigWithBase(map[string]interface{}{
				resourceFunctionOutputKey:          "old-output",
				resourceFunctionPCMaxPendingMsgKey: 0,
			}),
			want: &utils.ProducerConfig{
				MaxPendingMessages:                 0,
				MaxPendingMessagesAcrossPartitions: 50000,
				UseThreadLocalProducers:            true,
				BatchBuilder:                       "KEY_BASED",
				CompressionType:                    "ZSTD",
			},
		},
		{
			name:  "explicit zero and false values are sent on update",
			state: functionProducerState(t, remoteProducerConfig, "old-output"),
			config: functionConfigWithBase(map[string]interface{}{
				resourceFunctionOutputKey:                         "old-output",
				resourceFunctionPCMaxPendingMsgKey:                0,
				resourceFunctionPCMaxPendingMsgAcrossPartitionKey: 0,
				resourceFunctionPCUseThreadLocalProducersKey:      false,
				resourceFunctionPCCompressionTypeKey:              "LZ4",
			}),
			want: &utils.ProducerConfig{
				BatchBuilder:    "KEY_BASED",
				CompressionType: "LZ4",
			},
		},
		{
			name:  "omitted producer fields are not sent on unrelated update",
			state: functionProducerState(t, remoteProducerConfig, "old-output"),
			config: functionConfigWithBase(map[string]interface{}{
				resourceFunctionOutputKey: "new-output",
			}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res := resourcePulsarFunction()
			var sent *utils.ProducerConfig
			if test.state == nil {
				res.CreateContext = func(_ context.Context, d *schema.ResourceData, _ interface{}) diag.Diagnostics {
					functionConfig, err := marshalFunctionConfig(d)
					if err != nil {
						return diag.FromErr(err)
					}
					sent = functionConfig.ProducerConfig
					return nil
				}
			} else {
				res.UpdateContext = func(_ context.Context, d *schema.ResourceData, _ interface{}) diag.Diagnostics {
					functionConfig, err := marshalFunctionConfig(d)
					if err != nil {
						return diag.FromErr(err)
					}
					sent = functionConfig.ProducerConfig
					return nil
				}
			}

			diff, err := res.Diff(
				context.Background(),
				test.state,
				terraform.NewResourceConfigRaw(test.config),
				nil,
			)
			require.NoError(t, err)
			require.NotNil(t, diff)

			_, diags := res.Apply(context.Background(), test.state, diff, nil)
			require.False(t, diags.HasError(), "apply diagnostics: %#v", diags)
			assert.Equal(t, test.want, sent)
		})
	}
}

func functionConfigWithBase(values map[string]interface{}) map[string]interface{} {
	config := map[string]interface{}{
		resourceFunctionTenantKey:    "public",
		resourceFunctionNamespaceKey: "default",
		resourceFunctionNameKey:      "function-1",
	}
	for key, value := range values {
		config[key] = value
	}

	return config
}

func functionProducerState(
	t *testing.T,
	producerConfig utils.ProducerConfig,
	output string,
) *terraform.InstanceState {
	t.Helper()

	d := resourcePulsarFunction().Data(&terraform.InstanceState{
		ID: "public/default/function-1",
		Attributes: map[string]string{
			resourceFunctionTenantKey:    "public",
			resourceFunctionNamespaceKey: "default",
			resourceFunctionNameKey:      "function-1",
			resourceFunctionOutputKey:    output,
		},
	})
	require.NoError(t, unmarshalFunctionProducerConfig(utils.FunctionConfig{
		ProducerConfig: &producerConfig,
	}, d))

	return d.State()
}

func TestMergeFunctionCustomRuntimeOptions(t *testing.T) {
	base := `{"foo":"bar","sinkConfig":{"old":"value"},"sourceConfig":{"keep":"me"}}`
	sinkConfig := map[string]interface{}{
		runtimeOptionSinkConfigTypeField: "kafka",
		runtimeOptionConfigsKey:          map[string]interface{}{"new": "value"},
	}
	sourceConfig := map[string]interface{}{}

	merged, err := mergeFunctionCustomRuntimeOptions(base,
		runtimeConfigUpdate{key: runtimeOptionSinkConfigKey, config: sinkConfig},
		runtimeConfigUpdate{key: runtimeOptionSourceConfigKey, config: sourceConfig},
	)
	require.NoError(t, err)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(merged), &result))

	assert.Equal(t, "bar", result["foo"])
	assert.Equal(t, map[string]interface{}{
		runtimeOptionSinkConfigTypeField: "kafka",
		runtimeOptionConfigsKey:          map[string]interface{}{"new": "value"},
	}, result["sinkConfig"])
	_, hasSource := result["sourceConfig"]
	assert.False(t, hasSource)
}

func TestSplitFunctionCustomRuntimeOptions(t *testing.T) {
	///nolint:lll
	raw := `{"foo":"bar","sinkConfig":{"sinkType":"kafka","configs":{"alpha":"1","beta":true}},"sourceConfig":{"sourceType":"kinesis","configs":{"gamma":2}}}`

	sanitized, sinkConfig, sinkPresent, sourceConfig, sourcePresent, err := splitFunctionCustomRuntimeOptions(raw)
	require.NoError(t, err)
	assert.True(t, sinkPresent)
	assert.Equal(t, map[string]interface{}{
		runtimeOptionSinkConfigTypeField: "kafka",
		runtimeOptionConfigsKey:          map[string]interface{}{"alpha": "1", "beta": "true"},
	}, sinkConfig)
	assert.True(t, sourcePresent)
	assert.Equal(t, map[string]interface{}{
		runtimeOptionSourceConfigTypeField: "kinesis",
		runtimeOptionConfigsKey:            map[string]interface{}{"gamma": "2"},
	}, sourceConfig)
	assert.JSONEq(t, `{"foo":"bar"}`, sanitized)
}

func TestSplitFunctionCustomRuntimeOptionsWithoutSinkConfig(t *testing.T) {
	raw := `{"foo":"bar"}`

	sanitized, sinkConfig, sinkPresent, sourceConfig, sourcePresent, err := splitFunctionCustomRuntimeOptions(raw)
	require.NoError(t, err)
	assert.False(t, sinkPresent)
	assert.Nil(t, sinkConfig)
	assert.False(t, sourcePresent)
	assert.Nil(t, sourceConfig)
	assert.JSONEq(t, `{"foo":"bar"}`, sanitized)
}

func functionInputSpec(topic string, overrides map[string]interface{}) map[string]interface{} {
	spec := map[string]interface{}{
		resourceFunctionInputSpecTopicKey:              topic,
		resourceFunctionInputSpecReceiverQueueSizeKey:  defaultFunctionReceiverQueueSize,
		resourceFunctionInputSpecSchemaTypeKey:         "",
		resourceFunctionInputSpecSerdeClassNameKey:     "",
		resourceFunctionInputSpecRegexPatternKey:       false,
		resourceFunctionInputSpecPoolMessagesKey:       false,
		resourceFunctionInputSpecSchemaPropertiesKey:   map[string]interface{}{},
		resourceFunctionInputSpecConsumerPropertiesKey: map[string]interface{}{},
	}

	for key, value := range overrides {
		spec[key] = value
	}

	return spec
}

func functionResourceData(t *testing.T, values map[string]interface{}) *schema.ResourceData {
	t.Helper()

	d := schema.TestResourceDataRaw(t, resourcePulsarFunction().Schema, map[string]interface{}{})
	for key, value := range values {
		require.NoError(t, d.Set(key, value))
	}

	return d
}

func functionInputSpecsInState(t *testing.T, d *schema.ResourceData) map[string]map[string]interface{} {
	t.Helper()

	set, ok := d.Get(resourceFunctionInputSpecsKey).(*schema.Set)
	require.True(t, ok)

	specs := map[string]map[string]interface{}{}
	for _, item := range set.List() {
		spec := item.(map[string]interface{})
		specs[spec[resourceFunctionInputSpecTopicKey].(string)] = spec
	}

	return specs
}

func TestMarshalFunctionInputSpecs(t *testing.T) {
	d := functionResourceData(t, map[string]interface{}{
		resourceFunctionTenantKey:    "public",
		resourceFunctionNamespaceKey: "default",
		resourceFunctionNameKey:      "function-1",
		resourceFunctionInputsKey:    []interface{}{"public/default/in-1", "public/default/in-2"},
		resourceFunctionInputSpecsKey: []interface{}{
			functionInputSpec("public/default/in-1", map[string]interface{}{
				resourceFunctionInputSpecReceiverQueueSizeKey:  100,
				resourceFunctionInputSpecSchemaTypeKey:         "avro",
				resourceFunctionInputSpecConsumerPropertiesKey: map[string]interface{}{"application": "billing"},
			}),
			functionInputSpec("public/default/in-3", map[string]interface{}{
				resourceFunctionInputSpecReceiverQueueSizeKey: 500,
				resourceFunctionInputSpecPoolMessagesKey:      true,
			}),
		},
	})

	functionConfig, err := marshalFunctionConfig(d)
	require.NoError(t, err)

	require.Len(t, functionConfig.InputSpecs, 2)
	assert.Equal(t, 100, functionConfig.InputSpecs["public/default/in-1"].ReceiverQueueSize)
	assert.Equal(t, "avro", functionConfig.InputSpecs["public/default/in-1"].SchemaType)
	assert.Equal(t, map[string]string{"application": "billing"},
		functionConfig.InputSpecs["public/default/in-1"].ConsumerProperties)
	assert.Equal(t, 500, functionConfig.InputSpecs["public/default/in-3"].ReceiverQueueSize)
	assert.True(t, functionConfig.InputSpecs["public/default/in-3"].PoolMessages)

	// Empty maps are dropped rather than sent as {}.
	assert.Nil(t, functionConfig.InputSpecs["public/default/in-1"].SchemaProperties)

	// in-1 is declared in input_specs, so it must not also be sent in inputs: validateUpdate()
	// would fold it back in with a default ConsumerConfig and discard the receiver queue size.
	assert.Equal(t, []string{"public/default/in-2"}, functionConfig.Inputs)
}

func TestMarshalFunctionInputSpecsAbsent(t *testing.T) {
	d := functionResourceData(t, map[string]interface{}{
		resourceFunctionTenantKey:    "public",
		resourceFunctionNamespaceKey: "default",
		resourceFunctionNameKey:      "function-1",
		resourceFunctionInputsKey:    []interface{}{"public/default/in-1"},
	})

	functionConfig, err := marshalFunctionConfig(d)
	require.NoError(t, err)

	assert.Nil(t, functionConfig.InputSpecs)
	assert.Equal(t, []string{"public/default/in-1"}, functionConfig.Inputs)
}

func TestMarshalFunctionInputSpecsExplicitZeroQueueSize(t *testing.T) {
	d := functionResourceData(t, map[string]interface{}{
		resourceFunctionInputSpecsKey: []interface{}{
			functionInputSpec("public/default/in-1", map[string]interface{}{
				resourceFunctionInputSpecReceiverQueueSizeKey: 0,
			}),
		},
	})

	functionConfig, err := marshalFunctionConfig(d)
	require.NoError(t, err)
	consumerConfig := functionConfig.InputSpecs["public/default/in-1"]
	assert.True(t, consumerConfig.HasReceiverQueueSize())
	assert.Zero(t, consumerConfig.ReceiverQueueSize)

	payload, err := json.Marshal(functionConfig)
	require.NoError(t, err)
	assert.Contains(t, string(payload), `"receiverQueueSize":0`)
}

func TestMarshalFunctionInputSpecsRemoveEveryLegacyOverlap(t *testing.T) {
	const defaultSerde = "org.apache.pulsar.functions.api.utils.DefaultSerDe"

	d := functionResourceData(t, map[string]interface{}{
		resourceFunctionInputsKey: []interface{}{
			"public/default/plain",
			"public/default/keep-input",
		},
		resourceFunctionTopicsPatternKey: "public/default/pattern-.*",
		resourceFunctionCustomSerdeInputsKey: map[string]interface{}{
			"public/default/serde":      defaultSerde,
			"public/default/keep-serde": defaultSerde,
		},
		resourceFunctionCustomSchemaInputsKey: map[string]interface{}{
			"public/default/schema":      `{"schemaType":"STRING"}`,
			"public/default/keep-schema": `{"schemaType":"STRING"}`,
		},
		resourceFunctionInputSpecsKey: []interface{}{
			functionInputSpec("public/default/plain", nil),
			functionInputSpec("public/default/pattern-.*", map[string]interface{}{
				resourceFunctionInputSpecRegexPatternKey: true,
			}),
			functionInputSpec("public/default/serde", map[string]interface{}{
				resourceFunctionInputSpecSerdeClassNameKey: defaultSerde,
			}),
			functionInputSpec("public/default/schema", map[string]interface{}{
				resourceFunctionInputSpecSchemaTypeKey: "STRING",
			}),
		},
	})

	functionConfig, err := marshalFunctionConfig(d)
	require.NoError(t, err)

	assert.Equal(t, []string{"public/default/keep-input"}, functionConfig.Inputs)
	assert.Nil(t, functionConfig.TopicsPattern)
	assert.Equal(t, map[string]string{
		"public/default/keep-serde": defaultSerde,
	}, functionConfig.CustomSerdeInputs)
	assert.Equal(t, map[string]string{
		"public/default/keep-schema": `{"schemaType":"STRING"}`,
	}, functionConfig.CustomSchemaInputs)
	assert.Equal(t, defaultSerde,
		functionConfig.InputSpecs["public/default/serde"].SerdeClassName)
	assert.Equal(t, "STRING", functionConfig.InputSpecs["public/default/schema"].SchemaType)
}

func TestFunctionInputSpecsRejectIncompleteLegacyOverlap(t *testing.T) {
	const topic = "public/default/in-1"

	tests := []struct {
		name   string
		values map[string]interface{}
		want   string
	}{
		{
			name: "queue-only input spec cannot replace custom serde",
			values: map[string]interface{}{
				resourceFunctionCustomSerdeInputsKey: map[string]interface{}{
					topic: "com.acme.Serde",
				},
				resourceFunctionInputSpecsKey: []interface{}{
					functionInputSpec(topic, map[string]interface{}{
						resourceFunctionInputSpecReceiverQueueSizeKey: 100,
					}),
				},
			},
			want: resourceFunctionCustomSerdeInputsKey,
		},
		{
			name: "queue-only input spec cannot replace custom schema",
			values: map[string]interface{}{
				resourceFunctionCustomSchemaInputsKey: map[string]interface{}{
					topic: `{"schemaType":"STRING"}`,
				},
				resourceFunctionInputSpecsKey: []interface{}{
					functionInputSpec(topic, map[string]interface{}{
						resourceFunctionInputSpecReceiverQueueSizeKey: 100,
					}),
				},
			},
			want: resourceFunctionCustomSchemaInputsKey,
		},
		{
			name: "different serde is rejected",
			values: map[string]interface{}{
				resourceFunctionCustomSerdeInputsKey: map[string]interface{}{
					topic: "com.acme.LegacySerde",
				},
				resourceFunctionInputSpecsKey: []interface{}{
					functionInputSpec(topic, map[string]interface{}{
						resourceFunctionInputSpecSerdeClassNameKey: "com.acme.NewSerde",
					}),
				},
			},
			want: resourceFunctionCustomSerdeInputsKey,
		},
		{
			name: "different schema is rejected",
			values: map[string]interface{}{
				resourceFunctionCustomSchemaInputsKey: map[string]interface{}{
					topic: `{"schemaType":"STRING"}`,
				},
				resourceFunctionInputSpecsKey: []interface{}{
					functionInputSpec(topic, map[string]interface{}{
						resourceFunctionInputSpecSchemaTypeKey: "AVRO",
					}),
				},
			},
			want: resourceFunctionCustomSchemaInputsKey,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := marshalFunctionConfig(functionResourceData(t, test.values))
			require.ErrorContains(t, err, test.want)

			res := resourcePulsarFunction()
			config := functionConfigWithBase(test.values)
			d := schema.TestResourceDataRaw(t, res.Schema, map[string]interface{}{})
			_, err = res.Diff(
				context.Background(),
				d.State(),
				terraform.NewResourceConfigRaw(config),
				nil,
			)
			require.ErrorContains(t, err, test.want)
		})
	}
}

func TestFunctionInputSpecsAllowsEquivalentLegacyOverlapDuringPlan(t *testing.T) {
	const topic = "public/default/in-1"
	const schemaInput = `{
  "schemaType": "STRING",
  "schemaProperties": {"encoding": "utf8"},
  "consumerProperties": {"application": "billing"}
}`

	res := resourcePulsarFunction()
	config := functionConfigWithBase(map[string]interface{}{
		resourceFunctionCustomSerdeInputsKey: map[string]interface{}{
			topic: "com.acme.Serde",
		},
		resourceFunctionCustomSchemaInputsKey: map[string]interface{}{
			"public/default/schema": schemaInput,
		},
		resourceFunctionInputSpecsKey: []interface{}{
			functionInputSpec(topic, map[string]interface{}{
				resourceFunctionInputSpecSerdeClassNameKey: "com.acme.Serde",
			}),
			functionInputSpec("public/default/schema", map[string]interface{}{
				resourceFunctionInputSpecSchemaTypeKey:         "STRING",
				resourceFunctionInputSpecSchemaPropertiesKey:   map[string]interface{}{"encoding": "utf8"},
				resourceFunctionInputSpecConsumerPropertiesKey: map[string]interface{}{"application": "billing"},
			}),
		},
	})

	d := schema.TestResourceDataRaw(t, res.Schema, map[string]interface{}{})
	_, err := res.Diff(context.Background(), d.State(), terraform.NewResourceConfigRaw(config), nil)
	require.NoError(t, err)
}

func TestFunctionLegacyInputConfigurationPlansCleanly(t *testing.T) {
	const (
		plainTopic  = "public/default/plain"
		serdeTopic  = "public/default/serde"
		schemaTopic = "public/default/schema"
	)

	config := functionConfigWithBase(map[string]interface{}{
		resourceFunctionInputsKey: []interface{}{plainTopic},
		resourceFunctionCustomSerdeInputsKey: map[string]interface{}{
			serdeTopic: "com.acme.Serde",
		},
		resourceFunctionCustomSchemaInputsKey: map[string]interface{}{
			schemaTopic: `{"schemaType":"STRING"}`,
		},
	})
	res := resourcePulsarFunction()
	d := schema.TestResourceDataRaw(t, res.Schema, config)
	d.SetId("public/default/function-1")
	require.NoError(t, d.Set(resourceFunctionSinkConfigKey, nil))
	require.NoError(t, d.Set(resourceFunctionSourceConfigKey, nil))

	// v0.13 persisted these legacy fields. v0.14 must not invent input_specs from their remote
	// ConsumerConfig entries and create a follow-up plan.
	require.NoError(t, unmarshalFunctionInputSpecs(utils.FunctionConfig{
		InputSpecs: map[string]utils.ConsumerConfig{
			plainTopic:  {},
			serdeTopic:  {SerdeClassName: "com.acme.Serde"},
			schemaTopic: {SchemaType: "STRING"},
		},
	}, d))

	diff, err := res.Diff(context.Background(), d.State(), terraform.NewResourceConfigRaw(config), nil)
	require.NoError(t, err)
	if diff != nil {
		assert.True(t, diff.Empty(), "legacy inputs re-planned: %#v", diff.Attributes)
	}
}

func TestUnmarshalFunctionInputSpecs(t *testing.T) {
	// The broker returns a spec for every input topic, including ones the configuration declares
	// through inputs or topics_pattern, and never returns inputs at all.
	functionConfig := utils.FunctionConfig{
		InputSpecs: map[string]utils.ConsumerConfig{
			"public/default/in-1": {
				ReceiverQueueSize:  100,
				SchemaType:         "avro",
				SchemaProperties:   map[string]string{},
				ConsumerProperties: map[string]string{"application": "billing"},
			},
			"public/default/in-2": {
				SchemaProperties:   map[string]string{},
				ConsumerProperties: map[string]string{},
			},
			"public/default/pattern-.*": {
				RegexPattern:       true,
				SchemaProperties:   map[string]string{},
				ConsumerProperties: map[string]string{},
			},
			"public/default/serde": {},
			"public/default/schema": {
				SchemaType: "STRING",
			},
		},
	}

	d := functionResourceData(t, map[string]interface{}{
		resourceFunctionInputsKey:        []interface{}{"public/default/in-1", "public/default/in-2"},
		resourceFunctionTopicsPatternKey: "public/default/pattern-.*",
		resourceFunctionCustomSerdeInputsKey: map[string]interface{}{
			"public/default/serde": "org.apache.pulsar.functions.api.utils.DefaultSerDe",
		},
		resourceFunctionCustomSchemaInputsKey: map[string]interface{}{
			"public/default/schema": `{"schemaType":"STRING"}`,
		},
		resourceFunctionInputSpecsKey: []interface{}{
			functionInputSpec("public/default/in-1", map[string]interface{}{
				resourceFunctionInputSpecReceiverQueueSizeKey: 100,
			}),
		},
	})

	require.NoError(t, unmarshalFunctionInputSpecs(functionConfig, d))

	specs := functionInputSpecsInState(t, d)

	// in-1 is declared in input_specs, so it is refreshed even though inputs also lists it.
	require.Contains(t, specs, "public/default/in-1")
	assert.Equal(t, 100, specs["public/default/in-1"][resourceFunctionInputSpecReceiverQueueSizeKey])
	assert.Equal(t, "avro", specs["public/default/in-1"][resourceFunctionInputSpecSchemaTypeKey])
	assert.Equal(t, map[string]interface{}{"application": "billing"},
		specs["public/default/in-1"][resourceFunctionInputSpecConsumerPropertiesKey])

	// in-2 is represented by inputs and the pattern by topics_pattern; surfacing either would be a
	// block the configuration never wrote, and therefore a permanent diff.
	assert.NotContains(t, specs, "public/default/in-2")
	assert.NotContains(t, specs, "public/default/pattern-.*")
	assert.NotContains(t, specs, "public/default/serde")
	assert.NotContains(t, specs, "public/default/schema")
}

func TestUnmarshalFunctionInputSpecsPreservesConsumerProperties(t *testing.T) {
	topic := "public/default/in-1"
	d := functionResourceData(t, map[string]interface{}{
		resourceFunctionInputSpecsKey: []interface{}{
			functionInputSpec(topic, map[string]interface{}{
				resourceFunctionInputSpecConsumerPropertiesKey: map[string]interface{}{
					"application": "billing",
				},
			}),
		},
	})

	functionConfig := utils.FunctionConfig{
		InputSpecs: map[string]utils.ConsumerConfig{
			topic: {
				ReceiverQueueSize:  defaultFunctionReceiverQueueSize,
				ConsumerProperties: map[string]string{},
			},
		},
	}

	require.NoError(t, unmarshalFunctionInputSpecs(functionConfig, d))

	specs := functionInputSpecsInState(t, d)
	assert.Equal(t, map[string]interface{}{"application": "billing"},
		specs[topic][resourceFunctionInputSpecConsumerPropertiesKey])
}

func TestUnmarshalFunctionInputSpecsOnImport(t *testing.T) {
	// On import nothing is in state yet, so every spec the broker returns is the function's
	// complete input configuration.
	functionConfig := utils.FunctionConfig{
		InputSpecs: map[string]utils.ConsumerConfig{
			"public/default/in-1": {ReceiverQueueSize: 100},
			"public/default/in-2": {},
		},
	}

	d := functionResourceData(t, map[string]interface{}{})
	require.NoError(t, unmarshalFunctionInputSpecs(functionConfig, d))

	specs := functionInputSpecsInState(t, d)
	assert.Len(t, specs, 2)
	assert.Equal(t, 100, specs["public/default/in-1"][resourceFunctionInputSpecReceiverQueueSizeKey])
	assert.Equal(t, defaultFunctionReceiverQueueSize,
		specs["public/default/in-2"][resourceFunctionInputSpecReceiverQueueSizeKey])
}

func TestUnmarshalFunctionInputSpecsExplicitZeroQueueSize(t *testing.T) {
	consumerConfig := utils.ConsumerConfig{}
	consumerConfig.SetReceiverQueueSize(0)

	d := functionResourceData(t, map[string]interface{}{})
	require.NoError(t, unmarshalFunctionInputSpecs(utils.FunctionConfig{
		InputSpecs: map[string]utils.ConsumerConfig{
			"public/default/in-1": consumerConfig,
		},
	}, d))

	specs := functionInputSpecsInState(t, d)
	assert.Zero(t, specs["public/default/in-1"][resourceFunctionInputSpecReceiverQueueSizeKey])
}

func TestEffectiveFunctionInputTopics(t *testing.T) {
	d := functionResourceData(t, map[string]interface{}{
		resourceFunctionInputsKey:        []interface{}{"public/default/in-1", "public/default/in-2"},
		resourceFunctionTopicsPatternKey: "public/default/pattern-.*",
		resourceFunctionCustomSerdeInputsKey: map[string]interface{}{
			"public/default/serde": "org.apache.pulsar.functions.api.utils.DefaultSerDe",
		},
		resourceFunctionCustomSchemaInputsKey: map[string]interface{}{
			"public/default/schema": `{"schemaType":"STRING"}`,
		},
		resourceFunctionInputSpecsKey: []interface{}{
			functionInputSpec("public/default/in-1", map[string]interface{}{
				resourceFunctionInputSpecReceiverQueueSizeKey: 100,
			}),
			functionInputSpec("public/default/in-3", nil),
		},
	})

	topics := effectiveFunctionInputTopics(
		d.Get(resourceFunctionInputsKey),
		d.Get(resourceFunctionTopicsPatternKey),
		d.Get(resourceFunctionCustomSerdeInputsKey),
		d.Get(resourceFunctionCustomSchemaInputsKey),
		d.Get(resourceFunctionInputSpecsKey),
	)

	assert.Equal(t, map[string]bool{
		"public/default/in-1":       false,
		"public/default/in-2":       false,
		"public/default/in-3":       false,
		"public/default/pattern-.*": true,
		"public/default/serde":      false,
		"public/default/schema":     false,
	}, topics)
}

func TestFunctionInputSpecsValidation(t *testing.T) {
	base := map[string]interface{}{
		resourceFunctionTenantKey:    "public",
		resourceFunctionNamespaceKey: "default",
		resourceFunctionNameKey:      "function-1",
	}

	tests := []struct {
		name  string
		specs []interface{}
		want  string
	}{
		{
			name: "duplicate topic keys",
			specs: []interface{}{
				map[string]interface{}{
					resourceFunctionInputSpecTopicKey:             "public/default/in-1",
					resourceFunctionInputSpecReceiverQueueSizeKey: 100,
				},
				map[string]interface{}{
					resourceFunctionInputSpecTopicKey:             "public/default/in-1",
					resourceFunctionInputSpecReceiverQueueSizeKey: 200,
				},
			},
			want: `input_specs contains duplicate key "public/default/in-1"`,
		},
		{
			name: "schema and serde are mutually exclusive",
			specs: []interface{}{
				map[string]interface{}{
					resourceFunctionInputSpecTopicKey:          "public/default/in-1",
					resourceFunctionInputSpecSchemaTypeKey:     "STRING",
					resourceFunctionInputSpecSerdeClassNameKey: "example.StringSerde",
				},
			},
			want: "cannot set both schema_type and serde_class_name",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := map[string]interface{}{}
			for key, value := range base {
				config[key] = value
			}
			config[resourceFunctionInputSpecsKey] = test.specs

			res := resourcePulsarFunction()
			d := schema.TestResourceDataRaw(t, res.Schema, map[string]interface{}{})
			_, err := res.Diff(
				context.Background(),
				d.State(),
				terraform.NewResourceConfigRaw(config),
				nil,
			)
			require.ErrorContains(t, err, test.want)
		})
	}
}

func functionInputSpecsDiff(t *testing.T, state, config map[string]interface{}) *terraform.InstanceDiff {
	t.Helper()

	res := resourcePulsarFunction()

	d := schema.TestResourceDataRaw(t, res.Schema, map[string]interface{}{})
	for key, value := range state {
		require.NoError(t, d.Set(key, value))
	}
	d.SetId("public/default/function-1")

	diff, err := res.Diff(context.Background(), d.State(), terraform.NewResourceConfigRaw(config), nil)
	require.NoError(t, err)
	require.NotNil(t, diff)

	return diff
}

// inputSpecsDiffContains reports whether any input_specs attribute is planned to become want. The
// in-place cases assert on this so that a change the SDK dropped entirely - diffSet returns early
// when the element hashes match - cannot pass as "no replacement needed".
func inputSpecsDiffContains(diff *terraform.InstanceDiff, want string) bool {
	for key, attr := range diff.Attributes {
		if strings.HasPrefix(key, resourceFunctionInputSpecsKey+".") && attr != nil && attr.New == want {
			return true
		}
	}

	return false
}

// Pulsar accepts consumer-setting changes in place but rejects changes to the set of input topics,
// so the provider must replace the function in exactly the second case and no other.
func TestFunctionInputSpecsForceNew(t *testing.T) {
	base := map[string]interface{}{
		resourceFunctionTenantKey:    "public",
		resourceFunctionNamespaceKey: "default",
		resourceFunctionNameKey:      "function-1",
	}

	withBase := func(values map[string]interface{}) map[string]interface{} {
		merged := map[string]interface{}{}
		for key, value := range base {
			merged[key] = value
		}
		for key, value := range values {
			merged[key] = value
		}
		return merged
	}

	tests := []struct {
		name        string
		state       map[string]interface{}
		config      map[string]interface{}
		requiresNew bool
		// planned value that must appear somewhere in the input_specs diff
		plans string
	}{
		{
			name: "adopting input_specs for a topic already in inputs updates in place",
			state: withBase(map[string]interface{}{
				resourceFunctionInputsKey: []interface{}{"public/default/in-1", "public/default/in-2"},
			}),
			config: withBase(map[string]interface{}{
				resourceFunctionInputsKey: []interface{}{"public/default/in-1", "public/default/in-2"},
				resourceFunctionInputSpecsKey: []interface{}{
					map[string]interface{}{
						resourceFunctionInputSpecTopicKey:             "public/default/in-1",
						resourceFunctionInputSpecReceiverQueueSizeKey: 100,
					},
				},
			}),
			requiresNew: false,
			plans:       "100",
		},
		{
			name: "moving a topic from inputs to input_specs updates in place",
			state: withBase(map[string]interface{}{
				resourceFunctionInputsKey: []interface{}{"public/default/in-1"},
			}),
			config: withBase(map[string]interface{}{
				resourceFunctionInputSpecsKey: []interface{}{
					map[string]interface{}{
						resourceFunctionInputSpecTopicKey:             "public/default/in-1",
						resourceFunctionInputSpecReceiverQueueSizeKey: 100,
					},
				},
			}),
			requiresNew: false,
			plans:       "100",
		},
		{
			name: "moving a pattern to input_specs updates in place",
			state: withBase(map[string]interface{}{
				resourceFunctionTopicsPatternKey: "public/default/in-.*",
			}),
			config: withBase(map[string]interface{}{
				resourceFunctionInputSpecsKey: []interface{}{
					map[string]interface{}{
						resourceFunctionInputSpecTopicKey:        "public/default/in-.*",
						resourceFunctionInputSpecRegexPatternKey: true,
					},
				},
			}),
			requiresNew: false,
		},
		{
			name: "moving custom serde input to input_specs updates in place",
			state: withBase(map[string]interface{}{
				resourceFunctionCustomSerdeInputsKey: map[string]interface{}{
					"public/default/in-1": "org.apache.pulsar.functions.api.utils.DefaultSerDe",
				},
			}),
			config: withBase(map[string]interface{}{
				resourceFunctionInputSpecsKey: []interface{}{
					map[string]interface{}{
						resourceFunctionInputSpecTopicKey: "public/default/in-1",
						resourceFunctionInputSpecSerdeClassNameKey: "org.apache.pulsar.functions.api.utils." +
							"DefaultSerDe",
					},
				},
			}),
			requiresNew: false,
		},
		{
			name: "changing a custom serde for the same topic updates in place",
			state: withBase(map[string]interface{}{
				resourceFunctionCustomSerdeInputsKey: map[string]interface{}{
					"public/default/in-1": "example.OldSerde",
				},
			}),
			config: withBase(map[string]interface{}{
				resourceFunctionCustomSerdeInputsKey: map[string]interface{}{
					"public/default/in-1": "example.NewSerde",
				},
			}),
			requiresNew: false,
		},
		{
			name: "tuning receiver_queue_size updates in place",
			state: withBase(map[string]interface{}{
				resourceFunctionInputSpecsKey: []interface{}{
					functionInputSpec("public/default/in-1", map[string]interface{}{
						resourceFunctionInputSpecReceiverQueueSizeKey: 100,
					}),
				},
			}),
			config: withBase(map[string]interface{}{
				resourceFunctionInputSpecsKey: []interface{}{
					map[string]interface{}{
						resourceFunctionInputSpecTopicKey:             "public/default/in-1",
						resourceFunctionInputSpecReceiverQueueSizeKey: 250,
					},
				},
			}),
			requiresNew: false,
			plans:       "250",
		},
		{
			name: "adding a new input topic replaces the function",
			state: withBase(map[string]interface{}{
				resourceFunctionInputSpecsKey: []interface{}{
					functionInputSpec("public/default/in-1", map[string]interface{}{
						resourceFunctionInputSpecReceiverQueueSizeKey: 100,
					}),
				},
			}),
			config: withBase(map[string]interface{}{
				resourceFunctionInputSpecsKey: []interface{}{
					map[string]interface{}{
						resourceFunctionInputSpecTopicKey:             "public/default/in-1",
						resourceFunctionInputSpecReceiverQueueSizeKey: 100,
					},
					map[string]interface{}{
						resourceFunctionInputSpecTopicKey: "public/default/in-2",
					},
				},
			}),
			requiresNew: true,
		},
		{
			name: "renaming an input topic replaces the function",
			state: withBase(map[string]interface{}{
				resourceFunctionInputSpecsKey: []interface{}{
					functionInputSpec("public/default/in-1", nil),
				},
			}),
			config: withBase(map[string]interface{}{
				resourceFunctionInputSpecsKey: []interface{}{
					map[string]interface{}{
						resourceFunctionInputSpecTopicKey: "public/default/renamed",
					},
				},
			}),
			requiresNew: true,
		},
		{
			name: "flipping regex_pattern replaces the function",
			state: withBase(map[string]interface{}{
				resourceFunctionInputSpecsKey: []interface{}{
					functionInputSpec("public/default/in-.*", nil),
				},
			}),
			config: withBase(map[string]interface{}{
				resourceFunctionInputSpecsKey: []interface{}{
					map[string]interface{}{
						resourceFunctionInputSpecTopicKey:        "public/default/in-.*",
						resourceFunctionInputSpecRegexPatternKey: true,
					},
				},
			}),
			requiresNew: true,
		},
		{
			name: "dropping an input topic replaces the function",
			state: withBase(map[string]interface{}{
				resourceFunctionInputSpecsKey: []interface{}{
					functionInputSpec("public/default/in-1", nil),
					functionInputSpec("public/default/in-2", nil),
				},
			}),
			config: withBase(map[string]interface{}{
				resourceFunctionInputSpecsKey: []interface{}{
					map[string]interface{}{
						resourceFunctionInputSpecTopicKey: "public/default/in-1",
					},
				},
			}),
			requiresNew: true,
		},
		{
			name: "renaming a plain input replaces the function",
			state: withBase(map[string]interface{}{
				resourceFunctionInputsKey: []interface{}{"public/default/in-1"},
			}),
			config: withBase(map[string]interface{}{
				resourceFunctionInputsKey: []interface{}{"public/default/in-2"},
			}),
			requiresNew: true,
		},
		{
			name: "renaming a custom schema input replaces the function",
			state: withBase(map[string]interface{}{
				resourceFunctionCustomSchemaInputsKey: map[string]interface{}{
					"public/default/in-1": `{"schemaType":"STRING"}`,
				},
			}),
			config: withBase(map[string]interface{}{
				resourceFunctionCustomSchemaInputsKey: map[string]interface{}{
					"public/default/in-2": `{"schemaType":"STRING"}`,
				},
			}),
			requiresNew: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			diff := functionInputSpecsDiff(t, test.state, test.config)
			assert.Equal(t, test.requiresNew, diff.RequiresNew())
			if test.plans != "" {
				assert.True(t, inputSpecsDiffContains(diff, test.plans),
					"expected the diff to plan %q somewhere under %s, got %v",
					test.plans, resourceFunctionInputSpecsKey, diff.Attributes)
			}
		})
	}
}
