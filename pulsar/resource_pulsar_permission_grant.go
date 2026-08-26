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
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/admin"
	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/rest"
	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
	backoff "github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// permissionLocks provides per-resource-key mutexes to prevent concurrent
// GrantPermission / RevokePermission calls within the same provider process.
// Pulsar's grant API is read-modify-write against ZooKeeper; parallel writes
// to the same namespace or topic produce HTTP 409 conflicts.
var permissionLocks sync.Map

func getPermissionLock(key string) *sync.Mutex {
	mu, _ := permissionLocks.LoadOrStore(key, &sync.Mutex{})
	return mu.(*sync.Mutex)
}

func isConflictError(err error) bool {
	var adminErr rest.Error
	return errors.As(err, &adminErr) && adminErr.Code == http.StatusConflict
}

// retryOnConflict retries operation on HTTP 409 with exponential backoff.
// All other errors are treated as permanent and returned immediately.
func retryOnConflict(ctx context.Context, operation func(context.Context) error) error {
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 2 * time.Minute
	return backoff.Retry(func() error {
		err := operation(ctx)
		if err == nil {
			return nil
		}
		if isConflictError(err) {
			return err
		}
		return backoff.Permanent(err)
	}, backoff.WithContext(bo, ctx))
}

func resourcePulsarPermissionGrant() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourcePulsarPermissionGrantCreate,
		ReadContext:   resourcePulsarPermissionGrantRead,
		UpdateContext: resourcePulsarPermissionGrantUpdate,
		DeleteContext: resourcePulsarPermissionGrantDelete,

		Description: "Manages role permissions on exactly one Pulsar namespace or topic. Do not manage the same " +
			"role through this resource and a nested `permission_grant` block on `pulsar_namespace` or `pulsar_topic`.",

		Importer: &schema.ResourceImporter{
			StateContext: resourcePulsarPermissionGrantImport,
		},

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
				Description: "One or more Pulsar authorization actions granted to the role.",
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validateAuthAction,
				},
			},
		},
	}
}

// parsePermissionGrantImportID splits a permission grant import ID into its
// namespace or topic part and role. The ID mirrors the resource ID assigned on
// create: {namespace}/{role} (tenant/namespace/role) or {topic}/{role}
// ({domain}://tenant/namespace/topic/role). Topic IDs are recognized by the
// required domain scheme; anything else must be a tenant/namespace pair,
// so an ID without a domain scheme and without a tenant/namespace slash is
// rejected here. The role is the segment after the final slash, so roles
// containing "/" are not importable.
func parsePermissionGrantImportID(id string) (namespace string, topic string, role string, err error) {
	if id == "" {
		return "", "", "", fmt.Errorf("empty import ID, expected {namespace}/{role} or {topic}/{role}")
	}
	idx := strings.LastIndex(id, "/")
	if idx < 0 || idx == len(id)-1 {
		return "", "", "", fmt.Errorf("invalid import ID %q, expected {namespace}/{role} or {topic}/{role}", id)
	}
	role = id[idx+1:]
	target := id[:idx]
	switch {
	case strings.Contains(target, "://"):
		topic = target
	case strings.Contains(target, "/"):
		namespace = target
	default:
		return "", "", "", fmt.Errorf("invalid import ID %q, expected {namespace}/{role} or {topic}/{role}", id)
	}
	return namespace, topic, role, nil
}

