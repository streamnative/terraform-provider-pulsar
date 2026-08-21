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
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/rest"
	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	"github.com/streamnative/terraform-provider-pulsar/bytesize"
)

const (
	resourceFunctionTenantKey               = "tenant"
	resourceFunctionNamespaceKey            = "namespace"
	resourceFunctionNameKey                 = "name"
	resourceFunctionJarKey                  = "jar"
	resourceFunctionPyKey                   = "py"
	resourceFunctionGoKey                   = "go"
	resourceFunctionClassNameKey            = "classname"
	resourceFunctionInputsKey               = "inputs"
	resourceFunctionInputSpecsKey           = "input_specs"
	resourceFunctionTopicsPatternKey        = "topics_pattern"
	resourceFunctionOutputKey               = "output"
	resourceFunctionParallelismKey          = "parallelism"
	resourceFunctionProcessingGuaranteesKey = "processing_guarantees"
	resourceFunctionSubscriptionNameKey     = "subscription_name"
	resourceFunctionSubscriptionPositionKey = "subscription_position"
	resourceFunctionCleanupSubscriptionKey  = "cleanup_subscription"
	resourceFunctionSkipToLatestKey         = "skip_to_latest"
	resourceFunctionForwardSourceMessageKey = "forward_source_message_property"
	resourceFunctionRetainOrderingKey       = "retain_ordering"
	resourceFunctionRetainKeyOrderingKey    = "retain_key_ordering"
	resourceFunctionAutoACKKey              = "auto_ack"
	resourceFunctionMaxMessageRetriesKey    = "max_message_retries"
	resourceFunctionDeadLetterTopicKey      = "dead_letter_topic"
	resourceFunctionLogTopicKey             = "log_topic"
	resourceFunctionTimeoutKey              = "timeout_ms"
	resourceFunctionInputTypeClassNameKey   = "input_type_classname"
	resourceFunctionOutputTypeClassNameKey  = "output_type_classname"
	resourceFunctionOutputSerdeClassNameKey = "output_serde_classname"
	resourceFunctionOutputSchemaTypeKey     = "output_schema_type"
	resourceFunctionCustomSerdeInputsKey    = "custom_serde_inputs"
	resourceFunctionCustomSchemaInputsKey   = "custom_schema_inputs"
	resourceFunctionCustomSchemaOutputsKey  = "custom_schema_outputs"
	resourceFunctionCustomRuntimeOptionsKey = "custom_runtime_options"
	resourceFunctionSecretsKey              = "secrets"
	resourceFunctionCPUKey                  = "cpu"
	resourceFunctionRAMKey                  = "ram_mb"
	resourceFunctionDiskKey                 = "disk_mb"
	resourceFunctionUserConfig              = "user_config"
	resourceFunctionSinkConfigKey           = "sink_config"
	resourceFunctionSourceConfigKey         = "source_config"
	resourceFunctionSinkConfigTypeKey       = "sink_type"
	resourceFunctionSourceConfigTypeKey     = "source_type"
	resourceFunctionRuntimeConfigConfigsKey = "configs"
)

// Attributes of a single `input_specs` block, mapping onto utils.ConsumerConfig.
const (
	resourceFunctionInputSpecTopicKey              = "key"
	resourceFunctionInputSpecReceiverQueueSizeKey  = "receiver_queue_size"
	resourceFunctionInputSpecSchemaTypeKey         = "schema_type"
	resourceFunctionInputSpecSerdeClassNameKey     = "serde_class_name"
	resourceFunctionInputSpecRegexPatternKey       = "is_regex_pattern"
	resourceFunctionInputSpecPoolMessagesKey       = "pool_messages"
	resourceFunctionInputSpecSchemaPropertiesKey   = "schema_properties"
	resourceFunctionInputSpecConsumerPropertiesKey = "consumer_properties"
)

const defaultFunctionReceiverQueueSize = 1000

var functionInputSourceKeys = []string{
	resourceFunctionInputsKey,
	resourceFunctionTopicsPatternKey,
	resourceFunctionCustomSerdeInputsKey,
	resourceFunctionCustomSchemaInputsKey,
	resourceFunctionInputSpecsKey,
}

// Producer configuration for the function's output topic. The attribute names mirror the ones
// pulsar_source already exposes, so the two resources read the same way.
const (
	resourceFunctionPCMaxPendingMsgKey                = "max_pending_messages"
	resourceFunctionPCMaxPendingMsgAcrossPartitionKey = "max_pending_messages_across_partitions"
	resourceFunctionPCUseThreadLocalProducersKey      = "use_thread_local_producers"
	resourceFunctionPCBatchBuilderKey                 = "batch_builder"
	resourceFunctionPCCompressionTypeKey              = "compression_type"
)

const (
	runtimeOptionSinkConfigKey         = "sinkConfig"
	runtimeOptionSourceConfigKey       = "sourceConfig"
	runtimeOptionSinkConfigTypeField   = "sinkType"
	runtimeOptionSourceConfigTypeField = "sourceType"
	runtimeOptionConfigsKey            = "configs"
)

type runtimeConfigDefinition struct {
	schemaKey      string
	typeSchemaKey  string
	runtimeKey     string
	runtimeTypeKey string
}

var (
	sinkRuntimeConfigDefinition = runtimeConfigDefinition{
		schemaKey:      resourceFunctionSinkConfigKey,
		typeSchemaKey:  resourceFunctionSinkConfigTypeKey,
		runtimeKey:     runtimeOptionSinkConfigKey,
		runtimeTypeKey: runtimeOptionSinkConfigTypeField,
	}
	sourceRuntimeConfigDefinition = runtimeConfigDefinition{
		schemaKey:      resourceFunctionSourceConfigKey,
		typeSchemaKey:  resourceFunctionSourceConfigTypeKey,
		runtimeKey:     runtimeOptionSourceConfigKey,
		runtimeTypeKey: runtimeOptionSourceConfigTypeField,
	}
)

var resourceFunctionDescriptions = make(map[string]string)

func init() {
	//nolint:lll
	resourceFunctionDescriptions = map[string]string{
		resourceFunctionTenantKey:               "The tenant of the function.",
		resourceFunctionNamespaceKey:            "The namespace of the function.",
		resourceFunctionNameKey:                 "The name of the function.",
		resourceFunctionJarKey:                  "The path to the jar file.",
		resourceFunctionPyKey:                   "The path to the python file.",
		resourceFunctionGoKey:                   "The path to the go file.",
		resourceFunctionClassNameKey:            "The class name of the function.",
		resourceFunctionInputsKey:               "The input topics of the function.",
		resourceFunctionInputSpecsKey:           "Per-topic consumer configuration for the function's input topics, such as the receiver queue size. A topic configured here does not need to be repeated in `inputs`; if it is, this block takes precedence.",
		resourceFunctionTopicsPatternKey:        "The input topics pattern of the function. The pattern is a regex expression. The function consumes from all topics matching the pattern.",
		resourceFunctionOutputKey:               "The output topic of the function.",
		resourceFunctionParallelismKey:          "The parallelism of the function.",
		resourceFunctionProcessingGuaranteesKey: "The processing guarantees (aka delivery semantics) applied to the function. Possible values are `ATMOST_ONCE`, `ATLEAST_ONCE`, and `EFFECTIVELY_ONCE`.",
		resourceFunctionSubscriptionNameKey:     "The subscription name of the function.",
		resourceFunctionSubscriptionPositionKey: "The subscription position. Supported values: `Latest`, `Earliest`.",
		resourceFunctionCleanupSubscriptionKey:  "Whether to clean up subscription when the function is deleted.",
		resourceFunctionSkipToLatestKey:         "Whether to skip to the latest position when the function is restarted after failure.",
		resourceFunctionForwardSourceMessageKey: "Whether to forward source message property to the function output message.",
		resourceFunctionRetainOrderingKey:       "Whether to retain ordering when the function is restarted after failure.",
		resourceFunctionRetainKeyOrderingKey:    "Whether to retain key ordering when the function is restarted after failure.",
		resourceFunctionAutoACKKey:              "Whether to automatically acknowledge messages processed by the function.",
		resourceFunctionMaxMessageRetriesKey:    "The maximum number of times that a message will be retried when the function is configured with `EFFECTIVELY_ONCE` processing guarantees.",
		resourceFunctionDeadLetterTopicKey:      "The dead letter topic of the function.",
		resourceFunctionLogTopicKey:             "The log topic of the function.",
		resourceFunctionTimeoutKey:              "The timeout of the function in milliseconds.",
		resourceFunctionInputTypeClassNameKey:   "The input type class name of the function. ",
		resourceFunctionOutputTypeClassNameKey:  "The output type class name of the function. ",
		resourceFunctionOutputSerdeClassNameKey: "The output serde class name of the function. ",
		resourceFunctionOutputSchemaTypeKey:     "The output schema type of the function.",
		resourceFunctionCustomSerdeInputsKey:    "The custom serde inputs of the function.",
		resourceFunctionCustomSchemaInputsKey:   "The custom schema inputs of the function.",
		resourceFunctionCustomSchemaOutputsKey:  "The custom schema outputs of the function.",
		resourceFunctionCustomRuntimeOptionsKey: "The custom runtime options of the function.",
		resourceFunctionSecretsKey:              "The secrets of the function.",
		resourceFunctionCPUKey:                  "The CPU that needs to be allocated per function instance",
		resourceFunctionRAMKey:                  "The RAM that need to be allocated per function instance",
		resourceFunctionDiskKey:                 "The disk that need to be allocated per function instance",
		resourceFunctionUserConfig:              "User-defined config key/values",
		resourceFunctionSinkConfigKey:           "Sink configuration key/values serialized into custom_runtime_options.",
		resourceFunctionSourceConfigKey:         "Source configuration key/values serialized into custom_runtime_options.",
		//nolint:lll
		resourceFunctionPCMaxPendingMsgKey: "The maximum size of a queue holding pending messages",
		//nolint:lll
		resourceFunctionPCMaxPendingMsgAcrossPartitionKey: "The maximum number of pending messages across partitions",
		resourceFunctionPCUseThreadLocalProducersKey:      "Whether to use thread local producers",
		//nolint:lll
		resourceFunctionPCBatchBuilderKey: "BatchBuilder provides two types of batch construction methods, DEFAULT and KEY_BASED.",
		//nolint:lll
		resourceFunctionPCCompressionTypeKey: "Set the compression type for the producer. By default, message payloads are not compressed. Supported compression types are: LZ4, ZLIB, ZSTD, SNAPPY and NONE",
	}
}

