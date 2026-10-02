// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements. See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership. The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License. You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

package pulsar

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

func TestNamespacePropertiesLifecycle(t *testing.T) {
	ctx := context.Background()
	remote := map[string]string{"unmanaged": "keep"}
	failWrite, failRead, failDelete := false, false, false
	propertyRequests := 0
	const base = "/admin/v2/namespaces/tenant/namespace"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == base+"/properties":
			propertyRequests++
			if r.Method == http.MethodGet {
				if failRead {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				writeJSONResponse(t, w, http.StatusOK, remote)
				return
			}
			require.Equal(t, http.MethodPut, r.Method)
			if failWrite {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			var values map[string]string
			require.NoError(t, json.NewDecoder(r.Body).Decode(&values))
			for key, value := range values {
				remote[key] = value
			}
			w.WriteHeader(http.StatusNoContent)
		case strings.HasPrefix(r.URL.Path, base+"/property/"):
			propertyRequests++
			require.Equal(t, http.MethodDelete, r.Method)
			if failDelete {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			delete(remote, strings.TrimPrefix(r.URL.Path, base+"/property/"))
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPut && r.URL.Path == base:
			w.WriteHeader(http.StatusNoContent)
		case r.URL.Path == "/admin/v2/namespaces/tenant":
			writeJSONResponse(t, w, http.StatusOK, []string{"tenant/namespace"})
		case r.URL.Path == base+"/bundles":
			writeJSONResponse(t, w, http.StatusOK, map[string]int{"numBundles": 4})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	client := namespacePolicyTestClientBundle(t, server.URL)
	resource := resourcePulsarNamespace()
	var state *terraform.InstanceState
	config := func(properties map[string]interface{}) map[string]interface{} {
		values := map[string]interface{}{"tenant": "tenant", "namespace": "namespace"}
		if properties != nil {
			values["namespace_properties"] = properties
		}
		return values
	}
	apply := func(properties map[string]interface{}, wantError bool) {
		t.Helper()
		diff, err := resource.Diff(ctx, state, terraform.NewResourceConfigRaw(config(properties)), client)
		require.NoError(t, err)
		next, diags := resource.Apply(ctx, state, diff, client)
		require.Equal(t, wantError, diags.HasError(), "diagnostics: %#v", diags)
		state = next
	}
	enabled := map[string]interface{}{"cluster.sdt.enabled": "true", "cluster.sdt.catalog.name": "catalog"}
	apply(enabled, false)
	require.Equal(t, "true", remote["cluster.sdt.enabled"])
	require.Equal(t, "catalog", remote["cluster.sdt.catalog.name"])
	require.Equal(t, "keep", remote["unmanaged"])
	require.Equal(t, "2", state.Attributes["namespace_properties.%"])
	diff, err := resource.Diff(ctx, state, terraform.NewResourceConfigRaw(config(enabled)), client)
	require.NoError(t, err)
	require.True(t, diff.Empty(), "post-apply plan should be empty")

	// Refresh detects changed and missing keys without adopting unrelated properties.
	remote["cluster.sdt.enabled"] = "false"
	delete(remote, "cluster.sdt.catalog.name")
	refreshed, diags := resource.RefreshWithoutUpgrade(ctx, state, client)
	require.False(t, diags.HasError(), "%#v", diags)
	state = refreshed
	require.Equal(t, "false", state.Attributes["namespace_properties.cluster.sdt.enabled"])
	require.Equal(t, "1", state.Attributes["namespace_properties.%"])
	apply(enabled, false)
	require.Equal(t, "catalog", remote["cluster.sdt.catalog.name"])

	failRead = true
	_, diags = resource.RefreshWithoutUpgrade(ctx, state, client)
	require.True(t, diags.HasError())
	failRead = false

	// Failed writes must not remove old keys or hide the unapplied change in state.
	changed := map[string]interface{}{"cluster.sdt.enabled": "false"}
	failWrite = true
	apply(changed, true)
	require.Equal(t, "catalog", remote["cluster.sdt.catalog.name"])
	require.Equal(t, "true", state.Attributes["namespace_properties.cluster.sdt.enabled"])
	failWrite = false
	apply(changed, false)
	require.NotContains(t, remote, "cluster.sdt.catalog.name")
	require.Equal(t, "false", remote["cluster.sdt.enabled"])

	failDelete = true
	apply(map[string]interface{}{}, true)
	require.Equal(t, "false", state.Attributes["namespace_properties.cluster.sdt.enabled"])
	failDelete = false
	apply(map[string]interface{}{}, false)
	require.Equal(t, map[string]string{"unmanaged": "keep"}, remote)
	apply(enabled, false)
	apply(nil, false) // Removing the attribute removes previously managed keys only.
	require.Equal(t, map[string]string{"unmanaged": "keep"}, remote)

	// Omitted properties, including fresh import state, do not adopt remote keys.
	before := propertyRequests
	data := schema.TestResourceDataRaw(t, resource.Schema, config(nil))
	require.False(t, resourcePulsarNamespaceRead(ctx, data, client).HasError())
	require.Empty(t, data.Get("namespace_properties"))
	require.Equal(t, before, propertyRequests)
}
