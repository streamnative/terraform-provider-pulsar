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
	"strconv"
	"testing"

	"github.com/hashicorp/go-cty/cty/msgpack"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"
)

func TestPulsarNamespaceStateUpgradeV0(t *testing.T) {
	t.Parallel()

	resourceSchema := resourcePulsarNamespace()
	require.Equal(t, 1, resourceSchema.SchemaVersion)
	require.Len(t, resourceSchema.StateUpgraders, 1)
	require.Equal(t, 0, resourceSchema.StateUpgraders[0].Version)
	require.Equal(
		t,
		pulsarNamespaceStateTypeV0(),
		resourceSchema.StateUpgraders[0].Type,
	)
	require.True(t, resourceSchema.StateUpgraders[0].Type.HasAttribute(backlogQuotaManagedTypesStateAttr))

	t.Run("initializes v0.11 ownership state", func(t *testing.T) {
		rawState := map[string]interface{}{
			"tenant":    "tenant",
			"namespace": "namespace",
		}

		upgraded, err := resourceSchema.StateUpgraders[0].Upgrade(context.Background(), rawState, nil)
		require.NoError(t, err)
		require.Contains(t, upgraded, backlogQuotaManagedTypesStateAttr)
		require.Empty(t, upgraded[backlogQuotaManagedTypesStateAttr])
	})

	t.Run("normalizes null rc3 ownership state", func(t *testing.T) {
		rawState := map[string]interface{}{
			backlogQuotaManagedTypesStateAttr: nil,
		}

		upgraded, err := resourceSchema.StateUpgraders[0].Upgrade(context.Background(), rawState, nil)
		require.NoError(t, err)
		require.Empty(t, upgraded[backlogQuotaManagedTypesStateAttr])
		require.NotNil(t, upgraded[backlogQuotaManagedTypesStateAttr])
	})

	t.Run("preserves rc3 ownership state", func(t *testing.T) {
		existing := []interface{}{"destination_storage"}
		rawState := map[string]interface{}{
			backlogQuotaManagedTypesStateAttr: existing,
		}

		upgraded, err := resourceSchema.StateUpgraders[0].Upgrade(context.Background(), rawState, nil)
		require.NoError(t, err)
		require.Equal(t, existing, upgraded[backlogQuotaManagedTypesStateAttr])
	})
}

func TestPulsarNamespaceStateUpgradeV0Flatmap(t *testing.T) {
	t.Parallel()

	resourceSchema := resourcePulsarNamespace()
	server := schema.NewGRPCProviderServer(Provider())
	response, err := server.UpgradeResourceState(context.Background(), &tfprotov5.UpgradeResourceStateRequest{
		TypeName: "pulsar_namespace",
		Version:  0,
		RawState: &tfprotov5.RawState{Flatmap: map[string]string{
			"id":                               "tenant/namespace",
			"tenant":                           "tenant",
			"namespace":                        "namespace",
			"namespace_config.#":               "1",
			"namespace_config.0.anti_affinity": "group-a",
		}},
	})
	require.NoError(t, err)
	for _, diagnostic := range response.Diagnostics {
		require.NotEqual(t, tfprotov5.DiagnosticSeverityError, diagnostic.Severity, diagnostic.Summary)
	}
	require.NotNil(t, response.UpgradedState)

	upgradedState, err := msgpack.Unmarshal(
		response.UpgradedState.MsgPack,
		resourceSchema.CoreConfigSchema().ImpliedType(),
	)
	require.NoError(t, err)
	managedTypes := upgradedState.GetAttr(backlogQuotaManagedTypesStateAttr)
	require.True(t, managedTypes.IsKnown())
	require.True(t, managedTypes.Type().IsSetType())
	require.Zero(t, managedTypes.LengthInt())
}

func TestPulsarNamespaceStateUpgradeV0FlatmapPreservesRC3Ownership(t *testing.T) {
	t.Parallel()

	resourceSchema := resourcePulsarNamespace()
	server := schema.NewGRPCProviderServer(Provider())
	managedType := "destination_storage"
	response, err := server.UpgradeResourceState(context.Background(), &tfprotov5.UpgradeResourceStateRequest{
		TypeName: "pulsar_namespace",
		Version:  0,
		RawState: &tfprotov5.RawState{Flatmap: map[string]string{
			"id":                                     "tenant/namespace",
			"tenant":                                 "tenant",
			"namespace":                              "namespace",
			backlogQuotaManagedTypesStateAttr + ".#": "1",
			backlogQuotaManagedTypesStateAttr + "." +
				strconv.Itoa(schema.HashString(managedType)): managedType,
		}},
	})
	require.NoError(t, err)
	for _, diagnostic := range response.Diagnostics {
		require.NotEqual(t, tfprotov5.DiagnosticSeverityError, diagnostic.Severity, diagnostic.Summary)
	}
	require.NotNil(t, response.UpgradedState)

	upgradedState, err := msgpack.Unmarshal(
		response.UpgradedState.MsgPack,
		resourceSchema.CoreConfigSchema().ImpliedType(),
	)
	require.NoError(t, err)
	managedTypes := upgradedState.GetAttr(backlogQuotaManagedTypesStateAttr)
	require.True(t, managedTypes.IsKnown())
	require.True(t, managedTypes.Type().IsSetType())
	require.Equal(t, 1, managedTypes.LengthInt())

	iterator := managedTypes.ElementIterator()
	require.True(t, iterator.Next())
	_, value := iterator.Element()
	require.Equal(t, managedType, value.AsString())
}

