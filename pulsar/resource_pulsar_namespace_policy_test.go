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
	"net/http"
	"net/http/httptest"
	"sort"
	"sync"
	"testing"

	pulsaradmin "github.com/apache/pulsar-client-go/pulsaradmin/pkg/admin"
	adminconfig "github.com/apache/pulsar-client-go/pulsaradmin/pkg/admin/config"
	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"

	provideradmin "github.com/streamnative/terraform-provider-pulsar/pkg/admin"
)

func TestResourcePulsarNamespaceRead_MinimalRefreshOnlyReadsBundles(t *testing.T) {
	t.Parallel()

	var policyReads int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/admin/v2/namespaces/tenant" {
			writeJSONResponse(t, w, http.StatusOK, []string{"tenant/namespace"})
			return
		}
		if r.URL.Path == "/admin/v2/namespaces/tenant/namespace/bundles" {
			writeJSONResponse(t, w, http.StatusOK, map[string]interface{}{
				"boundaries": []string{"0x00000000", "0xffffffff"},
				"numBundles": 1,
			})
			return
		}
		policyReads++
		writeJSONResponse(t, w, http.StatusForbidden, map[string]string{"reason": "policy read forbidden"})
	}))
	defer server.Close()

	d := namespacePolicyTestResourceData(t, nil)
	diags := resourcePulsarNamespaceReadWithMode(
		context.Background(), d, namespacePolicyTestClientBundle(t, server.URL), namespaceReadRefresh,
	)
	require.False(t, diags.HasError(), "unexpected diagnostics: %#v", diags)
	require.Zero(t, policyReads)
}

func TestResourcePulsarNamespaceCreate_Bundles(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name            string
		configured      interface{}
		brokerBundles   int
		wantRequestBody bool
	}{
		{name: "broker default", brokerBundles: 4},
		{name: "explicit count", configured: 3, brokerBundles: 3, wantRequestBody: true},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == http.MethodPut && r.URL.Path == "/admin/v2/namespaces/tenant/namespace":
					if test.wantRequestBody {
						var policies utils.Policies
						require.NoError(t, json.NewDecoder(r.Body).Decode(&policies))
						require.NotNil(t, policies.Bundles)
						require.Equal(t, test.brokerBundles, policies.Bundles.NumBundles)
						require.NotNil(t, policies.ReplicationClusters)
					} else {
						require.Zero(t, r.ContentLength)
					}
					w.WriteHeader(http.StatusNoContent)
				case r.Method == http.MethodGet && r.URL.Path == "/admin/v2/namespaces/tenant":
					writeJSONResponse(t, w, http.StatusOK, []string{"tenant/namespace"})
				case r.Method == http.MethodGet && r.URL.Path == "/admin/v2/namespaces/tenant/namespace/bundles":
					writeJSONResponse(t, w, http.StatusOK, map[string]interface{}{"numBundles": test.brokerBundles})
				default:
					writeJSONResponse(t, w, http.StatusNotFound, map[string]string{"reason": "unexpected request"})
				}
			}))
			defer server.Close()

			config := map[string]interface{}{
				"tenant":    "tenant",
				"namespace": "namespace",
			}
			if test.configured != nil {
				config["bundles"] = test.configured
			}
			d := schema.TestResourceDataRaw(t, resourcePulsarNamespace().Schema, config)
			diags := resourcePulsarNamespaceCreate(
				context.Background(), d, namespacePolicyTestClientBundle(t, server.URL),
			)
			require.False(t, diags.HasError(), "unexpected diagnostics: %#v", diags)
			require.Equal(t, "tenant/namespace", d.Id())
			require.Equal(t, test.brokerBundles, d.Get("bundles"))
		})
	}
}

