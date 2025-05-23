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
	"net/url"
	"os"
	"strconv"

	pulsaradmin "github.com/apache/pulsar-client-go/pulsaradmin/pkg/admin"
	adminconfig "github.com/apache/pulsar-client-go/pulsaradmin/pkg/admin/config"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"

	"github.com/streamnative/terraform-provider-pulsar/pkg/admin"
)

const DefaultPulsarAPIVersion string = "0" // 0 will automatically match the default api version

var descriptions map[string]string

func init() {
	//nolint:lll
	descriptions = map[string]string{
		"web_service_url":                "Web service url is used to connect to your apache pulsar cluster",
		"token":                          "Authentication Token used to grant terraform permissions to modify Apace Pulsar Entities",
		"api_version":                    "Api Version to be used for the pulsar admin interaction",
		"tls_trust_certs_file_path":      "Path to a custom trusted TLS certificate file",
		"tls_key_file_path":              "Path to the key to use when using TLS client authentication",
		"tls_cert_file_path":             "Path to the cert to use when using TLS client authentication",
		"tls_allow_insecure_connection":  "Boolean flag to accept untrusted TLS certificates",
		"admin_roles":                    "Admin roles to be attached to tenant",
		"allowed_clusters":               "Tenant will be able to interact with these clusters",
		"namespace":                      "Pulsar namespaces are logical groupings of topics",
		"tenant":                         "An administrative unit for allocating capacity and enforcing an authentication/authorization scheme",
		"namespace_list":                 "List of namespaces for a given tenant",
		"enable_duplication":             "ensures that each message produced on Pulsar topics is persisted to disk only once, even if the message is produced more than once",
		"encrypt_topics":                 "encrypt messages at the producer and decrypt at the consumer",
		"max_producers_per_topic":        "Max number of producers per topic",
		"max_consumers_per_subscription": "Max number of consumers per subscription",
		"max_consumers_per_topic":        "Max number of consumers per topic",
		"message_ttl_seconds":            "Sets the message time to live",
		"dispatch_rate":                  "Data transfer rate for all the topics under the given namespace",
		"subscription_dispatch_rate":     "Data transfer rate for all the subscriptions under the given namespace",
		"persistence_policy":             "Policy for the namespace for data persistence",
		"backlog_quota":                  "",
		"issuer_url":                     "The OAuth 2.0 URL of the authentication provider which allows the Pulsar client to obtain an access token",
		"audience":                       "The OAuth 2.0 resource server identifier for the Pulsar cluster",
		"client_id":                      "The OAuth 2.0 client identifier",
		"scope":                          "The OAuth 2.0 scope(s) to request",
		"key_file_path":                  "The path of the private key file",
		"message_ttl":                    "The message time to live in seconds",
		"namespace_config":               "The namespace configuration",
		"topic_config":                   "The topic configuration",
	}
}

