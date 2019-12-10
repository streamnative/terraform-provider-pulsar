package pulsar

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/streamnative/pulsarctl/pkg/pulsar"
	"github.com/streamnative/pulsarctl/pkg/pulsar/common"
	"net/url"
	"strconv"
)

func Provider() terraform.ResourceProvider {

	provider := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"web_service_url": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions["web_service_url"],
				DefaultFunc: schema.EnvDefaultFunc("WEB_SERVICE_URL", nil),
			},
			"pulsar_auth_token": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				DefaultFunc: schema.EnvDefaultFunc("PULSAR_AUTH_TOKEN", nil),
				Description: descriptions["pulsar_auth_token"],
			},
			"api_version": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "1",
				Description: descriptions["api_version"],
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"pulsar_tenant":    resourcePulsarTenant(),
			"pulsar_namespace": resourcePulsarNamespace(),
			"pulsar_cluster":   resourcePulsarCluster(),
		},
	}

	provider.ConfigureFunc = func(d *schema.ResourceData) (interface{}, error) {
		tfVersion := provider.TerraformVersion
		if tfVersion == "" {
			// Terraform 0.12 introduced this field to the protocol, so if this field is missing,
			// we can assume Terraform version is <= 0.11
			tfVersion = "0.11+compatible"
		}

		if err := validatePulsarConfig(d); err != nil {
			return nil, err
		}

		return providerConfigure(d, tfVersion)
	}

	return provider
}

func providerConfigure(d *schema.ResourceData, tfVersion string) (interface{}, error) {
	clusterURL := d.Get("web_service_url").(string)
	token := d.Get("pulsar_auth_token").(string)
	pulsarApiVersion := d.Get("api_version").(string)

	apiVersion, err := strconv.Atoi(pulsarApiVersion)
	if err != nil {
		apiVersion = 1
	}

	config := &pulsar.Config{
		WebServiceURL: clusterURL,
		HTTPTimeout:   15,
		APIVersion:    common.APIVersion(apiVersion),
		Token:         token,
	}

	return pulsar.New(config)
}

func validatePulsarConfig(d *schema.ResourceData) error {
	webServiceURL, ok := d.Get("web_service_url").(string)
	if !ok {
		return fmt.Errorf("ERROR_PULSAR_CONFIG_INVALID")
	}

	if _, err := url.Parse(webServiceURL); err != nil {
		return fmt.Errorf("ERROR_PULSAR_CONFIG_INVALID_WEB_SERVICE_URL")
	}

	apiVersion, ok := d.Get("api_version").(string)
	if !ok {
		_ = d.Set("api_version", "1")
	}

	switch apiVersion {
	case "0":
		_ = d.Set("api_version", "0")
	case "1":
		// (@TODO pulsarctl) 1 is for v2, in pulsarctl, Version is set with iota, it should be iota+1
		_ = d.Set("api_version", "1")
	case "2":
		_ = d.Set("api_version", "2")
	default:
		_ = d.Set("api_version", "1")
	}

	return nil
}

var descriptions map[string]string

func init() {
	descriptions = map[string]string{
		"web_service_url":                "Web service url is used to connect to your apache pulsar cluster",
		"pulsar_auth_token":              "Authentication Token used to grant terraform permissions to modify Apace Pulsar Entities",
		"api_version":                    "Api Version to be used for the pulsar admin interaction",
		"admin_roles":                    "Admin roles to be attached to tenant",
		"allowed_clusters":               "Tenant will be able to interact with these clusters",
		"namespace":                      "",
		"tenant":                         "",
		"namespace_list":                 "",
		"enable_duplication":             "",
		"encrypt_topics":                 "",
		"max_producers_per_topic":        "",
		"max_consumers_per_subscription": "",
		"max_consumers_per_topic":        "",
		"dispatch_rate":                  "",
		"persistence_policy":             "",
		"backlog_quota":                  "",
	}
}