func TestPulsarNamespaceStateUpgradeV0JSONOwnership(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		rawState   string
		wantValues []string
	}{
		{
			name:     "v0.11 field absent",
			rawState: `{"id":"tenant/namespace","tenant":"tenant","namespace":"namespace"}`,
		},
		{
			name:     "rc3 null field",
			rawState: `{"id":"tenant/namespace","tenant":"tenant","namespace":"namespace","_backlog_quota_managed_types":null}`,
		},
		{
			name: "rc3 populated field",
			rawState: `{
				"id":"tenant/namespace",
				"tenant":"tenant",
				"namespace":"namespace",
				"_backlog_quota_managed_types":["destination_storage"]
			}`,
			wantValues: []string{"destination_storage"},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			resourceSchema := resourcePulsarNamespace()
			server := schema.NewGRPCProviderServer(Provider())
			response, err := server.UpgradeResourceState(context.Background(), &tfprotov5.UpgradeResourceStateRequest{
				TypeName: "pulsar_namespace",
				Version:  0,
				RawState: &tfprotov5.RawState{JSON: []byte(test.rawState)},
			})
			require.NoError(t, err)
			for _, diagnostic := range response.Diagnostics {
				require.NotEqual(t, tfprotov5.DiagnosticSeverityError, diagnostic.Severity, diagnostic.Summary)
			}
			require.NotNil(t, response.UpgradedState)

			upgradedState, err := msgpack.Unmarshal(
				response.UpgradedState.MsgPack,
				resourceSchema.CoreConfigSchema().ImpliedType(),
			)
			require.NoError(t, err)
			managedTypes := upgradedState.GetAttr(backlogQuotaManagedTypesStateAttr)
			require.True(t, managedTypes.IsKnown())
			require.Equal(t, len(test.wantValues), managedTypes.LengthInt())
			gotValues := make(map[string]struct{}, managedTypes.LengthInt())
			iterator := managedTypes.ElementIterator()
			for iterator.Next() {
				_, value := iterator.Element()
				gotValues[value.AsString()] = struct{}{}
			}
			for _, value := range test.wantValues {
				require.Contains(t, gotValues, value)
			}
		})
	}
}

func TestPulsarNamespacePolicyBlocksAreOptionalComputedSets(t *testing.T) {
	t.Parallel()

	resourceSchema := resourcePulsarNamespace().Schema
	for _, attr := range []string{
		"dispatch_rate",
		"subscription_dispatch_rate",
		"persistence_policies",
		"backlog_quota",
	} {
		attr := attr
		t.Run(attr, func(t *testing.T) {
			t.Parallel()
			policySchema := resourceSchema[attr]
			require.Equal(t, schema.TypeSet, policySchema.Type)
			require.True(t, policySchema.Optional)
			require.True(t, policySchema.Computed)
			require.False(t, policySchema.Required)
			require.False(t, policySchema.ForceNew)
		})
	}
}

func TestPulsarNamespaceBacklogQuotaOwnershipStateIsComputedOnly(t *testing.T) {
	t.Parallel()

	ownershipSchema := resourcePulsarNamespace().Schema[backlogQuotaManagedTypesStateAttr]
	require.Equal(t, schema.TypeSet, ownershipSchema.Type)
	require.True(t, ownershipSchema.Computed)
	require.False(t, ownershipSchema.Optional)
	require.False(t, ownershipSchema.Required)
	require.False(t, ownershipSchema.ForceNew)
	require.Equal(t, schema.TypeString, ownershipSchema.Elem.(*schema.Schema).Type)
}

func TestNamespaceAndTopicDispatchRateDescriptionsAreDistinct(t *testing.T) {
	t.Parallel()

	namespaceDescription := resourcePulsarNamespace().Schema["dispatch_rate"].Description
	topicDescription := resourcePulsarTopic().Schema["dispatch_rate"].Description

	require.Contains(t, namespaceDescription, "namespace")
	require.Contains(t, namespaceDescription, "terraform import")
	require.Equal(t, "Topic-level data transfer rate for the given topic", topicDescription)
}
