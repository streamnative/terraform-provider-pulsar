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
	"net/url"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/streamnative/pulsarctl/pkg/pulsar"
	"github.com/streamnative/pulsarctl/pkg/pulsar/common"
)

// Provider returns a terraform.ResourceProvider
func Provider() terraform.ResourceProvider {

	provider := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"web_service_url": {
				Type:        schema.TypeString,
				Required:    true,
				Description: descriptions["web_service_url"],
				DefaultFunc: schema.EnvDefaultFunc("WEB_SERVICE_URL", nil),
			},
			"token": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("PULSAR_AUTH_TOKEN", nil),
				Description: descriptions["token"],
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
			"pulsar_cluster":   resourcePulsarCluster(),
			"pulsar_namespace": resourcePulsarNamespace(),
			"pulsar_topic":     resourcePulsarTopic(),
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

	// can be used for version locking or version specific feature sets
	_ = tfVersion
	clusterURL := d.Get("web_service_url").(string)
	token := d.Get("token").(string)
	pulsarAPIVersion := d.Get("api_version").(string)

	apiVersion, err := strconv.Atoi(pulsarAPIVersion)
	if err != nil {
		apiVersion = 1
	}

	config := &common.Config{
		WebServiceURL:    clusterURL,
		Token:            token,
		PulsarAPIVersion: common.APIVersion(apiVersion),
	}

	return pulsar.New(config)
}

func validatePulsarConfig(d *schema.ResourceData) error {
	webServiceURL := d.Get("web_service_url").(string)

	if _, err := url.Parse(webServiceURL); err != nil {
		return fmt.Errorf("ERROR_PULSAR_CONFIG_INVALID_WEB_SERVICE_URL: %w", err)
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
		"web_service_url":  "Web service url is used to connect to your apache pulsar cluster",
		"token":            "Authentication Token used to grant terraform permissions to modify Apace Pulsar Entities",
		"api_version":      "Api Version to be used for the pulsar admin interaction",
		"admin_roles":      "Admin roles to be attached to tenant",
		"allowed_clusters": "Tenant will be able to interact with these clusters",
		"namespace":        "Pulsar namespaces are logical groupings of topics",
		"tenant": `An administrative unit for allocating capacity and enforcing an 
authentication/authorization scheme`,
		"namespace_list": "List of namespaces for a given tenant",
		"enable_duplication": `ensures that each message produced on Pulsar topics is persisted to disk 
only once, even if the message is produced more than once`,
		"encrypt_topics":                 "encrypt messages at the producer and decrypt at the consumer",
		"max_producers_per_topic":        "Max number of producers per topic",
		"max_consumers_per_subscription": "Max number of consumers per subscription",
		"max_consumers_per_topic":        "Max number of consumers per topic",
		"dispatch_rate":                  "Data transfer rate, in and out of the Pulsar Broker",
		"persistence_policy":             "Policy for the namespace for data persistence",
		"backlog_quota":                  "",
	}
}
