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

package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	adminconfig "github.com/apache/pulsar-client-go/pulsaradmin/pkg/admin/config"
	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNamespacePolicyClientRemoveBacklogQuotaByType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		quotaType utils.BacklogQuotaType
	}{
		{name: "destination storage", quotaType: utils.DestinationStorage},
		{name: "message age", quotaType: utils.MessageAge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodDelete, r.Method)
				assert.Equal(t, "/admin/v2/namespaces/public/default/backlogQuota", r.URL.Path)
				assert.Equal(t, tt.quotaType.String(), r.URL.Query().Get("backlogQuotaType"))
				assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()

			client, err := NewNamespacePolicyClient(&PulsarAdminConfig{Config: &adminconfig.Config{
				WebServiceURL: server.URL,
				Token:         "test-token",
			}})
			require.NoError(t, err)

			require.NoError(t, client.RemoveBacklogQuotaByType(
				context.Background(),
				"public/default",
				tt.quotaType,
			))
		})
	}
}