// PulsarClientBundle is a struct that holds the pulsar admin client for both v2 and v3 api versions
type PulsarClientBundle struct {
	Client   pulsaradmin.Client
	V3Client pulsaradmin.Client
}

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
			"tls_cert_file_path": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions["tls_cert_file_path"],
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"PULSAR_TLS_CERT_FILE_PATH"}, ""),
			},
			"tls_key_file_path": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions["tls_key_file_path"],
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"PULSAR_TLS_KEY_FILE_PATH"}, ""),
			},
			"tls_allow_insecure_connection": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: descriptions["tls_allow_insecure_connection"],
				DefaultFunc: schema.MultiEnvDefaultFunc(
					[]string{"PULSAR_TLS_ALLOW_INSECURE_CONNECTION", "TLS_ALLOW_INSECURE_CONNECTION"}, false),
			},
			"issuer_url": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions["issuer_url"],
			},
			"audience": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions["audience"],
			},
			"client_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions["client_id"],
			},
			"scope": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions["scope"],
			},
			"key_file_path": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions["key_file_path"],
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"PULSAR_KEY_FILE", "PULSAR_KEY_FILE_PATH"}, ""),
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"pulsar_cluster":      resourcePulsarCluster(),
			"pulsar_tenant":       resourcePulsarTenant(),
			"pulsar_namespace":    resourcePulsarNamespace(),
			"pulsar_topic":        resourcePulsarTopic(),
			"pulsar_source":       resourcePulsarSource(),
			"pulsar_sink":         resourcePulsarSink(),
			"pulsar_function":     resourcePulsarFunction(),
			"pulsar_subscription": resourcePulsarSubscription(),
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
	issuerEndpoint := d.Get("issuer_url").(string)
	clientID := d.Get("client_id").(string)
	audience := d.Get("audience").(string)
	scope := d.Get("scope").(string)
	keyFilePath := d.Get("key_file_path").(string)
	TLSCertFilePath := d.Get("tls_cert_file_path").(string)
	TLSKeyFilePath := d.Get("tls_key_file_path").(string)

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

	if TLSCertFilePath != "" && !FileExists(TLSCertFilePath) {
		return nil, diag.FromErr(fmt.Errorf("ERROR_PULSAR_CONFIG_CERT_FILE_NOTEXIST: %q", TLSCertFilePath))
	}

	if TLSKeyFilePath != "" && !FileExists(TLSKeyFilePath) {
		return nil, diag.FromErr(fmt.Errorf("ERROR_PULSAR_CONFIG_KEY_FILE_NOTEXIST: %q", TLSKeyFilePath))
	}

	if TLSTrustCertsFilePath != "" && !FileExists(TLSTrustCertsFilePath) {
		return nil, diag.FromErr(fmt.Errorf("ERROR_PULSAR_CONFIG_tls_TRUST_FILE_NOTEXIST: %q", TLSTrustCertsFilePath))
	}

	configVersion := adminconfig.APIVersion(apiVersion)
	// for backward compatibility, if user state api_version as 3
	// we will use v2 as the default client version because we have v3 as individual client
	if configVersion == adminconfig.V3 {
		configVersion = adminconfig.APIVersion(0) // v2 will be the default client version
	}

	config := &adminconfig.Config{
		WebServiceURL:              clusterURL,
		Token:                      token,
		PulsarAPIVersion:           configVersion,
		TLSTrustCertsFilePath:      TLSTrustCertsFilePath,
		TLSAllowInsecureConnection: TLSAllowInsecureConnection,
		IssuerEndpoint:             issuerEndpoint,
		ClientID:                   clientID,
		Audience:                   audience,
		Scope:                      scope,
		KeyFile:                    keyFilePath,
		TLSKeyFile:                 TLSKeyFilePath,
		TLSCertFile:                TLSCertFilePath,
	}

	client, err := admin.NewPulsarAdminClient(&admin.PulsarAdminConfig{
		Config: config,
	})
	if err != nil {
		return nil, diag.FromErr(err)
	}

	configV3 := &adminconfig.Config{
		WebServiceURL:              clusterURL,
		Token:                      token,
		PulsarAPIVersion:           adminconfig.V3,
		TLSTrustCertsFilePath:      TLSTrustCertsFilePath,
		TLSAllowInsecureConnection: TLSAllowInsecureConnection,
		IssuerEndpoint:             issuerEndpoint,
		ClientID:                   clientID,
		Audience:                   audience,
		Scope:                      scope,
		KeyFile:                    keyFilePath,
		TLSKeyFile:                 TLSKeyFilePath,
		TLSCertFile:                TLSCertFilePath,
	}

	clientV3, err := admin.NewPulsarAdminClient(&admin.PulsarAdminConfig{
		Config: configV3,
	})
	if err != nil {
		return nil, diag.FromErr(err)
	}

	clientBundle := PulsarClientBundle{
		Client:   client,
		V3Client: clientV3,
	}

	return clientBundle, nil
}

// Exists reports whether the named file or directory exists.
func FileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
