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

func TestIsDispatchRateConfigured(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		rate     utils.DispatchRate
		expected bool
	}{
		{
			name:     "zero_value_is_unset",
			rate:     utils.DispatchRate{},
			expected: false,
		},
		{
			name: "explicit_values",
			rate: utils.DispatchRate{
				DispatchThrottlingRateInMsg:  50,
				DispatchThrottlingRateInByte: 2048,
				RatePeriodInSecond:           50,
			},
			expected: true,
		},
		{
			name: "explicit_unlimited_still_configured",
			// A real configured "unlimited" dispatch rate is {-1,-1,1}; period is always >= 1.
			rate: utils.DispatchRate{
				DispatchThrottlingRateInMsg:  -1,
				DispatchThrottlingRateInByte: -1,
				RatePeriodInSecond:           1,
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.expected, isDispatchRateConfigured(tc.rate))
		})
	}
}

func TestIsRetentionConfigured(t *testing.T) {
	t.Parallel()

	require.False(t, isRetentionConfigured(nil))
	require.False(t, isRetentionConfigured(&utils.RetentionPolicies{}))
	require.True(t, isRetentionConfigured(&utils.RetentionPolicies{RetentionTimeInMinutes: 60}))
	require.True(t, isRetentionConfigured(&utils.RetentionPolicies{RetentionSizeInMB: 100}))
}

func TestIsPersistenceConfigured(t *testing.T) {
	t.Parallel()

	require.False(t, isPersistenceConfigured(nil))
	// An all-zero struct can only be a null/default sentinel: a real ensemble size is always >= 1.
	require.False(t, isPersistenceConfigured(&utils.PersistencePolicies{}))
	require.True(t, isPersistenceConfigured(&utils.PersistencePolicies{BookkeeperEnsemble: 2}))
}

// TestSetPermissionGrantAdoptAll locks the two-mode behavior the import fix relies on:
// adoptAll=true (import) hydrates ALL server grants even from an empty prior state, while
// adoptAll=false (refresh) preserves the existing managed-role filter that protects
// externally-managed grants.
func TestSetPermissionGrantAdoptAll(t *testing.T) {
	t.Parallel()

	produce, err := utils.ParseAuthAction("produce")
	require.NoError(t, err)
	consume, err := utils.ParseAuthAction("consume")
	require.NoError(t, err)

	grants := map[string][]utils.AuthAction{
		"role-a": {produce},
		"role-b": {consume},
	}

	// Import path: empty prior config, adoptAll=true -> all roles adopted.
	dImport := resourcePulsarNamespace().TestResourceData()
	setPermissionGrantFiltered(dImport, grants, true)
	require.Equal(t, 2, dImport.Get("permission_grant").(*schema.Set).Len())

	// Refresh path: empty prior config, adoptAll=false -> nothing adopted (external grants left alone).
	dRefresh := resourcePulsarNamespace().TestResourceData()
	setPermissionGrantFiltered(dRefresh, grants, false)
	require.Equal(t, 0, dRefresh.Get("permission_grant").(*schema.Set).Len())
}

func TestPolicyNullableIntToStateValue(t *testing.T) {
	t.Parallel()

	zero := 0
	positive := 123

	testCases := []struct {
		name     string
		input    *int
		expected int
	}{
		{
			name:     "nil_means_unset",
			input:    nil,
			expected: -1,
		},
		{
			name:     "zero_is_explicit_value",
			input:    &zero,
			expected: 0,
		},
		{
			name:     "positive_value",
			input:    &positive,
			expected: 123,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.expected, policyNullableIntToStateValue(tc.input))
		})
	}
}
