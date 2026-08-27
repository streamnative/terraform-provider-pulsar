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

func TestAccPulsarFunction_UpgradeFromV013RefreshFalsePlanIsClean(t *testing.T) {
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

	// Newly added Optional+Computed attributes must not create an upgrade diff for v0.13 state.
	output, err = runNamespaceUpgradeTerraformWithCurrentProvider(
		t, terraformPath, workingDir, terraformEnv,
		"plan", "-refresh=false", "-detailed-exitcode", "-input=false", "-no-color",
	)
	require.NoErrorf(t, err, "v0.13 state produced a non-empty refresh=false plan:\n%s", output)
	require.Zero(t, requestCount.Load(), "refresh=false plan must not call Pulsar APIs")
}
