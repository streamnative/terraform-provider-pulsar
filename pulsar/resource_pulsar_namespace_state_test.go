package pulsar

import (
	"testing"

	"github.com/hashicorp/go-cty/cty"
)

func TestRawValueHasNamespaceConfigStringField(t *testing.T) {
	rawConfig := cty.ObjectVal(map[string]cty.Value{
		"namespace_config": cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				"schema_compatibility_strategy": cty.StringVal("Undefined"),
			}),
		}),
	})

	if !rawValueHasNamespaceConfigStringField(rawConfig, "schema_compatibility_strategy") {
		t.Fatal("expected namespace schema compatibility strategy to be detected")
	}
}

func TestRawValueHasNamespaceConfigStringFieldForSchemaAutoUpdate(t *testing.T) {
	rawConfig := cty.ObjectVal(map[string]cty.Value{
		"namespace_config": cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				"schema_auto_update_compatibility_strategy": cty.StringVal("Forward"),
			}),
		}),
	})

	if !rawValueHasNamespaceConfigStringField(rawConfig, "schema_auto_update_compatibility_strategy") {
		t.Fatal("expected namespace schema auto update compatibility strategy to be detected")
	}
}

func TestRawValueHasNamespaceConfigStringFieldTuple(t *testing.T) {
	rawConfig := cty.ObjectVal(map[string]cty.Value{
		"namespace_config": cty.TupleVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				"schema_compatibility_strategy": cty.StringVal("Undefined"),
			}),
		}),
	})

	if !rawValueHasNamespaceConfigStringField(rawConfig, "schema_compatibility_strategy") {
		t.Fatal("expected tuple namespace_config to be supported")
	}
}

func TestRawValueHasNamespaceConfigStringFieldUnset(t *testing.T) {
	rawConfig := cty.ObjectVal(map[string]cty.Value{
		"namespace_config": cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				"schema_compatibility_strategy": cty.NullVal(cty.String),
			}),
		}),
	})

	if rawValueHasNamespaceConfigStringField(rawConfig, "schema_compatibility_strategy") {
		t.Fatal("expected missing namespace schema compatibility strategy to remain unset")
	}
}

func TestRawValueHasNamespaceConfigStringFieldWithoutBlock(t *testing.T) {
	rawConfig := cty.ObjectVal(map[string]cty.Value{})

	if rawValueHasNamespaceConfigStringField(rawConfig, "schema_compatibility_strategy") {
		t.Fatal("expected missing namespace_config block to remain unset")
	}
}

func TestRawConfigOrStateHasNamespaceConfigStringFieldFallsBackToRawState(t *testing.T) {
	rawConfig := cty.NullVal(cty.EmptyObject)
	rawState := cty.ObjectVal(map[string]cty.Value{
		"namespace_config": cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				"schema_compatibility_strategy": cty.StringVal("Undefined"),
			}),
		}),
	})

	if !rawConfigOrStateHasNamespaceConfigStringField(rawConfig, rawState, "schema_compatibility_strategy") {
		t.Fatal("expected null raw config to fall back to raw state")
	}
}

func TestRawConfigTakesPrecedenceOverRawStateForNamespaceConfigStringField(t *testing.T) {
	rawConfig := cty.ObjectVal(map[string]cty.Value{
		"namespace_config": cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				"schema_compatibility_strategy": cty.NullVal(cty.String),
			}),
		}),
	})
	rawState := cty.ObjectVal(map[string]cty.Value{
		"namespace_config": cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				"schema_compatibility_strategy": cty.StringVal("Undefined"),
			}),
		}),
	})

	if rawConfigOrStateHasNamespaceConfigStringField(rawConfig, rawState, "schema_compatibility_strategy") {
		t.Fatal("expected raw config to take precedence over raw state")
	}
}
