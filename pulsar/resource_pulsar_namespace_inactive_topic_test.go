package pulsar

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalInactiveTopicPolicies(t *testing.T) {
	t.Parallel()

	t.Run("valid_duration", func(t *testing.T) {
		t.Parallel()

		config := schema.NewSet(inactiveTopicPoliciesToHash, []interface{}{
			map[string]interface{}{
				"enable_delete_while_inactive": true,
				"max_inactive_duration":        "60s",
				"delete_mode":                  "delete_when_no_subscriptions",
			},
		})

		policies, err := unmarshalInactiveTopicPolicies(config)
		require.NoError(t, err)
		require.NotNil(t, policies)
		require.Equal(t, 60, policies.MaxInactiveDurationSeconds)
		require.True(t, policies.DeleteWhileInactive)
		require.NotNil(t, policies.InactiveTopicDeleteMode)
		require.Equal(t, "delete_when_no_subscriptions", policies.InactiveTopicDeleteMode.String())
	})

	t.Run("invalid_duration", func(t *testing.T) {
		t.Parallel()

		config := schema.NewSet(inactiveTopicPoliciesToHash, []interface{}{
			map[string]interface{}{
				"enable_delete_while_inactive": true,
				"max_inactive_duration":        "60m",
				"delete_mode":                  "delete_when_no_subscriptions",
			},
		})

		policies, err := unmarshalInactiveTopicPolicies(config)
		require.Error(t, err)
		require.Nil(t, policies)
		require.Contains(t, err.Error(), "invalid max_inactive_duration")
	})
}
