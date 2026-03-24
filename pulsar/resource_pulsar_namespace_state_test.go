package pulsar

import (
	"testing"

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

func namespaceConfigRawValue(fields map[string]cty.Value) cty.Value {
	return cty.ObjectVal(map[string]cty.Value{
		"namespace_config": cty.ListVal([]cty.Value{
			cty.ObjectVal(fields),
		}),
	})
}
