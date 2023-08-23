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

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/rest"
	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourcePulsarTenant() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourcePulsarTenantCreate,
		ReadContext:   resourcePulsarTenantRead,
		UpdateContext: resourcePulsarTenantUpdate,
		DeleteContext: resourcePulsarTenantDelete,
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				_ = d.Set("tenant", d.Id())
				err := resourcePulsarTenantRead(ctx, d, meta)
				if err.HasError() {
					return nil, fmt.Errorf("import %q: %s", d.Id(), err[0].Summary)
				}
				return []*schema.ResourceData{d}, nil
			},
		},
		Schema: map[string]*schema.Schema{
			"tenant": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions["tenant"],
			},
			"allowed_clusters": {
				Type:        schema.TypeSet,
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

func resourcePulsarTenantCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getClientFromMeta(meta).Tenants()

	tenant := d.Get("tenant").(string)
	adminRoles := handleHCLArray(d, "admin_roles")
	allowedClusters := handleHCLArrayV2(d.Get("allowed_clusters").(*schema.Set).List())

	input := utils.TenantData{
		Name:            tenant,
		AllowedClusters: allowedClusters,
		AdminRoles:      adminRoles,
	}

	if err := client.Create(input); err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_CREATE_TENANT: %w\n request_input: %#v", err, input))
	}

	return resourcePulsarTenantRead(ctx, d, meta)
}

func resourcePulsarTenantRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getClientFromMeta(meta).Tenants()

	tenant := d.Get("tenant").(string)

	td, err := client.Get(tenant)
	if err != nil {
		if cliErr, ok := err.(rest.Error); ok && cliErr.Code == 404 {
			return diag.Errorf("ERROR_TENANT_NOT_FOUND")
		}
		return diag.FromErr(fmt.Errorf("ERROR_READ_TENANT: %w", err))
	}

	_ = d.Set("tenant", tenant)
	_ = d.Set("admin_roles", td.AdminRoles)
	_ = d.Set("allowed_clusters", td.AllowedClusters)
	d.SetId(tenant)

	return nil
}

func resourcePulsarTenantUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getClientFromMeta(meta).Tenants()

	tenant := d.Get("tenant").(string)
	adminRoles := handleHCLArray(d, "admin_roles")
	allowedClusters := handleHCLArrayV2(d.Get("allowed_clusters").(*schema.Set).List())

	input := utils.TenantData{
		Name:            tenant,
		AllowedClusters: allowedClusters,
		AdminRoles:      adminRoles,
	}

	if err := client.Update(input); err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_UPDATE_TENANT: %w", err))
	}

	d.SetId(tenant)

	return nil
}

func resourcePulsarTenantDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := getClientFromMeta(meta).Tenants()

	tenant := d.Get("tenant").(string)

	if err := deleteExistingNamespacesForTenant(tenant, meta); err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_DELETING_EXISTING_NAMESPACES_FOR_TENANT: %w", err))
	}

	if err := client.Delete(tenant); err != nil {
		return diag.FromErr(fmt.Errorf("ERROR_DELETE_TENANT: %w", err))
	}

	_ = d.Set("tenant", "")

	return nil
}

func deleteExistingNamespacesForTenant(tenant string, meta interface{}) error {
	client := getClientFromMeta(meta).Namespaces()

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
	return handleHCLArrayV2(hclArray)
}

func handleHCLArrayV2(hclArray []interface{}) []string {
	out := make([]string, 0)

	if len(hclArray) == 0 {
		return out
	}

	for _, value := range hclArray {
		out = append(out, value.(string))
	}

	return out
}
