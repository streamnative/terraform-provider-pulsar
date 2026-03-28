package pulsar

import (
	"maps"
	"reflect"
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

	got, known := rawValueTopicPropertiesKeys(rawConfig)
	want := map[string]struct{}{
		"k1": {},
		"k2": {},
	}

	if !known {
		t.Fatal("expected topic_properties to be treated as a known source")
	}

	if !maps.Equal(got, want) {
		t.Fatalf("unexpected managed keys: got %#v, want %#v", got, want)
	}
}

func TestRawValueTopicPropertiesKeysUnset(t *testing.T) {
	rawConfig := cty.ObjectVal(map[string]cty.Value{})

	got, known := rawValueTopicPropertiesKeys(rawConfig)
	if known {
		t.Fatal("expected missing topic_properties to be treated as unknown")
	}

	if len(got) != 0 {
		t.Fatalf("expected no managed keys, got %#v", got)
	}
}

func TestRawValueTopicPropertiesKeysNull(t *testing.T) {
	rawConfig := cty.ObjectVal(map[string]cty.Value{
		"topic_properties": cty.NullVal(cty.Map(cty.String)),
	})

	got, known := rawValueTopicPropertiesKeys(rawConfig)
	if !known {
		t.Fatal("expected null topic_properties to be treated as an explicit source")
	}

	if len(got) != 0 {
		t.Fatalf("expected no managed keys, got %#v", got)
	}
}

func TestRawValueTopicPropertiesKeysEmptyMap(t *testing.T) {
	rawConfig := cty.ObjectVal(map[string]cty.Value{
		"topic_properties": cty.MapValEmpty(cty.String),
	})

	got, known := rawValueTopicPropertiesKeys(rawConfig)
	if !known {
		t.Fatal("expected empty topic_properties map to be treated as a known source")
	}

	if len(got) != 0 {
		t.Fatalf("expected no managed keys, got %#v", got)
	}
}

func TestTopicPropertiesManagedKeysFallsBackToRawState(t *testing.T) {
	rawConfig := cty.ObjectVal(map[string]cty.Value{})
	rawState := cty.ObjectVal(map[string]cty.Value{
		"topic_properties": cty.MapVal(map[string]cty.Value{
			"managed": cty.StringVal("v1"),
		}),
	})

	got, known := rawConfigOrStateTopicPropertiesKeys(rawConfig, rawState)
	want := map[string]struct{}{
		"managed": {},
	}

	if !known {
		t.Fatal("expected raw state to provide managed keys")
	}

	if !maps.Equal(got, want) {
		t.Fatalf("unexpected managed keys from raw state: got %#v, want %#v", got, want)
	}
}

func TestTopicPropertiesManagedKeysFallsBackToResourceData(t *testing.T) {
	d := resourcePulsarTopic().TestResourceData()
	d.SetId("persistent://public/default/test-topic")

	if !d.GetRawConfig().IsNull() {
		t.Fatal("expected raw config to be null")
	}

	if err := d.Set("topic_properties", map[string]string{"managed": "v1"}); err != nil {
		t.Fatalf("failed to set topic_properties: %v", err)
	}

	got, known := topicPropertiesManagedKeys(d)
	want := map[string]struct{}{
		"managed": {},
	}

	if !known {
		t.Fatal("expected resource data to provide managed keys")
	}

	if !maps.Equal(got, want) {
		t.Fatalf("unexpected managed keys from resource data: got %#v, want %#v", got, want)
	}
}

func TestTopicPropertiesManagedKeysFallsBackToEmptyResourceData(t *testing.T) {
	d := resourcePulsarTopic().TestResourceData()
	d.SetId("persistent://public/default/test-topic")

	if !d.GetRawConfig().IsNull() {
		t.Fatal("expected raw config to be null")
	}

	if err := d.Set("topic_properties", map[string]string{}); err != nil {
		t.Fatalf("failed to set empty topic_properties: %v", err)
	}

	got, known := topicPropertiesManagedKeys(d)
	if !known {
		t.Fatal("expected empty resource data topic_properties to be treated as known")
	}

	if len(got) != 0 {
		t.Fatalf("expected no managed keys, got %#v", got)
	}
}

