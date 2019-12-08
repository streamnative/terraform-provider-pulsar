package pulsar

import (
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
		return err
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

	d.Set("tenant", tenant)
	d.Set("admin_roles", td.AdminRoles)
	d.Set("allowed_clusters", td.AllowedClusters)
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

	err := client.Delete(tenant)
	if err != nil {
		log.Printf("[INFO] error deleting tenant: %s", err)
		return err
	}

	d.Set("tenant", "")

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
