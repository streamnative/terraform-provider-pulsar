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
	"testing"

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
	"github.com/stretchr/testify/require"
)

func TestParseSchemaCompatibilityStrategyVariants(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected utils.SchemaCompatibilityStrategy
	}{
		{"UndefinedUpper", "UNDEFINED", utils.SchemaCompatibilityStrategyUndefined},
		{"UndefinedCamel", "Undefined", utils.SchemaCompatibilityStrategyUndefined},
		{"UndefinedLower", "undefined", utils.SchemaCompatibilityStrategyUndefined},
		{"AlwaysIncompatibleUpper", "ALWAYS_INCOMPATIBLE", utils.SchemaCompatibilityStrategyAlwaysIncompatible},
		{"AlwaysIncompatibleCamel", "AlwaysIncompatible", utils.SchemaCompatibilityStrategyAlwaysIncompatible},
		{"AlwaysIncompatibleSnakeLower", "always_incompatible", utils.SchemaCompatibilityStrategyAlwaysIncompatible},
		{"AlwaysCompatibleUpper", "ALWAYS_COMPATIBLE", utils.SchemaCompatibilityStrategyAlwaysCompatible},
		{"AlwaysCompatibleCamel", "AlwaysCompatible", utils.SchemaCompatibilityStrategyAlwaysCompatible},
		{"AlwaysCompatibleSnakeLower", "always_compatible", utils.SchemaCompatibilityStrategyAlwaysCompatible},
		{"BackwardUpper", "BACKWARD", utils.SchemaCompatibilityStrategyBackward},
		{"BackwardCamel", "Backward", utils.SchemaCompatibilityStrategyBackward},
		{"BackwardLower", "backward", utils.SchemaCompatibilityStrategyBackward},
		{"ForwardUpper", "FORWARD", utils.SchemaCompatibilityStrategyForward},
		{"ForwardCamel", "Forward", utils.SchemaCompatibilityStrategyForward},
		{"ForwardLower", "forward", utils.SchemaCompatibilityStrategyForward},
		{"FullUpper", "FULL", utils.SchemaCompatibilityStrategyFull},
		{"FullCamel", "Full", utils.SchemaCompatibilityStrategyFull},
		{"FullLower", "full", utils.SchemaCompatibilityStrategyFull},
		{"BackwardTransitiveUpper", "BACKWARD_TRANSITIVE", utils.SchemaCompatibilityStrategyBackwardTransitive},
		{"BackwardTransitiveCamel", "BackwardTransitive", utils.SchemaCompatibilityStrategyBackwardTransitive},
		{"BackwardTransitiveSnakeLower", "backward_transitive", utils.SchemaCompatibilityStrategyBackwardTransitive},
		{"ForwardTransitiveUpper", "FORWARD_TRANSITIVE", utils.SchemaCompatibilityStrategyForwardTransitive},
		{"ForwardTransitiveCamel", "ForwardTransitive", utils.SchemaCompatibilityStrategyForwardTransitive},
		{"ForwardTransitiveSnakeLower", "forward_transitive", utils.SchemaCompatibilityStrategyForwardTransitive},
		{"FullTransitiveUpper", "FULL_TRANSITIVE", utils.SchemaCompatibilityStrategyFullTransitive},
		{"FullTransitiveCamel", "FullTransitive", utils.SchemaCompatibilityStrategyFullTransitive},
		{"FullTransitiveSnakeLower", "full_transitive", utils.SchemaCompatibilityStrategyFullTransitive},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseSchemaCompatibilityStrategy(tc.input)
			require.NoError(t, err)
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestParseSchemaCompatibilityStrategyInvalid(t *testing.T) {
	t.Parallel()

	_, err := parseSchemaCompatibilityStrategy("not-a-valid-strategy")
	require.Error(t, err)
}

func TestSchemaCompatibilityStrategyToTerraformValue(t *testing.T) {
	t.Parallel()

	testCases := map[utils.SchemaCompatibilityStrategy]string{
		utils.SchemaCompatibilityStrategyUndefined:          "Undefined",
		utils.SchemaCompatibilityStrategyAlwaysIncompatible: "AlwaysIncompatible",
		utils.SchemaCompatibilityStrategyAlwaysCompatible:   "AlwaysCompatible",
		utils.SchemaCompatibilityStrategyBackward:           "Backward",
		utils.SchemaCompatibilityStrategyForward:            "Forward",
		utils.SchemaCompatibilityStrategyFull:               "Full",
		utils.SchemaCompatibilityStrategyBackwardTransitive: "BackwardTransitive",
		utils.SchemaCompatibilityStrategyForwardTransitive:  "ForwardTransitive",
		utils.SchemaCompatibilityStrategyFullTransitive:     "FullTransitive",
		"": "",
	}

	for strategy, expected := range testCases {
		strategy := strategy
		expected := expected
		t.Run(expected, func(t *testing.T) {
			t.Parallel()
			actual := schemaCompatibilityStrategyToTerraformValue(strategy)
			require.Equal(t, expected, actual)
		})
	}
}