func resourcePulsarFunction() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourcePulsarFunctionCreate,
		ReadContext:   resourcePulsarFunctionRead,
		UpdateContext: resourcePulsarFunctionUpdate,
		DeleteContext: resourcePulsarFunctionDelete,
		CustomizeDiff: resourcePulsarFunctionCustomizeDiff,
		Description:   "Manages Pulsar Functions through the Functions Worker API.",
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				id := d.Id()

				parts := strings.Split(id, "/")
				if len(parts) != 3 {
					return nil, fmt.Errorf("id should be in tenant/namespace/function format, but get %s", id)
				}

				_ = d.Set(resourceFunctionTenantKey, parts[0])
				_ = d.Set(resourceFunctionNamespaceKey, parts[1])
				_ = d.Set(resourceFunctionNameKey, parts[2])

				diags := resourcePulsarFunctionRead(ctx, d, meta)
				if diags.HasError() {
					return nil, fmt.Errorf("import %q: %s", d.Id(), diags[0].Summary)
				}
				return []*schema.ResourceData{d}, nil
			},
		},
		Schema: map[string]*schema.Schema{
			resourceFunctionTenantKey: {
				Type:        schema.TypeString,
				Required:    true,
				Description: resourceFunctionDescriptions[resourceFunctionTenantKey],
			},
			resourceFunctionNamespaceKey: {
				Type:        schema.TypeString,
				Required:    true,
				Description: resourceFunctionDescriptions[resourceFunctionNamespaceKey],
			},
			resourceFunctionNameKey: {
				Type:        schema.TypeString,
				Required:    true,
				Description: resourceFunctionDescriptions[resourceFunctionNameKey],
			},
			resourceFunctionJarKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: resourceFunctionDescriptions[resourceFunctionJarKey],
			},
			resourceFunctionPyKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: resourceFunctionDescriptions[resourceFunctionPyKey],
			},
			resourceFunctionGoKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: resourceFunctionDescriptions[resourceFunctionGoKey],
			},
			resourceFunctionClassNameKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: resourceFunctionDescriptions[resourceFunctionClassNameKey],
			},
			resourceFunctionInputsKey: {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: resourceFunctionDescriptions[resourceFunctionInputsKey],
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			resourceFunctionInputSpecsKey: {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: resourceFunctionDescriptions[resourceFunctionInputSpecsKey],
				// Note the deliberate absence of ForceNew on the nested attributes. Changing any
				// of them rehashes the set element, which the SDK reads as a removal plus an
				// addition, so a nested ForceNew would replace the function on the very edits
				// Pulsar accepts in place. resourcePulsarFunctionCustomizeDiff decides
				// replacement instead, using the same rule the broker enforces.
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						resourceFunctionInputSpecTopicKey: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The input topic that this consumer configuration applies to.",
						},
						resourceFunctionInputSpecReceiverQueueSizeKey: {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  defaultFunctionReceiverQueueSize,
							//nolint:lll
							Description: "The consumer receiver queue size for this topic. Defaults to 1000, which buffers up to that many messages per function instance. Set to 0 to disable prefetch.",
							ValidateFunc: func(val interface{}, key string) ([]string, []error) {
								if v := val.(int); v < 0 {
									return nil, []error{
										fmt.Errorf("%s must be greater than or equal to 0, got %d", key, v),
									}
								}
								return nil, nil
							},
						},
						resourceFunctionInputSpecSchemaTypeKey: {
							Type:     schema.TypeString,
							Optional: true,
							//nolint:lll
							Description: "The schema type of this topic, either a builtin schema type such as `avro` or a Schema implementation class name.",
						},
						resourceFunctionInputSpecSerdeClassNameKey: {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "The serde class name of this topic. Cannot be set together with `schema_type`.",
						},
						resourceFunctionInputSpecRegexPatternKey: {
							Type:     schema.TypeBool,
							Optional: true,
							//nolint:lll
							Description: "Whether the topic is a regex pattern matching multiple topics. Cannot be changed in place; changing it replaces the function.",
						},
						resourceFunctionInputSpecPoolMessagesKey: {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "Whether the consumer pools messages for this topic.",
						},
						resourceFunctionInputSpecSchemaPropertiesKey: {
							Type:        schema.TypeMap,
							Optional:    true,
							Description: "Schema properties key/values for this topic.",
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						resourceFunctionInputSpecConsumerPropertiesKey: {
							Type:     schema.TypeMap,
							Optional: true,
							//nolint:lll
							Description: "Consumer properties key/values for this topic. Pulsar 4.0.x does not return this field on read, so the provider preserves the configured value in state; import cannot recover existing consumer properties.",
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			resourceFunctionTopicsPatternKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: resourceFunctionDescriptions[resourceFunctionTopicsPatternKey],
			},
			resourceFunctionOutputKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: resourceFunctionDescriptions[resourceFunctionOutputKey],
			},
			resourceFunctionParallelismKey: {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: resourceFunctionDescriptions[resourceFunctionParallelismKey],
			},
			resourceFunctionProcessingGuaranteesKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: resourceFunctionDescriptions[resourceFunctionProcessingGuaranteesKey],
			},
			resourceFunctionSubscriptionNameKey: {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: resourceFunctionDescriptions[resourceFunctionSubscriptionNameKey],
			},
			resourceFunctionSubscriptionPositionKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: resourceFunctionDescriptions[resourceFunctionSubscriptionPositionKey],
				ValidateFunc: func(val interface{}, key string) ([]string, []error) {
					v := val.(string)
					subscriptionPositionSupported := []string{
						SubscriptionPositionEarliest,
						SubscriptionPositionLatest,
					}

					found := false
					for _, item := range subscriptionPositionSupported {
						if v == item {
							found = true
							break
						}
					}
					if !found {
						return nil, []error{
							fmt.Errorf("%s is unsupported, shold be one of %s", v,
								strings.Join(subscriptionPositionSupported, ",")),
						}
					}

					return nil, nil
				},
			},
			resourceFunctionCleanupSubscriptionKey: {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: resourceFunctionDescriptions[resourceFunctionCleanupSubscriptionKey],
			},
			resourceFunctionSkipToLatestKey: {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: resourceFunctionDescriptions[resourceFunctionSkipToLatestKey],
			},
			resourceFunctionForwardSourceMessageKey: {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: resourceFunctionDescriptions[resourceFunctionForwardSourceMessageKey],
			},
			resourceFunctionRetainOrderingKey: {
				Type:        schema.TypeBool,
				ForceNew:    true,
				Optional:    true,
				Description: resourceFunctionDescriptions[resourceFunctionRetainOrderingKey],
			},
			resourceFunctionRetainKeyOrderingKey: {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: resourceFunctionDescriptions[resourceFunctionRetainKeyOrderingKey],
			},
			resourceFunctionAutoACKKey: {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: resourceFunctionDescriptions[resourceFunctionAutoACKKey],
			},
			resourceFunctionMaxMessageRetriesKey: {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: resourceFunctionDescriptions[resourceFunctionMaxMessageRetriesKey],
			},
			resourceFunctionDeadLetterTopicKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: resourceFunctionDescriptions[resourceFunctionDeadLetterTopicKey],
			},
			resourceFunctionLogTopicKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: resourceFunctionDescriptions[resourceFunctionLogTopicKey],
			},
			resourceFunctionTimeoutKey: {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: resourceFunctionDescriptions[resourceFunctionTimeoutKey],
			},
			resourceFunctionInputTypeClassNameKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: resourceFunctionDescriptions[resourceFunctionInputTypeClassNameKey],
			},
			resourceFunctionOutputTypeClassNameKey: {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				Description: resourceFunctionDescriptions[resourceFunctionOutputTypeClassNameKey],
			},
			resourceFunctionOutputSerdeClassNameKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: resourceFunctionDescriptions[resourceFunctionOutputSerdeClassNameKey],
			},
			resourceFunctionOutputSchemaTypeKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: resourceFunctionDescriptions[resourceFunctionOutputSchemaTypeKey],
			},
			resourceFunctionCustomSerdeInputsKey: {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: resourceFunctionDescriptions[resourceFunctionCustomSerdeInputsKey],
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			resourceFunctionCustomSchemaInputsKey: {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: resourceFunctionDescriptions[resourceFunctionCustomSchemaInputsKey],
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			resourceFunctionCustomSchemaOutputsKey: {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: resourceFunctionDescriptions[resourceFunctionCustomSchemaOutputsKey],
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			resourceFunctionCustomRuntimeOptionsKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: resourceFunctionDescriptions[resourceFunctionCustomRuntimeOptionsKey],
			},
			resourceFunctionSecretsKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: resourceFunctionDescriptions[resourceFunctionSecretsKey],
				ValidateFunc: func(val interface{}, key string) ([]string, []error) {
					v := val.(string)
					_, err := json.Marshal(v)
					if err != nil {
						return nil, []error{
							fmt.Errorf("cannot marshal %s: %s", v, err.Error()),
						}
					}
					return nil, nil
				},
			},
			resourceFunctionCPUKey: {
				Type:        schema.TypeFloat,
				Optional:    true,
				Computed:    true,
				Description: resourceFunctionDescriptions[resourceFunctionCPUKey],
			},
			resourceFunctionRAMKey: {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: resourceFunctionDescriptions[resourceFunctionRAMKey],
			},
			resourceFunctionDiskKey: {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: resourceFunctionDescriptions[resourceFunctionDiskKey],
			},
			resourceFunctionPCMaxPendingMsgKey: {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: resourceFunctionDescriptions[resourceFunctionPCMaxPendingMsgKey],
			},
			resourceFunctionPCMaxPendingMsgAcrossPartitionKey: {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: resourceFunctionDescriptions[resourceFunctionPCMaxPendingMsgAcrossPartitionKey],
			},
			resourceFunctionPCUseThreadLocalProducersKey: {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: resourceFunctionDescriptions[resourceFunctionPCUseThreadLocalProducersKey],
			},
			resourceFunctionPCBatchBuilderKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: resourceFunctionDescriptions[resourceFunctionPCBatchBuilderKey],
			},
			resourceFunctionPCCompressionTypeKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: resourceFunctionDescriptions[resourceFunctionPCCompressionTypeKey],
			},
			resourceFunctionUserConfig: {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: resourceFunctionDescriptions[resourceFunctionUserConfig],
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			resourceFunctionSinkConfigKey: {
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				MaxItems:    1,
				Description: resourceFunctionDescriptions[resourceFunctionSinkConfigKey],
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						resourceFunctionSinkConfigTypeKey: {
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: "Sink implementation identifier.",
						},
						resourceFunctionRuntimeConfigConfigsKey: {
							Type:        schema.TypeMap,
							Optional:    true,
							Computed:    true,
							Description: "Sink-specific key/value options.",
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			resourceFunctionSourceConfigKey: {
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				MaxItems:    1,
				Description: resourceFunctionDescriptions[resourceFunctionSourceConfigKey],
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						resourceFunctionSourceConfigTypeKey: {
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: "Source implementation identifier.",
						},
						resourceFunctionRuntimeConfigConfigsKey: {
							Type:        schema.TypeMap,
							Optional:    true,
							Computed:    true,
							Description: "Source-specific key/value options.",
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
		},
	}
}

// resourcePulsarFunctionCustomizeDiff validates input_specs and decides when a change to any input
// representation requires replacing the function.
//
// Pulsar's FunctionConfigUtils.validateUpdate() rejects any input topic the existing function does
// not already consume ("Input Topics cannot be altered") and any change to a topic's isRegexPattern
// flag, but accepts every other consumer setting - receiverQueueSize included. So replacement is
// keyed on the effective set of input topics rather than on any one schema attribute changing.
//
// The comparison includes every legacy input representation. Moving a topic from any legacy field
// into input_specs leaves that effective set untouched and must remain an in-place update.
func resourcePulsarFunctionCustomizeDiff(_ context.Context, diff *schema.ResourceDiff, _ interface{}) error {
	newSpecs := diff.Get(resourceFunctionInputSpecsKey)
	if err := validateFunctionInputSpecs(newSpecs); err != nil {
		return err
	}

	if diff.Id() == "" {
		// On create there is nothing to replace.
		return nil
	}

	inputChanged := false
	for _, key := range functionInputSourceKeys {
		if diff.HasChange(key) {
			inputChanged = true
			break
		}
	}
	if !inputChanged {
		return nil
	}

	oldInputs, newInputs := diff.GetChange(resourceFunctionInputsKey)
	oldPattern, newPattern := diff.GetChange(resourceFunctionTopicsPatternKey)
	oldCustomSerde, newCustomSerde := diff.GetChange(resourceFunctionCustomSerdeInputsKey)
	oldCustomSchema, newCustomSchema := diff.GetChange(resourceFunctionCustomSchemaInputsKey)
	oldSpecs, newSpecs := diff.GetChange(resourceFunctionInputSpecsKey)

	oldTopics := effectiveFunctionInputTopics(
		oldInputs, oldPattern, oldCustomSerde, oldCustomSchema, oldSpecs,
	)
	newTopics := effectiveFunctionInputTopics(
		newInputs, newPattern, newCustomSerde, newCustomSchema, newSpecs,
	)

	if len(oldTopics) != len(newTopics) {
		return forceNewFunctionInputTopology(diff, oldSpecs, newSpecs)
	}
	for topic, regexPattern := range newTopics {
		oldRegexPattern, ok := oldTopics[topic]
		if !ok || oldRegexPattern != regexPattern {
			return forceNewFunctionInputTopology(diff, oldSpecs, newSpecs)
		}
	}

	return nil
}

func validateFunctionInputSpecs(inputSpecs interface{}) error {
	set, ok := inputSpecs.(*schema.Set)
	if !ok || set.Len() == 0 {
		return nil
	}

	seenTopics := make(map[string]bool, set.Len())
	for _, item := range set.List() {
		spec, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		topic, _ := spec[resourceFunctionInputSpecTopicKey].(string)
		if topic == "" {
			// The SDK can include an empty placeholder while diffing TypeSet elements. The nested
			// Required schema validates user configuration, so ignore that internal value here.
			continue
		}
		if seenTopics[topic] {
			return fmt.Errorf("%s contains duplicate key %q", resourceFunctionInputSpecsKey, topic)
		}
		seenTopics[topic] = true

		schemaType, _ := spec[resourceFunctionInputSpecSchemaTypeKey].(string)
		serdeClassName, _ := spec[resourceFunctionInputSpecSerdeClassNameKey].(string)
		if schemaType != "" && serdeClassName != "" {
			return fmt.Errorf("%s %q cannot set both %s and %s",
				resourceFunctionInputSpecsKey,
				topic,
				resourceFunctionInputSpecSchemaTypeKey,
				resourceFunctionInputSpecSerdeClassNameKey,
			)
		}
	}

	return nil
}

func forceNewFunctionInputTopology(diff *schema.ResourceDiff, oldSpecs, newSpecs interface{}) error {
	if diff.HasChange(resourceFunctionInputSpecsKey) {
		return forceNewFunctionInputSpecs(diff, oldSpecs, newSpecs)
	}

	if diff.HasChange(resourceFunctionInputsKey) {
		return forceNewFunctionInputSet(diff, resourceFunctionInputsKey)
	}

	if diff.HasChange(resourceFunctionTopicsPatternKey) {
		return diff.ForceNew(resourceFunctionTopicsPatternKey)
	}

	for _, key := range []string{
		resourceFunctionCustomSerdeInputsKey,
		resourceFunctionCustomSchemaInputsKey,
	} {
		if diff.HasChange(key) {
			return forceNewFunctionInputMap(diff, key)
		}
	}

	return errors.New("input topology changed without an input attribute diff")
}

func forceNewFunctionInputSet(diff *schema.ResourceDiff, key string) error {
	if err := diff.ForceNew(key); err != nil {
		return err
	}

	oldValue, newValue := diff.GetChange(key)
	for _, value := range []interface{}{oldValue, newValue} {
		set, ok := value.(*schema.Set)
		if !ok {
			continue
		}
		for _, item := range set.List() {
			itemKey := fmt.Sprintf("%s.%d", key, set.F(item))
			if diff.HasChange(itemKey) {
				return diff.ForceNew(itemKey)
			}
		}
	}

	return nil
}

func forceNewFunctionInputMap(diff *schema.ResourceDiff, key string) error {
	if err := diff.ForceNew(key); err != nil {
		return err
	}

	oldValue, newValue := diff.GetChange(key)
	mapKeys := functionInputMapKeys(oldValue)
	for topic := range functionInputMapKeys(newValue) {
		mapKeys[topic] = true
	}
	for topic := range mapKeys {
		itemKey := key + "." + topic
		if diff.HasChange(itemKey) {
			return diff.ForceNew(itemKey)
		}
	}

	return nil
}

// forceNewFunctionInputSpecs flags an input_specs change as requiring replacement.
//
// A set-level ForceNew is not enough on its own. schemaMap.diffSet only carries it into the diff
// through the "input_specs.#" count attribute, and it emits that attribute solely when the number of
// elements changes. Renaming a topic or flipping regex_pattern leaves the count identical, so the
// replacement would be silently dropped. Flag the nested attribute that actually changed as well,
// which does reach the diff.
func forceNewFunctionInputSpecs(diff *schema.ResourceDiff, oldSpecs, newSpecs interface{}) error {
	if err := diff.ForceNew(resourceFunctionInputSpecsKey); err != nil {
		return err
	}

	for _, attribute := range []string{
		resourceFunctionInputSpecTopicKey,
		resourceFunctionInputSpecRegexPatternKey,
	} {
		for _, specs := range []interface{}{oldSpecs, newSpecs} {
			set, ok := specs.(*schema.Set)
			if !ok {
				continue
			}

			for _, item := range set.List() {
				key := fmt.Sprintf("%s.%d.%s", resourceFunctionInputSpecsKey, set.F(item), attribute)
				if !diff.HasChange(key) {
					continue
				}
				if err := diff.ForceNew(key); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// effectiveFunctionInputTopics maps every input topic the function consumes to its regex-pattern
// flag. Values are applied in Pulsar's create-path order, with input_specs last so it is the
// provider's canonical representation when a topic appears in more than one field.
func effectiveFunctionInputTopics(
	inputs, topicsPattern, customSerdeInputs, customSchemaInputs, inputSpecs interface{},
) map[string]bool {
	topics := map[string]bool{}

	if set, ok := inputs.(*schema.Set); ok {
		for _, item := range set.List() {
			if topic, ok := item.(string); ok && topic != "" {
				topics[topic] = false
			}
		}
	}

	if pattern, ok := topicsPattern.(string); ok && pattern != "" {
		topics[pattern] = true
	}

	for topic := range functionInputMapKeys(customSerdeInputs) {
		topics[topic] = false
	}
	for topic := range functionInputMapKeys(customSchemaInputs) {
		topics[topic] = false
	}

	// Applied last: an input_specs block wins over every legacy representation.
	for topic, consumerConfig := range functionInputSpecsFromSchema(inputSpecs) {
		topics[topic] = consumerConfig.RegexPattern
	}

	return topics
}

func functionInputMapKeys(value interface{}) map[string]bool {
	keys := map[string]bool{}

	switch values := value.(type) {
	case map[string]interface{}:
		for key := range values {
			if key != "" {
				keys[key] = true
			}
		}
	case map[string]string:
		for key := range values {
			if key != "" {
				keys[key] = true
			}
		}
	}

	return keys
}

// functionInputSpecsFromSchema converts an input_specs set into the map shape the admin API expects.
func functionInputSpecsFromSchema(inputSpecs interface{}) map[string]utils.ConsumerConfig {
	set, ok := inputSpecs.(*schema.Set)
	if !ok || set.Len() == 0 {
		return nil
	}

	specs := make(map[string]utils.ConsumerConfig, set.Len())
	for _, item := range set.List() {
		spec, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		topic, _ := spec[resourceFunctionInputSpecTopicKey].(string)
		if topic == "" {
			continue
		}

		consumerConfig := utils.ConsumerConfig{}
		if v, ok := spec[resourceFunctionInputSpecSchemaTypeKey].(string); ok {
			consumerConfig.SchemaType = v
		}
		if v, ok := spec[resourceFunctionInputSpecSerdeClassNameKey].(string); ok {
			consumerConfig.SerdeClassName = v
		}
		if v, ok := spec[resourceFunctionInputSpecRegexPatternKey].(bool); ok {
			consumerConfig.RegexPattern = v
		}
		if v, ok := spec[resourceFunctionInputSpecReceiverQueueSizeKey].(int); ok {
			consumerConfig.SetReceiverQueueSize(v)
		}
		if v, ok := spec[resourceFunctionInputSpecPoolMessagesKey].(bool); ok {
			consumerConfig.PoolMessages = v
		}
		consumerConfig.SchemaProperties = functionStringMap(spec[resourceFunctionInputSpecSchemaPropertiesKey])
		consumerConfig.ConsumerProperties = functionStringMap(spec[resourceFunctionInputSpecConsumerPropertiesKey])

		specs[topic] = consumerConfig
	}

	if len(specs) == 0 {
		return nil
	}

	return specs
}

// functionStringMap narrows a schema.TypeMap value to map[string]string, returning nil when empty so
// the field is omitted from the request payload.
func functionStringMap(value interface{}) map[string]string {
	interMap, ok := value.(map[string]interface{})
	if !ok || len(interMap) == 0 {
		return nil
	}

	stringMap := make(map[string]string, len(interMap))
	for key, item := range interMap {
		stringMap[key], _ = item.(string)
	}

	return stringMap
}

func functionLegacyInputMap(
	value interface{}, inputSpecs map[string]utils.ConsumerConfig,
) map[string]string {
	stringMap := functionStringMap(value)
	for topic := range inputSpecs {
		delete(stringMap, topic)
	}
	if len(stringMap) == 0 {
		return nil
	}

	return stringMap
}

func resourcePulsarFunctionRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getV3ClientFromMeta(meta).Functions()

	tenant := d.Get(resourceFunctionTenantKey).(string)
	namespace := d.Get(resourceFunctionNamespaceKey).(string)
	name := d.Get(resourceFunctionNameKey).(string)

	d.SetId(fmt.Sprintf("%s/%s/%s", tenant, namespace, name))

	functionConfig, err := client.GetFunction(tenant, namespace, name)
	if err != nil {
		if cliErr, ok := err.(rest.Error); ok && cliErr.Code == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(errors.Wrapf(err, "failed to get function %s", d.Id()))
	}

	err = unmarshalFunctionConfig(functionConfig, d)
	if err != nil {
		tflog.Debug(ctx, fmt.Sprintf("@@@Read function: %v", err))
		return diag.Errorf("ERROR_UNMARSHAL_FUNCTION_CONFIG: %v", err)
	}

	return nil
}

func resourcePulsarFunctionCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getV3ClientFromMeta(meta).Functions()

	functionConfig, err := marshalFunctionConfig(d)
	if err != nil {
		return diag.FromErr(err)
	}

	var archive string
	switch {
	case functionConfig.Jar != nil:
		archive = *functionConfig.Jar
	case functionConfig.Py != nil:
		archive = *functionConfig.Py
	case functionConfig.Go != nil:
		archive = *functionConfig.Go
	}

	if isPackageURLSupported(archive) {
		err = client.CreateFuncWithURL(functionConfig, archive)
	} else {
		err = client.CreateFunc(functionConfig, archive)
	}
	if err != nil {
		tflog.Debug(ctx, fmt.Sprintf("@@@Create function: %v", err))
		return diag.Errorf("ERROR_CREATE_FUNCTION: %v", err)
	}
	tflog.Debug(ctx, "@@@Create function: success")

	return resourcePulsarFunctionRead(ctx, d, meta)
}

func resourcePulsarFunctionUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getV3ClientFromMeta(meta).Functions()

	functionConfig, err := marshalFunctionConfig(d)
	if err != nil {
		return diag.FromErr(err)
	}

	var archive string
	switch {
	case functionConfig.Jar != nil:
		archive = *functionConfig.Jar
	case functionConfig.Py != nil:
		archive = *functionConfig.Py
	case functionConfig.Go != nil:
		archive = *functionConfig.Go
	}

	updateOptions := &utils.UpdateOptions{
		UpdateAuthData: true,
	}
	if isPackageURLSupported(archive) {
		err = client.UpdateFunctionWithURL(functionConfig, archive, updateOptions)
	} else {
		err = client.UpdateFunction(functionConfig, archive, updateOptions)
	}
	if err != nil {
		return diag.FromErr(err)
	}

	return resourcePulsarFunctionRead(ctx, d, meta)
}

func resourcePulsarFunctionDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getV3ClientFromMeta(meta).Functions()

	tenant := d.Get(resourceFunctionTenantKey).(string)
	namespace := d.Get(resourceFunctionNamespaceKey).(string)
	name := d.Get(resourceFunctionNameKey).(string)

	return diag.FromErr(client.DeleteFunction(tenant, namespace, name))
}

func marshalFunctionConfig(d *schema.ResourceData) (*utils.FunctionConfig, error) {
	functionConfig := &utils.FunctionConfig{}

	if inter, ok := d.GetOk(resourceFunctionTenantKey); ok {
		functionConfig.Tenant = inter.(string)
	}

	if inter, ok := d.GetOk(resourceFunctionNamespaceKey); ok {
		functionConfig.Namespace = inter.(string)
	}

	if inter, ok := d.GetOk(resourceFunctionNameKey); ok {
		functionConfig.Name = inter.(string)
	}

	inputSpecs := functionInputSpecsFromSchema(d.Get(resourceFunctionInputSpecsKey))
	if len(inputSpecs) != 0 {
		functionConfig.InputSpecs = inputSpecs
	}

	if inter, ok := d.GetOk(resourceFunctionInputsKey); ok {
		inputsSet := inter.(*schema.Set)
		var inputs []string

		for _, item := range inputsSet.List() {
			topic := item.(string)

			// A topic carried by an input_specs block is fully described there, and listing it in
			// both places is actively harmful on update: FunctionConfigUtils.validateUpdate() folds
			// inputs into the inputSpecs map with a default ConsumerConfig *before* reading that map
			// back, so the topic's consumer settings would be discarded on every apply. (The create
			// path applies inputSpecs last and does not have this problem, hence the asymmetry.)
			if _, ok := inputSpecs[topic]; ok {
				continue
			}

			inputs = append(inputs, topic)
		}

		functionConfig.Inputs = inputs
	}

	if inter, ok := d.GetOk(resourceFunctionOutputKey); ok {
		functionConfig.Output = inter.(string)
	}

	if inter, ok := d.GetOk(resourceFunctionTopicsPatternKey); ok {
		pattern := inter.(string)
		if _, isInputSpec := inputSpecs[pattern]; !isInputSpec {
			functionConfig.TopicsPattern = &pattern
		}
	}

	if inter, ok := d.GetOk(resourceFunctionJarKey); ok {
		jar := inter.(string)
		functionConfig.Jar = &jar
	}

	if inter, ok := d.GetOk(resourceFunctionPyKey); ok {
		py := inter.(string)
		functionConfig.Py = &py
	}

	if inter, ok := d.GetOk(resourceFunctionGoKey); ok {
		goLang := inter.(string)
		functionConfig.Go = &goLang
	}

	if inter, ok := d.GetOk(resourceFunctionClassNameKey); ok {
		functionConfig.ClassName = inter.(string)
	}

	if inter, ok := d.GetOk(resourceFunctionParallelismKey); ok {
		functionConfig.Parallelism = inter.(int)
	}

	if inter, ok := d.GetOk(resourceFunctionProcessingGuaranteesKey); ok {
		functionConfig.ProcessingGuarantees = inter.(string)
	}

	if inter, ok := d.GetOk(resourceFunctionSubscriptionNameKey); ok {
		functionConfig.SubName = inter.(string)
	}

	if inter, ok := d.GetOk(resourceFunctionSubscriptionPositionKey); ok {
		functionConfig.SubscriptionPosition = inter.(string)
	}

	if inter, ok := d.GetOk(resourceFunctionCleanupSubscriptionKey); ok {
		functionConfig.CleanupSubscription = inter.(bool)
	}

	if inter, ok := d.GetOk(resourceFunctionSkipToLatestKey); ok {
		functionConfig.SkipToLatest = inter.(bool)
	}

	if inter, ok := d.GetOk(resourceFunctionForwardSourceMessageKey); ok {
		functionConfig.ForwardSourceMessageProperty = inter.(bool)
	}

	if inter, ok := d.GetOk(resourceFunctionRetainOrderingKey); ok {
		functionConfig.RetainOrdering = inter.(bool)
	}

	if inter, ok := d.GetOk(resourceFunctionRetainKeyOrderingKey); ok {
		functionConfig.RetainKeyOrdering = inter.(bool)
	}

	if inter, ok := d.GetOk(resourceFunctionAutoACKKey); ok {
		functionConfig.AutoAck = inter.(bool)
	}

	if inter, ok := d.GetOk(resourceFunctionMaxMessageRetriesKey); ok {
		maxMessageRetries := inter.(int)
		functionConfig.MaxMessageRetries = &maxMessageRetries
	}

	if inter, ok := d.GetOk(resourceFunctionDeadLetterTopicKey); ok {
		functionConfig.DeadLetterTopic = inter.(string)
	}

	if inter, ok := d.GetOk(resourceFunctionLogTopicKey); ok {
		functionConfig.LogTopic = inter.(string)
	}

	if inter, ok := d.GetOk(resourceFunctionTimeoutKey); ok {
		timeout := int64(inter.(int))
		functionConfig.TimeoutMs = &timeout
	}

	if inter, ok := d.GetOk(resourceFunctionInputTypeClassNameKey); ok {
		functionConfig.InputTypeClassName = inter.(string)
	}

	if inter, ok := d.GetOk(resourceFunctionOutputTypeClassNameKey); ok {
		functionConfig.OutputTypeClassName = inter.(string)
	}

	if inter, ok := d.GetOk(resourceFunctionOutputSerdeClassNameKey); ok {
		functionConfig.OutputSerdeClassName = inter.(string)
	}

	if inter, ok := d.GetOk(resourceFunctionOutputSchemaTypeKey); ok {
		functionConfig.OutputSchemaType = inter.(string)
	}

	if inter, ok := d.GetOk(resourceFunctionCustomSerdeInputsKey); ok {
		functionConfig.CustomSerdeInputs = functionLegacyInputMap(inter, inputSpecs)
	}

	if inter, ok := d.GetOk(resourceFunctionCustomSchemaInputsKey); ok {
		functionConfig.CustomSchemaInputs = functionLegacyInputMap(inter, inputSpecs)
	}

	if inter, ok := d.GetOk(resourceFunctionCustomSchemaOutputsKey); ok {
		interMap := inter.(map[string]interface{})
		stringMap := make(map[string]string, len(interMap))

		for key, value := range interMap {
			stringMap[key] = value.(string)
		}

		functionConfig.CustomSchemaOutputs = stringMap
	}
	customRuntimeOptions, err := buildFunctionCustomRuntimeOptions(d)
	if err != nil {
		return nil, err
	}
	if customRuntimeOptions != "" {
		functionConfig.CustomRuntimeOptions = customRuntimeOptions
	}

	if inter, ok := d.GetOk(resourceFunctionSecretsKey); ok {
		var secrets map[string]interface{}
		secretsJSON := inter.(string)

		err := json.Unmarshal([]byte(secretsJSON), &secrets)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot unmarshal the secrets: %s", secretsJSON)
		}

		functionConfig.Secrets = secrets
	}

	resources := utils.NewDefaultResources()

	if inter, ok := d.GetOk(resourceFunctionCPUKey); ok {
		value := inter.(float64)
		resources.CPU = value
	}
	if inter, ok := d.GetOk(resourceFunctionRAMKey); ok {
		value := bytesize.FormMegaBytes(uint64(inter.(int))).ToBytes()
		resources.RAM = int64(value)
	}
	if inter, ok := d.GetOk(resourceFunctionDiskKey); ok {
		value := bytesize.FormMegaBytes(uint64(inter.(int))).ToBytes()
		resources.Disk = int64(value)
	}
	functionConfig.Resources = resources

	if inter, ok := d.GetOk(resourceFunctionUserConfig); ok {
		interMap := inter.(map[string]interface{})
		functionConfig.UserConfig = interMap
	}

	functionConfig.ProducerConfig = marshalFunctionProducerConfig(d)

	return functionConfig, nil
}

// unmarshalFunctionInputSpecs writes the server's inputSpecs into state.
//
// It cannot mirror the response verbatim. FunctionConfigUtils.convertFromDetails() returns an entry
// for *every* input topic the function consumes - including topics declared through all four legacy
// input fields - and never reconstructs those legacy fields. Copying all of it into state would
// invent input_specs blocks for configurations that never wrote one, and those would show as a diff
// on every plan forever.
//
// So a returned spec is skipped when the configuration already represents that topic through a
// legacy field, unless it also declares the topic in input_specs (in which case the block is
// genuinely the user's and must be refreshed rather than dropped). On import every legacy field is
// empty, so every spec lands in input_specs - the complete representation of the function.
func unmarshalFunctionInputSpecs(functionConfig utils.FunctionConfig, d *schema.ResourceData) error {
	covered := map[string]bool{}
	if inter, ok := d.GetOk(resourceFunctionInputsKey); ok {
		for _, item := range inter.(*schema.Set).List() {
			covered[item.(string)] = true
		}
	}
	if inter, ok := d.GetOk(resourceFunctionTopicsPatternKey); ok {
		covered[inter.(string)] = true
	}
	for _, key := range []string{
		resourceFunctionCustomSerdeInputsKey,
		resourceFunctionCustomSchemaInputsKey,
	} {
		if inter, ok := d.GetOk(key); ok {
			for topic := range functionInputMapKeys(inter) {
				covered[topic] = true
			}
		}
	}

	declared := functionInputSpecsFromSchema(d.Get(resourceFunctionInputSpecsKey))

	specs := make([]interface{}, 0, len(functionConfig.InputSpecs))
	for topic, consumerConfig := range functionConfig.InputSpecs {
		declaredConfig, isDeclared := declared[topic]
		if covered[topic] && !isDeclared {
			continue
		}

		// Pulsar 4.0.x persists consumerProperties but convertFromDetails() does not copy them into
		// the FunctionConfig returned by GET. Preserve the configured state until the server can
		// round-trip the field. A user removing the map has an empty declared value, so removal still
		// reaches both the request and state.
		if isDeclared && len(consumerConfig.ConsumerProperties) == 0 &&
			len(declaredConfig.ConsumerProperties) != 0 {
			consumerConfig.ConsumerProperties = declaredConfig.ConsumerProperties
		}

		specs = append(specs, flattenFunctionInputSpec(topic, consumerConfig))
	}

	return d.Set(resourceFunctionInputSpecsKey, specs)
}

func flattenFunctionInputSpec(topic string, consumerConfig utils.ConsumerConfig) map[string]interface{} {
	spec := map[string]interface{}{
		resourceFunctionInputSpecTopicKey:             topic,
		resourceFunctionInputSpecReceiverQueueSizeKey: defaultFunctionReceiverQueueSize,
		resourceFunctionInputSpecRegexPatternKey:      consumerConfig.RegexPattern,
		resourceFunctionInputSpecPoolMessagesKey:      consumerConfig.PoolMessages,
	}

	if consumerConfig.HasReceiverQueueSize() {
		spec[resourceFunctionInputSpecReceiverQueueSizeKey] = consumerConfig.ReceiverQueueSize
	}

	if consumerConfig.SchemaType != "" {
		spec[resourceFunctionInputSpecSchemaTypeKey] = consumerConfig.SchemaType
	}

	if consumerConfig.SerdeClassName != "" {
		spec[resourceFunctionInputSpecSerdeClassNameKey] = consumerConfig.SerdeClassName
	}

	// convertFromDetails always returns these maps non-nil, usually empty. Only surface them when
	// they hold something, so an empty map does not read as configuration the user never wrote.
	if len(consumerConfig.SchemaProperties) != 0 {
		spec[resourceFunctionInputSpecSchemaPropertiesKey] = convertToInterfaceMap(consumerConfig.SchemaProperties)
	}

	if len(consumerConfig.ConsumerProperties) != 0 {
		spec[resourceFunctionInputSpecConsumerPropertiesKey] = convertToInterfaceMap(consumerConfig.ConsumerProperties)
	}

	return spec
}

// marshalFunctionProducerConfig builds the output producer's configuration, mirroring how
// pulsar_source populates the same struct. It returns nil when nothing is configured so the
// request is unchanged for functions that do not set any of these.
func marshalFunctionProducerConfig(d *schema.ResourceData) *utils.ProducerConfig {
	producerConfig := &utils.ProducerConfig{}
	configured := false

	if inter, ok := d.GetOk(resourceFunctionPCMaxPendingMsgKey); ok {
		producerConfig.MaxPendingMessages = inter.(int)
		configured = true
	}

	if inter, ok := d.GetOk(resourceFunctionPCMaxPendingMsgAcrossPartitionKey); ok {
		producerConfig.MaxPendingMessagesAcrossPartitions = inter.(int)
		configured = true
	}

	if inter, ok := d.GetOk(resourceFunctionPCUseThreadLocalProducersKey); ok {
		producerConfig.UseThreadLocalProducers = inter.(bool)
		configured = true
	}

	if inter, ok := d.GetOk(resourceFunctionPCBatchBuilderKey); ok {
		producerConfig.BatchBuilder = inter.(string)
		configured = true
	}

	if inter, ok := d.GetOk(resourceFunctionPCCompressionTypeKey); ok {
		producerConfig.CompressionType = inter.(string)
		configured = true
	}

	if !configured {
		return nil
	}

	return producerConfig
}

// unmarshalFunctionProducerConfig writes the output producer's configuration into state. Each field
// is only surfaced when the server returned something for it, so a function that configures none of
// them does not gain a diff.
func unmarshalFunctionProducerConfig(functionConfig utils.FunctionConfig, d *schema.ResourceData) error {
	if functionConfig.ProducerConfig == nil {
		return nil
	}

	producerConfig := functionConfig.ProducerConfig

	if producerConfig.MaxPendingMessages != 0 {
		if err := d.Set(resourceFunctionPCMaxPendingMsgKey, producerConfig.MaxPendingMessages); err != nil {
			return err
		}
	}

	if producerConfig.MaxPendingMessagesAcrossPartitions != 0 {
		if err := d.Set(resourceFunctionPCMaxPendingMsgAcrossPartitionKey,
			producerConfig.MaxPendingMessagesAcrossPartitions); err != nil {
			return err
		}
	}

	if err := d.Set(resourceFunctionPCUseThreadLocalProducersKey,
		producerConfig.UseThreadLocalProducers); err != nil {
		return err
	}

	if producerConfig.BatchBuilder != "" {
		if err := d.Set(resourceFunctionPCBatchBuilderKey, producerConfig.BatchBuilder); err != nil {
			return err
		}
	}

	if producerConfig.CompressionType != "" {
		if err := d.Set(resourceFunctionPCCompressionTypeKey, producerConfig.CompressionType); err != nil {
			return err
		}
	}

	return nil
}

func unmarshalFunctionConfig(functionConfig utils.FunctionConfig, d *schema.ResourceData) error {
	if err := unmarshalFunctionProducerConfig(functionConfig, d); err != nil {
		return err
	}

	if functionConfig.Jar != nil {
		err := d.Set(resourceFunctionJarKey, *functionConfig.Jar)
		if err != nil {
			return err
		}
	}

	if functionConfig.Py != nil {
		err := d.Set(resourceFunctionPyKey, *functionConfig.Py)
		if err != nil {
			return err
		}
	}

	if functionConfig.Go != nil {
		err := d.Set(resourceFunctionGoKey, *functionConfig.Go)
		if err != nil {
			return err
		}
	}

	if functionConfig.ClassName != "" {
		err := d.Set(resourceFunctionClassNameKey, functionConfig.ClassName)
		if err != nil {
			return err
		}
	}

	if len(functionConfig.Inputs) != 0 {
		inputs := make([]string, len(functionConfig.Inputs))
		copy(inputs, functionConfig.Inputs)

		err := d.Set(resourceFunctionInputsKey, inputs)
		if err != nil {
			return err
		}
	}

	if functionConfig.TopicsPattern != nil {
		err := d.Set(resourceFunctionTopicsPatternKey, *functionConfig.TopicsPattern)
		if err != nil {
			return err
		}
	}

	if err := unmarshalFunctionInputSpecs(functionConfig, d); err != nil {
		return err
	}

	if functionConfig.Parallelism != 0 {
		err := d.Set(resourceFunctionParallelismKey, functionConfig.Parallelism)
		if err != nil {
			return err
		}
	}

	if functionConfig.Output != "" {
		err := d.Set(resourceFunctionOutputKey, functionConfig.Output)
		if err != nil {
			return err
		}
	}

	if functionConfig.Parallelism != 0 {
		err := d.Set(resourceFunctionParallelismKey, functionConfig.Parallelism)
		if err != nil {
			return err
		}
	}

	if functionConfig.ProcessingGuarantees != "" {
		err := d.Set(resourceFunctionProcessingGuaranteesKey, functionConfig.ProcessingGuarantees)
		if err != nil {
			return err
		}
	}

	if functionConfig.SubName != "" {
		err := d.Set(resourceFunctionSubscriptionNameKey, functionConfig.SubName)
		if err != nil {
			return err
		}
	}

	if functionConfig.SubscriptionPosition != "" {
		err := d.Set(resourceFunctionSubscriptionPositionKey, functionConfig.SubscriptionPosition)
		if err != nil {
			return err
		}
	}

	err := d.Set(resourceFunctionCleanupSubscriptionKey, functionConfig.CleanupSubscription)
	if err != nil {
		return err
	}

	err = d.Set(resourceFunctionSkipToLatestKey, functionConfig.SkipToLatest)
	if err != nil {
		return err
	}

	err = d.Set(resourceFunctionForwardSourceMessageKey, functionConfig.ForwardSourceMessageProperty)
	if err != nil {
		return err
	}

	err = d.Set(resourceFunctionRetainOrderingKey, functionConfig.RetainOrdering)
	if err != nil {
		return err
	}

	err = d.Set(resourceFunctionRetainKeyOrderingKey, functionConfig.RetainKeyOrdering)
	if err != nil {
		return err
	}

	err = d.Set(resourceFunctionAutoACKKey, functionConfig.AutoAck)
	if err != nil {
		return err
	}

	if functionConfig.MaxMessageRetries != nil {
		err = d.Set(resourceFunctionMaxMessageRetriesKey, *functionConfig.MaxMessageRetries)
		if err != nil {
			return err
		}
	}

	if functionConfig.DeadLetterTopic != "" {
		err = d.Set(resourceFunctionDeadLetterTopicKey, functionConfig.DeadLetterTopic)
		if err != nil {
			return err
		}
	}

	if functionConfig.LogTopic != "" {
		err = d.Set(resourceFunctionLogTopicKey, functionConfig.LogTopic)
		if err != nil {
			return err
		}
	}

	if functionConfig.TimeoutMs != nil {
		err = d.Set(resourceFunctionTimeoutKey, *functionConfig.TimeoutMs)
		if err != nil {
			return err
		}
	}

	if functionConfig.InputTypeClassName != "" {
		err = d.Set(resourceFunctionInputTypeClassNameKey, functionConfig.InputTypeClassName)
		if err != nil {
			return err
		}
	}

	if functionConfig.OutputTypeClassName != "" {
		err = d.Set(resourceFunctionOutputTypeClassNameKey, functionConfig.OutputTypeClassName)
		if err != nil {
			return err
		}
	}

	if functionConfig.OutputSerdeClassName != "" {
		err = d.Set(resourceFunctionOutputSerdeClassNameKey, functionConfig.OutputSerdeClassName)
		if err != nil {
			return err
		}
	}

	if functionConfig.OutputSchemaType != "" {
		err = d.Set(resourceFunctionOutputSchemaTypeKey, functionConfig.OutputSchemaType)
		if err != nil {
			return err
		}
	}

	if len(functionConfig.CustomSerdeInputs) != 0 {
		customSerdeInputs := make(map[string]interface{}, len(functionConfig.CustomSerdeInputs))
		for key, value := range functionConfig.CustomSerdeInputs {
			customSerdeInputs[key] = value
		}
		err = d.Set(resourceFunctionCustomSerdeInputsKey, customSerdeInputs)
		if err != nil {
			return err
		}
	}

	if len(functionConfig.CustomSchemaInputs) != 0 {
		customSchemaInputs := make(map[string]interface{}, len(functionConfig.CustomSchemaInputs))
		for key, value := range functionConfig.CustomSchemaInputs {
			customSchemaInputs[key] = value
		}
		err = d.Set(resourceFunctionCustomSchemaInputsKey, customSchemaInputs)
		if err != nil {
			return err
		}
	}

	if len(functionConfig.CustomSchemaOutputs) != 0 {
		customSchemaOutputs := make(map[string]interface{}, len(functionConfig.CustomSchemaOutputs))
		for key, value := range functionConfig.CustomSchemaOutputs {
			customSchemaOutputs[key] = value
		}
		err = d.Set(resourceFunctionCustomSchemaOutputsKey, customSchemaOutputs)
		if err != nil {
			return err
		}
	}

	if functionConfig.CustomRuntimeOptions != "" {
		sanitizedOptions, sinkConfig, sinkConfigPresent, sourceConfig, sourceConfigPresent,
			err := splitFunctionCustomRuntimeOptions(functionConfig.CustomRuntimeOptions)
		if err != nil {
			return err
		}

		if sinkConfigPresent {
			sinkState, err := flattenRuntimeConfigForState(sinkConfig, sinkRuntimeConfigDefinition)
			if err != nil {
				return err
			}
			if err = d.Set(resourceFunctionSinkConfigKey, sinkState); err != nil {
				return err
			}
		} else {
			if err = d.Set(resourceFunctionSinkConfigKey, nil); err != nil {
				return err
			}
		}

		if sourceConfigPresent {
			sourceState, err := flattenRuntimeConfigForState(sourceConfig, sourceRuntimeConfigDefinition)
			if err != nil {
				return err
			}
			if err = d.Set(resourceFunctionSourceConfigKey, sourceState); err != nil {
				return err
			}
		} else {
			if err = d.Set(resourceFunctionSourceConfigKey, nil); err != nil {
				return err
			}
		}

		if orig, ok := d.GetOk(resourceFunctionCustomRuntimeOptionsKey); ok {
			valueToSet := sanitizedOptions
			if origStr := orig.(string); origStr != "" && sanitizedOptions != "" {
				valueToSet, err = ignoreServerSetCustomRuntimeOptions(origStr, sanitizedOptions)
				if err != nil {
					return err
				}
			}
			if err = d.Set(resourceFunctionCustomRuntimeOptionsKey, valueToSet); err != nil {
				return err
			}
		}
	} else {
		if err := d.Set(resourceFunctionSinkConfigKey, nil); err != nil {
			return err
		}
		if err := d.Set(resourceFunctionSourceConfigKey, nil); err != nil {
			return err
		}
	}

	if len(functionConfig.Secrets) != 0 {
		s, err := json.Marshal(functionConfig.Secrets)
		if err != nil {
			return err
		}
		err = d.Set(resourceFunctionSecretsKey, string(s))
		if err != nil {
			return err
		}
	}

	if functionConfig.Resources != nil {
		err = d.Set(resourceFunctionCPUKey, functionConfig.Resources.CPU)
		if err != nil {
			return err
		}

		err = d.Set(resourceFunctionRAMKey, bytesize.FormBytes(uint64(functionConfig.Resources.RAM)).ToMegaBytes())
		if err != nil {
			return err
		}

		err = d.Set(resourceFunctionDiskKey, bytesize.FormBytes(uint64(functionConfig.Resources.Disk)).ToMegaBytes())
		if err != nil {
			return err
		}
	}

	if len(functionConfig.UserConfig) != 0 {
		userConfig := make(map[string]interface{}, len(functionConfig.UserConfig))
		for key, value := range functionConfig.UserConfig {
			userConfig[key] = value
		}
		err = d.Set(resourceFunctionUserConfig, userConfig)
		if err != nil {
			return err
		}
	}

	return nil
}

func buildFunctionCustomRuntimeOptions(d *schema.ResourceData) (string, error) {
	var base string
	if inter, ok := d.GetOk(resourceFunctionCustomRuntimeOptionsKey); ok {
		base = inter.(string)
	}

	sinkConfig, sinkConfigSet, err := expandFunctionRuntimeConfig(d, sinkRuntimeConfigDefinition)
	if err != nil {
		return "", err
	}
	sourceConfig, sourceConfigSet, err := expandFunctionRuntimeConfig(d, sourceRuntimeConfigDefinition)
	if err != nil {
		return "", err
	}

	if !sinkConfigSet && !sourceConfigSet {
		return base, nil
	}

	updates := make([]runtimeConfigUpdate, 0, 2)
	if sinkConfigSet {
		updates = append(updates, runtimeConfigUpdate{
			key:    runtimeOptionSinkConfigKey,
			config: sinkConfig,
		})
	}

	if sourceConfigSet {
		updates = append(updates, runtimeConfigUpdate{
			key:    runtimeOptionSourceConfigKey,
			config: sourceConfig,
		})
	}

	return mergeFunctionCustomRuntimeOptions(base, updates...)
}

func expandFunctionRuntimeConfig(d *schema.ResourceData,
	def runtimeConfigDefinition) (map[string]interface{}, bool, error) {
	inter, ok := d.GetOk(def.schemaKey)
	if !ok {
		return nil, false, nil
	}

	if inter == nil {
		return map[string]interface{}{}, true, nil
	}

	list, ok := inter.([]interface{})
	if !ok {
		return nil, false, fmt.Errorf("%s must be a list", def.schemaKey)
	}

	if len(list) == 0 || list[0] == nil {
		return map[string]interface{}{}, true, nil
	}

	item, ok := list[0].(map[string]interface{})
	if !ok {
		return nil, false, fmt.Errorf("%s must contain an object", def.schemaKey)
	}

	runtimeConfig := make(map[string]interface{})
	if typeValue, ok := item[def.typeSchemaKey].(string); ok && typeValue != "" {
		runtimeConfig[def.runtimeTypeKey] = typeValue
	}

	if configsRaw, ok := item[resourceFunctionRuntimeConfigConfigsKey].(map[string]interface{}); ok &&
		len(configsRaw) > 0 {
		runtimeConfig[runtimeOptionConfigsKey] = normalizeRuntimeConfigSchemaMap(configsRaw)
	}

	return runtimeConfig, true, nil
}

func normalizeRuntimeConfigSchemaMap(input map[string]interface{}) map[string]interface{} {
	normalized := make(map[string]interface{}, len(input))
	for key, value := range input {
		switch v := value.(type) {
		case string:
			normalized[key] = v
		case fmt.Stringer:
			normalized[key] = v.String()
		case nil:
			normalized[key] = ""
		default:
			normalized[key] = fmt.Sprintf("%v", v)
		}
	}

	return normalized
}

type runtimeConfigUpdate struct {
	key    string
	config map[string]interface{}
}

func mergeFunctionCustomRuntimeOptions(base string, updates ...runtimeConfigUpdate) (string, error) {
	if len(updates) == 0 {
		return base, nil
	}

	trimmed := strings.TrimSpace(base)
	runtimeOptions := make(map[string]interface{})
	if trimmed != "" {
		if err := json.Unmarshal([]byte(trimmed), &runtimeOptions); err != nil {
			return "", errors.Wrap(err, "cannot unmarshal custom_runtime_options")
		}
	}

	for _, update := range updates {
		delete(runtimeOptions, update.key)
		if len(update.config) > 0 {
			runtimeOptions[update.key] = update.config
		}
	}

	if len(runtimeOptions) == 0 {
		return "", nil
	}

	b, err := json.Marshal(runtimeOptions)
	if err != nil {
		return "", errors.Wrap(err, "cannot marshal custom_runtime_options")
	}

	return string(b), nil
}

func splitFunctionCustomRuntimeOptions(raw string) (
	string, map[string]interface{}, bool, map[string]interface{}, bool, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil, false, nil, false, nil
	}

	runtimeOptions := make(map[string]interface{})
	if err := json.Unmarshal([]byte(trimmed), &runtimeOptions); err != nil {
		return "", nil, false, nil, false, errors.Wrap(err, "cannot unmarshal custom_runtime_options from Pulsar")
	}

	sinkConfig, sinkPresent, err := extractRuntimeConfig(runtimeOptions, sinkRuntimeConfigDefinition)
	if err != nil {
		return "", nil, false, nil, false, err
	}

	sourceConfig, sourcePresent, err := extractRuntimeConfig(runtimeOptions, sourceRuntimeConfigDefinition)
	if err != nil {
		return "", nil, false, nil, false, err
	}

	sanitized := ""
	if len(runtimeOptions) > 0 {
		b, err := json.Marshal(runtimeOptions)
		if err != nil {
			return "", nil, false, nil, false, errors.Wrap(err, "cannot marshal custom_runtime_options")
		}
		sanitized = string(b)
	}

	return sanitized, sinkConfig, sinkPresent, sourceConfig, sourcePresent, nil
}

