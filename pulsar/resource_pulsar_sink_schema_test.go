// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
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
	"strings"
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/go-cty/cty/msgpack"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

func TestPulsarSinkStateUpgradeV0(t *testing.T) {
	resourceSchema := resourcePulsarSink()
	require.Equal(t, 1, resourceSchema.SchemaVersion)
	require.Len(t, resourceSchema.StateUpgraders, 1)
	require.Equal(t, 0, resourceSchema.StateUpgraders[0].Version)
	require.Equal(t, pulsarSinkStateTypeV0(), resourceSchema.StateUpgraders[0].Type)
	require.True(t, pulsarSinkStateTypeV0().HasAttribute(resourceSinkInputSpecsKey))

	inputSpecs := []interface{}{map[string]interface{}{
		resourceSinkInputSpecsSubsetTopicKey:             "persistent://public/default/in-1",
		resourceSinkInputSpecsSubsetSchemaTypeKey:        "",
		resourceSinkInputSpecsSubsetSerdeClassNameKey:    "",
		resourceSinkInputSpecsSubsetIsRegexPatternKey:    false,
		resourceSinkInputSpecsSubsetReceiverQueueSizeKey: 0,
	}}
	rawState := map[string]interface{}{
		resourceSinkInputsKey:     []interface{}{"persistent://public/default/in-1"},
		resourceSinkInputSpecsKey: inputSpecs,
	}

	upgraded, err := resourceSchema.StateUpgraders[0].Upgrade(context.Background(), rawState, nil)
	require.NoError(t, err)
	require.Equal(t, rawState, upgraded)
	require.Equal(t, inputSpecs, upgraded[resourceSinkInputSpecsKey])
}

func TestPulsarSinkStateUpgradeV0FlatmapRehashesInputSpecs(t *testing.T) {
	legacyState, config, topic := sinkV013ReceiverQueueZeroState(t)
	legacyHash := sinkInputSpecsStateHash(t, legacyState.Attributes)
	legacyQueueSizeKey := resourceSinkInputSpecsKey + "." + legacyHash + "." +
		resourceSinkInputSpecsSubsetReceiverQueueSizeKey
	require.Equal(t, "0", legacyState.Attributes[legacyQueueSizeKey])

	upgradedValue := upgradePulsarSinkV0State(t, &tfprotov5.RawState{
		Flatmap: legacyState.Attributes,
	})
	requireSinkInputSpecReceiverQueueSize(t, upgradedValue, topic, 0)

	resourceSchema := resourcePulsarSink()
	upgradedState, err := resourceSchema.ShimInstanceStateFromValue(upgradedValue)
	require.NoError(t, err)
	upgradedHash := sinkInputSpecsStateHash(t, upgradedState.Attributes)
	require.NotEqual(t, legacyHash, upgradedHash)
	upgradedQueueSizeKey := resourceSinkInputSpecsKey + "." + upgradedHash + "." +
		resourceSinkInputSpecsSubsetReceiverQueueSizeKey
	require.Equal(t, "0", upgradedState.Attributes[upgradedQueueSizeKey])
	requireSinkV013NoRefreshDiffEmpty(t, resourceSchema, upgradedState, config)
}

func TestPulsarSinkStateUpgradeV0JSONRehashesInputSpecs(t *testing.T) {
	_, config, topic := sinkV013ReceiverQueueZeroState(t)
	legacyJSON, err := json.Marshal(map[string]interface{}{
		"id":                               "public/default/sink-1",
		resourceSinkArchiveKey:             "builtin://jdbc",
		resourceSinkAutoACKKey:             true,
		resourceSinkCleanupSubscriptionKey: false,
		resourceSinkInputsKey:              []interface{}{topic},
		resourceSinkInputSpecsKey: []interface{}{map[string]interface{}{
			resourceSinkInputSpecsSubsetTopicKey:             topic,
			resourceSinkInputSpecsSubsetSchemaTypeKey:        "",
			resourceSinkInputSpecsSubsetSerdeClassNameKey:    "",
			resourceSinkInputSpecsSubsetIsRegexPatternKey:    false,
			resourceSinkInputSpecsSubsetReceiverQueueSizeKey: 0,
		}},
		resourceSinkNameKey:                 "sink-1",
		resourceSinkNamespaceKey:            "default",
		resourceSinkParallelismKey:          1,
		resourceSinkProcessingGuaranteesKey: ProcessingGuaranteesAtLeastOnce,
		resourceSinkRetainOrderingKey:       true,
		resourceSinkSubscriptionPositionKey: SubscriptionPositionEarliest,
		resourceSinkTenantKey:               "public",
	})
	require.NoError(t, err)

	upgradedValue := upgradePulsarSinkV0State(t, &tfprotov5.RawState{JSON: legacyJSON})
	requireSinkInputSpecReceiverQueueSize(t, upgradedValue, topic, 0)

	resourceSchema := resourcePulsarSink()
	upgradedState, err := resourceSchema.ShimInstanceStateFromValue(upgradedValue)
	require.NoError(t, err)
	requireSinkV013NoRefreshDiffEmpty(t, resourceSchema, upgradedState, config)
}