func TestResourcePulsarNamespaceRead_ImportRequiresPolicyReads(t *testing.T) {
	t.Parallel()

	var policyReads int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/admin/v2/namespaces/tenant" {
			writeJSONResponse(t, w, http.StatusOK, []string{"tenant/namespace"})
			return
		}
		if r.URL.Path == "/admin/v2/namespaces/tenant/namespace/bundles" {
			writeJSONResponse(t, w, http.StatusOK, map[string]interface{}{"numBundles": 4})
			return
		}
		policyReads++
		writeJSONResponse(t, w, http.StatusForbidden, map[string]string{"reason": "policy read forbidden"})
	}))
	defer server.Close()

	d := namespacePolicyTestResourceData(t, nil)
	diags := resourcePulsarNamespaceReadWithMode(
		context.Background(), d, namespacePolicyTestClientBundle(t, server.URL), namespaceReadImport,
	)
	require.True(t, diags.HasError())
	require.Equal(t, 1, policyReads)
	require.Contains(t, diags[0].Summary, "GetPersistence")
}

func TestResourcePulsarNamespaceRead_UnsetPoliciesClearTrackedState(t *testing.T) {
	t.Parallel()

	for _, response := range []struct {
		name   string
		status int
		body   interface{}
	}{
		{name: "no content", status: http.StatusNoContent},
		{name: "json null", status: http.StatusOK, body: nil},
	} {
		response := response
		t.Run(response.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/admin/v2/namespaces/tenant" {
					writeJSONResponse(t, w, http.StatusOK, []string{"tenant/namespace"})
					return
				}
				if r.URL.Path == "/admin/v2/namespaces/tenant/namespace/bundles" {
					writeJSONResponse(t, w, http.StatusOK, map[string]interface{}{"numBundles": 4})
					return
				}
				writeJSONResponse(t, w, response.status, response.body)
			}))
			defer server.Close()

			d := namespacePolicyTestResourceData(t, namespacePolicyTestBlocks())
			diags := resourcePulsarNamespaceReadWithMode(
				context.Background(), d, namespacePolicyTestClientBundle(t, server.URL), namespaceReadRefresh,
			)
			require.False(t, diags.HasError(), "unexpected diagnostics: %#v", diags)
			require.Zero(t, d.Get("dispatch_rate").(*schema.Set).Len())
			require.Zero(t, d.Get("subscription_dispatch_rate").(*schema.Set).Len())
			require.Zero(t, d.Get("persistence_policies").(*schema.Set).Len())
			require.Zero(t, d.Get("backlog_quota").(*schema.Set).Len())
		})
	}
}

func TestResourcePulsarNamespaceRead_PolicyNotFoundMarksResourceMissing(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/admin/v2/namespaces/tenant" {
			writeJSONResponse(t, w, http.StatusOK, []string{"tenant/namespace"})
			return
		}
		writeJSONResponse(t, w, http.StatusNotFound, map[string]string{"reason": "namespace missing"})
	}))
	defer server.Close()

	d := namespacePolicyTestResourceData(t, map[string]interface{}{
		"persistence_policies": namespacePolicyTestBlocks()["persistence_policies"],
	})
	diags := resourcePulsarNamespaceReadWithMode(
		context.Background(), d, namespacePolicyTestClientBundle(t, server.URL), namespaceReadRefresh,
	)
	require.False(t, diags.HasError(), "unexpected diagnostics: %#v", diags)
	require.Empty(t, d.Id())
}

func TestResourcePulsarNamespaceUpdate_UnrelatedChangeDoesNotWriteHydratedPolicies(t *testing.T) {
	t.Parallel()

	recorder := &namespacePolicyRequestRecorder{}
	server := httptest.NewServer(namespacePolicyConfiguredHandler(t, recorder))
	defer server.Close()

	d := schema.TestResourceDataRaw(t, resourcePulsarNamespace().Schema, map[string]interface{}{
		"tenant":               "tenant",
		"namespace":            "namespace",
		"enable_deduplication": true,
	})
	for attr, value := range namespacePolicyTestBlocks() {
		require.NoError(t, d.Set(attr, value))
		require.False(t, d.HasChange(attr), "%s unexpectedly marked changed", attr)
	}
	d.SetId("tenant/namespace")

	diags := resourcePulsarNamespaceUpdate(
		context.Background(), d, namespacePolicyTestClientBundle(t, server.URL),
	)
	require.False(t, diags.HasError(), "unexpected diagnostics: %#v", diags)
	require.Equal(t, []string{"/admin/v2/namespaces/tenant/namespace/deduplication"}, recorder.postPaths())
}

