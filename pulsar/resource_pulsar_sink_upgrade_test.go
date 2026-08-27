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

func TestAccPulsarSink_UpgradeFromV013RefreshFalseStateOnlyMigration(t *testing.T) {
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
		http.NotFound(w, nil)
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

resource "pulsar_sink" "test" {
  name                  = "sink-1"
  tenant                = "public"
  namespace             = "default"
  inputs                = ["persistent://public/default/in-1"]
  cleanup_subscription  = false
  subscription_position = "Earliest"
  parallelism           = 1
  auto_ack              = true
  processing_guarantees = "ATLEAST_ONCE"
  retain_ordering       = true
  archive               = "builtin://jdbc"
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

	legacyState, err := os.ReadFile(filepath.Join("testdata", "pulsar_sink_v013_state.json"))
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
	requireTerraformExitCode(t, err, 2, output)
	require.Zero(t, requestCount.Load(), "refresh=false state migration must not call Pulsar APIs")

	output, err = runNamespaceUpgradeTerraformWithCurrentProvider(
		t, terraformPath, workingDir, terraformEnv,
		"apply", "-refresh=false", "-auto-approve", "-input=false", "-no-color",
	)
	require.NoErrorf(t, err, "v0.13 state refresh=false apply failed:\n%s", output)
	require.Zero(t, requestCount.Load(), "state-only apply must not call Pulsar APIs")
	assertSinkStateUpgradedToV1(t, filepath.Join(workingDir, "terraform.tfstate"))

	output, err = runNamespaceUpgradeTerraformWithCurrentProvider(
		t, terraformPath, workingDir, terraformEnv,
		"plan", "-refresh=false", "-detailed-exitcode", "-input=false", "-no-color",
	)
	require.NoErrorf(t, err, "upgraded v0.13 state produced a non-empty refresh=false plan:\n%s", output)
	require.Zero(t, requestCount.Load(), "upgraded refresh=false plan must not call Pulsar APIs")
}

func requireTerraformExitCode(t *testing.T, err error, want int, output string) {
	t.Helper()

	exitErr, ok := err.(*exec.ExitError)
	require.Truef(t, ok, "terraform exited unexpectedly: %v\n%s", err, output)
	require.Equalf(t, want, exitErr.ExitCode(), "terraform output:\n%s", output)
}

func assertSinkStateUpgradedToV1(t *testing.T, statePath string) {
	t.Helper()

	type sinkInstance struct {
		SchemaVersion int                    `json:"schema_version"`
		Attributes    map[string]interface{} `json:"attributes"`
	}
	type terraformState struct {
		Resources []struct {
			Type      string         `json:"type"`
			Instances []sinkInstance `json:"instances"`
		} `json:"resources"`
	}

	stateJSON, err := os.ReadFile(statePath)
	require.NoError(t, err)

	var state terraformState
	require.NoError(t, json.Unmarshal(stateJSON, &state))

	for _, stateResource := range state.Resources {
		if stateResource.Type != "pulsar_sink" {
			continue
		}
		require.Len(t, stateResource.Instances, 1)
		instance := stateResource.Instances[0]
		require.Equal(t, 1, instance.SchemaVersion)

		inputs, ok := instance.Attributes[resourceSinkInputsKey].([]interface{})
		require.True(t, ok, "upgraded inputs state has type %T", instance.Attributes[resourceSinkInputsKey])
		require.Equal(t, []interface{}{"persistent://public/default/in-1"}, inputs)

		inputSpecs, ok := instance.Attributes[resourceSinkInputSpecsKey].([]interface{})
		require.True(t, ok, "upgraded input_specs state has type %T", instance.Attributes[resourceSinkInputSpecsKey])
		require.Len(t, inputSpecs, 1)
		inputSpec, ok := inputSpecs[0].(map[string]interface{})
		require.True(t, ok, "upgraded input_specs item has type %T", inputSpecs[0])
		require.Equal(t, "persistent://public/default/in-1", inputSpec[resourceSinkInputSpecsSubsetTopicKey])
		require.Equal(t, float64(0), inputSpec[resourceSinkInputSpecsSubsetReceiverQueueSizeKey])
		return
	}

	t.Fatal("pulsar_sink state not found")
}
