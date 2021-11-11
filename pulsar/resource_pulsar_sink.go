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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/streamnative/terraform-provider-pulsar/bytesize"

	"github.com/streamnative/pulsarctl/pkg/cli"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/pkg/errors"
	ctlutil "github.com/streamnative/pulsarctl/pkg/ctl/utils"
	"github.com/streamnative/pulsarctl/pkg/pulsar"
	"github.com/streamnative/pulsarctl/pkg/pulsar/utils"
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

	ProcessingGuaranteesAtLeastOnce     = "ATLEAST_ONCE"
	ProcessingGuaranteesAtMostOnce      = "ATMOST_ONCE"
	ProcessingGuaranteesEffectivelyOnce = "EFFECTIVELY_ONCE"
)

var resourceSinkDescriptions = make(map[string]string)

func init() {
	resourceSinkDescriptions[resourceSinkTenantKey] = "The sink's tenant"
	resourceSinkDescriptions[resourceSinkNamespaceKey] = "The sink's namespace"
	resourceSinkDescriptions[resourceSinkNameKey] = "The sink's name"
	resourceSinkDescriptions[resourceSinkInputsKey] = "The sink's input topics"
	resourceSinkDescriptions[resourceSinkTopicsPatternKey] =
		"TopicsPattern to consume from list of topics under a namespace that match the pattern"
	resourceSinkDescriptions[resourceSinkSubscriptionNameKey] =
		"Pulsar source subscription name if user wants a specific subscription-name for input-topic consumer"
	resourceSinkDescriptions[resourceSinkCleanupSubscriptionKey] =
		"Whether the subscriptions the functions created/used should be deleted when the functions was deleted"
	resourceSinkDescriptions[resourceSinkSubscriptionPositionKey] =
		"Pulsar source subscription position if user wants to consume messages from the specified location (Latest, Earliest)"
	resourceSinkDescriptions[resourceSinkCustomSerdeInputsKey] =
		"The map of input topics to SerDe class names (as a JSON string)"
	resourceSinkDescriptions[resourceSinkCustomSchemaInputsKey] =
		"The map of input topics to Schema types or class names (as a JSON string)"
	resourceSinkDescriptions[resourceSinkInputSpecsKey] = "The map of input topics specs"
	resourceSinkDescriptions[resourceSinkProcessingGuaranteesKey] =
		"Define the message delivery semantics, default to ATLEAST_ONCE (ATLEAST_ONCE, ATMOST_ONCE, EFFECTIVELY_ONCE)"
	resourceSinkDescriptions[resourceSinkRetainOrderingKey] = "Sink consumes and sinks messages in order"
	resourceSinkDescriptions[resourceSinkParallelismKey] = "The sink's parallelism factor"
	resourceSinkDescriptions[resourceSinkArchiveKey] =
		"Path to the archive file for the sink. " +
			"It also supports url-path [http/https/file (file protocol assumes that " +
			"file already exists on worker host)] from which worker can download the package"
	resourceSinkDescriptions[resourceSinkClassnameKey] = "The sink's class name if archive is file-url-path (file://)"
	resourceSinkDescriptions[resourceSinkCPUKey] =
		"The CPU that needs to be allocated per sink instance (applicable only to Docker runtime)"
	resourceSinkDescriptions[resourceSinkRAMKey] =
		"The RAM that need to be allocated per sink instance (applicable only to the process and Docker runtimes)"
	resourceSinkDescriptions[resourceSinkDiskKey] =
		"The disk that need to be allocated per sink instance (applicable only to Docker runtime)"
	resourceSinkDescriptions[resourceSinkConfigsKey] = "User defined configs key/values (JSON string)"
	resourceSinkDescriptions[resourceSinkAutoACKKey] =
		"Whether or not the framework will automatically acknowledge messages"
	resourceSinkDescriptions[resourceSinkTimeoutKey] = "The message timeout in milliseconds"
	resourceSinkDescriptions[resourceSinkCustomRuntimeOptionsKey] =
		"A string that encodes options to customize the runtime"
}

