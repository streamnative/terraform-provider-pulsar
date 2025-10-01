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

		Description: `Provides a resource for managing permissions on either Pulsar namespaces or topics. 
Permission can be granted to specific roles using this resource. This resource is optional and can be used
to manage permissions for roles outside of the namespace or topic resource lifecycle.

**Note:** It is not recommended to use this resource in conjunction with the ` + "`permission_grant`" + `
attributes of the ` + "`pulsar_namespace`" + ` or ` + "`pulsar_topic`" + ` resources for the same role.
Doing so will result in the resources continuously modifying the permission state.
See the ` + "`permission_grant`" + ` attribute of ` + "`pulsar_namespace`" + ` and ` + "`pulsar_topic`" +
			` resources for more information.`,

		Schema: map[string]*schema.Schema{
			"namespace": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Description: "The Pulsar namespace. Format: tenant/namespace. " +
					"One of namespace or topic **must** be specified.",
				ExactlyOneOf: []string{"namespace", "topic"},
			},
			"topic": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Description: "The Pulsar topic. Format: persistent://tenant/namespace/topic or " +
					"non-persistent://tenant/namespace/topic. " +
					"One of namespace or topic **must** be specified.",
				ExactlyOneOf: []string{"namespace", "topic"},
			},
			"role": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the Pulsar role to grant permissions to",
			},
			"actions": {
				Type:        schema.TypeSet,
				Required:    true,
				MinItems:    1,
				Description: "A set of authorization actions granted to the role.",
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validateAuthAction,
				},
			},
		},
	}
}

func resourcePulsarPermissionGrantCreate(ctx context.Context, d *schema.ResourceData,
	meta interface{}) diag.Diagnostics {
	client := getClientFromMeta(meta)

	role := d.Get("role").(string)
	actionsSet := d.Get("actions").(*schema.Set)

	actions := make([]utils.AuthAction, 0, actionsSet.Len())
	for _, action := range actionsSet.List() {
		auth, err := utils.ParseAuthAction(action.(string))
		if err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_PARSE_AUTH_ACTION: %w", err))
		}
		actions = append(actions, auth)
	}

	if namespace := d.Get("namespace").(string); namespace != "" {
		nsName, err := utils.GetNamespaceName(namespace)
		if err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_PARSE_NAMESPACE_NAME: %w", err))
		}

		if err = client.Namespaces().GrantNamespacePermission(*nsName, role, actions); err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_GRANT_NAMESPACE_PERMISSION: %w", err))
		}

		d.SetId(fmt.Sprintf("%s/%s", namespace, role))

	} else if topic := d.Get("topic").(string); topic != "" {
		topicName, err := utils.GetTopicName(topic)
		if err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_PARSE_TOPIC_NAME: %w", err))
		}

		if err = client.Topics().GrantPermission(*topicName, role, actions); err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_GRANT_TOPIC_PERMISSION: %w", err))
		}

		d.SetId(fmt.Sprintf("%s/%s", topic, role))
	}

	return resourcePulsarPermissionGrantRead(ctx, d, meta)
}

func resourcePulsarPermissionGrantRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getClientFromMeta(meta)
	role := d.Get("role").(string)

	var grants map[string][]utils.AuthAction
	var err error

	if namespace := d.Get("namespace").(string); namespace != "" {
		nsName, parseErr := utils.GetNamespaceName(namespace)
		if parseErr != nil {
			return diag.FromErr(fmt.Errorf("ERROR_PARSE_NAMESPACE_NAME: %w", parseErr))
		}

		grants, err = client.Namespaces().GetNamespacePermissions(*nsName)
		if err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE_PERMISSION_GRANT: %w", err))
		}

	} else if topic := d.Get("topic").(string); topic != "" {
		topicName, parseErr := utils.GetTopicName(topic)
		if parseErr != nil {
			return diag.FromErr(fmt.Errorf("ERROR_PARSE_TOPIC_NAME: %w", parseErr))
		}

		grants, err = client.Topics().GetPermissions(*topicName)
		if err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_READ_TOPIC_PERMISSION_GRANT: %w", err))
		}
	}

	if actions, exists := grants[role]; exists && len(actions) > 0 {
		actionsSet := schema.NewSet(schema.HashString, []interface{}{})
		for _, action := range actions {
			actionsSet.Add(action.String())
		}
		_ = d.Set("actions", actionsSet)
	} else {
		d.SetId("")
	}

	return nil
}

func resourcePulsarPermissionGrantUpdate(ctx context.Context, d *schema.ResourceData,
	meta interface{}) diag.Diagnostics {
	client := getClientFromMeta(meta)

	if d.HasChange("actions") {
		role := d.Get("role").(string)
		actionsSet := d.Get("actions").(*schema.Set)

		actions := make([]utils.AuthAction, 0, actionsSet.Len())
		for _, action := range actionsSet.List() {
			auth, err := utils.ParseAuthAction(action.(string))
			if err != nil {
				return diag.FromErr(fmt.Errorf("ERROR_PARSE_AUTH_ACTION: %w", err))
			}
			actions = append(actions, auth)
		}

		if namespace := d.Get("namespace").(string); namespace != "" {
			nsName, err := utils.GetNamespaceName(namespace)
			if err != nil {
				return diag.FromErr(fmt.Errorf("ERROR_PARSE_NAMESPACE_NAME: %w", err))
			}

			// Revoke and re-grant
			if err = client.Namespaces().RevokeNamespacePermission(*nsName, role); err != nil {
				return diag.FromErr(fmt.Errorf("ERROR_UPDATE_NAMESPACE_PERMISSION_GRANT: %w", err))
			}

			if err = client.Namespaces().GrantNamespacePermission(*nsName, role, actions); err != nil {
				return diag.FromErr(fmt.Errorf("ERROR_UPDATE_NAMESPACE_PERMISSION_GRANT: %w", err))
			}

		} else if topic := d.Get("topic").(string); topic != "" {
			topicName, err := utils.GetTopicName(topic)
			if err != nil {
				return diag.FromErr(fmt.Errorf("ERROR_PARSE_TOPIC_NAME: %w", err))
			}

			// Revoke and re-grant
			if err = client.Topics().RevokePermission(*topicName, role); err != nil {
				return diag.FromErr(fmt.Errorf("ERROR_UPDATE_TOPIC_PERMISSION_GRANT: %w", err))
			}

			if err = client.Topics().GrantPermission(*topicName, role, actions); err != nil {
				return diag.FromErr(fmt.Errorf("ERROR_UPDATE_TOPIC_PERMISSION_GRANT: %w", err))
			}
		}
	}

	return resourcePulsarPermissionGrantRead(ctx, d, meta)
}

func resourcePulsarPermissionGrantDelete(ctx context.Context, d *schema.ResourceData,
	meta interface{}) diag.Diagnostics {
	client := getClientFromMeta(meta)
	role := d.Get("role").(string)

	if namespace := d.Get("namespace").(string); namespace != "" {
		nsName, err := utils.GetNamespaceName(namespace)
		if err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_PARSE_NAMESPACE_NAME: %w", err))
		}

		if err = client.Namespaces().RevokeNamespacePermission(*nsName, role); err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_DELETE_NAMESPACE_PERMISSION_GRANT: %w", err))
		}

	} else if topic := d.Get("topic").(string); topic != "" {
		topicName, err := utils.GetTopicName(topic)
		if err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_PARSE_TOPIC_NAME: %w", err))
		}

		if err = client.Topics().RevokePermission(*topicName, role); err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_DELETE_TOPIC_PERMISSION_GRANT: %w", err))
		}
	}

	return nil
}
