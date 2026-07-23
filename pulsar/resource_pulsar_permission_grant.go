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
		CustomizeDiff: resourcePulsarPermissionGrantCustomizeDiff,

		Description: `Provides a resource for managing permissions on either Pulsar namespaces or topics.
Permission can be granted to specific roles using this resource.

**Important:** You must specify either ` + "`namespace`" + ` OR ` + "`topic`" + `, but not both. While both fields
are marked as optional in the schema, exactly one must be provided for the resource to be valid.

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

// resourcePulsarPermissionGrantCustomizeDiff runs during the plan phase and fails
// the plan when the grant references a namespace that is known not to exist. Without
// this check the missing namespace is only discovered at apply time as a
// "404 Namespace not found" error, after the change has already been merged.
func resourcePulsarPermissionGrantCustomizeDiff(_ context.Context, diff *schema.ResourceDiff,
	meta interface{}) error {
	// meta is nil when the provider is not configured (e.g. `terraform validate`).
	// There is no client to consult, so skip the check rather than failing.
	if meta == nil {
		return nil
	}
	client := getClientFromMeta(meta)

	namespace, _ := diff.GetOk("namespace")
	topic, _ := diff.GetOk("topic")

	return verifyGrantNamespaceExists(client, namespace.(string), topic.(string))
}

// verifyGrantNamespaceExists returns a diagnostic-style error when the namespace
// targeted by the grant (either directly, or the namespace owning the topic) is
// known not to exist.
//
// It intentionally errs on the side of NOT failing the plan: if the namespace
// cannot be determined (unparseable input, or the tenant listing call fails —
// which also covers the tenant itself not existing yet, and transient/network
// errors), the check is skipped. It only returns an error when the tenant's
// namespace listing succeeds AND the target namespace is provably absent. This
// avoids false positives during a fresh apply where the tenant/namespace are
// being created in the same run.
func verifyGrantNamespaceExists(client admin.Client, namespace, topic string) error {
	var nsName *utils.NameSpaceName

	switch {
	case namespace != "":
		parsed, err := utils.GetNamespaceName(namespace)
		if err != nil {
			// Malformed input is reported by the CRUD path; don't block the plan here.
			return nil
		}
		nsName = parsed
	case topic != "":
		topicName, err := utils.GetTopicName(topic)
		if err != nil {
			return nil
		}
		parsed, err := utils.GetNameSpaceName(topicName.GetTenant(), topicName.GetNamespace())
		if err != nil {
			return nil
		}
		nsName = parsed
	default:
		// Neither set: the schema's ExactlyOneOf validation handles this.
		return nil
	}

	// NameSpaceName only exposes String() ("tenant/namespace"); the tenant is the
	// first segment and is required to list the tenant's namespaces.
	tenant := strings.SplitN(nsName.String(), "/", 2)[0]
	namespaces, err := client.Namespaces().GetNamespaces(tenant)
	if err != nil {
		// Cannot determine existence (tenant missing, network error, etc.). Skip.
		return nil
	}

	if !contains(namespaces, nsName.String()) {
		return fmt.Errorf(
			"namespace %q does not exist; create it before granting permissions "+
				"(referenced by pulsar_permission_grant)", nsName.String())
	}

	return nil
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

		grants, err = getTopicSpecificPermissions(client, topicName)
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
