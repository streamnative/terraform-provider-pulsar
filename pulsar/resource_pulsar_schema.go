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
	"fmt"
	"strings"

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourcePulsarSchema() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourcePulsarSchemaCreate,
		ReadContext:   resourcePulsarSchemaRead,
		UpdateContext: resourcePulsarSchemaUpdate,
		DeleteContext: resourcePulsarSchemaDelete,
		Description: "Manages Pulsar schemas for topics. Schemas provide a way to enforce structure and compatibility" +
			" for messages in Pulsar topics. Schema updates create new versions rather than modifying existing" +
			" schemas, enabling schema evolution and compatibility validation.",

		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				// Parse import ID: tenant/namespace/topic
				parts := strings.Split(d.Id(), "/")
				if len(parts) != 3 {
					return nil, fmt.Errorf("invalid import ID format. Expected: tenant/namespace/topic")
				}

				_ = d.Set("tenant", parts[0])
				_ = d.Set("namespace", parts[1])
				_ = d.Set("topic", parts[2])

				diags := resourcePulsarSchemaRead(ctx, d, meta)
				if diags.HasError() {
					return nil, fmt.Errorf("import %q: %s", d.Id(), diags[0].Summary)
				}

				return []*schema.ResourceData{d}, nil
			},
		},

		Schema: map[string]*schema.Schema{
			"tenant": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The tenant name. Cannot be changed after creation.",
			},
			"namespace": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The namespace name. Cannot be changed after creation.",
			},
			"topic": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The topic name. Cannot be changed after creation.",
			},
			"type": {
				Type:     schema.TypeString,
				Required: true,
				Description: "Schema type. Supported types include:\n  - **Primitive types**: `STRING`, `INT8`," +
					" `INT16`, `INT32`, `INT64`, `FLOAT`, `DOUBLE`, `BOOLEAN`, `BYTES`, `TIMESTAMP`, `DATE`," +
					" `TIME`, `INSTANT`, `LOCAL_DATE`, `LOCAL_TIME`, `LOCAL_DATE_TIME`\n  - **Complex types**:" +
					" `AVRO`, `JSON`, `PROTOBUF`, `PROTOBUF_NATIVE`, `KEY_VALUE`\n  - **Auto types**: `AUTO_CONSUME`," +
					" `AUTO_PUBLISH`, `AUTO_PRODUCE_BYTES`",
				ValidateFunc: validateSchemaType,
			},
			"schema_data": {
				Type:     schema.TypeString,
				Optional: true,
				Description: " Required for complex types (`AVRO`, `JSON`, `PROTOBUF`, `PROTOBUF_NATIVE`," +
					" `KEY_VALUE`). Not needed for primitive types.",
			},
			"properties": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Additional schema properties as key-value pairs.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},

			// Computed fields
			"version": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Current schema version. Automatically incremented when schema is updated.",
			},
			"timestamp": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Schema creation/update timestamp",
			},
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of this resource in the format `tenant/namespace/topic`.",
			},
		},
	}
}

func resourcePulsarSchemaCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getClientFromMeta(meta).Schemas()

	tenant := d.Get("tenant").(string)
	namespace := d.Get("namespace").(string)
	topic := d.Get("topic").(string)
	schemaType := d.Get("type").(string)
	schemaData := d.Get("schema_data").(string)
	properties := d.Get("properties").(map[string]interface{})

	// Build topic name
	topicName, err := utils.GetTopicName(fmt.Sprintf("persistent://%s/%s/%s", tenant, namespace, topic))
	if err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_PARSE_TOPIC_NAME: %w", err))
	}

	// Validate schema data requirements based on type
	if err := validateSchemaDataRequirement(schemaType, schemaData); err != nil {
		return diag.FromErr(err)
	}

	// Convert properties to string map
	props := make(map[string]string)
	for k, v := range properties {
		if str, ok := v.(string); ok {
			props[k] = str
		} else {
			props[k] = fmt.Sprintf("%v", v)
		}
	}

	// Create schema payload
	payload := utils.PostSchemaPayload{
		SchemaType: schemaType,
		Schema:     schemaData,
		Properties: props,
	}

	// Create schema
	if err := client.CreateSchemaByPayload(topicName.String(), payload); err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_CREATE_SCHEMA: %w", err))
	}

	// Set resource ID
	d.SetId(fmt.Sprintf("%s/%s/%s", tenant, namespace, topic))

	// Read back to populate computed fields
	return resourcePulsarSchemaRead(ctx, d, meta)
}

func resourcePulsarSchemaRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getClientFromMeta(meta).Schemas()

	tenant := d.Get("tenant").(string)
	namespace := d.Get("namespace").(string)
	topic := d.Get("topic").(string)

	// Build topic name
	topicName, err := utils.GetTopicName(fmt.Sprintf("persistent://%s/%s/%s", tenant, namespace, topic))
	if err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_PARSE_TOPIC_NAME: %w", err))
	}

	// Get current schema with version
	schemaWithVersion, err := client.GetSchemaInfoWithVersion(topicName.String())
	if err != nil {
		// Handle 404 - schema doesn't exist
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("ERROR_READ_SCHEMA: %w", err))
	}

	// Update resource data
	_ = d.Set("tenant", tenant)
	_ = d.Set("namespace", namespace)
	_ = d.Set("topic", topic)
	_ = d.Set("type", schemaWithVersion.SchemaInfo.Type)
	_ = d.Set("schema_data", string(schemaWithVersion.SchemaInfo.Schema))
	_ = d.Set("properties", schemaWithVersion.SchemaInfo.Properties)

	// Set computed fields
	_ = d.Set("version", int(schemaWithVersion.Version))
	// TODO: timestamp support in pulsar-client-go: https://github.com/apache/pulsar-client-go/pull/1436
	// _ = d.Set("timestamp", time.Now().Format(time.RFC3339))

	return nil
}

func resourcePulsarSchemaUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Schema updates create new versions, so this is effectively a new upload
	client := getClientFromMeta(meta).Schemas()

	tenant := d.Get("tenant").(string)
	namespace := d.Get("namespace").(string)
	topic := d.Get("topic").(string)
	schemaType := d.Get("type").(string)
	schemaData := d.Get("schema_data").(string)
	properties := d.Get("properties").(map[string]interface{})

	// Build topic name
	topicName, err := utils.GetTopicName(fmt.Sprintf("persistent://%s/%s/%s", tenant, namespace, topic))
	if err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_PARSE_TOPIC_NAME: %w", err))
	}

	// Validate schema data requirements based on type
	if err := validateSchemaDataRequirement(schemaType, schemaData); err != nil {
		return diag.FromErr(err)
	}

	// Convert properties to string map
	props := make(map[string]string)
	for k, v := range properties {
		if str, ok := v.(string); ok {
			props[k] = str
		} else {
			props[k] = fmt.Sprintf("%v", v)
		}
	}

	// Create schema payload
	payload := utils.PostSchemaPayload{
		SchemaType: schemaType,
		Schema:     schemaData,
		Properties: props,
	}

	// Upload new schema version
	if err := client.CreateSchemaByPayload(topicName.String(), payload); err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_UPDATE_SCHEMA: %w", err))
	}

	// Read back to populate computed fields
	return resourcePulsarSchemaRead(ctx, d, meta)
}

func resourcePulsarSchemaDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getClientFromMeta(meta).Schemas()

	tenant := d.Get("tenant").(string)
	namespace := d.Get("namespace").(string)
	topic := d.Get("topic").(string)

	// Build topic name
	topicName, err := utils.GetTopicName(fmt.Sprintf("persistent://%s/%s/%s", tenant, namespace, topic))
	if err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_PARSE_TOPIC_NAME: %w", err))
	}

	// Delete all schema versions
	if err := client.DeleteSchema(topicName.String()); err != nil {
		// Ignore 404 errors - schema might have been deleted externally
		if !strings.Contains(err.Error(), "404") && !strings.Contains(err.Error(), "not found") {
			return diag.FromErr(fmt.Errorf("ERROR_DELETE_SCHEMA: %w", err))
		}
	}

	d.SetId("")
	return nil
}

// validateSchemaType validates that the schema type is one of the supported types
func validateSchemaType(v interface{}, k string) (warnings []string, errors []error) {
	value := v.(string)
	validTypes := []string{
		// Primitive types
		"STRING", "INT8", "INT16", "INT32", "INT64", "FLOAT", "DOUBLE", "BOOLEAN", "BYTES",
		"TIMESTAMP", "DATE", "TIME", "INSTANT", "LOCAL_DATE", "LOCAL_TIME", "LOCAL_DATE_TIME",
		// Complex types
		"AVRO", "JSON", "PROTOBUF", "PROTOBUF_NATIVE", "KEY_VALUE",
		// Auto types
		"AUTO_CONSUME", "AUTO_PUBLISH", "AUTO_PRODUCE_BYTES",
	}

	for _, validType := range validTypes {
		if value == validType {
			return
		}
	}

	errors = append(errors, fmt.Errorf("%s must be one of: %s", k, strings.Join(validTypes, ", ")))
	return
}

// validateSchemaDataRequirement validates that schema_data is provided when required
func validateSchemaDataRequirement(schemaType, schemaData string) error {
	// Complex types that require schema data
	complexTypes := map[string]bool{
		"AVRO":            true,
		"JSON":            true,
		"PROTOBUF":        true,
		"PROTOBUF_NATIVE": true,
		"KEY_VALUE":       true,
	}

	if complexTypes[schemaType] && strings.TrimSpace(schemaData) == "" {
		return fmt.Errorf("schema_data is required for schema type %s", schemaType)
	}

	return nil
}