func resourcePulsarSink() *schema.Resource {
	return &schema.Resource{
		Create: resourcePulsarSinkCreate,
		Read:   resourcePulsarSinkRead,
		Update: resourcePulsarSinkUpdate,
		Delete: resourcePulsarSinkDelete,
		Exists: resourcePulsarSinkExists,
		Importer: &schema.ResourceImporter{
			State: func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				id := d.Id()

				parts := strings.Split(id, "/")
				if len(parts) != 3 {
					return nil, errors.New("ID should be tenant/namespace/name format")
				}

				_ = d.Set(resourceSinkTenantKey, parts[0])
				_ = d.Set(resourceSinkNamespaceKey, parts[1])
				_ = d.Set(resourceSinkNameKey, parts[2])

				err := resourcePulsarSinkRead(d, meta)
				return []*schema.ResourceData{d}, err
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
				Default:     utils.Earliest,
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
		},
	}
}

func resourcePulsarSinkExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(pulsar.Client).Sinks()

	tenant := d.Get(resourceSinkTenantKey).(string)
	namespace := d.Get(resourceSinkNamespaceKey).(string)
	name := d.Get(resourceSinkNameKey).(string)

	_, err := client.GetSink(tenant, namespace, name)
	if err != nil {
		if cliErr, ok := err.(cli.Error); ok && cliErr.Code == 404 {
			// sink doesn't exist.
			return false, nil
		}

		return false, errors.Wrapf(err, "failed to get sink")
	}

	return true, nil
}

func resourcePulsarSinkDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(pulsar.Client).Sinks()

	tenant := d.Get(resourceSinkTenantKey).(string)
	namespace := d.Get(resourceSinkNamespaceKey).(string)
	name := d.Get(resourceSinkNameKey).(string)

	return client.DeleteSink(tenant, namespace, name)
}

func resourcePulsarSinkUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(pulsar.Client).Sinks()

	sinkConfig, err := marshalSinkConfig(d)
	if err != nil {
		return err
	}

	updateOptions := utils.NewUpdateOptions()
	if isLocalArchive(sinkConfig.Archive) {
		err = client.UpdateSink(sinkConfig, sinkConfig.Archive, updateOptions)
	} else {
		err = client.UpdateSinkWithURL(sinkConfig, sinkConfig.Archive, updateOptions)
	}
	if err != nil {
		return err
	}

	return resourcePulsarSinkRead(d, meta)
}

