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
	"io/ioutil"
	"strings"

	"github.com/streamnative/pulsarctl/pkg/cli"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/pkg/errors"
	ctlutil "github.com/streamnative/pulsarctl/pkg/ctl/utils"
	"github.com/streamnative/pulsarctl/pkg/pulsar"
	"github.com/streamnative/pulsarctl/pkg/pulsar/utils"

	"gopkg.in/yaml.v2"
)

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
					return nil, errors.New("id should be tenant/namespace/name format")
				}

				_ = d.Set("tenant", parts[0])
				_ = d.Set("namespace", parts[1])
				_ = d.Set("name", parts[1])

				err := resourcePulsarSinkRead(d, meta)
				return []*schema.ResourceData{d}, err
			},
		},
		Schema: map[string]*schema.Schema{
			"tenant": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions["tenant"],
			},
			"namespace": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions["namespace"],
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions["name"],
			},
			"inputs": {
				Type:        schema.TypeSet,
				Required:    false,
				Description: descriptions["inputs"],
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"topics_pattern": {
				Type:        schema.TypeString,
				Required:    false,
				Description: descriptions["topics_pattern"],
			},
			"subscription_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions["subscription_name"],
			},
			"subscription_position": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions["subscription_position"],
			},
			"custom_serde_inputs": {
				Type:        schema.TypeMap,
				Required:    false,
				Description: descriptions["custom_serde_inputs"],
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"custom_schema_inputs": {
				Type:        schema.TypeMap,
				Required:    false,
				Description: descriptions["custom_schema_inputs"],
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"input_specs": {
				Type:        schema.TypeMap,
				Required:    false,
				Description: descriptions["input_specs"],
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"schema_type":         {Type: schema.TypeString},
						"serde_class_name":    {Type: schema.TypeString},
						"is_regex_pattern":    {Type: schema.TypeBool},
						"receiver_queue_size": {Type: schema.TypeInt},
					},
				},
			},
			"processing_guarantees": {
				Type:        schema.TypeString,
				Required:    false,
				Description: descriptions["processing_guarantees"],
			},
			"retain_ordering": {
				Type:        schema.TypeBool,
				Required:    false,
				Description: descriptions["retain_ordering"],
			},
			"parallelism": {
				Type:        schema.TypeInt,
				Required:    false,
				Description: descriptions["parallelism"],
			},
			"archive": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions["archive"],
			},
			"classname": {
				Type:        schema.TypeString,
				Required:    false,
				Description: descriptions["classname"],
			},
			"sink_config_file": {
				Type:        schema.TypeString,
				Required:    false,
				Description: descriptions["sink_config_file"],
			},
			"cpu": {
				Type:        schema.TypeFloat,
				Required:    false,
				Description: descriptions["cpu"],
			},
			"ram": {
				Type:        schema.TypeFloat,
				Required:    false,
				Description: descriptions["ram"],
			},
			"disk": {
				Type:        schema.TypeFloat,
				Required:    false,
				Description: descriptions["disk"],
			},
			"sink_config": {
				Type:        schema.TypeMap,
				Required:    false,
				Description: descriptions["sink_config"],
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"auto_ack": {
				Type:        schema.TypeBool,
				Required:    false,
				Description: descriptions["auto_ack"],
			},
			"timeout_ms": {
				Type:        schema.TypeInt,
				Required:    false,
				Description: descriptions["timeout_ms"],
			},
			"custom_runtime_options": {
				Type:        schema.TypeString,
				Required:    false,
				Description: descriptions["custom_runtime_options"],
			},
		},
	}
}

func resourcePulsarSinkExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(pulsar.Client).Sinks()

	tenant := d.Get("tenant").(string)
	namespace := d.Get("namespace").(string)
	name := d.Get("name").(string)

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

	tenant := d.Get("tenant").(string)
	namespace := d.Get("namespace").(string)
	name := d.Get("name").(string)

	return client.DeleteSink(tenant, namespace, name)
}

func resourcePulsarSinkUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(pulsar.Client).Sinks()
	sinkConfig, err := getSinkConfig(d)
	if err != nil {
		return err
	}

	updateOptions := utils.NewUpdateOptions()
	if !ctlutil.IsPackageURLSupported(sinkConfig.Archive) &&
		!strings.HasPrefix(sinkConfig.Archive, ctlutil.BUILTIN) {
		return client.UpdateSink(sinkConfig, sinkConfig.Archive, updateOptions)
	} else {
		return client.UpdateSinkWithURL(sinkConfig, sinkConfig.Archive, updateOptions)
	}
}

func resourcePulsarSinkRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(pulsar.Client).Sinks()

	tenant := d.Get("tenant").(string)
	namespace := d.Get("namespace").(string)
	name := d.Get("name").(string)

	sinkConfig, err := client.GetSink(tenant, namespace, name)
	if err != nil {
		return errors.Wrapf(err, "failed to get %s sink from %s/%s", name, tenant, namespace)
	}

	//sinkConfig.Inputs
	var inputs []interface{}
	for _, input := range sinkConfig.Inputs {
		inputs = append(inputs, input)
	}

	err = d.Set("inputs", inputs)
	if err != nil {
		return errors.Wrapf(err, "failed to set inputs")
	}

	err = d.Set("topics_pattern", sinkConfig.TopicsPattern)
	if err != nil {
		return errors.Wrapf(err, "failed to set topics_pattern")
	}

	err = d.Set("subscription_name", sinkConfig.SourceSubscriptionName)
	if err != nil {
		return errors.Wrapf(err, "failed to set subscription_name")
	}

	err = d.Set("subscription_position", sinkConfig.SourceSubscriptionPosition)
	if err != nil {
		return errors.Wrapf(err, "failed to set subscription_position")
	}

	customSerdeInputs := make(map[string]interface{}, len(sinkConfig.TopicToSerdeClassName))
	for key, value := range sinkConfig.TopicToSerdeClassName {
		customSerdeInputs[key] = value
	}
	err = d.Set("custom_serde_inputs", customSerdeInputs)
	if err != nil {
		return errors.Wrapf(err, "failed to set custom_serde_inputs")
	}

	customSchemaInputs := make(map[string]interface{}, len(sinkConfig.TopicToSchemaType))
	for key, value := range sinkConfig.TopicToSchemaType {
		customSchemaInputs[key] = value
	}
	err = d.Set("custom_schema_inputs", customSchemaInputs)
	if err != nil {
		return errors.Wrapf(err, "failed to set custom_schema_inputs")
	}

	inputSpecs := make(map[string]interface{}, len(sinkConfig.InputSpecs))
	if len(sinkConfig.InputSpecs) > 0 {
		for key, config := range sinkConfig.InputSpecs {
			value := make(map[string]interface{})
			value["schema_type"] = config.SchemaType
			value["serde_class_name"] = config.SerdeClassName
			value["is_regex_pattern"] = config.IsRegexPattern
			value["receiver_queue_size"] = config.ReceiverQueueSize
			inputSpecs[key] = value
		}
	}

	err = d.Set("input_specs", inputSpecs)
	if err != nil {
		return errors.Wrapf(err, "failed to set input_specs")
	}

	err = d.Set("processing_guarantees", sinkConfig.ProcessingGuarantees)
	if err != nil {
		return errors.Wrapf(err, "failed to set processing_guarantees")
	}

	err = d.Set("retain_ordering", sinkConfig.RetainOrdering)
	if err != nil {
		return errors.Wrapf(err, "failed to set retain_ordering")
	}

	err = d.Set("parallelism", sinkConfig.Parallelism)
	if err != nil {
		return errors.Wrapf(err, "failed to set parallelism")
	}

	err = d.Set("archive", sinkConfig.Archive)
	if err != nil {
		return errors.Wrapf(err, "failed to set archive")
	}

	err = d.Set("classname", sinkConfig.ClassName)
	if err != nil {
		return errors.Wrapf(err, "failed to set classname")
	}

	if sinkConfig.Resources != nil {
		err = d.Set("cpu", sinkConfig.Resources.CPU)
		if err != nil {
			return errors.Wrapf(err, "failed to set cpu")
		}

		err = d.Set("ram", float64(sinkConfig.Resources.RAM))
		if err != nil {
			return errors.Wrapf(err, "failed to set ram")
		}

		err = d.Set("disk", float64(sinkConfig.Resources.Disk))
		if err != nil {
			return errors.Wrapf(err, "failed to set disk")
		}
	}

	err = d.Set("sink_config", sinkConfig.Configs)
	if err != nil {
		return errors.Wrapf(err, "failed to set sink_config")
	}

	err = d.Set("auto_ack", sinkConfig.AutoAck)
	if err != nil {
		return errors.Wrapf(err, "failed to set auto_ack")
	}

	if sinkConfig.TimeoutMs != nil {
		err = d.Set("timeout_ms", int(*sinkConfig.TimeoutMs))
		if err != nil {
			return errors.Wrapf(err, "failed to set timeout_ms")
		}
	}

	err = d.Set("custom_runtime_options", sinkConfig.CustomRuntimeOptions)
	if err != nil {
		return errors.Wrapf(err, "failed to set custom_runtime_options")
	}

	return nil
}

func resourcePulsarSinkCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(pulsar.Client).Sinks()

	sinkConfig, err := getSinkConfig(d)
	if err != nil {
		return err
	}

	if !ctlutil.IsPackageURLSupported(sinkConfig.Archive) &&
		!strings.HasPrefix(sinkConfig.Archive, ctlutil.BUILTIN) {
		return client.CreateSink(sinkConfig, sinkConfig.Archive)
	} else {
		return client.CreateSinkWithURL(sinkConfig, sinkConfig.Archive)
	}
}

