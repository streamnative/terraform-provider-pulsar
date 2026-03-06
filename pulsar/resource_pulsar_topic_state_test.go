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
