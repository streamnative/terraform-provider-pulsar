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
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"github.com/stretchr/testify/require"
)

const pulsarProviderAddress = "registry.terraform.io/streamnative/pulsar"

var pulsarV011ExternalProvider = map[string]resource.ExternalProvider{
	"pulsar": {
		Source:            pulsarProviderAddress,
		VersionConstraint: "= 0.11.0",
	},
}

func TestAccPulsarNamespace_UpgradeFromV011RefreshFalseDoesNotWritePolicies(t *testing.T) {
	if os.Getenv(resource.EnvTfAcc) == "" {
		t.Skipf("set %s to run acceptance tests", resource.EnvTfAcc)
	}

	terraformPath := os.Getenv("TF_ACC_TERRAFORM_PATH")
	if terraformPath == "" {
		var err error
		terraformPath, err = exec.LookPath("terraform")
		require.NoError(t, err)
	}

	var requestCount atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestCount.Add(1)
		http.Error(w, "unexpected Pulsar request", http.StatusInternalServerError)
	}))
	defer server.Close()

	workingDir := t.TempDir()
	config := fmt.Sprintf(`
terraform {
  required_providers {
    pulsar = {
      source  = "streamnative/pulsar"
      version = "= 0.11.0"
    }
  }
}

provider "pulsar" {
  web_service_url = %q
}

resource "pulsar_namespace" "test" {
  tenant    = "tenant"
  namespace = "namespace"

  namespace_config {
    anti_affinity = "group-a"
  }
}
`, server.URL)
	require.NoError(t, os.WriteFile(filepath.Join(workingDir, "main.tf"), []byte(config), 0o600))

	cliConfigPath := filepath.Join(workingDir, "terraform.rc")
	require.NoError(t, os.WriteFile(cliConfigPath, []byte(`
disable_checkpoint = true
provider_installation {
  direct {}
}
`), 0o600))

	// Fixture captures state written by provider v0.11.0 before ownership metadata existed.
	legacyState, err := os.ReadFile(filepath.Join("testdata", "pulsar_namespace_v011_state.json"))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(workingDir, "terraform.tfstate"), legacyState, 0o600))

	terraformEnv := append(os.Environ(),
		"CHECKPOINT_DISABLE=1",
		"TF_CLI_CONFIG_FILE="+cliConfigPath,
		"TF_IN_AUTOMATION=1",
	)

	output, err := runNamespaceUpgradeTerraform(
		context.Background(), terraformPath, workingDir, terraformEnv,
		"init", "-input=false", "-no-color",
	)
	require.NoErrorf(t, err, "terraform init failed:\n%s", output)

	output, err = runNamespaceUpgradeTerraformWithCurrentProvider(
		t, terraformPath, workingDir, terraformEnv,
		"plan", "-refresh=false", "-detailed-exitcode", "-input=false", "-no-color",
	)
	require.NoErrorf(t, err, "v0.11 state produced a non-empty refresh=false plan:\n%s", output)

	output, err = runNamespaceUpgradeTerraformWithCurrentProvider(
		t, terraformPath, workingDir, terraformEnv,
		"apply", "-refresh=false", "-auto-approve", "-input=false", "-no-color",
	)
	require.NoErrorf(t, err, "v0.11 state refresh=false apply failed:\n%s", output)
	require.Zero(t, requestCount.Load(), "refresh=false state upgrade must not call Pulsar APIs")

	assertNamespaceStateUpgradedToV1(t, filepath.Join(workingDir, "terraform.tfstate"))
}

