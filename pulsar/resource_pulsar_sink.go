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

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	"github.com/streamnative/pulsar-admin-go/pkg/admin"
	"github.com/streamnative/pulsar-admin-go/pkg/rest"
	"github.com/streamnative/pulsar-admin-go/pkg/utils"

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
	resourceSinkProcessingGuaranteesKey              = "processing_guarantees"
	resourceSinkRetainOrderingKey                    = "retain_ordering"
	resourceSinkParallelismKey                       = "parallelism"
	resourceSinkArchiveKey                           = "archive"
	resourceSinkClassnameKey                         = "classname"
	resourceSinkCPUKey                               = "cpu"
	resourceSinkRAMKey                               = "ram_mb"
	resourceSinkDiskKey                              = "disk_mb"
	resourceSinkConfigsKey                           = "configs"
	resourceSinkAutoACKKey                           = "auto_ack"
	resourceSinkTimeoutKey                           = "timeout_ms"
	resourceSinkCustomRuntimeOptionsKey              = "custom_runtime_options"
	resourceSinkDeadLetterTopicKey                   = "dead_letter_topic"
	resourceSinkMaxRedeliverCountKey                 = "max_redeliver_count"
	resourceSinkNegativeCountRedeliveryDelayKey      = "negative_ack_redelivery_delay_ms"
	resourceSinkRetainKeyOrderingKey                 = "retain_key_ordering"
	resourceSinkSinkTypeKey                          = "sink_type"
	resourceSinkSecretsKey                           = "secrets"
)

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
		resourceSinkParallelismKey:                  "The sink's parallelism factor",
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
				Default:     "Earliest",
				Description: resourceSinkDescriptions[resourceSinkSubscriptionPositionKey],
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
				Computed:    true,
				Description: resourceSinkDescriptions[resourceSinkInputSpecsKey],
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						resourceSinkInputSpecsSubsetTopicKey:             {Type: schema.TypeString, Required: true},
						resourceSinkInputSpecsSubsetSchemaTypeKey:        {Type: schema.TypeString, Required: true},
						resourceSinkInputSpecsSubsetSerdeClassNameKey:    {Type: schema.TypeString, Required: true},
						resourceSinkInputSpecsSubsetIsRegexPatternKey:    {Type: schema.TypeBool, Required: true},
						resourceSinkInputSpecsSubsetReceiverQueueSizeKey: {Type: schema.TypeInt, Required: true},
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
				Default:     utils.NewDefaultResources().CPU,
				Description: resourceSinkDescriptions[resourceSinkCPUKey],
			},
			resourceSinkRAMKey: {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     int(bytesize.FormBytes(uint64(utils.NewDefaultResources().RAM)).ToMegaBytes()),
				Description: resourceSinkDescriptions[resourceSinkRAMKey],
			},
			resourceSinkDiskKey: {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     int(bytesize.FormBytes(uint64(utils.NewDefaultResources().Disk)).ToMegaBytes()),
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
				Type:        schema.TypeString,
				Optional:    true,
				Description: resourceSinkDescriptions[resourceSinkCustomRuntimeOptionsKey],
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
				Type:        schema.TypeString,
				Optional:    true,
				Description: resourceSinkDescriptions[resourceSinkSecretsKey],
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
		},
	}
}

func resourcePulsarSinkCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(admin.Client).Sinks()

	sinkConfig, err := marshalSinkConfig(d)
	if err != nil {
		return diag.FromErr(err)
	}

	if isLocalArchive(sinkConfig.Archive) {
		err = client.CreateSink(sinkConfig, sinkConfig.Archive)
	} else {
		err = client.CreateSinkWithURL(sinkConfig, sinkConfig.Archive)
	}
	if err != nil {
		return diag.FromErr(err)
	}

	return resourcePulsarSinkRead(ctx, d, meta)
}

func resourcePulsarSinkRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// NOTE: Pulsar cannot returns the fields correctly, so ignore these:
	// - resourceSinkSubscriptionPositionKey
	// - resourceSinkProcessingGuaranteesKey
	// - resourceSinkRetainOrderingKey

	client := meta.(admin.Client).Sinks()

	tenant := d.Get(resourceSinkTenantKey).(string)
	namespace := d.Get(resourceSinkNamespaceKey).(string)
	name := d.Get(resourceSinkNameKey).(string)

	d.SetId(fmt.Sprintf("%s/%s/%s", tenant, namespace, name))

	sinkConfig, err := client.GetSink(tenant, namespace, name)
	if err != nil {
		if cliErr, ok := err.(rest.Error); ok && cliErr.Code == 404 {
			return diag.Errorf("ERROR_SINK_NOT_FOUND")
		}
		return diag.FromErr(errors.Wrapf(err, "failed to get %s sink from %s/%s", name, tenant, namespace))
	}

	inputs := make([]string, len(sinkConfig.Inputs))
	copy(inputs, sinkConfig.Inputs)

	err = d.Set(resourceSinkInputsKey, inputs)
	if err != nil {
		return diag.FromErr(err)
	}

	if sinkConfig.TopicsPattern != nil {
		err = d.Set(resourceSinkTopicsPatternKey, sinkConfig.TopicsPattern)
		if err != nil {
			return diag.FromErr(err)
		}
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

	if len(sinkConfig.TopicToSerdeClassName) != 0 {
		customSerdeInputs := make(map[string]interface{}, len(sinkConfig.TopicToSerdeClassName))
		for key, value := range sinkConfig.TopicToSerdeClassName {
			customSerdeInputs[key] = value
		}
		err = d.Set(resourceSinkCustomSerdeInputsKey, customSerdeInputs)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if len(sinkConfig.TopicToSchemaType) != 0 {
		customSchemaInputs := make(map[string]interface{}, len(sinkConfig.TopicToSchemaType))
		for key, value := range sinkConfig.TopicToSchemaType {
			customSchemaInputs[key] = value
		}

		err = d.Set(resourceSinkCustomSchemaInputsKey, customSchemaInputs)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if len(sinkConfig.InputSpecs) > 0 {
		var inputSpecs []interface{}
		for key, config := range sinkConfig.InputSpecs {
			item := make(map[string]interface{})
			item[resourceSinkInputSpecsSubsetTopicKey] = key
			item[resourceSinkInputSpecsSubsetSchemaTypeKey] = config.SchemaType
			item[resourceSinkInputSpecsSubsetSerdeClassNameKey] = config.SerdeClassName
			item[resourceSinkInputSpecsSubsetIsRegexPatternKey] = config.RegexPattern
			item[resourceSinkInputSpecsSubsetReceiverQueueSizeKey] = config.ReceiverQueueSize
			inputSpecs = append(inputSpecs, item)
		}
		err = d.Set(resourceSinkInputSpecsKey, inputSpecs)
		if err != nil {
			return diag.FromErr(err)
		}
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

	if len(sinkConfig.CustomRuntimeOptions) != 0 {
		err = d.Set(resourceSinkCustomRuntimeOptionsKey, sinkConfig.CustomRuntimeOptions)
		if err != nil {
			return diag.FromErr(err)
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
		s, err := json.Marshal(sinkConfig.Configs)
		if err != nil {
			return diag.FromErr(errors.Wrap(err, "cannot marshal secrets from sinkConfig"))
		}
		err = d.Set(resourceSinkSecretsKey, string(s))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

func resourcePulsarSinkUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(admin.Client).Sinks()

	sinkConfig, err := marshalSinkConfig(d)
	if err != nil {
		return diag.FromErr(err)
	}

	updateOptions := utils.NewUpdateOptions()
	if isLocalArchive(sinkConfig.Archive) {
		err = client.UpdateSink(sinkConfig, sinkConfig.Archive, updateOptions)
	} else {
		err = client.UpdateSinkWithURL(sinkConfig, sinkConfig.Archive, updateOptions)
	}
	if err != nil {
		return diag.FromErr(err)
	}

	return resourcePulsarSinkRead(ctx, d, meta)
}

func resourcePulsarSinkDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(admin.Client).Sinks()

	tenant := d.Get(resourceSinkTenantKey).(string)
	namespace := d.Get(resourceSinkNamespaceKey).(string)
	name := d.Get(resourceSinkNameKey).(string)

	return diag.FromErr(client.DeleteSink(tenant, namespace, name))
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

	if inter, ok := d.GetOk(resourceSinkInputsKey); ok {
		inputsSet := inter.(*schema.Set)
		var inputs []string

		for _, item := range inputsSet.List() {
			inputs = append(inputs, item.(string))
		}

		sinkConfig.Inputs = inputs
	}

	if inter, ok := d.GetOk(resourceSinkTopicsPatternKey); ok {
		pattern := inter.(string)
		sinkConfig.TopicsPattern = &pattern
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
		interMap := inter.(map[string]interface{})
		stringMap := make(map[string]string, len(interMap))

		for key, value := range interMap {
			stringMap[key] = value.(string)
		}

		sinkConfig.TopicToSerdeClassName = stringMap
	}

	if inter, ok := d.GetOk(resourceSinkCustomSchemaInputsKey); ok {
		interMap := inter.(map[string]interface{})
		stringMap := make(map[string]string, len(interMap))

		for key, value := range interMap {
			stringMap[key] = value.(string)
		}

		sinkConfig.TopicToSchemaType = stringMap
	}

	if inter, ok := d.GetOk(resourceSinkInputSpecsKey); ok {
		set := inter.(*schema.Set)
		if set.Len() > 0 {
			inputSpecs := make(map[string]utils.ConsumerConfig)
			for _, n := range set.List() {
				m := n.(map[string]interface{})
				inputSpec := utils.ConsumerConfig{
					SchemaType:        m[resourceSinkInputSpecsSubsetSchemaTypeKey].(string),
					SerdeClassName:    m[resourceSinkInputSpecsSubsetSerdeClassNameKey].(string),
					RegexPattern:      m[resourceSinkInputSpecsSubsetIsRegexPatternKey].(bool),
					ReceiverQueueSize: m[resourceSinkInputSpecsSubsetReceiverQueueSizeKey].(int),
				}
				inputSpecs[m[resourceSinkInputSpecsSubsetTopicKey].(string)] = inputSpec
			}
			sinkConfig.InputSpecs = inputSpecs
		}
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
