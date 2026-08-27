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
	"testing"

	pulsaradmin "github.com/apache/pulsar-client-go/pulsaradmin/pkg/admin"
	adminconfig "github.com/apache/pulsar-client-go/pulsaradmin/pkg/admin/config"
	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourcePulsarFunctionUpdate_UnrelatedChangesOmitProducerConfig(t *testing.T) {
	brokerProducerConfig := utils.ProducerConfig{
		MaxPendingMessages:                 1000,
		MaxPendingMessagesAcrossPartitions: 50000,
		UseThreadLocalProducers:            true,
		BatchBuilder:                       "KEY_BASED",
		CompressionType:                    "ZSTD",
	}

	for _, test := range []struct {
		name    string
		changes map[string]interface{}
	}{
		{
			name: "output",
			changes: map[string]interface{}{
				resourceFunctionOutputKey: "new-output",
			},
		},
		{
			name: "parallelism",
			changes: map[string]interface{}{
				resourceFunctionParallelismKey: 2,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			res := resourcePulsarFunction()
			state := functionProducerState(t, brokerProducerConfig, "old-output")
			config := functionConfigWithBase(map[string]interface{}{
				resourceFunctionOutputKey:            "old-output",
				resourceFunctionPCMaxPendingMsgKey:   brokerProducerConfig.MaxPendingMessages,
				resourceFunctionPCCompressionTypeKey: brokerProducerConfig.CompressionType,
			})
			for key, value := range test.changes {
				config[key] = value
			}

			var getCount int
			var sent *utils.FunctionConfig
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/admin/v3/functions/public/default/function-1" {
					http.NotFound(w, r)
					return
				}

				switch r.Method {
				case http.MethodGet:
					getCount++
					writeFunctionProducerConfigResponse(t, w, utils.FunctionConfig{
						Tenant:         "public",
						Namespace:      "default",
						Name:           "function-1",
						ProducerConfig: &brokerProducerConfig,
					})
				case http.MethodPut:
					functionConfig, err := functionConfigFromMultipartRequest(r)
					if err != nil {
						t.Errorf("decode update request: %v", err)
						http.Error(w, err.Error(), http.StatusBadRequest)
						return
					}
					sent = &functionConfig
					w.WriteHeader(http.StatusNoContent)
				default:
					http.Error(w, "unexpected method", http.StatusMethodNotAllowed)
				}
			}))
			defer server.Close()

			diff := functionProducerRawConfigDiff(t, res, state, config)
			_, diags := res.Apply(
				context.Background(),
				state,
				diff,
				functionProducerTestClientBundle(t, server.URL),
			)
			require.False(t, diags.HasError(), "apply diagnostics: %#v", diags)
			require.NotNil(t, sent)
			assert.Nil(t, sent.ProducerConfig)
			// The one GET is the normal post-update refresh. No pre-update merge GET is needed.
			assert.Equal(t, 1, getCount)
		})
	}

}

