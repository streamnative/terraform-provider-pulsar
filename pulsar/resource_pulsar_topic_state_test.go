package pulsar

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestTopicConfigHasSchemaCompatibilityStrategy(t *testing.T) {
	resource := resourcePulsarTopic()
	data := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{
		"tenant":     "public",
		"namespace":  "default",
		"topic_type": "persistent",
		"topic_name": "test-topic",
		"partitions": 0,
	})

	if err := data.Set("topic_config", []interface{}{
		map[string]interface{}{
			"schema_compatibility_strategy": "Undefined",
		},
	}); err != nil {
		t.Fatalf("failed to set topic_config: %v", err)
	}

	if !topicConfigHasSchemaCompatibilityStrategy(data) {
		t.Fatal("expected schema compatibility strategy to be detected")
	}
}

func TestTopicConfigHasSchemaCompatibilityStrategyUnset(t *testing.T) {
	resource := resourcePulsarTopic()
	data := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{
		"tenant":     "public",
		"namespace":  "default",
		"topic_type": "persistent",
		"topic_name": "test-topic",
		"partitions": 0,
	})

	if err := data.Set("topic_config", []interface{}{
		map[string]interface{}{},
	}); err != nil {
		t.Fatalf("failed to set topic_config: %v", err)
	}

	if topicConfigHasSchemaCompatibilityStrategy(data) {
		t.Fatal("expected missing schema compatibility strategy to remain unset")
	}
}