func TestAccPulsarNamespace_UpgradeFromV011WithManagedPolicyBlocks(t *testing.T) {
	cluster := acctest.RandString(10)
	tenant := acctest.RandString(10)
	namespace := acctest.RandString(10)
	config := testPulsarNamespacePolicyBlocks(testWebServiceURL, cluster, tenant, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testPulsarNamespaceDestroy,
		Steps: []resource.TestStep{
			{
				Config:            config,
				ExternalProviders: pulsarV011ExternalProvider,
			},
			{
				Config:             config,
				ProviderFactories:  testAccProviderFactories,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccPulsarNamespace_UpgradeFromV011WithOmittedPolicyBlocks(t *testing.T) {
	cluster := acctest.RandString(10)
	tenant := acctest.RandString(10)
	namespace := acctest.RandString(10)
	config := testPulsarNamespaceNoPolicyBlocks(testWebServiceURL, cluster, tenant, namespace)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testPulsarNamespaceDestroy,
		Steps: []resource.TestStep{
			{
				Config:            config,
				ExternalProviders: pulsarV011ExternalProvider,
			},
			{
				Config:             config,
				ProviderFactories:  testAccProviderFactories,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func runNamespaceUpgradeTerraformWithCurrentProvider(
	t *testing.T,
	terraformPath string,
	workingDir string,
	env []string,
	args ...string,
) (output string, err error) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	grpcProvider := schema.NewGRPCProviderServer(Provider())
	reattachConfig, closeCh, err := plugin.DebugServe(ctx, &plugin.ServeOpts{
		GRPCProviderFunc: func() tfprotov5.ProviderServer {
			return grpcProvider
		},
		NoLogOutputOverride: true,
		ProviderAddr:        pulsarProviderAddress,
		UseTFLogSink:        t,
	})
	if err != nil {
		cancel()
		return "", err
	}
	defer func() {
		cancel()
		_, _ = grpcProvider.StopProvider(context.Background(), nil)
		select {
		case <-closeCh:
		case <-time.After(5 * time.Second):
			if err == nil {
				err = fmt.Errorf("timed out stopping test provider")
			}
		}
	}()

	reattachJSON, err := json.Marshal(map[string]plugin.ReattachConfig{
		pulsarProviderAddress: reattachConfig,
	})
	if err != nil {
		return "", err
	}

	providerEnv := append(append([]string{}, env...),
		"PLUGIN_PROTOCOL_VERSIONS=5",
		"TF_REATTACH_PROVIDERS="+string(reattachJSON),
	)
	return runNamespaceUpgradeTerraform(
		ctx, terraformPath, workingDir, providerEnv, args...,
	)
}

func runNamespaceUpgradeTerraform(
	ctx context.Context,
	terraformPath string,
	workingDir string,
	env []string,
	args ...string,
) (string, error) {
	command := exec.CommandContext(ctx, terraformPath, args...)
	command.Dir = workingDir
	command.Env = env
	output, err := command.CombinedOutput()
	return string(output), err
}

func assertNamespaceStateUpgradedToV1(t *testing.T, statePath string) {
	t.Helper()

	type namespaceInstance struct {
		SchemaVersion int                    `json:"schema_version"`
		Attributes    map[string]interface{} `json:"attributes"`
	}
	type terraformState struct {
		Resources []struct {
			Type      string              `json:"type"`
			Instances []namespaceInstance `json:"instances"`
		} `json:"resources"`
	}

	stateJSON, err := os.ReadFile(statePath)
	require.NoError(t, err)

	var state terraformState
	require.NoError(t, json.Unmarshal(stateJSON, &state))

	for _, stateResource := range state.Resources {
		if stateResource.Type != "pulsar_namespace" {
			continue
		}
		require.Len(t, stateResource.Instances, 1)
		instance := stateResource.Instances[0]
		require.Equal(t, 1, instance.SchemaVersion)

		managedTypes, ok := instance.Attributes[backlogQuotaManagedTypesStateAttr].([]interface{})
		require.True(t, ok, "upgraded ownership state has type %T", instance.Attributes[backlogQuotaManagedTypesStateAttr])
		require.Empty(t, managedTypes)
		return
	}

	t.Fatal("pulsar_namespace state not found")
}
