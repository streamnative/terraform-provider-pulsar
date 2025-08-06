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
	"context" // For handling request cancellation and timeouts
	"fmt"     // For string formatting and error messages

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"   // Pulsar utilities, includes AuthAction type
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"          // For returning errors to Terraform
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema" // Core Terraform SDK for resource definitions
)

// This function defines what our new resource looks like and what it can do
// It returns a *schema.Resource which tells Terraform everything about our resource
func resourcePulsarPermissionGrant() *schema.Resource {
	return &schema.Resource{
		// These 4 functions handle the complete lifecycle of our resource
		CreateContext: resourcePulsarPermissionGrantCreate, // Called when "terraform apply" creates this resource
		ReadContext:   resourcePulsarPermissionGrantRead,   // Called to check current state (refreshes, plans, etc)
		UpdateContext: resourcePulsarPermissionGrantUpdate, // Called when user changes the resource config
		DeleteContext: resourcePulsarPermissionGrantDelete, // Called when "terraform destroy" or resource removed

		// Schema defines what fields this resource has and their validation rules
		// This is like defining the structure of your Terraform config block
		Schema: map[string]*schema.Schema{
			// The "namespace" field - user must specify which namespace to grant permissions on
			"namespace": {
				Type:        schema.TypeString,                            // This field accepts a string value
				Required:    true,                                         // User MUST provide this field
				ForceNew:    true,                                         // If this changes, destroy and recreate the resource
				Description: "The namespace in format 'tenant/namespace'", // Help text for documentation
			},

			// The "role" field - user must specify which role gets the permissions
			"role": {
				Type:        schema.TypeString,                  // This field accepts a string value
				Required:    true,                               // User MUST provide this field
				ForceNew:    true,                               // If this changes, destroy and recreate the resource
				Description: "The role to grant permissions to", // Help text for documentation
			},

			// The "actions" field - user must specify what permissions to grant
			"actions": {
				Type:        schema.TypeSet,                        // This is a set (no duplicates, order doesn't matter)
				Required:    true,                                  // User MUST provide this field
				MinItems:    1,                                     // Must have at least 1 action
				Description: "Set of actions to grant to the role", // Help text for documentation

				// Elem defines what each item in the set looks like
				Elem: &schema.Schema{
					Type:         schema.TypeString,  // Each action is a string
					ValidateFunc: validateAuthAction, // This function validates each action is valid
				},
			},
		},
	}
}

// CREATE function - called when Terraform needs to create this resource
// This happens during "terraform apply" when the resource doesn't exist yet
func resourcePulsarPermissionGrantCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Get the Pulsar admin client from the provider configuration
	// meta contains the connection info (URL, tokens, etc) configured in the provider block
	// .Namespaces() gives us the client specifically for namespace operations
	client := getClientFromMeta(meta).Namespaces()

	// Extract values from the Terraform configuration
	// d.Get() retrieves the values the user specified in their .tf file
	namespace := d.Get("namespace").(string)     // e.g., "public/default"
	role := d.Get("role").(string)               // e.g., "my-role"
	actionsSet := d.Get("actions").(*schema.Set) // e.g., ["produce", "consume"]

	// Convert the Terraform set of actions to the format Pulsar API expects
	// Terraform gives us a *schema.Set, but Pulsar API wants []utils.AuthAction
	actions := make([]utils.AuthAction, 0, actionsSet.Len()) // Create slice with capacity
	for _, action := range actionsSet.List() {               // Iterate through each action
		// Convert each string action to utils.AuthAction type and add to slice
		actions = append(actions, utils.AuthAction(action.(string)))
	}

	// Convert the namespace string to Pulsar's namespace type
	// Pulsar API expects utils.NameSpaceName, not a plain string
	nsName, err := utils.GetNamespaceName(namespace)
	if err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_PARSE_NAMESPACE_NAME: %w", err))
	}

	// Make the actual API call to Pulsar to grant the permission
	// This is where the real work happens - we're telling Pulsar:
	// "Give role X these actions Y on namespace Z"
	if err := client.GrantNamespacePermission(*nsName, role, actions); err != nil {
		// If the API call fails, return an error to Terraform
		// diag.FromErr converts a Go error to Terraform's diagnostic format
		return diag.FromErr(fmt.Errorf("ERROR_GRANT_NAMESPACE_PERMISSION: %w", err))
	}

	// Set the resource ID so Terraform can track this resource
	// We use "namespace/role" as a unique identifier
	// Example: "public/default/my-role"
	d.SetId(fmt.Sprintf("%s/%s", namespace, role))

	// After creating, immediately read the resource to populate the state
	// This ensures Terraform's state matches what actually exists in Pulsar
	return resourcePulsarPermissionGrantRead(ctx, d, meta)
}