func resourcePulsarPermissionGrantImport(ctx context.Context, d *schema.ResourceData,
	meta interface{}) ([]*schema.ResourceData, error) {
	importID := d.Id()

	namespace, topic, role, err := parsePermissionGrantImportID(importID)
	if err != nil {
		return nil, fmt.Errorf("ERROR_PARSE_PERMISSION_GRANT_IMPORT_ID: %w", err)
	}

	// Set the fields Read consumes and normalize the ID to the canonical
	// {namespace}/{role} or {topic}/{role} form assigned on create.
	canonicalID := ""
	if topic != "" {
		topicName, parseErr := utils.GetTopicName(topic)
		if parseErr != nil {
			return nil, fmt.Errorf("ERROR_PARSE_TOPIC_NAME: %w", parseErr)
		}
		_ = d.Set("topic", topicName.String())
		canonicalID = fmt.Sprintf("%s/%s", topicName.String(), role)
	} else {
		nsName, parseErr := utils.GetNamespaceName(namespace)
		if parseErr != nil {
			return nil, fmt.Errorf("ERROR_PARSE_NAMESPACE_NAME: %w", parseErr)
		}
		_ = d.Set("namespace", nsName.String())
		canonicalID = fmt.Sprintf("%s/%s", nsName.String(), role)
	}
	_ = d.Set("role", role)
	d.SetId(canonicalID)

	client := getClientFromMeta(meta)

	grants, err := fetchPermissionGrants(client, namespace, topic)
	if err != nil {
		if isIgnorableNotFoundError(err) {
			return nil, fmt.Errorf("ERROR_PERMISSION_GRANT_NOT_FOUND: %s", importID)
		}
		return nil, fmt.Errorf("import %q: %w", importID, err)
	}
	if !applyPermissionGrantsToState(d, grants, role) {
		return nil, fmt.Errorf("ERROR_PERMISSION_GRANT_NOT_FOUND: %s", importID)
	}
	return []*schema.ResourceData{d}, nil
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

		mu := getPermissionLock(nsName.String())
		mu.Lock()
		defer mu.Unlock()

		if err = retryOnConflict(ctx, func(callCtx context.Context) error {
			return client.Namespaces().GrantNamespacePermissionWithContext(callCtx, *nsName, role, actions)
		}); err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_GRANT_NAMESPACE_PERMISSION: %w", err))
		}

		d.SetId(fmt.Sprintf("%s/%s", namespace, role))

	} else if topic := d.Get("topic").(string); topic != "" {
		topicName, err := utils.GetTopicName(topic)
		if err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_PARSE_TOPIC_NAME: %w", err))
		}

		mu := getPermissionLock(topicName.String())
		mu.Lock()
		defer mu.Unlock()

		if err = retryOnConflict(ctx, func(callCtx context.Context) error {
			return client.Topics().GrantPermissionWithContext(callCtx, *topicName, role, actions)
		}); err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_GRANT_TOPIC_PERMISSION: %w", err))
		}

		d.SetId(fmt.Sprintf("%s/%s", topic, role))
	}

	return resourcePulsarPermissionGrantRead(ctx, d, meta)
}

// getTopicSpecificPermissions returns only topic-level permissions (excluding
// inherited namespace permissions) by reading the namespace policies directly.
// client.Topics().GetPermissions() merges namespace and topic permissions,
// which causes Terraform drift when a role has permissions at both levels.
func getTopicSpecificPermissions(client admin.Client, topicName *utils.TopicName) (
	map[string][]utils.AuthAction, error) {
	ns := fmt.Sprintf("%s/%s", topicName.GetTenant(), topicName.GetNamespace())
	policies, err := client.Namespaces().GetPolicies(ns)
	if err != nil {
		return nil, err
	}
	if policies == nil {
		return map[string][]utils.AuthAction{}, nil
	}
	if perms, ok := policies.AuthPolicies.DestinationAuth[topicName.String()]; ok {
		return perms, nil
	}
	return map[string][]utils.AuthAction{}, nil
}

// fetchPermissionGrants reads the raw role-to-actions grants map for a
// namespace or topic. Errors are wrapped with the same prefixes the Read
// path has always reported.
func fetchPermissionGrants(client admin.Client, namespace string, topic string) (
	map[string][]utils.AuthAction, error) {
	if namespace != "" {
		nsName, err := utils.GetNamespaceName(namespace)
		if err != nil {
			return nil, fmt.Errorf("ERROR_PARSE_NAMESPACE_NAME: %w", err)
		}
		grants, err := client.Namespaces().GetNamespacePermissions(*nsName)
		if err != nil {
			return nil, fmt.Errorf("ERROR_READ_NAMESPACE_PERMISSION_GRANT: %w", err)
		}
		return grants, nil
	}

	topicName, err := utils.GetTopicName(topic)
	if err != nil {
		return nil, fmt.Errorf("ERROR_PARSE_TOPIC_NAME: %w", err)
	}
	grants, err := getTopicSpecificPermissions(client, topicName)
	if err != nil {
		return nil, fmt.Errorf("ERROR_READ_TOPIC_PERMISSION_GRANT: %w", err)
	}
	return grants, nil
}

