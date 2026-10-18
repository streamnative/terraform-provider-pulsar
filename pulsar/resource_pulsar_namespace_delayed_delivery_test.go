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
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

func delayedDeliveryTestBlock(enabled bool, tick, maxDelay int) []interface{} {
	return []interface{}{map[string]interface{}{
		"enabled": enabled, "tick_duration_ms": tick, "max_delivery_delay_ms": maxDelay,
	}}
}

func TestPulsarNamespaceDelayedDelivery_Schema(t *testing.T) {
	block := resourcePulsarNamespace().Schema["delayed_delivery"]
	require.True(t, block.Optional)
	require.False(t, block.Computed)
	require.False(t, block.ForceNew)
	require.Equal(t, 1, block.MaxItems)
	fields := block.Elem.(*schema.Resource).Schema
	require.True(t, fields["enabled"].Required)
	require.True(t, fields["tick_duration_ms"].Required)
	require.Equal(t, 0, fields["max_delivery_delay_ms"].Default)
	for _, test := range []struct {
		field string
		value int
		valid bool
	}{
		{"tick_duration_ms", 1, true},
		{"tick_duration_ms", 100, true},
		{"tick_duration_ms", 0, false},
		{"tick_duration_ms", -1, false},
		{"tick_duration_ms", 1 << 53, false},
		{"max_delivery_delay_ms", 0, true},
		{"max_delivery_delay_ms", 60000, true},
		{"max_delivery_delay_ms", -1, false},
	} {
		_, errs := fields[test.field].ValidateFunc(test.value, test.field)
		require.Equal(t, test.valid, len(errs) == 0)
	}
}

func TestPulsarNamespaceDelayedDelivery_Read(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		body   interface{}
	}{
		{"disabled", http.StatusOK, map[string]interface{}{
			"active": false, "tickTime": 100, "maxDeliveryDelayInMillis": 60000,
		}},
		{"empty", http.StatusNoContent, nil},
		{"null", http.StatusOK, nil},
		{"forbidden", http.StatusForbidden, nil},
		{"not found", http.StatusNotFound, nil},
	} {
		for _, mode := range []namespaceReadMode{namespaceReadRefresh, namespaceReadImport} {
			t.Run(test.name+string(rune('0'+mode)), func(t *testing.T) {
				base := namespacePolicyConfiguredHandler(t, &namespacePolicyRequestRecorder{})
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/admin/v2/namespaces/tenant/namespace/delayedDelivery" {
						writeJSONResponse(t, w, test.status, test.body)
						return
					}
					base.ServeHTTP(w, r)
				}))
				t.Cleanup(server.Close)
				var config map[string]interface{}
				if mode == namespaceReadRefresh {
					config = map[string]interface{}{"delayed_delivery": delayedDeliveryTestBlock(true, 1000, 0)}
				}
				d := namespacePolicyTestResourceData(t, config)
				diags := resourcePulsarNamespaceReadWithMode(
					context.Background(), d, namespacePolicyTestClientBundle(t, server.URL), mode,
				)
				require.Equal(t, test.status >= 400, diags.HasError(), "%#v", diags)
				require.Equal(t, "tenant/namespace", d.Id())
				switch {
				case test.status >= 400:
					require.Contains(t, diags[0].Summary, "GetDelayedDelivery")
				case test.name == "disabled":
					require.Equal(t, delayedDeliveryTestBlock(false, 100, 60000), d.Get("delayed_delivery").(*schema.Set).List())
				default:
					require.Zero(t, d.Get("delayed_delivery").(*schema.Set).Len())
				}
			})
		}
	}
}

func TestPulsarNamespaceDelayedDelivery_Apply(t *testing.T) {
	for _, test := range []struct {
		name    string
		old     []interface{}
		next    []interface{}
		method  string
		failure bool
	}{
		{name: "create", next: delayedDeliveryTestBlock(true, 100, 0), method: http.MethodPost},
		{name: "disable", old: delayedDeliveryTestBlock(true, 100, 0),
			next: delayedDeliveryTestBlock(false, 200, 60000), method: http.MethodPost},
		{name: "remove", old: delayedDeliveryTestBlock(true, 100, 0), method: http.MethodDelete},
		{name: "unmanaged"},
		{name: "unchanged", old: delayedDeliveryTestBlock(false, 100, 60000),
			next: delayedDeliveryTestBlock(false, 100, 60000)},
		{name: "failed update", old: delayedDeliveryTestBlock(true, 100, 0),
			next: delayedDeliveryTestBlock(false, 200, 60000), method: http.MethodPost, failure: true},
		{name: "failed removal", old: delayedDeliveryTestBlock(true, 100, 0),
			method: http.MethodDelete, failure: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var writes []string
			base := namespacePolicyConfiguredHandler(t, &namespacePolicyRequestRecorder{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/admin/v2/namespaces/tenant/namespace/delayedDelivery" {
					base.ServeHTTP(w, r)
					return
				}
				if r.Method != http.MethodGet {
					writes = append(writes, r.Method)
					if test.failure {
						w.WriteHeader(http.StatusForbidden)
						return
					}
				}
				if r.Method == http.MethodPost {
					var body map[string]interface{}
					require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
					data := test.next[0].(map[string]interface{})
					require.Equal(t, data["enabled"], body["active"])
					require.Equal(t, float64(data["tick_duration_ms"].(int)), body["tickTime"])
					if max := data["max_delivery_delay_ms"].(int); max > 0 {
						require.Equal(t, float64(max), body["maxDeliveryDelayInMillis"])
					}
				} else if r.Method == http.MethodGet && len(test.next) > 0 {
					data := test.next[0].(map[string]interface{})
					writeJSONResponse(t, w, http.StatusOK, map[string]interface{}{
						"active": data["enabled"], "tickTime": data["tick_duration_ms"],
						"maxDeliveryDelayInMillis": data["max_delivery_delay_ms"],
					})
					return
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			t.Cleanup(server.Close)
			res := resourcePulsarNamespace()
			meta := namespacePolicyTestClientBundle(t, server.URL)
			oldData := schema.TestResourceDataRaw(t, res.Schema, map[string]interface{}{
				"tenant": "tenant", "namespace": "namespace", "delayed_delivery": test.old,
			})
			oldData.SetId("tenant/namespace")
			var state *terraform.InstanceState
			if test.name != "create" {
				state = oldData.State()
			}
			config := terraform.NewResourceConfigRaw(map[string]interface{}{
				"tenant": "tenant", "namespace": "namespace", "delayed_delivery": test.next,
				"enable_deduplication": true,
			})
			diff, err := res.Diff(context.Background(), state, config, meta)
			require.NoError(t, err)
			if state != nil {
				require.False(t, diff.RequiresNew())
			}
			result, diags := res.Apply(context.Background(), state, diff, meta)
			require.Equal(t, test.failure, diags.HasError(), "%#v", diags)
			if test.method == "" {
				require.Empty(t, writes)
			} else {
				require.Equal(t, []string{test.method}, writes)
			}
			want := test.next
			if test.failure {
				want = test.old
			}
			got := res.Data(result).Get("delayed_delivery").(*schema.Set).List()
			require.Len(t, got, len(want))
			if len(want) > 0 {
				require.Equal(t, want, got)
			}
			if !test.failure {
				nextDiff, err := res.Diff(context.Background(), result, config, meta)
				require.NoError(t, err)
				require.True(t, nextDiff.Empty(), "%#v", nextDiff)
			}
		})
	}
}
