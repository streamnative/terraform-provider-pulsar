package pulsar

import (
	"testing"

	"github.com/hashicorp/go-cty/cty"
)

func TestRawConfigHasTopicSchemaCompatibilityStrategy(t *testing.T) {
	rawConfig := cty.ObjectVal(map[string]cty.Value{
		"topic_config": cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				"schema_compatibility_strategy": cty.StringVal("Undefined"),
			}),
		}),
	})

	if !rawConfigHasTopicSchemaCompatibilityStrategy(rawConfig) {
		t.Fatal("expected schema compatibility strategy to be detected")
	}
}

func TestRawConfigHasTopicSchemaCompatibilityStrategyTuple(t *testing.T) {
	rawConfig := cty.ObjectVal(map[string]cty.Value{
		"topic_config": cty.TupleVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				"schema_compatibility_strategy": cty.StringVal("Undefined"),
			}),
		}),
	})

	if !rawConfigHasTopicSchemaCompatibilityStrategy(rawConfig) {
		t.Fatal("expected tuple topic_config to be supported")
	}
}

func TestRawConfigHasTopicSchemaCompatibilityStrategyUnset(t *testing.T) {
	rawConfig := cty.ObjectVal(map[string]cty.Value{
		"topic_config": cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				"schema_compatibility_strategy": cty.NullVal(cty.String),
			}),
		}),
	})

	if rawConfigHasTopicSchemaCompatibilityStrategy(rawConfig) {
		t.Fatal("expected missing schema compatibility strategy to remain unset")
	}
}

func TestRawConfigHasTopicSchemaCompatibilityStrategyWithoutBlock(t *testing.T) {
	rawConfig := cty.ObjectVal(map[string]cty.Value{})

	if rawConfigHasTopicSchemaCompatibilityStrategy(rawConfig) {
		t.Fatal("expected missing topic_config block to remain unset")
	}
}

func TestRawConfigHasTopicSchemaCompatibilityStrategyInvalidTopicConfigShape(t *testing.T) {
	rawConfig := cty.ObjectVal(map[string]cty.Value{
		"topic_config": cty.ObjectVal(map[string]cty.Value{
			"schema_compatibility_strategy": cty.StringVal("Undefined"),
		}),
	})

	if rawConfigHasTopicSchemaCompatibilityStrategy(rawConfig) {
		t.Fatal("expected invalid topic_config shape to be ignored")
	}
}

func TestTopicConfigHasSchemaCompatibilityStrategyFallsBackToRawState(t *testing.T) {
	rawConfig := cty.NullVal(cty.EmptyObject)
	rawState := cty.ObjectVal(map[string]cty.Value{
		"topic_config": cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				"schema_compatibility_strategy": cty.StringVal("Undefined"),
			}),
		}),
	})

	if !rawConfigOrStateHasTopicSchemaCompatibilityStrategy(rawConfig, rawState) {
		t.Fatal("expected null raw config to fall back to raw state")
	}
}

func TestRawConfigTakesPrecedenceOverRawStateForSchemaCompatibilityStrategy(t *testing.T) {
	rawConfig := cty.ObjectVal(map[string]cty.Value{
		"topic_config": cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				"schema_compatibility_strategy": cty.NullVal(cty.String),
			}),
		}),
	})
	rawState := cty.ObjectVal(map[string]cty.Value{
		"topic_config": cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				"schema_compatibility_strategy": cty.StringVal("Undefined"),
			}),
		}),
	})

	if rawConfigOrStateHasTopicSchemaCompatibilityStrategy(rawConfig, rawState) {
		t.Fatal("expected raw config to take precedence over raw state")
	}
}
