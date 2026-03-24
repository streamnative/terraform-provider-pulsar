package pulsar

import (
	"testing"

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
	"github.com/hashicorp/go-cty/cty"
)

func TestRawValueHasNamespaceConfigStringField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		rawValue cty.Value
		field    string
		want     bool
	}{
		{
			name: "schema compatibility strategy set",
			rawValue: namespaceConfigRawValue(map[string]cty.Value{
				"schema_compatibility_strategy": cty.StringVal("Undefined"),
			}),
			field: "schema_compatibility_strategy",
			want:  true,
		},
		{
			name: "schema auto update compatibility strategy set",
			rawValue: namespaceConfigRawValue(map[string]cty.Value{
				"schema_auto_update_compatibility_strategy": cty.StringVal("Full"),
			}),
			field: "schema_auto_update_compatibility_strategy",
			want:  true,
		},
		{
			name:     "field missing from empty block",
			rawValue: namespaceConfigRawValue(map[string]cty.Value{}),
			field:    "schema_compatibility_strategy",
			want:     false,
		},
		{
			name: "null field is ignored",
			rawValue: namespaceConfigRawValue(map[string]cty.Value{
				"schema_auto_update_compatibility_strategy": cty.NullVal(cty.String),
			}),
			field: "schema_auto_update_compatibility_strategy",
			want:  false,
		},
		{
			name: "invalid namespace config shape is ignored",
			rawValue: cty.ObjectVal(map[string]cty.Value{
				"namespace_config": cty.ObjectVal(map[string]cty.Value{
					"schema_compatibility_strategy": cty.StringVal("Undefined"),
				}),
			}),
			field: "schema_compatibility_strategy",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := rawValueHasNamespaceConfigStringField(tt.rawValue, tt.field)
			if got != tt.want {
				t.Fatalf("unexpected result: got %t, want %t", got, tt.want)
			}
		})
	}
}

func TestRawConfigOrStateHasNamespaceConfigStringField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		rawConfig cty.Value
		rawState  cty.Value
		field     string
		want      bool
	}{
		{
			name:      "falls back to raw state when config is null",
			rawConfig: cty.NullVal(cty.EmptyObject),
			rawState: namespaceConfigRawValue(map[string]cty.Value{
				"schema_compatibility_strategy": cty.StringVal("Undefined"),
			}),
			field: "schema_compatibility_strategy",
			want:  true,
		},
		{
			name:      "unknown config does not fall back to raw state",
			rawConfig: cty.UnknownVal(cty.EmptyObject),
			rawState: namespaceConfigRawValue(map[string]cty.Value{
				"schema_compatibility_strategy": cty.StringVal("Undefined"),
			}),
			field: "schema_compatibility_strategy",
			want:  false,
		},
		{
			name: "raw config takes precedence over raw state",
			rawConfig: namespaceConfigRawValue(map[string]cty.Value{
				"schema_compatibility_strategy": cty.NullVal(cty.String),
			}),
			rawState: namespaceConfigRawValue(map[string]cty.Value{
				"schema_compatibility_strategy": cty.StringVal("Undefined"),
			}),
			field: "schema_compatibility_strategy",
			want:  false,
		},
		{
			name: "missing field in config suppresses previous auto update state",
			rawConfig: namespaceConfigRawValue(map[string]cty.Value{
				"max_producers_per_topic": cty.NumberIntVal(50),
			}),
			rawState: namespaceConfigRawValue(map[string]cty.Value{
				"schema_auto_update_compatibility_strategy": cty.StringVal("Full"),
			}),
			field: "schema_auto_update_compatibility_strategy",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := rawConfigOrStateHasNamespaceConfigStringField(tt.rawConfig, tt.rawState, tt.field)
			if got != tt.want {
				t.Fatalf("unexpected result: got %t, want %t", got, tt.want)
			}
		})
	}
}

func TestNamespaceSchemaCompatibilityStrategyStateValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		strategy         utils.SchemaCompatibilityStrategy
		hasExplicitValue bool
		wantValue        string
		wantOK           bool
	}{
		{
			name:             "undefined without explicit value",
			strategy:         utils.SchemaCompatibilityStrategyUndefined,
			hasExplicitValue: false,
			wantValue:        "",
			wantOK:           false,
		},
		{
			name:             "undefined with explicit value",
			strategy:         utils.SchemaCompatibilityStrategyUndefined,
			hasExplicitValue: true,
			wantValue:        "Undefined",
			wantOK:           true,
		},
		{
			name:             "empty strategy with explicit value",
			strategy:         "",
			hasExplicitValue: true,
			wantValue:        "Undefined",
			wantOK:           true,
		},
		{
			name:             "backward transitive strategy",
			strategy:         utils.SchemaCompatibilityStrategyBackwardTransitive,
			hasExplicitValue: true,
			wantValue:        "BackwardTransitive",
			wantOK:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotValue, gotOK := namespaceSchemaCompatibilityStrategyStateValue(tt.strategy, tt.hasExplicitValue)
			if gotValue != tt.wantValue || gotOK != tt.wantOK {
				t.Fatalf(
					"unexpected state value: got (%q, %t), want (%q, %t)",
					gotValue,
					gotOK,
					tt.wantValue,
					tt.wantOK,
				)
			}
		})
	}
}

func TestNamespaceSchemaAutoUpdateCompatibilityStrategyStateValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		strategy         utils.SchemaAutoUpdateCompatibilityStrategy
		hasExplicitValue bool
		wantValue        string
		wantOK           bool
	}{
		{
			name:             "unset without explicit value",
			strategy:         utils.Full,
			hasExplicitValue: false,
			wantValue:        "",
			wantOK:           false,
		},
		{
			name:             "empty strategy with explicit value",
			strategy:         "",
			hasExplicitValue: true,
			wantValue:        "",
			wantOK:           false,
		},
		{
			name:             "explicit full strategy",
			strategy:         utils.Full,
			hasExplicitValue: true,
			wantValue:        "Full",
			wantOK:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotValue, gotOK := namespaceSchemaAutoUpdateCompatibilityStrategyStateValue(
				tt.strategy,
				tt.hasExplicitValue,
			)
			if gotValue != tt.wantValue || gotOK != tt.wantOK {
				t.Fatalf(
					"unexpected state value: got (%q, %t), want (%q, %t)",
					gotValue,
					gotOK,
					tt.wantValue,
					tt.wantOK,
				)
			}
		})
	}
}

func namespaceConfigRawValue(fields map[string]cty.Value) cty.Value {
	return cty.ObjectVal(map[string]cty.Value{
		"namespace_config": cty.ListVal([]cty.Value{
			cty.ObjectVal(fields),
		}),
	})
}