func TestResourcePulsarNamespaceUpdate_ExplicitPoliciesAreWritten(t *testing.T) {
	t.Parallel()

	recorder := &namespacePolicyRequestRecorder{}
	server := httptest.NewServer(namespacePolicyConfiguredHandler(t, recorder))
	defer server.Close()

	config := map[string]interface{}{
		"tenant":    "tenant",
		"namespace": "namespace",
	}
	for attr, value := range namespacePolicyTestBlocks() {
		config[attr] = value
	}
	d := schema.TestResourceDataRaw(t, resourcePulsarNamespace().Schema, config)
	d.SetId("tenant/namespace")

	diags := resourcePulsarNamespaceUpdate(
		context.Background(), d, namespacePolicyTestClientBundle(t, server.URL),
	)
	require.False(t, diags.HasError(), "unexpected diagnostics: %#v", diags)
	require.ElementsMatch(t, []string{
		"/admin/v2/namespaces/tenant/namespace/backlogQuota",
		"/admin/v2/namespaces/tenant/namespace/dispatchRate",
		"/admin/v2/namespaces/tenant/namespace/persistence",
		"/admin/v2/namespaces/tenant/namespace/subscriptionDispatchRate",
	}, recorder.postPaths())
}

func TestResourcePulsarNamespaceUpdate_BacklogQuotaWriteFailureSkipsRemoval(t *testing.T) {
	t.Parallel()

	recorder := &namespacePolicyRequestRecorder{}
	baseHandler := namespacePolicyConfiguredHandler(t, recorder)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/admin/v2/namespaces/tenant/namespace/backlogQuota" {
			recorder.record(r.Method, r.URL.Path)
			writeJSONResponse(t, w, http.StatusInternalServerError, map[string]string{"reason": "write failed"})
			return
		}
		baseHandler.ServeHTTP(w, r)
	}))
	defer server.Close()

	resourceSchema := resourcePulsarNamespace()
	oldQuota := namespacePolicyTestBacklogQuota(utils.DestinationStorage)
	newQuota := namespacePolicyTestBacklogQuota(utils.MessageAge)
	oldData := schema.TestResourceDataRaw(t, resourceSchema.Schema, map[string]interface{}{
		"tenant":        "tenant",
		"namespace":     "namespace",
		"backlog_quota": []interface{}{oldQuota},
	})
	oldData.SetId("tenant/namespace")
	require.NoError(t, oldData.Set(
		backlogQuotaManagedTypesStateAttr,
		schema.NewSet(schema.HashString, []interface{}{utils.DestinationStorage.String()}),
	))
	state := oldData.State()
	require.NotNil(t, state)
	state.RawState = testBacklogQuotaOwnershipRawState(utils.DestinationStorage)
	state.RawConfig = testBacklogQuotaRawConfig(newQuota)

	diff, err := resourceSchema.Diff(
		context.Background(),
		state,
		terraform.NewResourceConfigRaw(map[string]interface{}{
			"tenant":        "tenant",
			"namespace":     "namespace",
			"backlog_quota": []interface{}{newQuota},
		}),
		namespacePolicyTestClientBundle(t, server.URL),
	)
	require.NoError(t, err)
	require.NotNil(t, diff)

	failedState, diags := resourceSchema.Apply(
		context.Background(),
		state,
		diff,
		namespacePolicyTestClientBundle(t, server.URL),
	)
	require.True(t, diags.HasError())
	require.Empty(t, recorder.deletePaths())
	requireNamespaceBacklogQuotaStateTypes(t, failedState, utils.DestinationStorage)
	require.Equal(t, "1", failedState.Attributes[backlogQuotaManagedTypesStateAttr+".#"])
}

