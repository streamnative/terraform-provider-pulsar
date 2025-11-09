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
		resourceFunctionTopicsPatternKey:        "The input topics pattern of the function. The pattern is a regex expression. The function consumes from all topics matching the pattern.",
		resourceFunctionOutputKey:               "The output topic of the function.",
		resourceFunctionParallelismKey:          "The parallelism of the function.",
		resourceFunctionProcessingGuaranteesKey: "The processing guarantees (aka delivery semantics) applied to the function. Possible values are `ATMOST_ONCE`, `ATLEAST_ONCE`, and `EFFECTIVELY_ONCE`.",
		resourceFunctionSubscriptionNameKey:     "The subscription name of the function.",
		resourceFunctionSubscriptionPositionKey: "The subscription position of the function. Possible values are `LATEST`, `EARLIEST`, and `CUSTOM`.",
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
	}
}

func resourcePulsarFunction() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourcePulsarFunctionCreate,
		ReadContext:   resourcePulsarFunctionRead,
		UpdateContext: resourcePulsarFunctionUpdate,
		DeleteContext: resourcePulsarFunctionDelete,
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
				ForceNew:    true,
				Description: resourceFunctionDescriptions[resourceFunctionInputsKey],
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			resourceFunctionTopicsPatternKey: {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
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
				Description: resourceFunctionDescriptions[resourceFunctionInputTypeClassNameKey],
			},
			resourceFunctionOutputTypeClassNameKey: {
				Type:        schema.TypeString,
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

	updateOptions := utils.NewUpdateOptions()
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

	if inter, ok := d.GetOk(resourceFunctionInputsKey); ok {
		inputsSet := inter.(*schema.Set)
		var inputs []string

		for _, item := range inputsSet.List() {
			inputs = append(inputs, item.(string))
		}

		functionConfig.Inputs = inputs
	}

	if inter, ok := d.GetOk(resourceFunctionOutputKey); ok {
		functionConfig.Output = inter.(string)
	}

	if inter, ok := d.GetOk(resourceFunctionTopicsPatternKey); ok {
		pattern := inter.(string)
		functionConfig.TopicsPattern = &pattern
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
		interMap := inter.(map[string]interface{})
		stringMap := make(map[string]string, len(interMap))

		for key, value := range interMap {
			stringMap[key] = value.(string)
		}

		functionConfig.CustomSerdeInputs = stringMap
	}

	if inter, ok := d.GetOk(resourceFunctionCustomSchemaInputsKey); ok {
		interMap := inter.(map[string]interface{})
		stringMap := make(map[string]string, len(interMap))

		for key, value := range interMap {
			stringMap[key] = value.(string)
		}

		functionConfig.CustomSchemaInputs = stringMap
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

	return functionConfig, nil
}

func unmarshalFunctionConfig(functionConfig utils.FunctionConfig, d *schema.ResourceData) error {
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
		sanitizedOptions, sinkConfig, sinkConfigPresent, sourceConfig, sourceConfigPresent, err := splitFunctionCustomRuntimeOptions(functionConfig.CustomRuntimeOptions)
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

func expandFunctionRuntimeConfig(d *schema.ResourceData, def runtimeConfigDefinition) (map[string]interface{}, bool, error) {
	inter, ok := d.GetOkExists(def.schemaKey)
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

	if configsRaw, ok := item[resourceFunctionRuntimeConfigConfigsKey].(map[string]interface{}); ok && len(configsRaw) > 0 {
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

func splitFunctionCustomRuntimeOptions(raw string) (string, map[string]interface{}, bool, map[string]interface{}, bool, error) {
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

func extractRuntimeConfig(runtimeOptions map[string]interface{}, def runtimeConfigDefinition) (map[string]interface{}, bool, error) {
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
	if config == nil || len(config) == 0 {
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
