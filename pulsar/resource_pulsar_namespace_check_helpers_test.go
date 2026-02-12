package pulsar

import (
	"errors"
	"fmt"
	"testing"

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/rest"
	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"
)

func TestIsInactiveTopicPoliciesUnset(t *testing.T) {
	t.Parallel()

	deleteMode := utils.DeleteWhenNoSubscriptions

	testCases := []struct {
		name     string
		policies utils.InactiveTopicPolicies
		expected bool
	}{
		{
			name:     "zero_value_struct",
			policies: utils.InactiveTopicPolicies{},
			expected: true,
		},
		{
			name: "mode_only",
			policies: utils.InactiveTopicPolicies{
				InactiveTopicDeleteMode: &deleteMode,
			},
			expected: false,
		},
		{
			name: "duration_only",
			policies: utils.InactiveTopicPolicies{
				MaxInactiveDurationSeconds: 60,
			},
			expected: false,
		},
		{
			name: "delete_while_inactive_only",
			policies: utils.InactiveTopicPolicies{
				DeleteWhileInactive: true,
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.expected, isInactiveTopicPoliciesUnset(tc.policies))
		})
	}
}

func TestIsNotFoundError(t *testing.T) {
	t.Parallel()

	require.True(t, isNotFoundError(rest.Error{Code: 404, Reason: "Not Found"}))
	require.False(t, isNotFoundError(rest.Error{Code: 500, Reason: "Internal Server Error"}))
	require.True(t, isNotFoundError(errors.New("code: 404 reason: Not Found")))
	require.False(t, isNotFoundError(errors.New("connection reset by peer")))
}

func TestHasInactiveTopicPoliciesConfigured(t *testing.T) {
	t.Parallel()

	require.False(t, hasInactiveTopicPoliciesConfigured(nil))
	require.False(t, hasInactiveTopicPoliciesConfigured(schema.NewSet(schema.HashString, []interface{}{})))
	require.True(t, hasInactiveTopicPoliciesConfigured(schema.NewSet(schema.HashString, []interface{}{"configured"})))
}

func TestIsIgnorableNotFoundError(t *testing.T) {
	t.Parallel()

	require.True(t, isIgnorableNotFoundError(rest.Error{Code: 404, Reason: "Not Found"}))
	require.True(t, isIgnorableNotFoundError(fmt.Errorf("wrapped: %w", rest.Error{Code: 404, Reason: "Not Found"})))
	require.False(t, isIgnorableNotFoundError(rest.Error{Code: 500, Reason: "Internal Server Error"}))
	require.True(t, isIgnorableNotFoundError(errors.New("code: 404 reason: Not Found")))
	require.True(t, isIgnorableNotFoundError(errors.New("resource not found")))
	require.False(t, isIgnorableNotFoundError(errors.New("connection reset by peer")))
}