func resourcePulsarSinkRead(d *schema.ResourceData, meta interface{}) error {
	// NOTE: Pulsar cannot returns the fields correctly, so ignore these:
	// - resourceSinkSubscriptionPositionKey
	// - resourceSinkProcessingGuaranteesKey
	// - resourceSinkRetainOrderingKey

	client := meta.(pulsar.Client).Sinks()

	tenant := d.Get(resourceSinkTenantKey).(string)
	namespace := d.Get(resourceSinkNamespaceKey).(string)
	name := d.Get(resourceSinkNameKey).(string)

	d.SetId(fmt.Sprintf("%s/%s/%s", tenant, namespace, name))

	sinkConfig, err := client.GetSink(tenant, namespace, name)
	if err != nil {
		return errors.Wrapf(err, "failed to get %s sink from %s/%s", name, tenant, namespace)
	}

	inputs := make([]string, len(sinkConfig.Inputs))
	for index, input := range sinkConfig.Inputs {
		inputs[index] = input
	}

	err = d.Set(resourceSinkInputsKey, inputs)
	if err != nil {
		return err
	}

	if sinkConfig.TopicsPattern != nil {
		err = d.Set(resourceSinkTopicsPatternKey, sinkConfig.TopicsPattern)
		if err != nil {
			return err
		}
	}

	if len(sinkConfig.SourceSubscriptionName) != 0 {
		err = d.Set(resourceSinkSubscriptionNameKey, sinkConfig.SourceSubscriptionName)
		if err != nil {
			return err
		}
	}

	err = d.Set(resourceSinkCleanupSubscriptionKey, sinkConfig.CleanupSubscription)
	if err != nil {
		return err
	}

	if len(sinkConfig.TopicToSerdeClassName) != 0 {
		customSerdeInputs := make(map[string]interface{}, len(sinkConfig.TopicToSerdeClassName))
		for key, value := range sinkConfig.TopicToSerdeClassName {
			customSerdeInputs[key] = value
		}
		err = d.Set(resourceSinkCustomSerdeInputsKey, customSerdeInputs)
		if err != nil {
			return err
		}
	}

	if len(sinkConfig.TopicToSchemaType) != 0 {
		customSchemaInputs := make(map[string]interface{}, len(sinkConfig.TopicToSchemaType))
		for key, value := range sinkConfig.TopicToSchemaType {
			customSchemaInputs[key] = value
		}

		err = d.Set(resourceSinkCustomSchemaInputsKey, customSchemaInputs)
		if err != nil {
			return err
		}
	}

	if len(sinkConfig.InputSpecs) > 0 {
		var inputSpecs []interface{}
		for key, config := range sinkConfig.InputSpecs {
			item := make(map[string]interface{})
			item[resourceSinkInputSpecsSubsetTopicKey] = key
			item[resourceSinkInputSpecsSubsetSchemaTypeKey] = config.SchemaType
			item[resourceSinkInputSpecsSubsetSerdeClassNameKey] = config.SerdeClassName
			item[resourceSinkInputSpecsSubsetIsRegexPatternKey] = config.IsRegexPattern
			item[resourceSinkInputSpecsSubsetReceiverQueueSizeKey] = config.ReceiverQueueSize
			inputSpecs = append(inputSpecs, item)
		}
		err = d.Set(resourceSinkInputSpecsKey, inputSpecs)
		if err != nil {
			return err
		}
	}

	err = d.Set(resourceSinkParallelismKey, sinkConfig.Parallelism)
	if err != nil {
		return err
	}

	// When the archive is built-in resource, it is not empty, otherwise it is empty.
	if sinkConfig.Archive != "" {
		err = d.Set(resourceSinkArchiveKey, sinkConfig.Archive)
		if err != nil {
			return err
		}
	}

	err = d.Set(resourceSinkClassnameKey, sinkConfig.ClassName)
	if err != nil {
		return err
	}

	if sinkConfig.Resources != nil {
		err = d.Set(resourceSinkCPUKey, sinkConfig.Resources.CPU)
		if err != nil {
			return err
		}

		err = d.Set(resourceSinkRAMKey, bytesize.FormBytes(uint64(sinkConfig.Resources.RAM)).ToMegaBytes())
		if err != nil {
			return err
		}

		err = d.Set(resourceSinkDiskKey, bytesize.FormBytes(uint64(sinkConfig.Resources.Disk)).ToMegaBytes())
		if err != nil {
			return err
		}
	}

	if len(sinkConfig.Configs) != 0 {
		b, err := json.Marshal(sinkConfig.Configs)
		if err != nil {
			return errors.Wrap(err, "cannot marshal configs from sinkConfig")
		}

		err = d.Set(resourceSinkConfigsKey, string(b))
		if err != nil {
			return err
		}
	}

	err = d.Set(resourceSinkAutoACKKey, sinkConfig.AutoAck)
	if err != nil {
		return err
	}

	if sinkConfig.TimeoutMs != nil {
		err = d.Set(resourceSinkTimeoutKey, int(*sinkConfig.TimeoutMs))
		if err != nil {
			return err
		}
	}

	if len(sinkConfig.CustomRuntimeOptions) != 0 {
		err = d.Set(resourceSinkCustomRuntimeOptionsKey, sinkConfig.CustomRuntimeOptions)
		if err != nil {
			return err
		}
	}

	return nil
}

func resourcePulsarSinkCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(pulsar.Client).Sinks()

	sinkConfig, err := marshalSinkConfig(d)
	if err != nil {
		return err
	}

	if isLocalArchive(sinkConfig.Archive) {
		err = client.CreateSink(sinkConfig, sinkConfig.Archive)
	} else {
		err = client.CreateSinkWithURL(sinkConfig, sinkConfig.Archive)
	}
	if err != nil {
		return err
	}

	return resourcePulsarSinkRead(d, meta)
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
					IsRegexPattern:    m[resourceSinkInputSpecsSubsetIsRegexPatternKey].(bool),
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

	return sinkConfig, nil
}

func isLocalArchive(archive string) bool {
	return !ctlutil.IsPackageURLSupported(archive) &&
		!strings.HasPrefix(archive, ctlutil.BUILTIN)
}