func TestResourcePulsarNamespaceUpdate_UnrelatedFailureDoesNotClaimImportedBacklogQuota(t *testing.T) {
	t.Parallel()

	recorder := &namespacePolicyRequestRecorder{}
	baseHandler := namespacePolicyConfiguredHandler(t, recorder)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/admin/v2/namespaces/tenant/namespace/deduplication" {
			recorder.record(r.Method, r.URL.Path)
			writeJSONResponse(t, w, http.StatusInternalServerError, map[string]string{"reason": "write failed"})
			return
		}
		baseHandler.ServeHTTP(w, r)
	}))
	defer server.Close()

	resourceSchema := resourcePulsarNamespace()
	destination := namespacePolicyTestBacklogQuota(utils.DestinationStorage)
	messageAge := namespacePolicyTestBacklogQuota(utils.MessageAge)
	oldData := schema.TestResourceDataRaw(t, resourceSchema.Schema, map[string]interface{}{
		"tenant":        "tenant",
		"namespace":     "namespace",
		"backlog_quota": []interface{}{destination, messageAge},
	})
	oldData.SetId("tenant/namespace")
	require.NoError(t, oldData.Set(backlogQuotaManagedTypesStateAttr, []interface{}{}))
	state := oldData.State()
	require.NotNil(t, state)
	state.RawState = testBacklogQuotaOwnershipRawState()
	state.RawConfig = testBacklogQuotaRawConfig(destination)

	diff, err := resourceSchema.Diff(
		context.Background(),
		state,
		terraform.NewResourceConfigRaw(map[string]interface{}{
			"tenant":               "tenant",
			"namespace":            "namespace",
			"enable_deduplication": true,
			"backlog_quota":        []interface{}{destination},
		}),
		namespacePolicyTestClientBundle(t, server.URL),
	)
	require.NoError(t, err)
	require.NotNil(t, diff)

	failedState, diags := resourceSchema.Apply(
		context.Background(),
		state,
		diff,
		namespacePolicyTestClientBundle(t, server.URL),
	)
	require.True(t, diags.HasError())
	requireNamespaceBacklogQuotaStateTypes(
		t,
		failedState,
		utils.DestinationStorage,
		utils.MessageAge,
	)
	require.Equal(t, "0", failedState.Attributes[backlogQuotaManagedTypesStateAttr+".#"])
}

func requireNamespaceBacklogQuotaStateTypes(
	t *testing.T,
	state *terraform.InstanceState,
	expected ...utils.BacklogQuotaType,
) {
	t.Helper()
	require.NotNil(t, state)
	data := resourcePulsarNamespace().Data(state)
	types, err := backlogQuotaTypes(data.Get("backlog_quota").(*schema.Set))
	require.NoError(t, err)
	require.Len(t, types, len(expected))
	for _, quotaType := range expected {
		require.Contains(t, types, quotaType)
	}
}

func namespacePolicyTestBacklogQuota(quotaType utils.BacklogQuotaType) map[string]interface{} {
	if quotaType == utils.MessageAge {
		return map[string]interface{}{
			"limit_bytes":   "-1",
			"limit_seconds": "3600",
			"policy":        utils.ConsumerBacklogEviction.String(),
			"type":          quotaType.String(),
		}
	}
	return map[string]interface{}{
		"limit_bytes":   "100",
		"limit_seconds": "-1",
		"policy":        utils.ProducerRequestHold.String(),
		"type":          quotaType.String(),
	}
}

func namespacePolicyTestResourceData(t *testing.T, blocks map[string]interface{}) *schema.ResourceData {
	t.Helper()
	d := resourcePulsarNamespace().TestResourceData()
	require.NoError(t, d.Set("tenant", "tenant"))
	require.NoError(t, d.Set("namespace", "namespace"))
	for attr, value := range blocks {
		require.NoError(t, d.Set(attr, value))
	}
	d.SetId("tenant/namespace")
	return d
}