// applyPermissionGrantsToState stores the role's actions in state. It reports
// whether the grant exists; a missing or empty grant clears the resource ID so
// Terraform treats the resource as gone.
func applyPermissionGrantsToState(d *schema.ResourceData, grants map[string][]utils.AuthAction, role string) bool {
	if actions, exists := grants[role]; exists && len(actions) > 0 {
		actionsSet := schema.NewSet(schema.HashString, []interface{}{})
		for _, action := range actions {
			actionsSet.Add(action.String())
		}
		_ = d.Set("actions", actionsSet)
		return true
	}
	d.SetId("")
	return false
}

func resourcePulsarPermissionGrantRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getClientFromMeta(meta)
	role := d.Get("role").(string)

	grants, err := fetchPermissionGrants(client, d.Get("namespace").(string), d.Get("topic").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	applyPermissionGrantsToState(d, grants, role)

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

			mu := getPermissionLock(nsName.String())
			mu.Lock()
			defer mu.Unlock()

			// Revoke and re-grant under the same lock to keep them atomic
			if err = retryOnConflict(ctx, func(callCtx context.Context) error {
				return client.Namespaces().RevokeNamespacePermissionWithContext(callCtx, *nsName, role)
			}); err != nil {
				return diag.FromErr(fmt.Errorf("ERROR_UPDATE_NAMESPACE_PERMISSION_GRANT: %w", err))
			}

			if err = retryOnConflict(ctx, func(callCtx context.Context) error {
				return client.Namespaces().GrantNamespacePermissionWithContext(callCtx, *nsName, role, actions)
			}); err != nil {
				return diag.FromErr(fmt.Errorf("ERROR_UPDATE_NAMESPACE_PERMISSION_GRANT: %w", err))
			}

		} else if topic := d.Get("topic").(string); topic != "" {
			topicName, err := utils.GetTopicName(topic)
			if err != nil {
				return diag.FromErr(fmt.Errorf("ERROR_PARSE_TOPIC_NAME: %w", err))
			}

			mu := getPermissionLock(topicName.String())
			mu.Lock()
			defer mu.Unlock()

			// Revoke and re-grant under the same lock to keep them atomic
			if err = retryOnConflict(ctx, func(callCtx context.Context) error {
				return client.Topics().RevokePermissionWithContext(callCtx, *topicName, role)
			}); err != nil {
				return diag.FromErr(fmt.Errorf("ERROR_UPDATE_TOPIC_PERMISSION_GRANT: %w", err))
			}

			if err = retryOnConflict(ctx, func(callCtx context.Context) error {
				return client.Topics().GrantPermissionWithContext(callCtx, *topicName, role, actions)
			}); err != nil {
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

		mu := getPermissionLock(nsName.String())
		mu.Lock()
		defer mu.Unlock()

		if err = retryOnConflict(ctx, func(callCtx context.Context) error {
			return client.Namespaces().RevokeNamespacePermissionWithContext(callCtx, *nsName, role)
		}); err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_DELETE_NAMESPACE_PERMISSION_GRANT: %w", err))
		}

	} else if topic := d.Get("topic").(string); topic != "" {
		topicName, err := utils.GetTopicName(topic)
		if err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_PARSE_TOPIC_NAME: %w", err))
		}

		mu := getPermissionLock(topicName.String())
		mu.Lock()
		defer mu.Unlock()

		if err = retryOnConflict(ctx, func(callCtx context.Context) error {
			return client.Topics().RevokePermissionWithContext(callCtx, *topicName, role)
		}); err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_DELETE_TOPIC_PERMISSION_GRANT: %w", err))
		}
	}

	return nil
}
