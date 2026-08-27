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

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/stretchr/testify/require"
)

func TestAccPulsarFunction_UpgradeFromV013RefreshFalseMigratesSensitiveMetadata(t *testing.T) {
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
      version = "= 0.13.0"
    }
  }
}

provider "pulsar" {
  web_service_url = %q
}

resource "pulsar_function" "upgrade" {
  name                            = "producer-upgrade-function"
  tenant                          = "public"
  namespace                       = "default"
  jar                             = "function://public/default/api-examples@v1"
  classname                       = "org.apache.pulsar.functions.api.examples.WordCountFunction"
  inputs                          = ["public/default/producer-upgrade-input"]
  output                          = "public/default/producer-upgrade-output"
  parallelism                     = 1
  forward_source_message_property = true
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

	legacyState, err := os.ReadFile(filepath.Join("testdata", "pulsar_function_v013_state.json"))
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

	// v0.13 state predates user_config's Sensitive schema flag. Terraform records the missing
	// sensitive path as a state-only update; no Pulsar API operation must be planned or run.
	output, err = runNamespaceUpgradeTerraformWithCurrentProvider(
		t, terraformPath, workingDir, terraformEnv,
		"plan", "-refresh=false", "-detailed-exitcode", "-input=false", "-no-color",
	)
	require.Error(t, err, "v0.13 state must plan the sensitive metadata migration")
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	require.Equalf(t, 2, exitErr.ExitCode(), "unexpected plan result:\n%s", output)
	require.Zero(t, requestCount.Load(), "refresh=false plan must not call Pulsar APIs")

	output, err = runNamespaceUpgradeTerraformWithCurrentProvider(
		t, terraformPath, workingDir, terraformEnv,
		"apply", "-refresh=false", "-auto-approve", "-input=false", "-no-color",
	)
	require.NoErrorf(t, err, "v0.13 state metadata migration apply failed:\n%s", output)
	require.Zero(t, requestCount.Load(), "state-only apply must not call Pulsar APIs")

	output, err = runNamespaceUpgradeTerraformWithCurrentProvider(
		t, terraformPath, workingDir, terraformEnv,
		"plan", "-refresh=false", "-detailed-exitcode", "-input=false", "-no-color",
	)
	require.NoErrorf(t, err, "migrated v0.13 state produced a non-empty refresh=false plan:\n%s", output)
	require.Zero(t, requestCount.Load(), "post-migration plan must not call Pulsar APIs")

	assertFunctionStateRecordsSensitiveUserConfig(t, filepath.Join(workingDir, "terraform.tfstate"))
}

func assertFunctionStateRecordsSensitiveUserConfig(t *testing.T, statePath string) {
	t.Helper()

	type sensitiveAttributeStep struct {
		Type  string      `json:"type"`
		Value interface{} `json:"value"`
	}
	type functionInstance struct {
		SchemaVersion       int                        `json:"schema_version"`
		SensitiveAttributes [][]sensitiveAttributeStep `json:"sensitive_attributes"`
	}
	type terraformState struct {
		Resources []struct {
			Type      string             `json:"type"`
			Instances []functionInstance `json:"instances"`
		} `json:"resources"`
	}

	stateJSON, err := os.ReadFile(statePath)
	require.NoError(t, err)

	var state terraformState
	require.NoError(t, json.Unmarshal(stateJSON, &state))
	for _, stateResource := range state.Resources {
		if stateResource.Type != "pulsar_function" {
			continue
		}
		require.Len(t, stateResource.Instances, 1)
		instance := stateResource.Instances[0]
		require.Zero(t, instance.SchemaVersion)
		for _, path := range instance.SensitiveAttributes {
			for _, step := range path {
				if step.Type == "get_attr" && step.Value == resourceFunctionUserConfig {
					return
				}
			}
		}
		require.Fail(t, "missing user_config sensitive path", "state: %s", stateJSON)
		return
	}

	t.Fatal("pulsar_function state not found")
}
