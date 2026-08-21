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
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sinkInputSpec(topic string, overrides map[string]interface{}) map[string]interface{} {
	spec := map[string]interface{}{
		resourceSinkInputSpecsSubsetTopicKey:             topic,
		resourceSinkInputSpecsSubsetSchemaTypeKey:        "",
		resourceSinkInputSpecsSubsetSerdeClassNameKey:    "",
		resourceSinkInputSpecsSubsetIsRegexPatternKey:    false,
		resourceSinkInputSpecsSubsetReceiverQueueSizeKey: 0,
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
	assert.Empty(t, spec.SchemaType)
	assert.Empty(t, spec.SerdeClassName)
	assert.Nil(t, spec.SchemaProperties)
	assert.Nil(t, spec.ConsumerProperties)
}

func TestMarshalSinkInputSpecsNewFields(t *testing.T) {
	d := sinkResourceData(t, map[string]interface{}{
		resourceSinkTenantKey:    "public",
		resourceSinkNamespaceKey: "default",
		resourceSinkNameKey:      "sink-1",
		resourceSinkInputSpecsKey: []interface{}{
			sinkInputSpec("persistent://public/default/in-1", map[string]interface{}{
				resourceSinkInputSpecsSubsetPoolMessagesKey: true,
				resourceSinkInputSpecsSubsetSchemaPropertiesKey: map[string]interface{}{
					"schema-key": "schema-value",
				},
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
	assert.Equal(t, map[string]string{"schema-key": "schema-value"}, spec.SchemaProperties)
	assert.Equal(t, map[string]string{"application": "billing"}, spec.ConsumerProperties)
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
