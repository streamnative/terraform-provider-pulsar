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
	"strings"

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/rest"
	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
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
			// terraform doesn't nested map, so use TypeSet.
			resourceSinkInputSpecsKey: {
				Type:        schema.TypeSet,
				Optional:    true,
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
							Description: "The consumer receiver queue size for this topic. Defaults to 1000, which buffers up to that many messages per sink instance. Set to 0 to disable prefetch.",
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
	if err := validateSinkInputSpecs(newSpecs); err != nil {
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
	oldSpecs, newSpecs := diff.GetChange(resourceSinkInputSpecsKey)

	oldTopics := effectiveSinkInputTopics(
		oldInputs, oldPattern, oldCustomSerde, oldCustomSchema, oldSpecs,
	)
	newTopics := effectiveSinkInputTopics(
		newInputs, newPattern, newCustomSerde, newCustomSchema, newSpecs,
	)

	if len(oldTopics) != len(newTopics) {
		return forceNewSinkInputTopology(diff, oldSpecs, newSpecs)
	}
	for topic, regexPattern := range newTopics {
		oldRegexPattern, ok := oldTopics[topic]
		if !ok || oldRegexPattern != regexPattern {
			return forceNewSinkInputTopology(diff, oldSpecs, newSpecs)
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
// precedence order. input_specs is applied last and is therefore the canonical representation.
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

// unmarshalSinkInputSpecs keeps the input representation chosen in configuration. Pulsar returns
// every sink input through both Inputs and InputSpecs and does not reconstruct the legacy pattern or
// custom maps, so mirroring the response would invent state and cause replacement drift. On import,
// no legacy field is populated and InputSpecs becomes the canonical complete representation.
func unmarshalSinkInputSpecs(sinkConfig utils.SinkConfig, d *schema.ResourceData) error {
	covered := map[string]bool{}
	if inter, ok := d.GetOk(resourceSinkInputsKey); ok {
		for _, item := range inter.(*schema.Set).List() {
			covered[item.(string)] = true
		}
	}
	if inter, ok := d.GetOk(resourceSinkTopicsPatternKey); ok {
		covered[inter.(string)] = true
	}
	for _, key := range []string{
		resourceSinkCustomSerdeInputsKey,
		resourceSinkCustomSchemaInputsKey,
	} {
		if inter, ok := d.GetOk(key); ok {
			for topic := range sinkInputMapKeys(inter) {
				covered[topic] = true
			}
		}
	}

	declared := sinkInputSpecsFromSchema(d.Get(resourceSinkInputSpecsKey))
	specs := make([]interface{}, 0, len(sinkConfig.InputSpecs))
	for topic, consumerConfig := range sinkConfig.InputSpecs {
		_, isDeclared := declared[topic]
		if covered[topic] && !isDeclared {
			continue
		}
		specs = append(specs, flattenSinkInputSpec(topic, consumerConfig))
	}

	return d.Set(resourceSinkInputSpecsKey, specs)
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

	if err := validateSinkInputSpecs(d.Get(resourceSinkInputSpecsKey)); err != nil {
		return nil, err
	}

	inputSpecs := sinkInputSpecsFromSchema(d.Get(resourceSinkInputSpecsKey))
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