func TestResourcePulsarFunctionUpdate_MergesFreshProducerConfig(t *testing.T) {
	staleProducerConfig := utils.ProducerConfig{
		MaxPendingMessages:                 1000,
		MaxPendingMessagesAcrossPartitions: 500,
		UseThreadLocalProducers:            false,
		BatchBuilder:                       "DEFAULT",
		CompressionType:                    "LZ4",
	}
	batchingMaxMessages := 50
	brokerProducerConfig := utils.ProducerConfig{
		MaxPendingMessages:                 2000,
		MaxPendingMessagesAcrossPartitions: 50000,
		UseThreadLocalProducers:            true,
		BatchBuilder:                       "KEY_BASED",
		CompressionType:                    "ZSTD",
		CryptoConfig: &utils.CryptoConfig{
			CryptoKeyReaderClassName:    "com.acme.Reader",
			CryptoKeyReaderConfig:       map[string]interface{}{"key": "value"},
			EncryptionKeys:              []string{"key-a"},
			ProducerCryptoFailureAction: "FAIL",
			ConsumerCryptoFailureAction: "CONSUME",
		},
		BatchingConfig: &utils.BatchingConfig{
			Enabled:             false,
			BatchingMaxMessages: &batchingMaxMessages,
			BatchBuilder:        "KEY_BASED",
		},
	}

	res := resourcePulsarFunction()
	state := functionProducerState(t, staleProducerConfig, "old-output")
	config := functionConfigWithBase(map[string]interface{}{
		resourceFunctionOutputKey:          "old-output",
		resourceFunctionPCMaxPendingMsgKey: 0,
	})

	var getCount int
	var updateCount int
	var sent *utils.FunctionConfig
	functionConfigResponse := utils.FunctionConfig{
		Tenant:         "public",
		Namespace:      "default",
		Name:           "function-1",
		Output:         "old-output",
		ProducerConfig: &brokerProducerConfig,
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/admin/v3/functions/public/default/function-1" {
			http.NotFound(w, r)
			return
		}

		switch r.Method {
		case http.MethodGet:
			getCount++
			writeFunctionProducerConfigResponse(t, w, functionConfigResponse)
		case http.MethodPut:
			functionConfig, err := functionConfigFromMultipartRequest(r)
			if err != nil {
				t.Errorf("decode update request: %v", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			sent = &functionConfig
			functionConfigResponse.ProducerConfig = functionConfig.ProducerConfig
			updateCount++
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "unexpected method", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	diff := functionProducerRawConfigDiff(t, res, state, config)
	_, diags := res.Apply(
		context.Background(),
		state,
		diff,
		functionProducerTestClientBundle(t, server.URL),
	)
	require.False(t, diags.HasError(), "apply diagnostics: %#v", diags)
	require.NotNil(t, sent)
	require.NotNil(t, sent.ProducerConfig)

	expectedProducerConfig := brokerProducerConfig
	expectedProducerConfig.MaxPendingMessages = 0
	assert.Equal(t, &expectedProducerConfig, sent.ProducerConfig)
	assert.Equal(t, brokerProducerConfig.CryptoConfig, sent.ProducerConfig.CryptoConfig)
	assert.Equal(t, brokerProducerConfig.BatchingConfig, sent.ProducerConfig.BatchingConfig)
	// Omitted fields came from the fresh broker GET, not stale Terraform state.
	assert.Equal(t, brokerProducerConfig.MaxPendingMessagesAcrossPartitions,
		sent.ProducerConfig.MaxPendingMessagesAcrossPartitions)
	assert.NotEqual(t, staleProducerConfig.MaxPendingMessagesAcrossPartitions,
		sent.ProducerConfig.MaxPendingMessagesAcrossPartitions)
	assert.Equal(t, 1, updateCount)
	// One pre-update merge GET plus the normal post-update refresh.
	assert.Equal(t, 2, getCount)
}

func TestResourcePulsarFunctionCreate_ProducerConfigBehaviorUnchanged(t *testing.T) {
	res := resourcePulsarFunction()
	config := functionConfigWithBase(map[string]interface{}{
		resourceFunctionPCMaxPendingMsgKey:           0,
		resourceFunctionPCUseThreadLocalProducersKey: false,
	})

	var getCount int
	var createCount int
	var sent *utils.FunctionConfig
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/admin/v3/functions/public/default/function-1" {
			http.NotFound(w, r)
			return
		}

		switch r.Method {
		case http.MethodGet:
			getCount++
			var producerConfig *utils.ProducerConfig
			if sent != nil {
				producerConfig = sent.ProducerConfig
			}
			writeFunctionProducerConfigResponse(t, w, utils.FunctionConfig{
				Tenant:         "public",
				Namespace:      "default",
				Name:           "function-1",
				ProducerConfig: producerConfig,
			})
		case http.MethodPost:
			functionConfig, err := functionConfigFromMultipartRequest(r)
			if err != nil {
				t.Errorf("decode create request: %v", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			sent = &functionConfig
			createCount++
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "unexpected method", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	diff, err := res.Diff(
		context.Background(),
		nil,
		terraform.NewResourceConfigRaw(config),
		nil,
	)
	require.NoError(t, err)
	require.NotNil(t, diff)

	_, diags := res.Apply(
		context.Background(),
		nil,
		diff,
		functionProducerTestClientBundle(t, server.URL),
	)
	require.False(t, diags.HasError(), "create diagnostics: %#v", diags)
	require.NotNil(t, sent)
	assert.Equal(t, &utils.ProducerConfig{CompressionType: "LZ4"}, sent.ProducerConfig)
	assert.Equal(t, 1, createCount)
	// The one GET is the normal post-create refresh; creation never needs a merge read.
	assert.Equal(t, 1, getCount)
}

func functionProducerRawConfigDiff(
	t *testing.T,
	res *schema.Resource,
	state *terraform.InstanceState,
	config map[string]interface{},
) *terraform.InstanceDiff {
	t.Helper()

	legacyDiff, err := res.Diff(context.Background(), state, terraform.NewResourceConfigRaw(config), nil)
	require.NoError(t, err)
	require.NotNil(t, legacyDiff)

	prior, err := schema.StateValueFromInstanceState(state, res.CoreConfigSchema().ImpliedType())
	require.NoError(t, err)
	planned, err := schema.ApplyDiff(prior, legacyDiff, res.CoreConfigSchema())
	require.NoError(t, err)
	rawConfig, err := schema.JSONMapToStateValue(config, res.CoreConfigSchema())
	require.NoError(t, err)
	diff, err := schema.DiffFromValues(context.Background(), prior, planned, rawConfig, res)
	require.NoError(t, err)
	require.NotNil(t, diff)

	return diff
}

func functionProducerTestClientBundle(t *testing.T, serverURL string) PulsarClientBundle {
	t.Helper()

	client, err := pulsaradmin.New(&adminconfig.Config{
		WebServiceURL:    serverURL,
		PulsarAPIVersion: adminconfig.V3,
	})
	require.NoError(t, err)

	return PulsarClientBundle{V3Client: client}
}

func functionConfigFromMultipartRequest(r *http.Request) (utils.FunctionConfig, error) {
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		return utils.FunctionConfig{}, err
	}
	if r.MultipartForm == nil || len(r.MultipartForm.Value["functionConfig"]) != 1 {
		return utils.FunctionConfig{}, fmt.Errorf("missing functionConfig multipart field")
	}

	var functionConfig utils.FunctionConfig
	if err := json.Unmarshal([]byte(r.MultipartForm.Value["functionConfig"][0]), &functionConfig); err != nil {
		return utils.FunctionConfig{}, err
	}

	return functionConfig, nil
}

func writeFunctionProducerConfigResponse(t *testing.T, w http.ResponseWriter, functionConfig utils.FunctionConfig) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(functionConfig); err != nil {
		t.Errorf("encode function config response: %v", err)
	}
}
