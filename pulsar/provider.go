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
	"net/url"
	"strconv"

	"github.com/pkg/errors"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/streamnative/pulsarctl/pkg/pulsar"
	"github.com/streamnative/pulsarctl/pkg/pulsar/common"
)

const DefaultPulsarAPIVersion string = "0" // 0 will automatically match the default api version

// Provider returns a schema.Provider
func Provider() *schema.Provider {
	provider := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"web_service_url": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions["web_service_url"],
				DefaultFunc: schema.MultiEnvDefaultFunc(
					[]string{"PUSLAR_WEB_SERVICE_URL", "WEB_SERVICE_URL"}, ""),
			},
			"token": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions["token"],
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"PULSAR_TOKEN", "PULSAR_AUTH_TOKEN"}, ""),
			},
			"api_version": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions["api_version"],
				DefaultFunc: schema.EnvDefaultFunc("PULSAR_API_VERSION", DefaultPulsarAPIVersion),
			},
			"tls_trust_certs_file_path": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions["tls_trust_certs_file_path"],
				DefaultFunc: schema.MultiEnvDefaultFunc(
					[]string{"PULSAR_TLS_TRUST_CERTS_FILE_PATH", "TLS_TRUST_CERTS_FILE_PATH"}, ""),
			},
			"tls_allow_insecure_connection": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: descriptions["tls_allow_insecure_connection"],
				DefaultFunc: schema.MultiEnvDefaultFunc(
					[]string{"PULSAR_TLS_TRUST_CERTS_FILE_PATH", "TLS_ALLOW_INSECURE_CONNECTION"}, false),
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"pulsar_tenant":    resourcePulsarTenant(),
			"pulsar_cluster":   resourcePulsarCluster(),
			"pulsar_namespace": resourcePulsarNamespace(),
			"pulsar_topic":     resourcePulsarTopic(),
		},
	}

	provider.ConfigureContextFunc = func(_ context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		return providerConfigure(d, provider.TerraformVersion)
	}

	return provider
}

func providerConfigure(d *schema.ResourceData, tfVersion string) (interface{}, diag.Diagnostics) {
	// can be used for version locking or version specific feature sets
	_ = tfVersion
	clusterURL := d.Get("web_service_url").(string)
	token := d.Get("token").(string)
	pulsarAPIVersion := d.Get("api_version").(string)
	TLSTrustCertsFilePath := d.Get("tls_trust_certs_file_path").(string)
	TLSAllowInsecureConnection := d.Get("tls_allow_insecure_connection").(bool)

	if clusterURL == "" {
		clusterURL = "http://localhost:8080"
	}

	if _, err := url.Parse(clusterURL); err != nil {
		return nil, diag.FromErr(errors.Wrap(err, "invalid web_service_url"))
	}

	apiVersion, err := strconv.Atoi(pulsarAPIVersion)
	if err != nil {
		return nil, diag.FromErr(errors.Wrap(err, "invalid api_version"))
	}

	config := &common.Config{
		WebServiceURL:              clusterURL,
		Token:                      token,
		PulsarAPIVersion:           common.APIVersion(apiVersion),
		TLSTrustCertsFilePath:      TLSTrustCertsFilePath,
		TLSAllowInsecureConnection: TLSAllowInsecureConnection,
	}

	client, err := pulsar.New(config)
	if err != nil {
		return nil, diag.FromErr(errors.Wrapf(err, "failed to create pulsar client"))
	}

	return client, nil
}

var descriptions map[string]string

func init() {
	descriptions = map[string]string{
		"web_service_url": "Web service url is used to connect to your apache pulsar cluster",
		"token": `Authentication Token used to grant terraform permissions
to modify Apace Pulsar Entities`,
		"api_version":                   "Api Version to be used for the pulsar admin interaction",
		"tls_trust_certs_file_path":     "Path to a custom trusted TLS certificate file",
		"tls_allow_insecure_connection": "Boolean flag to accept untrusted TLS certificates",
		"admin_roles":                   "Admin roles to be attached to tenant",
		"allowed_clusters":              "Tenant will be able to interact with these clusters",
		"namespace":                     "Pulsar namespaces are logical groupings of topics",
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
