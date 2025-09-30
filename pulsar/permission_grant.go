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
	"bytes"
	"fmt"

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/streamnative/terraform-provider-pulsar/hashcode"
	"github.com/streamnative/terraform-provider-pulsar/types"
)

func permissionGrantToHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%s-", m["role"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["actions"].([]string)))

	return hashcode.String(buf.String())
}

func unmarshalPermissionGrants(v *schema.Set) ([]*types.PermissionGrant, error) {
	grants := v.List()
	permissionGrants := make([]*types.PermissionGrant, 0, len(grants))
	for _, grant := range grants {
		data := grant.(map[string]interface{})

		var permissionGrant types.PermissionGrant
		permissionGrant.Role = data["role"].(string)

		var actions []utils.AuthAction
		for _, action := range data["actions"].(*schema.Set).List() {
			authAction, err := utils.ParseAuthAction(action.(string))
			if err != nil {
				return nil, fmt.Errorf("ERROR_INVALID_AUTH_ACTION: %w", err)
			}
			actions = append(actions, authAction)
		}
		permissionGrant.Actions = actions

		permissionGrants = append(permissionGrants, &permissionGrant)
	}

	return permissionGrants, nil
}

// setPermissionGrantFiltered only sets permissions for roles that are explicitly defined in the Terraform configuration
func setPermissionGrantFiltered(d *schema.ResourceData, grants map[string][]utils.AuthAction) {
	// Get the current permission_grant configuration to see which roles are managed by Terraform
	currentConfig := d.Get("permission_grant").(*schema.Set)
	managedRoles := make(map[string]bool)

	for _, grant := range currentConfig.List() {
		grantMap := grant.(map[string]interface{})
		role := grantMap["role"].(string)
		managedRoles[role] = true
	}

	// Only include permissions for roles that are explicitly managed by this Terraform resource
	permissionGrants := []interface{}{}
	for role, roleActions := range grants {
		if managedRoles[role] {
			actions := []string{}
			for _, action := range roleActions {
				actions = append(actions, action.String())
			}
			permissionGrants = append(permissionGrants, map[string]interface{}{
				"role":    role,
				"actions": actions,
			})
		}
	}

	_ = d.Set("permission_grant", schema.NewSet(permissionGrantToHash, permissionGrants))
}
