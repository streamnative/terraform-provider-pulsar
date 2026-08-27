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
	"sort"
	"strings"

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/rest"
	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"

	"github.com/streamnative/terraform-provider-pulsar/bytesize"
)

const (
	resourceSinkTenantKey                            = "tenant"
	resourceSinkNamespaceKey                         = "namespace"
	resourceSinkNameKey                              = "name"
	resourceSinkInputsKey                            = "inputs"
	resourceSinkTopicsPatternKey                     = "topics_pattern"
	resourceSinkSubscriptionNameKey                  = "subscription_name"
	resourceSinkCleanupSubscriptionKey               = "cleanup_subscription"
	resourceSinkSubscriptionPositionKey              = "subscription_position"
	resourceSinkCustomSerdeInputsKey                 = "custom_serde_inputs"
	resourceSinkCustomSchemaInputsKey                = "custom_schema_inputs"
	resourceSinkInputSpecsKey                        = "input_specs"
	resourceSinkInputSpecsSubsetTopicKey             = "key"
	resourceSinkInputSpecsSubsetSchemaTypeKey        = "schema_type"
	resourceSinkInputSpecsSubsetSerdeClassNameKey    = "serde_class_name"
	resourceSinkInputSpecsSubsetIsRegexPatternKey    = "is_regex_pattern"
	resourceSinkInputSpecsSubsetReceiverQueueSizeKey = "receiver_queue_size"
	resourceSinkInputSpecsSubsetPoolMessagesKey      = "pool_messages"
	//nolint:lll
	resourceSinkInputSpecsSubsetConsumerPropertiesKey = "consumer_properties"
	resourceSinkProcessingGuaranteesKey               = "processing_guarantees"
	resourceSinkRetainOrderingKey                     = "retain_ordering"
	resourceSinkParallelismKey                        = "parallelism"
	resourceSinkArchiveKey                            = "archive"
	resourceSinkClassnameKey                          = "classname"
	resourceSinkCPUKey                                = "cpu"
	resourceSinkRAMKey                                = "ram_mb"
	resourceSinkDiskKey                               = "disk_mb"
	resourceSinkConfigsKey                            = "configs"
	resourceSinkAutoACKKey                            = "auto_ack"
	resourceSinkTimeoutKey                            = "timeout_ms"
	resourceSinkCustomRuntimeOptionsKey               = "custom_runtime_options"
	resourceSinkDeadLetterTopicKey                    = "dead_letter_topic"
	resourceSinkMaxRedeliverCountKey                  = "max_redeliver_count"
	resourceSinkNegativeCountRedeliveryDelayKey       = "negative_ack_redelivery_delay_ms"
	resourceSinkRetainKeyOrderingKey                  = "retain_key_ordering"
	resourceSinkSinkTypeKey                           = "sink_type"
	resourceSinkSecretsKey                            = "secrets"
)

const defaultSinkReceiverQueueSize = 1000

var sinkInputSourceKeys = []string{
	resourceSinkInputsKey,
	resourceSinkTopicsPatternKey,
	resourceSinkCustomSerdeInputsKey,
	resourceSinkCustomSchemaInputsKey,
	resourceSinkInputSpecsKey,
}

var resourceSinkDescriptions = make(map[string]string)

func init() {
	//nolint:lll
	resourceSinkDescriptions = map[string]string{
		resourceSinkTenantKey:                       "The sink's tenant",
		resourceSinkNamespaceKey:                    "The sink's namespace",
		resourceSinkNameKey:                         "The sink's name",
		resourceSinkInputsKey:                       "The sink's input topics",
		resourceSinkTopicsPatternKey:                "TopicsPattern to consume from list of topics under a namespace that match the pattern",
		resourceSinkSubscriptionNameKey:             "Pulsar source subscription name if user wants a specific subscription-name for input-topic consumer",
		resourceSinkCleanupSubscriptionKey:          "Whether the subscriptions the functions created/used should be deleted when the functions was deleted",
		resourceSinkSubscriptionPositionKey:         "Pulsar source subscription position if user wants to consume messages from the specified location (Latest, Earliest). Default to Earliest.",
		resourceSinkCustomSerdeInputsKey:            "The map of input topics to SerDe class names (as a JSON string)",
		resourceSinkCustomSchemaInputsKey:           "The map of input topics to Schema types or class names (as a JSON string)",
		resourceSinkInputSpecsKey:                   "The map of input topics specs",
		resourceSinkProcessingGuaranteesKey:         "Define the message delivery semantics, default to ATLEAST_ONCE (ATLEAST_ONCE, ATMOST_ONCE, EFFECTIVELY_ONCE)",
		resourceSinkRetainOrderingKey:               "Sink consumes and sinks messages in order",
		resourceSinkParallelismKey:                  "The sink's parallelism factor. Defaults to `1`.",
		resourceSinkArchiveKey:                      "Path to the archive file for the sink. It also supports url-path [http/https/file (file protocol assumes that file already exists on worker host)] from which worker can download the package",
		resourceSinkClassnameKey:                    "The sink's class name if archive is file-url-path (file://)",
		resourceSinkCPUKey:                          "The CPU that needs to be allocated per sink instance (applicable only to Docker runtime)",
		resourceSinkRAMKey:                          "The RAM that need to be allocated per sink instance (applicable only to the process and Docker runtimes)",
		resourceSinkDiskKey:                         "The disk that need to be allocated per sink instance (applicable only to Docker runtime)",
		resourceSinkConfigsKey:                      "User defined configs key/values (JSON string)",
		resourceSinkAutoACKKey:                      "Whether or not the framework will automatically acknowledge messages",
		resourceSinkTimeoutKey:                      "The message timeout in milliseconds",
		resourceSinkCustomRuntimeOptionsKey:         "A string that encodes options to customize the runtime",
		resourceSinkDeadLetterTopicKey:              "Name of the dead topic where the failing messages will be sent",
		resourceSinkMaxRedeliverCountKey:            "Maximum number of times that a message will be redelivered before being sent to the dead letter topic",
		resourceSinkNegativeCountRedeliveryDelayKey: "The negative ack message redelivery delay in milliseconds",
		resourceSinkRetainKeyOrderingKey:            "Sink consumes and processes messages in key order",
		resourceSinkSinkTypeKey:                     "The sinks's connector provider",
		resourceSinkSecretsKey:                      "The map of secretName to an object that encapsulates how the secret is fetched by the underlying secrets provider",
	}
}

