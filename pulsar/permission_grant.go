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

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/streamnative/pulsar-admin-go/pkg/utils"

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

func setPermissionGrant(d *schema.ResourceData, grants map[string][]utils.AuthAction) {
	permissionGrants := []interface{}{}
	for role, roleActions := range grants {
		actions := []string{}
		for _, action := range roleActions {
			actions = append(actions, action.String())
		}
		permissionGrants = append(permissionGrants, map[string]interface{}{
			"role":    role,
			"actions": actions,
		})
	}

	_ = d.Set("permission_grant", schema.NewSet(permissionGrantToHash, permissionGrants))
}
