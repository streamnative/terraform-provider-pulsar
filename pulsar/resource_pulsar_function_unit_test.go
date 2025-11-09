package pulsar

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeFunctionCustomRuntimeOptions(t *testing.T) {
	base := `{"foo":"bar","sinkConfig":{"old":"value"},"sourceConfig":{"keep":"me"}}`
	sinkConfig := map[string]interface{}{runtimeOptionSinkConfigTypeField: "kafka", runtimeOptionConfigsKey: map[string]interface{}{"new": "value"}}
	sourceConfig := map[string]interface{}{}

	merged, err := mergeFunctionCustomRuntimeOptions(base,
		runtimeConfigUpdate{key: runtimeOptionSinkConfigKey, config: sinkConfig},
		runtimeConfigUpdate{key: runtimeOptionSourceConfigKey, config: sourceConfig},
	)
	require.NoError(t, err)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(merged), &result))

	assert.Equal(t, "bar", result["foo"])
	assert.Equal(t, map[string]interface{}{runtimeOptionSinkConfigTypeField: "kafka", runtimeOptionConfigsKey: map[string]interface{}{"new": "value"}}, result["sinkConfig"])
	_, hasSource := result["sourceConfig"]
	assert.False(t, hasSource)
}

func TestSplitFunctionCustomRuntimeOptions(t *testing.T) {
	raw := `{"foo":"bar","sinkConfig":{"sinkType":"kafka","configs":{"alpha":"1","beta":true}},"sourceConfig":{"sourceType":"kinesis","configs":{"gamma":2}}}`

	sanitized, sinkConfig, sinkPresent, sourceConfig, sourcePresent, err := splitFunctionCustomRuntimeOptions(raw)
	require.NoError(t, err)
	assert.True(t, sinkPresent)
	assert.Equal(t, map[string]interface{}{
		runtimeOptionSinkConfigTypeField: "kafka",
		runtimeOptionConfigsKey:          map[string]interface{}{"alpha": "1", "beta": "true"},
	}, sinkConfig)
	assert.True(t, sourcePresent)
	assert.Equal(t, map[string]interface{}{
		runtimeOptionSourceConfigTypeField: "kinesis",
		runtimeOptionConfigsKey:            map[string]interface{}{"gamma": "2"},
	}, sourceConfig)
	assert.JSONEq(t, `{"foo":"bar"}`, sanitized)
}

func TestSplitFunctionCustomRuntimeOptionsWithoutSinkConfig(t *testing.T) {
	raw := `{"foo":"bar"}`

	sanitized, sinkConfig, sinkPresent, sourceConfig, sourcePresent, err := splitFunctionCustomRuntimeOptions(raw)
	require.NoError(t, err)
	assert.False(t, sinkPresent)
	assert.Nil(t, sinkConfig)
	assert.False(t, sourcePresent)
	assert.Nil(t, sourceConfig)
	assert.JSONEq(t, `{"foo":"bar"}`, sanitized)
}