func extractRuntimeConfig(runtimeOptions map[string]interface{},
	def runtimeConfigDefinition) (map[string]interface{}, bool, error) {
	raw, ok := runtimeOptions[def.runtimeKey]
	if !ok {
		return nil, false, nil
	}

	delete(runtimeOptions, def.runtimeKey)
	configState := map[string]interface{}{}
	if raw == nil {
		return configState, true, nil
	}

	configMap, ok := raw.(map[string]interface{})
	if !ok {
		return nil, false, fmt.Errorf("%s in custom_runtime_options must be a JSON object", def.runtimeKey)
	}

	if typeVal, ok := configMap[def.runtimeTypeKey]; ok {
		if typeStr, ok := typeVal.(string); ok && typeStr != "" {
			configState[def.runtimeTypeKey] = typeStr
		}
	}

	if configsRaw, ok := configMap[runtimeOptionConfigsKey]; ok {
		configsMap, ok := configsRaw.(map[string]interface{})
		if !ok {
			return nil, false, fmt.Errorf("configs in %s must be a JSON object", def.runtimeKey)
		}
		flattened, err := stringifyRuntimeConfigMap(configsMap)
		if err != nil {
			return nil, false, err
		}
		configState[runtimeOptionConfigsKey] = flattened
	}

	return configState, true, nil
}