func TestPulsarSinkStateUpgradeV0FlatmapPreservesV014RCInputSpecFields(t *testing.T) {
	topic := "persistent://public/default/in-1"
	config := sinkConfigWithBase(map[string]interface{}{
		resourceSinkAutoACKKey: true,
		resourceSinkInputSpecsKey: []interface{}{sinkInputSpec(topic, map[string]interface{}{
			resourceSinkInputSpecsSubsetReceiverQueueSizeKey: 0,
			resourceSinkInputSpecsSubsetPoolMessagesKey:      true,
			resourceSinkInputSpecsSubsetConsumerPropertiesKey: map[string]interface{}{
				"application": "billing",
			},
		})},
	})
	resourceSchema := resourcePulsarSink()
	legacyData := schema.TestResourceDataRaw(t, resourceSchema.Schema, config)
	legacyData.SetId("public/default/sink-1")

	upgradedValue := upgradePulsarSinkV0State(t, &tfprotov5.RawState{
		Flatmap: legacyData.State().Attributes,
	})
	inputSpecs := upgradedValue.GetAttr(resourceSinkInputSpecsKey)
	iterator := inputSpecs.ElementIterator()
	require.True(t, iterator.Next())
	_, spec := iterator.Element()
	require.True(t, spec.GetAttr(resourceSinkInputSpecsSubsetPoolMessagesKey).RawEquals(cty.True))
	require.True(t, spec.GetAttr(resourceSinkInputSpecsSubsetConsumerPropertiesKey).
		Index(cty.StringVal("application")).RawEquals(cty.StringVal("billing")))
}

func sinkV013ReceiverQueueZeroState(
	t *testing.T,
) (*terraform.InstanceState, map[string]interface{}, string) {
	t.Helper()

	topic := "persistent://public/default/in-1"
	config := sinkLegacyInputConfig(topic)
	inputSpec := sinkV013InputSpec(topic)
	inputSpec[resourceSinkInputSpecsSubsetReceiverQueueSizeKey] = 0
	state := sinkV013StateWithComputedInputSpecs(t, config, []interface{}{inputSpec})

	return state, config, topic
}

func upgradePulsarSinkV0State(t *testing.T, rawState *tfprotov5.RawState) cty.Value {
	t.Helper()

	resourceSchema := resourcePulsarSink()
	server := schema.NewGRPCProviderServer(Provider())
	response, err := server.UpgradeResourceState(context.Background(), &tfprotov5.UpgradeResourceStateRequest{
		TypeName: "pulsar_sink",
		Version:  0,
		RawState: rawState,
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
	return upgradedState
}

func requireSinkInputSpecReceiverQueueSize(
	t *testing.T,
	state cty.Value,
	topic string,
	want int64,
) {
	t.Helper()

	inputSpecs := state.GetAttr(resourceSinkInputSpecsKey)
	require.True(t, inputSpecs.IsKnown())
	require.Equal(t, 1, inputSpecs.LengthInt())
	iterator := inputSpecs.ElementIterator()
	require.True(t, iterator.Next())
	_, spec := iterator.Element()
	require.Equal(t, topic, spec.GetAttr(resourceSinkInputSpecsSubsetTopicKey).AsString())
	require.True(t, spec.GetAttr(resourceSinkInputSpecsSubsetReceiverQueueSizeKey).
		RawEquals(cty.NumberIntVal(want)))
}

func sinkInputSpecsStateHash(t *testing.T, attributes map[string]string) string {
	t.Helper()

	prefix := resourceSinkInputSpecsKey + "."
	suffix := "." + resourceSinkInputSpecsSubsetTopicKey
	for key := range attributes {
		if strings.HasPrefix(key, prefix) && strings.HasSuffix(key, suffix) {
			return strings.TrimSuffix(strings.TrimPrefix(key, prefix), suffix)
		}
	}

	t.Fatalf("input_specs key is missing from %#v", attributes)
	return ""
}

func requireSinkV013NoRefreshDiffEmpty(
	t *testing.T,
	resourceSchema *schema.Resource,
	state *terraform.InstanceState,
	config map[string]interface{},
) {
	t.Helper()

	state.RawConfig = sinkRawConfigWithoutInputSpecs()
	diff, err := resourceSchema.Diff(
		context.Background(), state, terraform.NewResourceConfigRaw(config), nil,
	)
	require.NoError(t, err)
	if diff != nil {
		require.True(t, diff.Empty(), "upgraded v0.13 state re-planned: %#v", diff.Attributes)
	}
}
