package pulsar

import (
	"errors"
	"testing"

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/rest"
	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
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