func TestTopicPropertiesManagedKeysUnknownWithoutConfigStateOrResourceData(t *testing.T) {
	d := resourcePulsarTopic().TestResourceData()
	d.SetId("persistent://public/default/test-topic")

	got, known := topicPropertiesManagedKeys(d)
	if known {
		t.Fatal("expected managed keys to remain unknown")
	}

	if len(got) != 0 {
		t.Fatalf("expected no managed keys, got %#v", got)
	}
}

func TestRawConfigTopicPropertiesTakesPrecedenceOverRawState(t *testing.T) {
	rawConfig := cty.ObjectVal(map[string]cty.Value{
		"topic_properties": cty.MapValEmpty(cty.String),
	})
	rawState := cty.ObjectVal(map[string]cty.Value{
		"topic_properties": cty.MapVal(map[string]cty.Value{
			"managed": cty.StringVal("v1"),
		}),
	})

	got, known := rawConfigOrStateTopicPropertiesKeys(rawConfig, rawState)
	if !known {
		t.Fatal("expected explicit empty config to take precedence")
	}

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

func TestRawValueTopicPropertiesValues(t *testing.T) {
	rawState := cty.ObjectVal(map[string]cty.Value{
		"topic_properties": cty.MapVal(map[string]cty.Value{
			"managed": cty.StringVal("v1"),
			"index":   cty.StringVal("-1"),
		}),
	})

	got, known := rawValueTopicPropertiesValues(rawState)
	want := map[string]string{
		"managed": "v1",
		"index":   "-1",
	}

	if !known {
		t.Fatal("expected topic_properties values to be treated as known")
	}

	if !maps.Equal(got, want) {
		t.Fatalf("unexpected topic_properties values: got %#v, want %#v", got, want)
	}
}

func TestRawStateHasImportedTopicPropertiesMarker(t *testing.T) {
	rawState := cty.ObjectVal(map[string]cty.Value{
		"topic_properties": cty.MapVal(map[string]cty.Value{
			"managed": cty.StringVal("v1"),
			"index":   cty.StringVal("-1"),
		}),
	})

	if !rawStateHasImportedTopicPropertiesMarker(rawState) {
		t.Fatal("expected import marker to be detected")
	}
}

func TestTopicPropertiesPlannedMapForImportDriftWithoutConfiguredTopicProperties(t *testing.T) {
	rawConfig := cty.ObjectVal(map[string]cty.Value{})
	rawState := cty.ObjectVal(map[string]cty.Value{
		"topic_properties": cty.MapVal(map[string]cty.Value{
			"index": cty.StringVal("-1"),
		}),
	})

	got, changed := topicPropertiesPlannedMapForImportDrift(
		rawConfig,
		rawState,
		map[string]interface{}{"index": "-1"},
		nil,
	)
	if !changed {
		t.Fatal("expected implicit topic_properties drift to be suppressed")
	}

	want := map[string]interface{}{"index": "-1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected planned topic_properties: got %#v, want %#v", got, want)
	}
}

func TestTopicPropertiesPlannedMapForImportDriftForUndeclaredKeys(t *testing.T) {
	rawConfig := cty.ObjectVal(map[string]cty.Value{
		"topic_properties": cty.MapVal(map[string]cty.Value{
			"managed": cty.StringVal("v1"),
		}),
	})
	rawState := cty.ObjectVal(map[string]cty.Value{
		"topic_properties": cty.MapVal(map[string]cty.Value{
			"managed":  cty.StringVal("v1"),
			"external": cty.StringVal("persisted-outside-state"),
			"index":    cty.StringVal("-1"),
		}),
	})

	got, changed := topicPropertiesPlannedMapForImportDrift(
		rawConfig,
		rawState,
		map[string]interface{}{
			"managed":  "v1",
			"external": "persisted-outside-state",
			"index":    "-1",
		},
		map[string]interface{}{
			"managed": "v1",
		},
	)
	if !changed {
		t.Fatal("expected undeclared topic properties drift to be suppressed")
	}

	want := map[string]interface{}{
		"managed":  "v1",
		"external": "persisted-outside-state",
		"index":    "-1",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected planned topic_properties: got %#v, want %#v", got, want)
	}
}

