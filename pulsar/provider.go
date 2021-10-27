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
	"io/ioutil"
	"net/url"
	"os"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/pkg/errors"
	"github.com/streamnative/pulsarctl/pkg/pulsar"
	"github.com/streamnative/pulsarctl/pkg/pulsar/common"

	vault "github.com/hashicorp/vault/api"
	vaultgo "github.com/mittwald/vaultgo"
)

const DefaultPulsarAPIVersion string = "0" // 0 will automatically match the default api version

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
				Default:     DefaultPulsarAPIVersion,
				Description: descriptions["api_version"],
			},
			"tls_trust_certs_file_path": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions["tls_trust_certs_file_path"],
				DefaultFunc: schema.EnvDefaultFunc("TLS_TRUST_CERTS_FILE_PATH", nil),
			},
			"tls_allow_insecure_connection": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: descriptions["tls_allow_insecure_connection"],
				DefaultFunc: schema.EnvDefaultFunc("TLS_ALLOW_INSECURE_CONNECTION", false),
			},
			"vault_address": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("VAULT_ADDR", nil),
				Description: "URL of the root of the target Vault server.",
			},
			"vault_token": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("VAULT_TOKEN", ""),
				Description: "Token to use to authenticate to Vault.",
			},
			"vault_role": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("VAULT_ROLE", ""),
				Description: "Role to use to get to a certificate from Vault.",
			},
			"vault_pki": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("VAULT_PKI", ""),
				Description: "PKI to use to get a certificate from Vault.",
			},
			"vault_skip_tls_verify": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("VAULT_SKIP_VERIFY", false),
				Description: "Set this to true only if the target Vault server is an insecure development instance.",
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
	TLSTrustCertsFilePath := d.Get("tls_trust_certs_file_path").(string)
	TLSAllowInsecureConnection := d.Get("tls_allow_insecure_connection").(bool)

	vaultAddr := d.Get("vault_address").(string)
	vaultToken := d.Get("vault_token").(string)
	vaultRole := d.Get("vault_role").(string)
	vaultPki := d.Get("vault_pki").(string)
	vaultSkipTLSVerify := d.Get("vault_skip_tls_verify").(bool)

	apiVersion, err := strconv.Atoi(pulsarAPIVersion)
	if err != nil {
		return nil, err
	}

	config := &common.Config{
		WebServiceURL:              clusterURL,
		Token:                      token,
		PulsarAPIVersion:           common.APIVersion(apiVersion),
		TLSTrustCertsFilePath:      TLSTrustCertsFilePath,
		TLSAllowInsecureConnection: TLSAllowInsecureConnection,
	}

	if vaultAddr != "" && vaultToken != "" && vaultRole != "" && vaultPki != "" {
		// Get certificate from Vault
		crtPath, keyPath, errVault := getCertificateFromVault(vaultAddr, vaultToken, vaultRole, vaultPki, vaultSkipTLSVerify)
		if errVault != nil {
			return nil, errVault
		}

		config.TLSCertFile = crtPath
		config.TLSKeyFile = keyPath
	}

	return pulsar.New(config)
}

func getCertificateFromVault(addr, token, role, pki string, skipTLSVerify bool) (keyPath, certPath string, err error) {

	c, errVaultClient := vaultgo.NewClient(
		addr,
		&vaultgo.TLSConfig{
			TLSConfig: &vault.TLSConfig{
				Insecure: skipTLSVerify,
			},
		},
	)
	if errVaultClient != nil {
		err = errors.Wrap(errVaultClient, "failed to init vault client")
		return
	}

	if token != "" {
		c.SetToken(token)
	}

	body := map[string]string{
		"common_name": "terraform-provider-pulsar",
		"ttl":         "15m",
	}
	type vaultPkiIssueResponse struct {
		Data struct {
			Certificate []byte   `json:"certificate"`
			IssuingCA   []byte   `json:"issuing_ca"`
			CAChain     [][]byte `json:"ca_chain"`
			PrivateKey  []byte   `json:"private_key"`
		} `json:"data"`
	}
	var response vaultPkiIssueResponse

	errPost := c.Write([]string{fmt.Sprintf("v1/%s/issue/%s", pki, role)}, &body, &response, nil)
	if errPost != nil {
		err = errors.Wrap(errPost, "unable to request certificate from Vault")
		return
	}

	crtFile, errTmpCrt := ioutil.TempFile("terraform-provider-pulsar", "pki_*.crt")
	if errTmpCrt != nil {
		err = errors.Wrap(errTmpCrt, "unable to create temporary .crt file")
		return
	}
	certPath = crtFile.Name()

	errWriteCrt := os.WriteFile(certPath, response.Data.Certificate, 0644)
	if errWriteCrt != nil {
		err = errors.Wrap(errWriteCrt, "unable to output certificate to .crt file")
		return
	}

	keyFile, errTmpKey := ioutil.TempFile("terraform-provider-pulsar", "pki_*.key")
	if errTmpKey != nil {
		err = errors.Wrap(errTmpKey, "unable to create temporary .key file")
		return
	}
	keyPath = keyFile.Name()

	errWriteKey := os.WriteFile(keyPath, response.Data.PrivateKey, 0644)
	if errWriteKey != nil {
		err = errors.Wrap(errWriteKey, "unable to output key to .key file")
		return
	}

	return
}

func validatePulsarConfig(d *schema.ResourceData) error {
	webServiceURL := d.Get("web_service_url").(string)

	if _, err := url.Parse(webServiceURL); err != nil {
		return fmt.Errorf("ERROR_PULSAR_CONFIG_INVALID_WEB_SERVICE_URL: %w", err)
	}

	return nil
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
