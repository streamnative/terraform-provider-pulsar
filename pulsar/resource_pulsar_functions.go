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
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/streamnative/pulsarctl/pkg/pulsar/utils"
)

func resourcePulsarFunctions() *schema.Resource {
	return &schema.Resource{
		Create: resourcePulsarFunctionCreate,
		Read:   resourcePulsarFunctionRead,
		Update: resourcePulsarFunctionUpdate,
		Delete: resourcePulsarFunctionDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"pattern": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"tenant": {
				Type:     schema.TypeString,
				Required: true,
			},
			"namespace": {
				Type:     schema.TypeString,
				Required: true,
			},
			"timeout": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  100,
			},
			"auto_acknowledgement": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"output": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"log_topic": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"output_schema": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"function_path": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"class_name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"jar_file": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"py_file": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"go_file": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"file_path": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"fqfn": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"inputs": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func resourcePulsarFunctionCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(tfPulsarClient).Functions()

	tenant := d.Get("tenant").(string)
	namespace := d.Get("namespace").(string)
	name := d.Get("name").(string)
	inputs := handleHCLArray(d, "inputs")
	output := d.Get("output").(string)
	className := d.Get("class_name").(string)
	jarFile := d.Get("jar_file").(string)

	fqfn := fmt.Sprintf("%s/%s/%s", tenant, namespace, name)

	fn := &utils.FunctionConfig{
		Tenant:               tenant,
		Namespace:            namespace,
		Name:                 name,
		Parallelism:          1,
		Inputs:               inputs,
		Output:               output,
		ClassName:            className,
		Jar:                  &jarFile,
		Resources:            utils.NewDefaultResources(),
		Runtime:              utils.JavaRuntime,
		ProcessingGuarantees: "EFFECTIVELY_ONCE",
		FQFN:                 fqfn,
		//WindowConfig:         &utils.WindowConfig{
		//	WindowLengthCount:             nil,
		//	WindowLengthDurationMs:        nil,
		//	SlidingIntervalCount:          nil,
		//	SlidingIntervalDurationMs:     nil,
		//	LateDataTopic:                 nil,
		//	MaxLagMs:                      nil,
		//	WatermarkEmitIntervalMs:       nil,
		//	TimestampExtractorClassName:   nil,
		//	ActualWindowFunctionClassName: nil,
		//},
		//DeadLetterTopic:      "",
		//OutputSerdeClassName: "",
		//TopicsPattern:        nil,
	}

	// functions localrun --inputs test_input --output test_output
	// --className org.apache.pulsar.functions.api.examples.JavaNativeExclamationFunction
	//--jar exclaim.jar

	if err := client.CreateFunc(fn, jarFile); err != nil {
		return fmt.Errorf("ERROR_CREATE_PULSAR_FUNCTION: %w \n input: %v -- %s", err, fn.Jar, *fn.Jar)
	}

	_ = d.Set("fqfn", fqfn)
	return resourcePulsarFunctionRead(d, meta)
}

func resourcePulsarFunctionRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(tfPulsarClient).Functions()

	tenant := d.Get("tenant").(string)
	name := d.Get("name").(string)
	namespace := d.Get("namespace").(string)
	fqfn := d.Get("fqfn").(string)

	_, err := client.GetFunction(tenant, namespace, name)
	if err != nil {
		return fmt.Errorf("ERROR_READING_PULSAR_FUNCTION: %w", err)
	}

	_ = d.Set("tenant", tenant)
	_ = d.Set("namespace", namespace)
	_ = d.Set("name", name)
	d.SetId(fqfn)

	return nil
}

func resourcePulsarFunctionUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(tfPulsarClient).Functions()

	tenant := d.Get("tenant").(string)
	namespace := d.Get("namespace").(string)
	name := d.Get("name").(string)
	inputs := handleHCLArray(d, "inputs")
	output := d.Get("output").(string)
	className := d.Get("class_name").(string)
	jarFile := d.Get("jar_file").(string)
	fqfn := fmt.Sprintf("%s/%s/%s", tenant, namespace, name)

	fn := &utils.FunctionConfig{
		Tenant:               tenant,
		Namespace:            namespace,
		Name:                 name,
		Parallelism:          1,
		Inputs:               inputs,
		Output:               output,
		ClassName:            className,
		Jar:                  &jarFile,
		Resources:            utils.NewDefaultResources(),
		Runtime:              utils.JavaRuntime,
		ProcessingGuarantees: "EFFECTIVELY_ONCE",
		FQFN:                 fqfn,
	}

	if err := client.UpdateFunction(fn, jarFile, nil); err != nil {
		return fmt.Errorf("ERROR_UPDATE_PULSAR_FUNCTION: %w", err)
	}

	return resourcePulsarFunctionRead(d, meta)
}

func resourcePulsarFunctionDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(tfPulsarClient).Functions()

	tenant := d.Get("tenant").(string)
	namespace := d.Get("namespace").(string)
	name := d.Get("name").(string)

	if err := client.DeleteFunction(tenant, namespace, name); err != nil {
		return fmt.Errorf("ERROR_DELETE_PULSAR_FUNCTION: %w", err)
	}

	_ = d.Set("name", "")
	_ = d.Set("tenant", "")
	_ = d.Set("namespace", "")

	return nil
}

func Int64P(v int) *int64 {
	result := int64(v)
	return &result
}

func IntP(v int) *int {
	return &v
}

func StringP(v string) *string {
	return &v
}