func getSinkConfig(d *schema.ResourceData) (*utils.SinkConfig, error) {
	sinkConfig := &utils.SinkConfig{}

	if configFilePathInter, ok := d.GetOk("sink_config_file"); ok {
		configFilePath := configFilePathInter.(string)

		bytes, err := ioutil.ReadFile(configFilePath)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read config file: %s", configFilePath)
		}

		err = yaml.Unmarshal(bytes, &sinkConfig)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse the config file: %s", configFilePath)
		}
		// continue to override the sink config
	}

	if inter, ok := d.GetOk("tenant"); ok {
		sinkConfig.Tenant = inter.(string)
	}

	if inter, ok := d.GetOk("namespace"); ok {
		sinkConfig.Namespace = inter.(string)
	}

	if inter, ok := d.GetOk("name"); ok {
		sinkConfig.Name = inter.(string)
	}

	if inter, ok := d.GetOk("inputs"); ok {
		inputsSet := inter.(*schema.Set)
		inputs := make([]string, inputsSet.Len())

		for _, item := range inputsSet.List() {
			inputs = append(inputs, item.(string))
		}

		sinkConfig.Inputs = inputs
	}

	if inter, ok := d.GetOk("topics_pattern"); ok {
		pattern := inter.(string)
		sinkConfig.TopicsPattern = &pattern
	}

	if inter, ok := d.GetOk("subscription_name"); ok {
		sinkConfig.SourceSubscriptionName = inter.(string)
	}

	if inter, ok := d.GetOk("subscription_position"); ok {
		sinkConfig.SourceSubscriptionPosition = inter.(string)
	}

	if inter, ok := d.GetOk("custom_serde_inputs"); ok {
		interMap := inter.(map[string]interface{})
		stringMap := make(map[string]string, len(interMap))

		for key, value := range interMap {
			stringMap[key] = value.(string)
		}

		sinkConfig.TopicToSerdeClassName = stringMap
	}

	if inter, ok := d.GetOk("custom_schema_inputs"); ok {
		interMap := inter.(map[string]interface{})
		stringMap := make(map[string]string, len(interMap))

		for key, value := range interMap {
			stringMap[key] = value.(string)
		}

		sinkConfig.TopicToSchemaType = stringMap
	}

	if inter, ok := d.GetOk("input_specs"); ok {
		interMap := inter.(map[string]interface{})
		inputSpecs := make(map[string]utils.ConsumerConfig, len(interMap))

		for inputSpecKey, inputSpecValueInter := range interMap {
			inputSpec := utils.ConsumerConfig{}
			interMap := inputSpecValueInter.(map[string]interface{})

			value, found := interMap["schema_type"]
			if found {
				inputSpec.SchemaType = value.(string)
			}

			value, found = interMap["serde_class_name"]
			if found {
				inputSpec.SerdeClassName = value.(string)
			}

			value, found = interMap["is_regex_pattern"]
			if found {
				inputSpec.IsRegexPattern = value.(bool)
			}

			value, found = interMap["receiver_queue_size"]
			if found {
				inputSpec.ReceiverQueueSize = value.(int)
			}

			inputSpecs[inputSpecKey] = inputSpec
		}

		sinkConfig.InputSpecs = inputSpecs
	}

	if inter, ok := d.GetOk("processing_guarantees"); ok {
		sinkConfig.ProcessingGuarantees = inter.(string)
	}

	if inter, ok := d.GetOk("retain_ordering"); ok {
		sinkConfig.RetainOrdering = inter.(bool)
	}

	if inter, ok := d.GetOk("parallelism"); ok {
		sinkConfig.Parallelism = inter.(int)
	}

	if inter, ok := d.GetOk("archive"); ok {
		sinkConfig.Archive = inter.(string)
	}

	if inter, ok := d.GetOk("classname"); ok {
		sinkConfig.ClassName = inter.(string)
	}

	var resource *utils.Resources

	if inter, ok := d.GetOk("cpu"); ok {
		if resource == nil {
			resource = utils.NewDefaultResources()
		}

		value := inter.(float64)
		resource.CPU = value
	}

	if inter, ok := d.GetOk("ram"); ok {
		if resource == nil {
			resource = utils.NewDefaultResources()
		}
		value := int64(inter.(float64))
		resource.RAM = value
	}

	if inter, ok := d.GetOk("disk"); ok {
		if resource == nil {
			resource = utils.NewDefaultResources()
		}
		value := int64(inter.(float64))
		resource.Disk = value
	}

	if resource != nil {
		sinkConfig.Resources = resource
	}

	if inter, ok := d.GetOk("sink_config"); ok {
		sinkConfig.Configs = inter.(map[string]interface{})
	}

	if inter, ok := d.GetOk("auto_ack"); ok {
		sinkConfig.AutoAck = inter.(bool)
	}

	if inter, ok := d.GetOk("timeout_ms"); ok {
		value := int64(inter.(int))
		sinkConfig.TimeoutMs = &value
	}

	if inter, ok := d.GetOk("custom_runtime_options"); ok {
		sinkConfig.CustomRuntimeOptions = inter.(string)
	}

	return sinkConfig, nil
}