func resourcePulsarSink() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourcePulsarSinkCreate,
		ReadContext:   resourcePulsarSinkRead,
		UpdateContext: resourcePulsarSinkUpdate,
		DeleteContext: resourcePulsarSinkDelete,
		CustomizeDiff: resourcePulsarSinkCustomizeDiff,
		Description:   "Manages Pulsar IO sinks through the Functions Worker API.",
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Type:    pulsarSinkStateTypeV0(),
				Upgrade: resourcePulsarSinkStateUpgradeV0,
				Version: 0,
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				id := d.Id()

				parts := strings.Split(id, "/")
				if len(parts) != 3 {
					return nil, errors.New("ID should be tenant/namespace/name format")
				}

				_ = d.Set(resourceSinkTenantKey, parts[0])
				_ = d.Set(resourceSinkNamespaceKey, parts[1])
				_ = d.Set(resourceSinkNameKey, parts[2])

				diags := resourcePulsarSinkRead(ctx, d, meta)
				if diags.HasError() {
					return nil, fmt.Errorf("import %q: %s", d.Id(), diags[0].Summary)
				}
				return []*schema.ResourceData{d}, nil
			},
		},
		Schema: map[string]*schema.Schema{
			resourceSinkTenantKey: {
				Type:        schema.TypeString,
				Required:    true,
				Description: resourceSinkDescriptions[resourceSinkTenantKey],
			},
			resourceSinkNamespaceKey: {
				Type:        schema.TypeString,
				Required:    true,
				Description: resourceSinkDescriptions[resourceSinkNamespaceKey],
			},
			resourceSinkNameKey: {
				Type:        schema.TypeString,
				Required:    true,
				Description: resourceSinkDescriptions[resourceSinkNameKey],
			},
			resourceSinkInputsKey: {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: resourceSinkDescriptions[resourceSinkInputsKey],
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			resourceSinkTopicsPatternKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: resourceSinkDescriptions[resourceSinkTopicsPatternKey],
			},
			resourceSinkSubscriptionNameKey: {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: resourceSinkDescriptions[resourceSinkSubscriptionNameKey],
			},
			resourceSinkCleanupSubscriptionKey: {
				Type:        schema.TypeBool,
				Required:    true,
				Description: resourceSinkDescriptions[resourceSinkCleanupSubscriptionKey],
			},
			resourceSinkSubscriptionPositionKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     SubscriptionPositionEarliest,
				Description: resourceSinkDescriptions[resourceSinkSubscriptionPositionKey],
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
			resourceSinkCustomSerdeInputsKey: {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: resourceSinkDescriptions[resourceSinkCustomSerdeInputsKey],
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			resourceSinkCustomSchemaInputsKey: {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: resourceSinkDescriptions[resourceSinkCustomSchemaInputsKey],
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			// Terraform does not support nested maps, so use TypeSet. v0.13 populated this field for
			// legacy input representations; keep those values computed so an upgrade does not plan
			// their removal.
			resourceSinkInputSpecsKey: {
				Type:        schema.TypeSet,
				Optional:    true,
				Computed:    true,
				Description: resourceSinkDescriptions[resourceSinkInputSpecsKey],
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						resourceSinkInputSpecsSubsetTopicKey: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The input topic that this consumer configuration applies to.",
						},
						resourceSinkInputSpecsSubsetSchemaTypeKey: {
							Type:     schema.TypeString,
							Optional: true,
							//nolint:lll
							Description: "The schema type of this topic, either a builtin schema type such as `avro` or a Schema implementation class name. Cannot be set together with `serde_class_name`.",
						},
						resourceSinkInputSpecsSubsetSerdeClassNameKey: {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "The serde class name of this topic. Cannot be set together with `schema_type`.",
						},
						resourceSinkInputSpecsSubsetIsRegexPatternKey: {
							Type:     schema.TypeBool,
							Optional: true,
							//nolint:lll
							Description: "Whether the topic is a regex pattern matching multiple topics. Pulsar rejects a change to this on an existing sink.",
						},
						resourceSinkInputSpecsSubsetReceiverQueueSizeKey: {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  defaultSinkReceiverQueueSize,
							//nolint:lll
							Description: "The consumer receiver queue size for this topic. When omitted, the provider sends 1000, which buffers up to that many messages per sink instance. Set to 0 to disable prefetch.",
							ValidateFunc: func(val interface{}, key string) ([]string, []error) {
								if v := val.(int); v < 0 {
									return nil, []error{
										fmt.Errorf("%s must be greater than or equal to 0, got %d", key, v),
									}
								}
								return nil, nil
							},
						},
						resourceSinkInputSpecsSubsetPoolMessagesKey: {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "Whether the consumer pools messages for this topic.",
						},
						resourceSinkInputSpecsSubsetConsumerPropertiesKey: {
							Type:        schema.TypeMap,
							Optional:    true,
							Description: "Consumer properties key/values for this topic.",
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			resourceSinkProcessingGuaranteesKey: {
				Type:     schema.TypeString,
				Optional: true,
				Default:  ProcessingGuaranteesAtLeastOnce,
				ValidateFunc: func(val interface{}, key string) ([]string, []error) {
					v := val.(string)
					supported := []string{
						ProcessingGuaranteesAtLeastOnce,
						ProcessingGuaranteesAtMostOnce,
						ProcessingGuaranteesEffectivelyOnce,
					}

					found := false
					for _, item := range supported {
						if v == item {
							found = true
							break
						}
					}
					if !found {
						return nil, []error{
							fmt.Errorf("%s is unsupported, shold be one of %s", v,
								strings.Join(supported, ",")),
						}
					}

					return nil, nil
				},
				Description: resourceSinkDescriptions[resourceSinkProcessingGuaranteesKey],
			},
			resourceSinkRetainOrderingKey: {
				Type:        schema.TypeBool,
				ForceNew:    true,
				Optional:    true,
				Default:     true,
				Description: resourceSinkDescriptions[resourceSinkRetainOrderingKey],
			},
			resourceSinkParallelismKey: {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     1,
				Description: resourceSinkDescriptions[resourceSinkParallelismKey],
			},
			resourceSinkArchiveKey: {
				Type:        schema.TypeString,
				Required:    true,
				Description: resourceSinkDescriptions[resourceSinkArchiveKey],
			},
			resourceSinkClassnameKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: resourceSinkDescriptions[resourceSinkClassnameKey],
			},
			resourceSinkCPUKey: {
				Type:        schema.TypeFloat,
				Optional:    true,
				Computed:    true,
				Description: resourceSinkDescriptions[resourceSinkCPUKey],
			},
			resourceSinkRAMKey: {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: resourceSinkDescriptions[resourceSinkRAMKey],
			},
			resourceSinkDiskKey: {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: resourceSinkDescriptions[resourceSinkDiskKey],
			},
			resourceSinkConfigsKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Sensitive:   true,
				Description: resourceSinkDescriptions[resourceSinkConfigsKey],
			},
			resourceSinkAutoACKKey: {
				Type:        schema.TypeBool,
				Required:    true,
				Description: resourceSinkDescriptions[resourceSinkAutoACKKey],
			},
			resourceSinkTimeoutKey: {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: resourceSinkDescriptions[resourceSinkTimeoutKey],
			},
			resourceSinkCustomRuntimeOptionsKey: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  resourceSinkDescriptions[resourceSinkCustomRuntimeOptionsKey],
				ValidateFunc: jsonValidateFunc,
			},
			resourceSinkDeadLetterTopicKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: resourceSinkDescriptions[resourceSinkDeadLetterTopicKey],
			},
			resourceSinkMaxRedeliverCountKey: {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: resourceSinkDescriptions[resourceSinkMaxRedeliverCountKey],
			},
			resourceSinkNegativeCountRedeliveryDelayKey: {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: resourceSinkDescriptions[resourceSinkNegativeCountRedeliveryDelayKey],
			},
			resourceSinkRetainKeyOrderingKey: {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: resourceSinkDescriptions[resourceSinkRetainKeyOrderingKey],
			},
			resourceSinkSinkTypeKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: resourceSinkDescriptions[resourceSinkSinkTypeKey],
			},
			resourceSinkSecretsKey: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  resourceSinkDescriptions[resourceSinkSecretsKey],
				ValidateFunc: jsonValidateFunc,
			},
		},
	}
}

