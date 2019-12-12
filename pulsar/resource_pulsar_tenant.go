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
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/streamnative/pulsarctl/pkg/pulsar"
	"github.com/streamnative/pulsarctl/pkg/pulsar/utils"
)

func resourcePulsarTenant() *schema.Resource {
	return &schema.Resource{
		Create: resourcePulsarTenantCreate,
		Read:   resourcePulsarTenantRead,
		Update: resourcePulsarTenantUpdate,
		Delete: resourcePulsarTenantDelete,

		Schema: map[string]*schema.Schema{
			"tenant": {
				Type:     schema.TypeString,
				Required: true,
			},
			"allowed_clusters": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: descriptions["allowed_clusters"],
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"admin_roles": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: descriptions["admin_roles"],
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourcePulsarTenantCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(pulsar.Client).Tenants()

	tenant := d.Get("tenant").(string)
	adminRoles := handleHCLArray(d, "admin_roles")
	allowedClusters := handleHCLArray(d, "allowed_clusters")

	input := utils.TenantData{
		Name:            tenant,
		AllowedClusters: allowedClusters,
		AdminRoles:      adminRoles,
	}

	if err := client.Create(input); err != nil {
		return fmt.Errorf("ERROR_CREATE_TENANT: %w\n request_input: %s", err, input)
	}

	return resourcePulsarTenantRead(d, meta)
}

func resourcePulsarTenantRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(pulsar.Client).Tenants()

	tenant := d.Get("tenant").(string)

	td, err := client.Get(tenant)
	if err != nil {
		return err
	}

	_ = d.Set("tenant", tenant)
	_ = d.Set("admin_roles", td.AdminRoles)
	_ = d.Set("allowed_clusters", td.AllowedClusters)
	d.SetId(tenant)

	return nil
}

func resourcePulsarTenantUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(pulsar.Client).Tenants()

	d.Partial(true)
	tenant := d.Get("tenant").(string)
	adminRoles := handleHCLArray(d, "admin_roles")
	allowedClusters := handleHCLArray(d, "allowed_clusters")

	input := utils.TenantData{
		Name:            tenant,
		AllowedClusters: allowedClusters,
		AdminRoles:      adminRoles,
	}

	if err := client.Update(input); err != nil {
		return err
	}

	d.SetId(tenant)

	return nil
}

func resourcePulsarTenantDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(pulsar.Client).Tenants()

	tenant := d.Get("tenant").(string)

	if err := deleteExistingNamespacesForTenant(tenant, meta); err != nil {
		return fmt.Errorf("ERROR_DELETING_EXISTING_NAMESPACES_FOR_TENANT: %w", err)
	}

	if err := client.Delete(tenant); err != nil {
		return err
	}

	_ = d.Set("tenant", "")

	return nil
}

func deleteExistingNamespacesForTenant(tenant string, meta interface{}) error {
	client := meta.(pulsar.Client).Namespaces()

	nsList, err := client.GetNamespaces(tenant)
	if err != nil {
		return err
	}

	if len(nsList) > 0 {
		for _, ns := range nsList {
			if strings.Contains(ns, tenant) {
				if err = client.DeleteNamespace(ns); err != nil {
					return err
				}

				return nil
			}

			fullNamespacePath := fmt.Sprintf("%s/%s", tenant, ns)
			if err = client.DeleteNamespace(fullNamespacePath); err != nil {
				return err
			}
		}
	}

	return nil
}

func handleHCLArray(d *schema.ResourceData, key string) []string {
	hclArray := d.Get(key).([]interface{})
	out := make([]string, 0)

	if len(hclArray) == 0 {
		return out
	}

	for _, value := range hclArray {
		out = append(out, value.(string))
	}

	return out
}