func stringifyRuntimeConfigMap(input map[string]interface{}) (map[string]interface{}, error) {
	config := make(map[string]interface{}, len(input))
	for key, value := range input {
		stringValue, err := stringifyRuntimeConfigValue(value)
		if err != nil {
			return nil, err
		}
		config[key] = stringValue
	}

	return config, nil
}

func stringifyRuntimeConfigValue(value interface{}) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case bool:
		return strconv.FormatBool(v), nil
	case float64:
		if math.Trunc(v) == v {
			return strconv.FormatInt(int64(v), 10), nil
		}
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case nil:
		return "", nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
}

func flattenRuntimeConfigForState(config map[string]interface{}, def runtimeConfigDefinition) ([]interface{}, error) {
	if len(config) == 0 {
		return nil, nil
	}

	state := make(map[string]interface{})
	if typeVal, ok := config[def.runtimeTypeKey]; ok {
		if typeStr, ok := typeVal.(string); ok && typeStr != "" {
			state[def.typeSchemaKey] = typeStr
		}
	}

	if configsVal, ok := config[runtimeOptionConfigsKey]; ok {
		configsMap, ok := configsVal.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("configs in %s must be a map of strings", def.runtimeKey)
		}
		state[resourceFunctionRuntimeConfigConfigsKey] = configsMap
	}

	if len(state) == 0 {
		return nil, nil
	}

	return []interface{}{state}, nil
}