// pulsarSinkStateTypeV0 is the frozen schema-version 0 shape. It accepts v0.13's five-field
// input_specs elements and the two fields v0.14.0-rc.1 added without a schema version bump. Keep
// it independent from the current schema so Terraform decodes legacy TypeSet elements before
// re-encoding them under the current schema.
func pulsarSinkStateTypeV0() cty.Type {
	inputSpecsType := cty.Set(cty.Object(map[string]cty.Type{
		resourceSinkInputSpecsSubsetTopicKey:              cty.String,
		resourceSinkInputSpecsSubsetSchemaTypeKey:         cty.String,
		resourceSinkInputSpecsSubsetSerdeClassNameKey:     cty.String,
		resourceSinkInputSpecsSubsetIsRegexPatternKey:     cty.Bool,
		resourceSinkInputSpecsSubsetReceiverQueueSizeKey:  cty.Number,
		resourceSinkInputSpecsSubsetPoolMessagesKey:       cty.Bool,
		resourceSinkInputSpecsSubsetConsumerPropertiesKey: cty.Map(cty.String),
	}))

	return cty.Object(map[string]cty.Type{
		"id":                                        cty.String,
		resourceSinkArchiveKey:                      cty.String,
		resourceSinkAutoACKKey:                      cty.Bool,
		resourceSinkClassnameKey:                    cty.String,
		resourceSinkCleanupSubscriptionKey:          cty.Bool,
		resourceSinkConfigsKey:                      cty.String,
		resourceSinkCPUKey:                          cty.Number,
		resourceSinkCustomRuntimeOptionsKey:         cty.String,
		resourceSinkCustomSchemaInputsKey:           cty.Map(cty.String),
		resourceSinkCustomSerdeInputsKey:            cty.Map(cty.String),
		resourceSinkDeadLetterTopicKey:              cty.String,
		resourceSinkDiskKey:                         cty.Number,
		resourceSinkInputsKey:                       cty.Set(cty.String),
		resourceSinkInputSpecsKey:                   inputSpecsType,
		resourceSinkMaxRedeliverCountKey:            cty.Number,
		resourceSinkNameKey:                         cty.String,
		resourceSinkNamespaceKey:                    cty.String,
		resourceSinkNegativeCountRedeliveryDelayKey: cty.Number,
		resourceSinkParallelismKey:                  cty.Number,
		resourceSinkProcessingGuaranteesKey:         cty.String,
		resourceSinkRAMKey:                          cty.Number,
		resourceSinkRetainKeyOrderingKey:            cty.Bool,
		resourceSinkRetainOrderingKey:               cty.Bool,
		resourceSinkSecretsKey:                      cty.String,
		resourceSinkSinkTypeKey:                     cty.String,
		resourceSinkSubscriptionNameKey:             cty.String,
		resourceSinkSubscriptionPositionKey:         cty.String,
		resourceSinkTenantKey:                       cty.String,
		resourceSinkTimeoutKey:                      cty.Number,
		resourceSinkTopicsPatternKey:                cty.String,
	})
}

func resourcePulsarSinkStateUpgradeV0(
	_ context.Context,
	rawState map[string]interface{},
	_ interface{},
) (map[string]interface{}, error) {
	if rawState == nil {
		return nil, fmt.Errorf("pulsar_sink state upgrade from version 0: state is nil")
	}

	return rawState, nil
}

func resourcePulsarSinkCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getV3ClientFromMeta(meta).Sinks()

	sinkConfig, err := marshalSinkConfig(d)
	if err != nil {
		return diag.FromErr(err)
	}

	if isPackageURLSupported(sinkConfig.Archive) {
		err = client.CreateSinkWithURL(sinkConfig, sinkConfig.Archive)
	} else {
		err = client.CreateSink(sinkConfig, sinkConfig.Archive)
	}
	if err != nil {
		return diag.FromErr(err)
	}

	return resourcePulsarSinkRead(ctx, d, meta)
}

func resourcePulsarSinkRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getV3ClientFromMeta(meta).Sinks()

	tenant := d.Get(resourceSinkTenantKey).(string)
	namespace := d.Get(resourceSinkNamespaceKey).(string)
	name := d.Get(resourceSinkNameKey).(string)

	d.SetId(fmt.Sprintf("%s/%s/%s", tenant, namespace, name))

	sinkConfig, err := client.GetSink(tenant, namespace, name)
	if err != nil {
		if cliErr, ok := err.(rest.Error); ok && cliErr.Code == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(errors.Wrapf(err, "failed to get %s sink from %s/%s", name, tenant, namespace))
	}

	if err = unmarshalSinkInputSpecs(sinkConfig, d); err != nil {
		return diag.FromErr(err)
	}

	if len(sinkConfig.SourceSubscriptionName) != 0 {
		err = d.Set(resourceSinkSubscriptionNameKey, sinkConfig.SourceSubscriptionName)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	err = d.Set(resourceSinkCleanupSubscriptionKey, sinkConfig.CleanupSubscription)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set(resourceSinkParallelismKey, sinkConfig.Parallelism)
	if err != nil {
		return diag.FromErr(err)
	}

	// When the archive is built-in resource, it is not empty, otherwise it is empty.
	if sinkConfig.Archive != "" {
		err = d.Set(resourceSinkArchiveKey, sinkConfig.Archive)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	err = d.Set(resourceSinkClassnameKey, sinkConfig.ClassName)
	if err != nil {
		return diag.FromErr(err)
	}

	if sinkConfig.Resources != nil {
		err = d.Set(resourceSinkCPUKey, sinkConfig.Resources.CPU)
		if err != nil {
			return diag.FromErr(err)
		}

		err = d.Set(resourceSinkRAMKey, bytesize.FormBytes(uint64(sinkConfig.Resources.RAM)).ToMegaBytes())
		if err != nil {
			return diag.FromErr(err)
		}

		err = d.Set(resourceSinkDiskKey, bytesize.FormBytes(uint64(sinkConfig.Resources.Disk)).ToMegaBytes())
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if len(sinkConfig.Configs) != 0 {
		b, err := json.Marshal(sinkConfig.Configs)
		if err != nil {
			return diag.FromErr(errors.Wrap(err, "cannot marshal configs from sinkConfig"))
		}

		err = d.Set(resourceSinkConfigsKey, string(b))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	err = d.Set(resourceSinkAutoACKKey, sinkConfig.AutoAck)
	if err != nil {
		return diag.FromErr(err)
	}

	if sinkConfig.TimeoutMs != nil {
		err = d.Set(resourceSinkTimeoutKey, int(*sinkConfig.TimeoutMs))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if sinkConfig.CustomRuntimeOptions != "" {
		orig, ok := d.GetOk(resourceSinkCustomRuntimeOptionsKey)
		if ok {
			s, err := ignoreServerSetCustomRuntimeOptions(orig.(string), sinkConfig.CustomRuntimeOptions)
			if err != nil {
				return diag.FromErr(err)
			}
			err = d.Set(resourceSinkCustomRuntimeOptionsKey, s)
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}

	if len(sinkConfig.DeadLetterTopic) != 0 {
		err = d.Set(resourceSinkDeadLetterTopicKey, sinkConfig.DeadLetterTopic)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	err = d.Set(resourceSinkMaxRedeliverCountKey, sinkConfig.MaxMessageRetries)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set(resourceSinkNegativeCountRedeliveryDelayKey, sinkConfig.NegativeAckRedeliveryDelayMs)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set(resourceSinkRetainKeyOrderingKey, sinkConfig.RetainKeyOrdering)
	if err != nil {
		return diag.FromErr(err)
	}

	if len(sinkConfig.SinkType) != 0 {
		err = d.Set(resourceSinkSinkTypeKey, sinkConfig.SinkType)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if len(sinkConfig.Secrets) != 0 {
		s, err := json.Marshal(sinkConfig.Secrets)
		if err != nil {
			return diag.FromErr(errors.Wrap(err, "cannot marshal secrets from sinkConfig"))
		}
		err = d.Set(resourceSinkSecretsKey, string(s))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	err = d.Set(resourceSinkRetainOrderingKey, sinkConfig.RetainOrdering)
	if err != nil {
		return diag.FromErr(err)
	}

	if sinkConfig.ProcessingGuarantees != "" {
		err = d.Set(resourceSinkProcessingGuaranteesKey, sinkConfig.ProcessingGuarantees)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if sinkConfig.SourceSubscriptionPosition != "" {
		err = d.Set(resourceSinkSubscriptionPositionKey, sinkConfig.SourceSubscriptionPosition)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func resourcePulsarSinkUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getV3ClientFromMeta(meta).Sinks()

	sinkConfig, err := marshalSinkConfig(d)
	if err != nil {
		return diag.FromErr(err)
	}
	if sinkConfigHasUnownedLegacyInputTopics(sinkConfig) {
		currentSinkConfig, err := client.GetSink(sinkConfig.Tenant, sinkConfig.Namespace, sinkConfig.Name)
		if err != nil {
			return diag.FromErr(errors.Wrapf(err, "failed to get %s sink from %s/%s",
				sinkConfig.Name, sinkConfig.Tenant, sinkConfig.Namespace))
		}
		mergeSinkLegacyInputSpecsFromBroker(sinkConfig, currentSinkConfig)
	}

	updateOptions := &utils.UpdateOptions{
		UpdateAuthData: true,
	}
	if isPackageURLSupported(sinkConfig.Archive) {
		err = client.UpdateSinkWithURL(sinkConfig, sinkConfig.Archive, updateOptions)
	} else {
		err = client.UpdateSink(sinkConfig, sinkConfig.Archive, updateOptions)
	}
	if err != nil {
		return diag.FromErr(err)
	}

	return resourcePulsarSinkRead(ctx, d, meta)
}

func resourcePulsarSinkDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getV3ClientFromMeta(meta).Sinks()

	tenant := d.Get(resourceSinkTenantKey).(string)
	namespace := d.Get(resourceSinkNamespaceKey).(string)
	name := d.Get(resourceSinkNameKey).(string)

	return diag.FromErr(client.DeleteSink(tenant, namespace, name))
}

// resourcePulsarSinkCustomizeDiff validates input_specs and mirrors the broker's update rules:
// consumer settings can change in place, while changing a topic or its regex flag replaces the sink.
// Moving an unchanged topic between a legacy input field and input_specs remains an in-place update.
func resourcePulsarSinkCustomizeDiff(_ context.Context, diff *schema.ResourceDiff, _ interface{}) error {
	newSpecs := diff.Get(resourceSinkInputSpecsKey)
	if rawConfigOmitsSinkInputSpecs(diff.GetRawConfig()) {
		newSpecs = nil
	}
	if err := validateSinkInputSpecs(newSpecs); err != nil {
		return err
	}
	if err := mergeSinkLegacyTypesIntoInputSpecs(
		sinkInputSpecsFromSchema(newSpecs),
		sinkStringMap(diff.Get(resourceSinkCustomSerdeInputsKey)),
		sinkStringMap(diff.Get(resourceSinkCustomSchemaInputsKey)),
	); err != nil {
		return err
	}

	if diff.Id() == "" {
		return nil
	}

	inputChanged := false
	for _, key := range sinkInputSourceKeys {
		if diff.HasChange(key) {
			inputChanged = true
			break
		}
	}
	if !inputChanged {
		return nil
	}

	oldInputs, newInputs := diff.GetChange(resourceSinkInputsKey)
	oldPattern, newPattern := diff.GetChange(resourceSinkTopicsPatternKey)
	oldCustomSerde, newCustomSerde := diff.GetChange(resourceSinkCustomSerdeInputsKey)
	oldCustomSchema, newCustomSchema := diff.GetChange(resourceSinkCustomSchemaInputsKey)
	oldSpecs, plannedSpecs := diff.GetChange(resourceSinkInputSpecsKey)
	if rawConfigOmitsSinkInputSpecs(diff.GetRawConfig()) {
		plannedSpecs = nil
	}

	oldTopics := effectiveSinkInputTopics(
		oldInputs, oldPattern, oldCustomSerde, oldCustomSchema, oldSpecs,
	)
	newTopics := effectiveSinkInputTopics(
		newInputs, newPattern, newCustomSerde, newCustomSchema, plannedSpecs,
	)

	if len(oldTopics) != len(newTopics) {
		return forceNewSinkInputTopology(diff, oldSpecs, plannedSpecs)
	}
	for topic, regexPattern := range newTopics {
		oldRegexPattern, ok := oldTopics[topic]
		if !ok || oldRegexPattern != regexPattern {
			return forceNewSinkInputTopology(diff, oldSpecs, plannedSpecs)
		}
	}

	return nil
}

func forceNewSinkInputTopology(diff *schema.ResourceDiff, oldSpecs, newSpecs interface{}) error {
	if diff.HasChange(resourceSinkInputSpecsKey) {
		return forceNewSinkInputSpecs(diff, oldSpecs, newSpecs)
	}

	if diff.HasChange(resourceSinkInputsKey) {
		return forceNewSinkInputSet(diff, resourceSinkInputsKey)
	}

	if diff.HasChange(resourceSinkTopicsPatternKey) {
		return diff.ForceNew(resourceSinkTopicsPatternKey)
	}

	for _, key := range []string{
		resourceSinkCustomSerdeInputsKey,
		resourceSinkCustomSchemaInputsKey,
	} {
		if diff.HasChange(key) {
			return forceNewSinkInputMap(diff, key)
		}
	}

	return errors.New("input topology changed without an input attribute diff")
}

func forceNewSinkInputSet(diff *schema.ResourceDiff, key string) error {
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
				// The aggregate set is already ForceNew; one changed element is sufficient to
				// preserve that decision when the SDK rehashes an element.
				return diff.ForceNew(itemKey)
			}
		}
	}

	return nil
}

func forceNewSinkInputMap(diff *schema.ResourceDiff, key string) error {
	if err := diff.ForceNew(key); err != nil {
		return err
	}

	oldValue, newValue := diff.GetChange(key)
	mapKeys := sinkInputMapKeys(oldValue)
	for topic := range sinkInputMapKeys(newValue) {
		mapKeys[topic] = true
	}
	for topic := range mapKeys {
		itemKey := key + "." + topic
		if diff.HasChange(itemKey) {
			// The aggregate map is already ForceNew; one changed entry is sufficient to
			// preserve that decision in the flattened diff.
			return diff.ForceNew(itemKey)
		}
	}

	return nil
}

// A set-level ForceNew is insufficient when a set element changes but the element count does not.
// Mark the nested topology attribute too so Terraform preserves the replacement decision.
func forceNewSinkInputSpecs(diff *schema.ResourceDiff, oldSpecs, newSpecs interface{}) error {
	if err := diff.ForceNew(resourceSinkInputSpecsKey); err != nil {
		return err
	}

	for _, attribute := range []string{
		resourceSinkInputSpecsSubsetTopicKey,
		resourceSinkInputSpecsSubsetIsRegexPatternKey,
	} {
		for _, specs := range []interface{}{oldSpecs, newSpecs} {
			set, ok := specs.(*schema.Set)
			if !ok {
				continue
			}

			for _, item := range set.List() {
				key := fmt.Sprintf("%s.%d.%s", resourceSinkInputSpecsKey, set.F(item), attribute)
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

// effectiveSinkInputTopics maps every input topic to its regex flag in the broker's create-path
// precedence order. All representations share the broker's inputSpecs keyspace, so an identical
// topic/pattern string follows the same last-write-wins behavior as SinkConfigUtils. input_specs is
// applied last and is therefore the canonical representation.
func effectiveSinkInputTopics(
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

	for topic := range sinkInputMapKeys(customSerdeInputs) {
		topics[topic] = false
	}
	for topic := range sinkInputMapKeys(customSchemaInputs) {
		topics[topic] = false
	}

	for topic, consumerConfig := range sinkInputSpecsFromSchema(inputSpecs) {
		topics[topic] = consumerConfig.RegexPattern
	}

	return topics
}

func sinkInputMapKeys(value interface{}) map[string]bool {
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

// sinkStringMap narrows a schema.TypeMap value to map[string]string, returning nil when empty so
// the field is omitted from the request payload.
func sinkStringMap(value interface{}) map[string]string {
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

func sinkInputSpecsFromSchema(inputSpecs interface{}) map[string]utils.ConsumerConfig {
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

		topic, _ := spec[resourceSinkInputSpecsSubsetTopicKey].(string)
		if topic == "" {
			continue
		}

		consumerConfig := utils.ConsumerConfig{}
		if v, ok := spec[resourceSinkInputSpecsSubsetSchemaTypeKey].(string); ok {
			consumerConfig.SchemaType = v
		}
		if v, ok := spec[resourceSinkInputSpecsSubsetSerdeClassNameKey].(string); ok {
			consumerConfig.SerdeClassName = v
		}
		if v, ok := spec[resourceSinkInputSpecsSubsetIsRegexPatternKey].(bool); ok {
			consumerConfig.RegexPattern = v
		}
		if v, ok := spec[resourceSinkInputSpecsSubsetReceiverQueueSizeKey].(int); ok {
			consumerConfig.SetReceiverQueueSize(v)
		}
		if v, ok := spec[resourceSinkInputSpecsSubsetPoolMessagesKey].(bool); ok {
			consumerConfig.PoolMessages = v
		}
		consumerConfig.ConsumerProperties = sinkStringMap(
			spec[resourceSinkInputSpecsSubsetConsumerPropertiesKey])

		specs[topic] = consumerConfig
	}

	if len(specs) == 0 {
		return nil
	}

	return specs
}

// rawConfigOmitsSinkInputSpecs is true only when the current HCL is available and omits the
// attribute. A refresh has no raw config, so callers conservatively retain state in that case.
func rawConfigOmitsSinkInputSpecs(rawConfig cty.Value) bool {
	return rawConfig.IsKnown() && !rawConfig.IsNull() &&
		!rawValueHasTopLevelAttribute(rawConfig, resourceSinkInputSpecsKey)
}

func configuredSinkInputSpecsValue(d *schema.ResourceData) interface{} {
	if rawConfigOmitsSinkInputSpecs(d.GetRawConfig()) {
		return nil
	}

	return d.Get(resourceSinkInputSpecsKey)
}

// configuredSinkInputSpecs separates HCL-owned blocks from v0.13's computed state. During an
// apply the raw config is authoritative. Refresh requests carry state but no config, so preserve
// the existing state in that case; input_specs being Optional+Computed keeps that fallback clean.
func configuredSinkInputSpecs(d *schema.ResourceData) map[string]utils.ConsumerConfig {
	return sinkInputSpecsFromSchema(configuredSinkInputSpecsValue(d))
}

// mergeSinkLegacyTypesIntoInputSpecs preserves a legacy type when a topic also has input_specs.
// The queue-only case is unambiguous. All other conflicting or incomplete combinations fail rather
// than relying on SinkConfigUtils' precedence and silently dropping a type.
func mergeSinkLegacyTypesIntoInputSpecs(
	inputSpecs map[string]utils.ConsumerConfig,
	serdeInputs, schemaInputs map[string]string,
) error {
	for topic, inputSpec := range inputSpecs {
		serdeClassName, hasSerde := serdeInputs[topic]
		schemaType, hasSchema := schemaInputs[topic]
		if !hasSerde && !hasSchema {
			continue
		}

		if hasSerde && hasSchema {
			return fmt.Errorf("%s %q overlaps both %s and %s",
				resourceSinkInputSpecsKey, topic,
				resourceSinkCustomSerdeInputsKey, resourceSinkCustomSchemaInputsKey)
		}

		if hasSerde {
			if serdeClassName == "" {
				return fmt.Errorf("%s %q overlaps %s with an empty value",
					resourceSinkInputSpecsKey, topic, resourceSinkCustomSerdeInputsKey)
			}
			if inputSpec.SchemaType != "" {
				return fmt.Errorf("%s %q cannot combine %s with %s",
					resourceSinkInputSpecsKey, topic,
					resourceSinkInputSpecsSubsetSchemaTypeKey, resourceSinkCustomSerdeInputsKey)
			}
			if inputSpec.SerdeClassName != "" && inputSpec.SerdeClassName != serdeClassName {
				return fmt.Errorf("%s %q has conflicting %s values between %s and %s",
					resourceSinkInputSpecsKey, topic, resourceSinkInputSpecsSubsetSerdeClassNameKey,
					resourceSinkInputSpecsKey, resourceSinkCustomSerdeInputsKey)
			}
			inputSpec.SerdeClassName = serdeClassName
		}

		if hasSchema {
			if schemaType == "" {
				return fmt.Errorf("%s %q overlaps %s with an empty value",
					resourceSinkInputSpecsKey, topic, resourceSinkCustomSchemaInputsKey)
			}
			if inputSpec.SerdeClassName != "" {
				return fmt.Errorf("%s %q cannot combine %s with %s",
					resourceSinkInputSpecsKey, topic,
					resourceSinkInputSpecsSubsetSerdeClassNameKey, resourceSinkCustomSchemaInputsKey)
			}
			if inputSpec.SchemaType != "" && inputSpec.SchemaType != schemaType {
				return fmt.Errorf("%s %q has conflicting %s values between %s and %s",
					resourceSinkInputSpecsKey, topic, resourceSinkInputSpecsSubsetSchemaTypeKey,
					resourceSinkInputSpecsKey, resourceSinkCustomSchemaInputsKey)
			}
			inputSpec.SchemaType = schemaType
		}

		inputSpecs[topic] = inputSpec
	}

	return nil
}

func sinkLegacyInputMap(
	value interface{}, inputSpecs map[string]utils.ConsumerConfig,
) map[string]string {
	stringMap := sinkStringMap(value)
	for topic := range inputSpecs {
		delete(stringMap, topic)
	}
	if len(stringMap) == 0 {
		return nil
	}

	return stringMap
}

// sinkConfigHasUnownedLegacyInputTopics reports whether an update request contains a legacy input
// representation that is not already represented by an HCL-owned input_specs entry.
func sinkConfigHasUnownedLegacyInputTopics(sinkConfig *utils.SinkConfig) bool {
	for _, topic := range sinkConfig.Inputs {
		if !sinkInputSpecIsHCLConfigured(sinkConfig, topic) {
			return true
		}
	}
	if sinkConfig.TopicsPattern != nil && !sinkInputSpecIsHCLConfigured(sinkConfig, *sinkConfig.TopicsPattern) {
		return true
	}
	for _, inputMap := range []map[string]string{
		sinkConfig.TopicToSerdeClassName,
		sinkConfig.TopicToSchemaType,
	} {
		for topic := range inputMap {
			if !sinkInputSpecIsHCLConfigured(sinkConfig, topic) {
				return true
			}
		}
	}

	return false
}

func sinkInputSpecIsHCLConfigured(sinkConfig *utils.SinkConfig, topic string) bool {
	_, configured := sinkConfig.InputSpecs[topic]
	return configured
}

// mergeSinkLegacyInputSpecsFromBroker converts broker-known legacy inputs into canonical
// InputSpecs. UpdateSink validates legacy fields by rebuilding their ConsumerConfig, so retain the
// broker's complete config and only overlay settings legacy HCL actually owns. Topics absent from
// the broker's InputSpecs stay in the legacy request as a compatibility fallback.
func mergeSinkLegacyInputSpecsFromBroker(sinkConfig *utils.SinkConfig, currentSinkConfig utils.SinkConfig) {
	hclInputSpecs := make(map[string]bool, len(sinkConfig.InputSpecs))
	for topic := range sinkConfig.InputSpecs {
		hclInputSpecs[topic] = true
	}
	mergedTopics := map[string]bool{}
	mergeTopic := func(topic string, overlay func(*utils.ConsumerConfig)) {
		if hclInputSpecs[topic] {
			return
		}

		consumerConfig, exists := sinkConfig.InputSpecs[topic]
		if !exists {
			consumerConfig, exists = currentSinkConfig.InputSpecs[topic]
			if !exists {
				return
			}
		}

		overlay(&consumerConfig)
		if sinkConfig.InputSpecs == nil {
			sinkConfig.InputSpecs = map[string]utils.ConsumerConfig{}
		}
		sinkConfig.InputSpecs[topic] = consumerConfig
		mergedTopics[topic] = true
	}

	for _, topic := range sinkConfig.Inputs {
		mergeTopic(topic, func(consumerConfig *utils.ConsumerConfig) {
			consumerConfig.RegexPattern = false
		})
	}
	if sinkConfig.TopicsPattern != nil {
		mergeTopic(*sinkConfig.TopicsPattern, func(consumerConfig *utils.ConsumerConfig) {
			consumerConfig.RegexPattern = true
		})
	}
	for topic, serdeClassName := range sinkConfig.TopicToSerdeClassName {
		mergeTopic(topic, func(consumerConfig *utils.ConsumerConfig) {
			consumerConfig.RegexPattern = false
			consumerConfig.SchemaType = ""
			consumerConfig.SerdeClassName = serdeClassName
		})
	}
	for topic, schemaType := range sinkConfig.TopicToSchemaType {
		mergeTopic(topic, func(consumerConfig *utils.ConsumerConfig) {
			consumerConfig.RegexPattern = false
			consumerConfig.SerdeClassName = ""
			consumerConfig.SchemaType = schemaType
		})
	}

	if len(mergedTopics) == 0 {
		return
	}

	inputs := make([]string, 0, len(sinkConfig.Inputs))
	for _, topic := range sinkConfig.Inputs {
		if !mergedTopics[topic] {
			inputs = append(inputs, topic)
		}
	}
	if len(inputs) == 0 {
		sinkConfig.Inputs = nil
	} else {
		sinkConfig.Inputs = inputs
	}
	if sinkConfig.TopicsPattern != nil && mergedTopics[*sinkConfig.TopicsPattern] {
		sinkConfig.TopicsPattern = nil
	}
	sinkConfig.TopicToSerdeClassName = removeMergedSinkLegacyInputMap(
		sinkConfig.TopicToSerdeClassName, mergedTopics)
	sinkConfig.TopicToSchemaType = removeMergedSinkLegacyInputMap(
		sinkConfig.TopicToSchemaType, mergedTopics)
}

func removeMergedSinkLegacyInputMap(inputMap map[string]string, mergedTopics map[string]bool) map[string]string {
	for topic := range mergedTopics {
		delete(inputMap, topic)
	}
	if len(inputMap) == 0 {
		return nil
	}

	return inputMap
}

// unmarshalSinkInputSpecs keeps the input representation chosen in configuration while refreshing
// its values from the broker's canonical InputSpecs map. Pulsar returns every sink input through both
// Inputs and InputSpecs and does not reconstruct the legacy pattern or custom maps, so copying the
// response verbatim would invent state and cause replacement drift.
func unmarshalSinkInputSpecs(sinkConfig utils.SinkConfig, d *schema.ResourceData) error {
	remoteSpecs := sinkConfig.InputSpecs
	if remoteSpecs == nil {
		remoteSpecs = map[string]utils.ConsumerConfig{}
	}
	// Current Pulsar versions return every entry through InputSpecs. Retain a defensive fallback
	// for older or partial responses that expose a plain input only through Inputs.
	for _, topic := range sinkConfig.Inputs {
		if _, ok := remoteSpecs[topic]; !ok {
			remoteSpecs[topic] = utils.ConsumerConfig{}
		}
	}

	declared := configuredSinkInputSpecs(d)
	if !hasConfiguredSinkInputs(d, declared) {
		return unmarshalImportedSinkInputs(remoteSpecs, d)
	}

	legacySerdeInputs := sinkStringMap(d.Get(resourceSinkCustomSerdeInputsKey))
	legacySchemaInputs := sinkStringMap(d.Get(resourceSinkCustomSchemaInputsKey))
	covered, err := refreshSinkLegacyInputs(remoteSpecs, declared, d)
	if err != nil {
		return err
	}

	specs := make([]interface{}, 0, len(remoteSpecs))
	for topic, consumerConfig := range remoteSpecs {
		_, isDeclared := declared[topic]
		if covered[topic] && !isDeclared {
			continue
		}
		consumerConfig = sinkInputSpecStateConfig(
			topic, consumerConfig, declared, legacySerdeInputs, legacySchemaInputs,
		)
		specs = append(specs, flattenSinkInputSpec(topic, consumerConfig))
	}

	return d.Set(resourceSinkInputSpecsKey, specs)
}

// sinkInputSpecStateConfig retains the HCL representation when SinkConfigUtils has merged a
// legacy type into a queue-only input_specs request. Otherwise Read would write that type into the
// TypeSet, change its hash, and produce a perpetual follow-up diff.
func sinkInputSpecStateConfig(
	topic string,
	consumerConfig utils.ConsumerConfig,
	declared map[string]utils.ConsumerConfig,
	legacySerdeInputs, legacySchemaInputs map[string]string,
) utils.ConsumerConfig {
	declaredConfig, isDeclared := declared[topic]
	if !isDeclared {
		return consumerConfig
	}

	if _, ownedByLegacySerde := legacySerdeInputs[topic]; ownedByLegacySerde &&
		declaredConfig.SerdeClassName == "" {
		consumerConfig.SerdeClassName = ""
	}
	if _, ownedByLegacySchema := legacySchemaInputs[topic]; ownedByLegacySchema &&
		declaredConfig.SchemaType == "" {
		consumerConfig.SchemaType = ""
	}

	return consumerConfig
}

func hasConfiguredSinkInputs(d *schema.ResourceData, declared map[string]utils.ConsumerConfig) bool {
	if len(declared) != 0 {
		return true
	}
	for _, key := range []string{
		resourceSinkInputsKey,
		resourceSinkTopicsPatternKey,
		resourceSinkCustomSerdeInputsKey,
		resourceSinkCustomSchemaInputsKey,
	} {
		if _, ok := d.GetOk(key); ok {
			return true
		}
	}
	return false
}

func refreshSinkLegacyInputs(
	remoteSpecs, declared map[string]utils.ConsumerConfig, d *schema.ResourceData,
) (map[string]bool, error) {
	covered := map[string]bool{}

	if inter, ok := d.GetOk(resourceSinkInputsKey); ok {
		inputs := make([]string, 0, inter.(*schema.Set).Len())
		for _, item := range inter.(*schema.Set).List() {
			topic := item.(string)
			remote, exists := remoteSpecs[topic]
			_, isDeclared := declared[topic]
			if !exists || (!isDeclared && remote.RegexPattern) {
				continue
			}
			inputs = append(inputs, topic)
			covered[topic] = true
		}
		if err := d.Set(resourceSinkInputsKey, inputs); err != nil {
			return nil, err
		}
	}

	if inter, ok := d.GetOk(resourceSinkTopicsPatternKey); ok {
		pattern := inter.(string)
		remote, exists := remoteSpecs[pattern]
		_, isDeclared := declared[pattern]
		if exists && (isDeclared || remote.RegexPattern) {
			covered[pattern] = true
		} else if err := d.Set(resourceSinkTopicsPatternKey, ""); err != nil {
			return nil, err
		}
	}

	type legacyMap struct {
		key   string
		value func(utils.ConsumerConfig) string
	}
	for _, legacy := range []legacyMap{
		{resourceSinkCustomSerdeInputsKey, func(config utils.ConsumerConfig) string {
			return config.SerdeClassName
		}},
		{resourceSinkCustomSchemaInputsKey, func(config utils.ConsumerConfig) string {
			return config.SchemaType
		}},
	} {
		inter, ok := d.GetOk(legacy.key)
		if !ok {
			continue
		}
		values := sinkStringMap(inter)
		refreshed := make(map[string]string, len(values))
		for topic, value := range values {
			remote, exists := remoteSpecs[topic]
			_, isDeclared := declared[topic]
			if !exists {
				continue
			}
			if isDeclared {
				// input_specs wins on the wire, so preserve an overlapped legacy value that the
				// broker cannot reconstruct independently.
				refreshed[topic] = value
				covered[topic] = true
				continue
			}
			if remote.RegexPattern {
				continue
			}
			if remoteValue := legacy.value(remote); remoteValue != "" {
				refreshed[topic] = remoteValue
				covered[topic] = true
			}
		}
		if err := d.Set(legacy.key, refreshed); err != nil {
			return nil, err
		}
	}

	return covered, nil
}

func unmarshalImportedSinkInputs(
	remoteSpecs map[string]utils.ConsumerConfig, d *schema.ResourceData,
) error {
	topics := make([]string, 0, len(remoteSpecs))
	regexCandidates := 0
	for topic, consumerConfig := range remoteSpecs {
		topics = append(topics, topic)
		if sinkInputSpecUsesOnlyLegacyDefaults(consumerConfig) && consumerConfig.RegexPattern &&
			consumerConfig.SerdeClassName == "" && consumerConfig.SchemaType == "" {
			regexCandidates++
		}
	}
	sort.Strings(topics)

	inputs := make([]string, 0, len(remoteSpecs))
	customSerdeInputs := map[string]string{}
	customSchemaInputs := map[string]string{}
	pattern := ""
	specs := make([]interface{}, 0, len(remoteSpecs))
	for _, topic := range topics {
		consumerConfig := remoteSpecs[topic]
		if sinkInputSpecUsesOnlyLegacyDefaults(consumerConfig) {
			switch {
			case consumerConfig.RegexPattern && regexCandidates == 1 &&
				consumerConfig.SerdeClassName == "" && consumerConfig.SchemaType == "":
				pattern = topic
				continue
			case !consumerConfig.RegexPattern && consumerConfig.SerdeClassName != "":
				customSerdeInputs[topic] = consumerConfig.SerdeClassName
				continue
			case !consumerConfig.RegexPattern && consumerConfig.SchemaType != "":
				customSchemaInputs[topic] = consumerConfig.SchemaType
				continue
			case !consumerConfig.RegexPattern && consumerConfig.SerdeClassName == "" &&
				consumerConfig.SchemaType == "":
				inputs = append(inputs, topic)
				continue
			}
		}
		specs = append(specs, flattenSinkInputSpec(topic, consumerConfig))
	}

	for key, value := range map[string]interface{}{
		resourceSinkInputsKey:             inputs,
		resourceSinkTopicsPatternKey:      pattern,
		resourceSinkCustomSerdeInputsKey:  customSerdeInputs,
		resourceSinkCustomSchemaInputsKey: customSchemaInputs,
		resourceSinkInputSpecsKey:         specs,
	} {
		if err := d.Set(key, value); err != nil {
			return err
		}
	}
	return nil
}

func sinkInputSpecUsesOnlyLegacyDefaults(consumerConfig utils.ConsumerConfig) bool {
	return !consumerConfig.HasReceiverQueueSize() &&
		!consumerConfig.PoolMessages &&
		len(consumerConfig.ConsumerProperties) == 0 &&
		len(consumerConfig.SchemaProperties) == 0 &&
		consumerConfig.CryptoConfig == nil &&
		(consumerConfig.SchemaType == "" || consumerConfig.SerdeClassName == "")
}

func flattenSinkInputSpec(topic string, consumerConfig utils.ConsumerConfig) map[string]interface{} {
	spec := map[string]interface{}{
		resourceSinkInputSpecsSubsetTopicKey:             topic,
		resourceSinkInputSpecsSubsetReceiverQueueSizeKey: defaultSinkReceiverQueueSize,
		resourceSinkInputSpecsSubsetIsRegexPatternKey:    consumerConfig.RegexPattern,
		resourceSinkInputSpecsSubsetPoolMessagesKey:      consumerConfig.PoolMessages,
	}

	if consumerConfig.HasReceiverQueueSize() {
		spec[resourceSinkInputSpecsSubsetReceiverQueueSizeKey] = consumerConfig.ReceiverQueueSize
	}
	if consumerConfig.SchemaType != "" {
		spec[resourceSinkInputSpecsSubsetSchemaTypeKey] = consumerConfig.SchemaType
	}
	if consumerConfig.SerdeClassName != "" {
		spec[resourceSinkInputSpecsSubsetSerdeClassNameKey] = consumerConfig.SerdeClassName
	}
	if len(consumerConfig.ConsumerProperties) != 0 {
		spec[resourceSinkInputSpecsSubsetConsumerPropertiesKey] =
			convertToInterfaceMap(consumerConfig.ConsumerProperties)
	}

	return spec
}

// validateSinkInputSpecs enforces the two rules Pulsar applies to inputSpecs that the schema cannot
// express: topics are the map key so they must be unique, and SinkConfigUtils rejects a spec that
// sets both schemaType and serdeClassName.
func validateSinkInputSpecs(inputSpecs interface{}) error {
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

		topic, _ := spec[resourceSinkInputSpecsSubsetTopicKey].(string)
		if topic == "" {
			// The SDK can include an empty placeholder while diffing TypeSet elements. The nested
			// Required schema still validates actual user configuration.
			continue
		}
		if seenTopics[topic] {
			return fmt.Errorf("%s contains duplicate %s %q",
				resourceSinkInputSpecsKey, resourceSinkInputSpecsSubsetTopicKey, topic)
		}
		seenTopics[topic] = true

		schemaType, _ := spec[resourceSinkInputSpecsSubsetSchemaTypeKey].(string)
		serdeClassName, _ := spec[resourceSinkInputSpecsSubsetSerdeClassNameKey].(string)
		if schemaType != "" && serdeClassName != "" {
			return fmt.Errorf("%s %q cannot set both %s and %s",
				resourceSinkInputSpecsKey, topic,
				resourceSinkInputSpecsSubsetSchemaTypeKey,
				resourceSinkInputSpecsSubsetSerdeClassNameKey)
		}
	}

	return nil
}

func marshalSinkConfig(d *schema.ResourceData) (*utils.SinkConfig, error) {
	sinkConfig := &utils.SinkConfig{}

	if inter, ok := d.GetOk(resourceSinkTenantKey); ok {
		sinkConfig.Tenant = inter.(string)
	}

	if inter, ok := d.GetOk(resourceSinkNamespaceKey); ok {
		sinkConfig.Namespace = inter.(string)
	}

	if inter, ok := d.GetOk(resourceSinkNameKey); ok {
		sinkConfig.Name = inter.(string)
	}

	configuredInputSpecs := configuredSinkInputSpecsValue(d)
	if err := validateSinkInputSpecs(configuredInputSpecs); err != nil {
		return nil, err
	}

	inputSpecs := sinkInputSpecsFromSchema(configuredInputSpecs)
	if err := mergeSinkLegacyTypesIntoInputSpecs(
		inputSpecs,
		sinkStringMap(d.Get(resourceSinkCustomSerdeInputsKey)),
		sinkStringMap(d.Get(resourceSinkCustomSchemaInputsKey)),
	); err != nil {
		return nil, err
	}
	if len(inputSpecs) != 0 {
		sinkConfig.InputSpecs = inputSpecs
	}

	if inter, ok := d.GetOk(resourceSinkInputsKey); ok {
		inputsSet := inter.(*schema.Set)
		var inputs []string

		for _, item := range inputsSet.List() {
			topic := item.(string)
			if _, isInputSpec := inputSpecs[topic]; isInputSpec {
				continue
			}
			inputs = append(inputs, topic)
		}

		if len(inputs) != 0 {
			sinkConfig.Inputs = inputs
		}
	}

	if inter, ok := d.GetOk(resourceSinkTopicsPatternKey); ok {
		pattern := inter.(string)
		if _, isInputSpec := inputSpecs[pattern]; !isInputSpec {
			sinkConfig.TopicsPattern = &pattern
		}
	}

	if inter, ok := d.GetOk(resourceSinkSubscriptionNameKey); ok {
		sinkConfig.SourceSubscriptionName = inter.(string)
	}

	if inter, ok := d.GetOk(resourceSinkCleanupSubscriptionKey); ok {
		sinkConfig.CleanupSubscription = inter.(bool)
	}

	if inter, ok := d.GetOk(resourceSinkSubscriptionPositionKey); ok {
		sinkConfig.SourceSubscriptionPosition = inter.(string)
	}

	if inter, ok := d.GetOk(resourceSinkCustomSerdeInputsKey); ok {
		sinkConfig.TopicToSerdeClassName = sinkLegacyInputMap(inter, inputSpecs)
	}

	if inter, ok := d.GetOk(resourceSinkCustomSchemaInputsKey); ok {
		sinkConfig.TopicToSchemaType = sinkLegacyInputMap(inter, inputSpecs)
	}

	if inter, ok := d.GetOk(resourceSinkProcessingGuaranteesKey); ok {
		sinkConfig.ProcessingGuarantees = inter.(string)
	}

	if inter, ok := d.GetOk(resourceSinkRetainOrderingKey); ok {
		sinkConfig.RetainOrdering = inter.(bool)
	}

	if inter, ok := d.GetOk(resourceSinkParallelismKey); ok {
		sinkConfig.Parallelism = inter.(int)
	}

	if inter, ok := d.GetOk(resourceSinkArchiveKey); ok {
		sinkConfig.Archive = inter.(string)
	}

	if inter, ok := d.GetOk(resourceSinkClassnameKey); ok {
		sinkConfig.ClassName = inter.(string)
	}

	resources := utils.NewDefaultResources()

	if inter, ok := d.GetOk(resourceSinkCPUKey); ok {
		value := inter.(float64)
		resources.CPU = value
	}

	if inter, ok := d.GetOk(resourceSinkRAMKey); ok {
		value := bytesize.FormMegaBytes(uint64(inter.(int))).ToBytes()
		resources.RAM = int64(value)
	}

	if inter, ok := d.GetOk(resourceSinkDiskKey); ok {
		value := bytesize.FormMegaBytes(uint64(inter.(int))).ToBytes()
		resources.Disk = int64(value)
	}

	sinkConfig.Resources = resources

	if inter, ok := d.GetOk(resourceSinkConfigsKey); ok {
		var configs map[string]interface{}
		configsJSON := inter.(string)

		err := json.Unmarshal([]byte(configsJSON), &configs)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot unmarshal the configs: %s", configsJSON)
		}

		sinkConfig.Configs = configs
	}

	if inter, ok := d.GetOk(resourceSinkAutoACKKey); ok {
		sinkConfig.AutoAck = inter.(bool)
	}

	if inter, ok := d.GetOk(resourceSinkTimeoutKey); ok {
		value := int64(inter.(int))
		sinkConfig.TimeoutMs = &value
	}

	if inter, ok := d.GetOk(resourceSinkCustomRuntimeOptionsKey); ok {
		sinkConfig.CustomRuntimeOptions = inter.(string)
	}

	if inter, ok := d.GetOk(resourceSinkDeadLetterTopicKey); ok {
		sinkConfig.DeadLetterTopic = inter.(string)
	}

	if inter, ok := d.GetOk(resourceSinkMaxRedeliverCountKey); ok {
		sinkConfig.MaxMessageRetries = inter.(int)
	}

	if inter, ok := d.GetOk(resourceSinkNegativeCountRedeliveryDelayKey); ok {
		sinkConfig.NegativeAckRedeliveryDelayMs = int64(inter.(int))
	}

	if inter, ok := d.GetOk(resourceSinkRetainKeyOrderingKey); ok {
		sinkConfig.RetainOrdering = inter.(bool)
	}

	if inter, ok := d.GetOk(resourceSinkSinkTypeKey); ok {
		sinkConfig.SinkType = inter.(string)
	}

	if inter, ok := d.GetOk(resourceSinkSecretsKey); ok {
		var secrets map[string]interface{}
		secretsJSON := inter.(string)

		err := json.Unmarshal([]byte(secretsJSON), &secrets)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot unmarshal the secrets: %s", secretsJSON)
		}

		sinkConfig.Secrets = secrets
	}

	return sinkConfig, nil
}