// READ function - called to check the current state of the resource
// This happens during "terraform plan", "terraform refresh", and after create/update
func resourcePulsarPermissionGrantRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Get the Pulsar admin client (same as in create)
	client := getClientFromMeta(meta).Namespaces()

	// Get the namespace and role from the resource's current state
	namespace := d.Get("namespace").(string)
	role := d.Get("role").(string)

	// Convert the namespace string to Pulsar's namespace type
	nsName, err := utils.GetNamespaceName(namespace)
	if err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_PARSE_NAMESPACE_NAME: %w", err))
	}

	// Ask Pulsar: "What are ALL the permissions on this namespace?"
	// This returns a map like: {"role1": ["produce", "consume"], "role2": ["produce"]}
	permissions, err := client.GetNamespacePermissions(*nsName)
	if err != nil {
		// If we can't read permissions, return an error
		return diag.FromErr(fmt.Errorf("ERROR_READ_NAMESPACE_PERMISSIONS: %w", err))
	}

	// Look for our specific role in the permissions map
	roleActions, exists := permissions[role]
	if !exists {
		// If our role doesn't have any permissions, the resource no longer exists
		// Clear the ID to tell Terraform "this resource is gone"
		d.SetId("")
		return nil // No error, just removed from state
	}

	// Convert Pulsar's action format back to Terraform's format
	// Pulsar gives us []utils.AuthAction, but Terraform wants []string
	actions := make([]string, len(roleActions))
	for i, action := range roleActions {
		actions[i] = string(action) // Convert each AuthAction to string
	}

	// Update Terraform's state with what we found in Pulsar
	// This keeps Terraform's state in sync with reality
	if err := d.Set("actions", actions); err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_SET_ACTIONS: %w", err))
	}

	// Return nil diagnostics = success, no errors
	return nil
}

// UPDATE function - called when user changes the resource configuration
// This happens during "terraform apply" when the resource exists but config changed
func resourcePulsarPermissionGrantUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Get the Pulsar admin client (same as in create/read)
	client := getClientFromMeta(meta).Namespaces()

	// Get the namespace and role (these can't change due to ForceNew: true)
	namespace := d.Get("namespace").(string)
	role := d.Get("role").(string)

	// Convert the namespace string to Pulsar's namespace type
	nsName, err := utils.GetNamespaceName(namespace)
	if err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_PARSE_NAMESPACE_NAME: %w", err))
	}

	// Check if the actions field changed
	// d.HasChange() returns true if this field is different from the previous apply
	if d.HasChange("actions") {
		// Strategy: revoke all existing permissions, then grant the new ones
		// This is simpler than trying to figure out what to add/remove

		// Step 1: Remove all existing permissions for this role
		if err := client.RevokeNamespacePermission(*nsName, role); err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_REVOKE_NAMESPACE_PERMISSION: %w", err))
		}

		// Step 2: Grant the new permissions
		// Convert the new actions from Terraform format to Pulsar format
		actionsSet := d.Get("actions").(*schema.Set)
		actions := make([]utils.AuthAction, 0, actionsSet.Len())
		for _, action := range actionsSet.List() {
			actions = append(actions, utils.AuthAction(action.(string)))
		}

		// Grant the new permissions
		if err := client.GrantNamespacePermission(*nsName, role, actions); err != nil {
			return diag.FromErr(fmt.Errorf("ERROR_GRANT_NAMESPACE_PERMISSION: %w", err))
		}
	}

	// After updating, read the resource to refresh the state
	return resourcePulsarPermissionGrantRead(ctx, d, meta)
}

// DELETE function - called when Terraform needs to destroy this resource
// This happens during "terraform destroy" or when the resource is removed from config
func resourcePulsarPermissionGrantDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Get the Pulsar admin client (same as other functions)
	client := getClientFromMeta(meta).Namespaces()

	// Get the namespace and role from the resource state
	namespace := d.Get("namespace").(string)
	role := d.Get("role").(string)

	// Convert the namespace string to Pulsar's namespace type
	nsName, err := utils.GetNamespaceName(namespace)
	if err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_PARSE_NAMESPACE_NAME: %w", err))
	}

	// Tell Pulsar: "Remove ALL permissions for this role on this namespace"
	// This completely removes the role's access to the namespace
	if err := client.RevokeNamespacePermission(*nsName, role); err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_REVOKE_NAMESPACE_PERMISSION: %w", err))
	}

	// Return nil = success, resource is deleted
	// Terraform will automatically remove this from state
	return nil
}