func namespacePolicyTestBlocks() map[string]interface{} {
	return map[string]interface{}{
		"dispatch_rate": []interface{}{map[string]interface{}{
			"dispatch_msg_throttling_rate":  50,
			"rate_period_seconds":           50,
			"dispatch_byte_throttling_rate": 2048,
		}},
		"subscription_dispatch_rate": []interface{}{map[string]interface{}{
			"dispatch_msg_throttling_rate":  50,
			"rate_period_seconds":           50,
			"dispatch_byte_throttling_rate": 2048,
		}},
		"persistence_policies": []interface{}{map[string]interface{}{
			"bookkeeper_ensemble":                 2,
			"bookkeeper_write_quorum":             2,
			"bookkeeper_ack_quorum":               2,
			"managed_ledger_max_mark_delete_rate": 0.0,
		}},
		"backlog_quota": []interface{}{map[string]interface{}{
			"limit_bytes":   "100",
			"limit_seconds": "-1",
			"policy":        string(utils.ProducerRequestHold),
			"type":          string(utils.DestinationStorage),
		}},
	}
}

func namespacePolicyTestClientBundle(t *testing.T, serverURL string) PulsarClientBundle {
	t.Helper()
	config := &adminconfig.Config{WebServiceURL: serverURL}
	client, err := pulsaradmin.New(config)
	require.NoError(t, err)
	policyClient, err := provideradmin.NewNamespacePolicyClient(&provideradmin.PulsarAdminConfig{Config: config})
	require.NoError(t, err)
	return PulsarClientBundle{
		Client:                client,
		V3Client:              client,
		NamespacePolicyClient: policyClient,
	}
}

func namespacePolicyConfiguredHandler(
	t *testing.T,
	recorder *namespacePolicyRequestRecorder,
) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder.record(r.Method, r.URL.Path)
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		switch r.URL.Path {
		case "/admin/v2/namespaces/tenant":
			writeJSONResponse(t, w, http.StatusOK, []string{"tenant/namespace"})
		case "/admin/v2/namespaces/tenant/namespace/bundles":
			writeJSONResponse(t, w, http.StatusOK, map[string]interface{}{"numBundles": 4})
		case "/admin/v2/namespaces/tenant/namespace/persistence":
			writeJSONResponse(t, w, http.StatusOK, map[string]interface{}{
				"bookkeeperEnsemble":             2,
				"bookkeeperWriteQuorum":          2,
				"bookkeeperAckQuorum":            2,
				"managedLedgerMaxMarkDeleteRate": 0.0,
			})
		case "/admin/v2/namespaces/tenant/namespace/backlogQuotaMap":
			writeJSONResponse(t, w, http.StatusOK, map[string]interface{}{
				"destination_storage": map[string]interface{}{
					"limitSize": 100,
					"limitTime": -1,
					"policy":    string(utils.ProducerRequestHold),
				},
			})
		case "/admin/v2/namespaces/tenant/namespace/dispatchRate",
			"/admin/v2/namespaces/tenant/namespace/subscriptionDispatchRate":
			writeJSONResponse(t, w, http.StatusOK, map[string]interface{}{
				"dispatchThrottlingRateInMsg":  50,
				"dispatchThrottlingRateInByte": 2048,
				"ratePeriodInSecond":           50,
			})
		default:
			writeJSONResponse(t, w, http.StatusNotFound, map[string]string{"reason": "not found"})
		}
	})
}

func writeJSONResponse(t *testing.T, w http.ResponseWriter, status int, body interface{}) {
	t.Helper()
	w.WriteHeader(status)
	if status == http.StatusNoContent {
		return
	}
	if err := json.NewEncoder(w).Encode(body); err != nil {
		t.Errorf("encode HTTP response: %v", err)
	}
}

type namespacePolicyRequestRecorder struct {
	mu       sync.Mutex
	requests []namespacePolicyRequest
}

type namespacePolicyRequest struct {
	method string
	path   string
}

func (r *namespacePolicyRequestRecorder) record(method, path string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requests = append(r.requests, namespacePolicyRequest{method: method, path: path})
}

func (r *namespacePolicyRequestRecorder) postPaths() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	paths := make([]string, 0)
	for _, request := range r.requests {
		if request.method == http.MethodPost {
			paths = append(paths, request.path)
		}
	}
	sort.Strings(paths)
	return paths
}

func (r *namespacePolicyRequestRecorder) deletePaths() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	paths := make([]string, 0)
	for _, request := range r.requests {
		if request.method == http.MethodDelete {
			paths = append(paths, request.path)
		}
	}
	sort.Strings(paths)
	return paths
}
