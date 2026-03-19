package pulsar

import (
	"maps"
	"testing"

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
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

func TestRawValueTopicPropertiesKeys(t *testing.T) {
	rawConfig := cty.ObjectVal(map[string]cty.Value{
		"topic_properties": cty.ObjectVal(map[string]cty.Value{
			"k1": cty.StringVal("v1"),
			"k2": cty.StringVal("v2"),
		}),
	})

	got := rawValueTopicPropertiesKeys(rawConfig)
	want := map[string]struct{}{
		"k1": {},
		"k2": {},
	}

	if !maps.Equal(got, want) {
		t.Fatalf("unexpected managed keys: got %#v, want %#v", got, want)
	}
}

func TestRawValueTopicPropertiesKeysUnset(t *testing.T) {
	rawConfig := cty.ObjectVal(map[string]cty.Value{})

	got := rawValueTopicPropertiesKeys(rawConfig)
	if len(got) != 0 {
		t.Fatalf("expected no managed keys, got %#v", got)
	}
}

func TestRawValueTopicPropertiesKeysNull(t *testing.T) {
	rawConfig := cty.ObjectVal(map[string]cty.Value{
		"topic_properties": cty.NullVal(cty.Map(cty.String)),
	})

	got := rawValueTopicPropertiesKeys(rawConfig)
	if len(got) != 0 {
		t.Fatalf("expected no managed keys, got %#v", got)
	}
}

func TestTopicPropertiesManagedKeysFallsBackToRawState(t *testing.T) {
	rawConfig := cty.NullVal(cty.EmptyObject)
	rawState := cty.ObjectVal(map[string]cty.Value{
		"topic_properties": cty.MapVal(map[string]cty.Value{
			"managed": cty.StringVal("v1"),
		}),
	})

	got := rawConfigOrStateTopicPropertiesKeys(rawConfig, rawState)
	want := map[string]struct{}{
		"managed": {},
	}

	if !maps.Equal(got, want) {
		t.Fatalf("unexpected managed keys from raw state: got %#v, want %#v", got, want)
	}
}

func TestTopicPropertiesManagedKeysFallsBackToResourceData(t *testing.T) {
	d := resourcePulsarTopic().TestResourceData()

	if !d.GetRawConfig().IsNull() {
		t.Fatal("expected raw config to be null")
	}

	if err := d.Set("topic_properties", map[string]string{"managed": "v1"}); err != nil {
		t.Fatalf("failed to set topic_properties: %v", err)
	}

	got := topicPropertiesManagedKeys(d)
	want := map[string]struct{}{
		"managed": {},
	}

	if !maps.Equal(got, want) {
		t.Fatalf("unexpected managed keys from resource data: got %#v, want %#v", got, want)
	}
}

func TestRawConfigTopicPropertiesTakesPrecedenceOverRawState(t *testing.T) {
	rawConfig := cty.ObjectVal(map[string]cty.Value{})
	rawState := cty.ObjectVal(map[string]cty.Value{
		"topic_properties": cty.MapVal(map[string]cty.Value{
			"managed": cty.StringVal("v1"),
		}),
	})

	got := rawConfigOrStateTopicPropertiesKeys(rawConfig, rawState)
	if len(got) != 0 {
		t.Fatalf("expected raw config to take precedence, got %#v", got)
	}
}

func TestIgnoreServerSetTopicProperties(t *testing.T) {
	properties := map[string]string{
		"managed":        "v1",
		"kafkaTopicUUID": "uuid-like-value",
		"external_key":   "external",
	}
	managedKeys := map[string]struct{}{
		"managed": {},
	}

	got := ignoreServerSetTopicProperties(managedKeys, properties)
	want := map[string]string{
		"managed": "v1",
	}

	if !maps.Equal(got, want) {
		t.Fatalf("unexpected filtered properties: got %#v, want %#v", got, want)
	}
}

func TestTopicSchemaCompatibilityStrategyStateValue(t *testing.T) {
	tests := []struct {
		name             string
		strategy         utils.SchemaCompatibilityStrategy
		hasExplicitValue bool
		wantValue        string
		wantOK           bool
	}{
		{
			name:             "UndefinedWithoutExplicitValue",
			strategy:         utils.SchemaCompatibilityStrategyUndefined,
			hasExplicitValue: false,
			wantValue:        "",
			wantOK:           false,
		},
		{
			name:             "UndefinedWithExplicitValue",
			strategy:         utils.SchemaCompatibilityStrategyUndefined,
			hasExplicitValue: true,
			wantValue:        "Undefined",
			wantOK:           true,
		},
		{
			name:             "Backward",
			strategy:         utils.SchemaCompatibilityStrategyBackward,
			hasExplicitValue: false,
			wantValue:        "Backward",
			wantOK:           true,
		},
		{
			name:             "Full",
			strategy:         utils.SchemaCompatibilityStrategyFull,
			hasExplicitValue: false,
			wantValue:        "Full",
			wantOK:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValue, gotOK := topicSchemaCompatibilityStrategyStateValue(tt.strategy, tt.hasExplicitValue)
			if gotValue != tt.wantValue || gotOK != tt.wantOK {
				t.Fatalf(
					"unexpected state value for %s: got (%q, %t), want (%q, %t)",
					tt.strategy,
					gotValue,
					gotOK,
					tt.wantValue,
					tt.wantOK,
				)
			}
		})
	}
}
