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

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourcePulsarPermissionGrant() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourcePulsarPermissionGrantCreate,
		ReadContext:   resourcePulsarPermissionGrantRead,
		UpdateContext: resourcePulsarPermissionGrantUpdate,
		DeleteContext: resourcePulsarPermissionGrantDelete,

		Schema: map[string]*schema.Schema{
			"namespace": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The namespace in format 'tenant/namespace'",
			},
			"role": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The role to grant permissions to",
			},
			"actions": {
				Type:        schema.TypeSet,
				Required:    true,
				MinItems:    1,
				Description: "Set of actions to grant to the role",
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validateAuthAction,
				},
			},
		},
	}
}

func resourcePulsarPermissionGrantCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getClientFromMeta(meta).Namespaces()

	namespace := d.Get("namespace").(string)
	role := d.Get("role").(string)
	actionsSet := d.Get("actions").(*schema.Set)

	// Convert actions to Pulsar API format
	actions := make([]utils.AuthAction, 0, actionsSet.Len())
	for _, action := range actionsSet.List() {
		actions = append(actions, utils.AuthAction(action.(string)))
	}

	nsName, err := utils.GetNamespaceName(namespace)
	if err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_PARSE_NAMESPACE_NAME: %w", err))
	}

	if err := client.GrantNamespacePermission(*nsName, role, actions); err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_GRANT_NAMESPACE_PERMISSION: %w", err))
	}

	d.SetId(fmt.Sprintf("%s/%s", namespace, role))

	return resourcePulsarPermissionGrantRead(ctx, d, meta)
}

func resourcePulsarPermissionGrantRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getClientFromMeta(meta).Namespaces()

	namespace := d.Get("namespace").(string)
	role := d.Get("role").(string)

	nsName, err := utils.GetNamespaceName(namespace)
	if err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_PARSE_NAMESPACE_NAME: %w", err))
	}

	permissions, err := client.GetNamespacePermissions(*nsName)
	if err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE_PERMISSIONS: %w", err))
	}

	roleActions, exists := permissions[role]
	if !exists {
		d.SetId("")
		return nil
	}

	// Convert actions back to string slice
	actions := make([]string, len(roleActions))
	for i, action := range roleActions {
		actions[i] = string(action)
	}

	if err := d.Set("actions", actions); err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_SET_ACTIONS: %w", err))
	}

	return nil
}

func resourcePulsarPermissionGrantUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getClientFromMeta(meta).Namespaces()

	namespace := d.Get("namespace").(string)
	role := d.Get("role").(string)

	nsName, err := utils.GetNamespaceName(namespace)
	if err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_PARSE_NAMESPACE_NAME: %w", err))
	}

	if d.HasChange("actions") {
		// Revoke existing permissions and grant new ones
		if err := client.RevokeNamespacePermission(*nsName, role); err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_REVOKE_NAMESPACE_PERMISSION: %w", err))
		}

		actionsSet := d.Get("actions").(*schema.Set)
		actions := make([]utils.AuthAction, 0, actionsSet.Len())
		for _, action := range actionsSet.List() {
			actions = append(actions, utils.AuthAction(action.(string)))
		}

		if err := client.GrantNamespacePermission(*nsName, role, actions); err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_GRANT_NAMESPACE_PERMISSION: %w", err))
		}
	}

	return resourcePulsarPermissionGrantRead(ctx, d, meta)
}

func resourcePulsarPermissionGrantDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getClientFromMeta(meta).Namespaces()

	namespace := d.Get("namespace").(string)
	role := d.Get("role").(string)

	nsName, err := utils.GetNamespaceName(namespace)
	if err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_PARSE_NAMESPACE_NAME: %w", err))
	}

	if err := client.RevokeNamespacePermission(*nsName, role); err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_REVOKE_NAMESPACE_PERMISSION: %w", err))
	}

	return nil
}