func TestTopicPropertiesPlannedMapForImportDriftDoesNotSuppressExplicitEmptyMap(t *testing.T) {
	rawConfig := cty.ObjectVal(map[string]cty.Value{
		"topic_properties": cty.MapValEmpty(cty.String),
	})
	rawState := cty.ObjectVal(map[string]cty.Value{
		"topic_properties": cty.MapVal(map[string]cty.Value{
			"managed": cty.StringVal("v1"),
			"index":   cty.StringVal("-1"),
		}),
	})

	if _, changed := topicPropertiesPlannedMapForImportDrift(
		rawConfig,
		rawState,
		map[string]interface{}{"managed": "v1", "index": "-1"},
		map[string]interface{}{},
	); changed {
		t.Fatal("expected explicit empty topic_properties to preserve delete semantics")
	}
}

func TestTopicPropertiesPlannedMapForImportDriftDoesNotSuppressCleanState(t *testing.T) {
	rawConfig := cty.ObjectVal(map[string]cty.Value{
		"topic_properties": cty.MapVal(map[string]cty.Value{
			"managed": cty.StringVal("v1"),
		}),
	})
	rawState := cty.ObjectVal(map[string]cty.Value{
		"topic_properties": cty.MapVal(map[string]cty.Value{
			"managed": cty.StringVal("v1"),
			"k2":      cty.StringVal("v2"),
		}),
	})

	if _, changed := topicPropertiesPlannedMapForImportDrift(
		rawConfig,
		rawState,
		map[string]interface{}{"managed": "v1", "k2": "v2"},
		map[string]interface{}{"managed": "v1"},
	); changed {
		t.Fatal("expected clean state deletions to remain visible")
	}
}

func TestRawValueHasTopLevelAttribute(t *testing.T) {
	rawValue := cty.ObjectVal(map[string]cty.Value{
		"replication_clusters": cty.SetVal([]cty.Value{cty.StringVal("standalone")}),
	})

	if !rawValueHasTopLevelAttribute(rawValue, "replication_clusters") {
		t.Fatal("expected replication_clusters to be detected")
	}
}

func TestRawValueHasTopLevelAttributeUnset(t *testing.T) {
	rawValue := cty.ObjectVal(map[string]cty.Value{
		"replication_clusters": cty.NullVal(cty.Set(cty.String)),
	})

	if rawValueHasTopLevelAttribute(rawValue, "replication_clusters") {
		t.Fatal("expected null replication_clusters to be treated as unset")
	}
}

func TestRawValueHasTopLevelAttributeMissing(t *testing.T) {
	rawValue := cty.ObjectVal(map[string]cty.Value{})

	if rawValueHasTopLevelAttribute(rawValue, "replication_clusters") {
		t.Fatal("expected missing replication_clusters to be treated as absent")
	}
}

func TestShouldReadTopicReplicationClustersFromResourceData(t *testing.T) {
	d := resourcePulsarTopic().TestResourceData()
	d.SetId("persistent://public/default/test-topic")

	if err := d.Set("replication_clusters", []string{"standalone"}); err != nil {
		t.Fatalf("failed to set replication_clusters: %v", err)
	}

	if !shouldReadTopicReplicationClusters(d) {
		t.Fatal("expected replication_clusters in resource data to be treated as managed")
	}
}

func TestShouldReadTopicReplicationClustersFromEmptyResourceData(t *testing.T) {
	d := resourcePulsarTopic().TestResourceData()
	d.SetId("persistent://public/default/test-topic")

	if err := d.Set("replication_clusters", []string{}); err != nil {
		t.Fatalf("failed to set empty replication_clusters: %v", err)
	}

	if !shouldReadTopicReplicationClusters(d) {
		t.Fatal("expected empty replication_clusters in resource data to be treated as managed")
	}
}

func TestShouldReadTopicReplicationClustersUnsetWithoutConfigStateOrResourceData(t *testing.T) {
	d := resourcePulsarTopic().TestResourceData()
	d.SetId("persistent://public/default/test-topic")

	if shouldReadTopicReplicationClusters(d) {
		t.Fatal("expected replication_clusters to remain unmanaged without config, state, or resource data")
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
