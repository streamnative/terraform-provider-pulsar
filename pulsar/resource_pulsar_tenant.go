package pulsar

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/streamnative/pulsarctl/pkg/pulsar"
	"github.com/streamnative/pulsarctl/pkg/pulsar/utils"
	"log"
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
	client := meta.(pulsar.Client)

	tenant := d.Get("tenant").(string)
	adminRoles := handleHCLArray(d, "admin_roles")
	allowedClusters := handleHCLArray(d, "allowed_clusters")

	input := utils.TenantData{
		Name:            tenant,
		AllowedClusters: allowedClusters,
		AdminRoles:      adminRoles,
	}

	if err := client.Tenants().Create(input); err != nil {
		return fmt.Errorf("ERROR_CREATE_NEW_TENANT: \n%w", err)
	}

	return resourcePulsarTenantRead(d, meta)
}

func resourcePulsarTenantRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(pulsar.Client)

	tenant := d.Get("tenant").(string)

	td, err := client.Tenants().Get(tenant)
	if err != nil {
		return fmt.Errorf("ERROR_READ_TENANT: \n%w", err)
	}

	_ = d.Set("tenant", tenant)
	_ = d.Set("admin_roles", td.AdminRoles)
	_ = d.Set("allowed_clusters", td.AllowedClusters)
	d.SetId(tenant)

	return nil
}

func resourcePulsarTenantUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(pulsar.Client)
	log.Println("[INFO] deleted tenant")

	d.Partial(true)
	tenant := d.Get("tenant").(string)
	adminRoles := handleHCLArray(d, "admin_roles")
	allowedClusters := handleHCLArray(d, "allowed_clusters")

	input := utils.TenantData{
		Name:            tenant,
		AllowedClusters: allowedClusters,
		AdminRoles:      adminRoles,
	}

	if err := client.Tenants().Update(input); err != nil {
		return fmt.Errorf("ERROR_UPDATE_TENANT:  \n%w", err)
	}

	d.SetId(tenant)

	return nil
}

func resourcePulsarTenantDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(pulsar.Client)

	tenant := d.Get("tenant").(string)

	err := client.Tenants().Delete(tenant)
	if err != nil {
		return fmt.Errorf("ERROR_DELETE_TENANT: \n%w", err)
	}

	_ = d.Set("tenant", "")

	return nil
}

func handleHCLArray(d *schema.ResourceData, key string) []string {
	hclArray := d.Get(key).([]interface{})
	out := make([]string, len(hclArray))

	if len(hclArray) > 0 {

		for _, value := range hclArray {
			out = append(out, value.(string))
		}
	}

	return out
}
