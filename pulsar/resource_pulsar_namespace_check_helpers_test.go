package pulsar

import (
	"errors"
	"fmt"
	"testing"

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/rest"
	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
	"github.com/hashicorp/go-cty/cty"
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
			// A real configured "unlimited" dispatch rate is {-1,-1,1}; the period is always >= 1.
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

func TestIsPersistenceConfigured(t *testing.T) {
	t.Parallel()

	require.False(t, isPersistenceConfigured(nil))
	// An all-zero struct can only be a null/default sentinel: a real ensemble size is always >= 1.
	require.False(t, isPersistenceConfigured(&utils.PersistencePolicies{}))
	require.True(t, isPersistenceConfigured(&utils.PersistencePolicies{BookkeeperEnsemble: 2}))
}

// TestSetBacklogQuotaFiltered locks the non-authoritative refresh semantics of backlog_quota:
// an empty prior state (import) adopts every quota type the server reports, while a non-empty prior
// state only refreshes the types Terraform already tracks, leaving out-of-band types alone.
func TestSetBacklogQuotaFiltered(t *testing.T) {
	t.Parallel()

	quotas := map[utils.BacklogQuotaType]utils.BacklogQuota{
		utils.DestinationStorage: {LimitSize: 10000, LimitTime: -1, Policy: utils.ProducerRequestHold},
		utils.MessageAge:         {LimitSize: -1, LimitTime: 3600, Policy: utils.ConsumerBacklogEviction},
	}

	// Import path: empty prior state -> adopt every type.
	dImport := resourcePulsarNamespace().TestResourceData()
	require.NoError(t, setBacklogQuotaFiltered(dImport, quotas))
	require.Equal(t, 2, dImport.Get("backlog_quota").(*schema.Set).Len())

	// Refresh path: only the tracked type is kept, the out-of-band one is ignored.
	dRefresh := resourcePulsarNamespace().TestResourceData()
	require.NoError(t, dRefresh.Set("backlog_quota", schema.NewSet(hashBacklogQuotaSubset(), []interface{}{
		map[string]interface{}{
			"limit_bytes":   "1",
			"limit_seconds": "-1",
			"policy":        string(utils.ProducerRequestHold),
			"type":          string(utils.DestinationStorage),
		},
	})))
	require.NoError(t, setBacklogQuotaFiltered(dRefresh, quotas))

	kept := dRefresh.Get("backlog_quota").(*schema.Set).List()
	require.Len(t, kept, 1)
	require.Equal(t, string(utils.DestinationStorage), kept[0].(map[string]interface{})["type"])
	require.Equal(t, "10000", kept[0].(map[string]interface{})["limit_bytes"])
}

func TestRemovedBacklogQuotaTypes(t *testing.T) {
	t.Parallel()

	quota := func(quotaType utils.BacklogQuotaType, limit string) map[string]interface{} {
		return map[string]interface{}{
			"limit_bytes":   limit,
			"limit_seconds": "-1",
			"policy":        string(utils.ProducerRequestHold),
			"type":          quotaType.String(),
		}
	}

	tests := []struct {
		name string
		old  []interface{}
		new  []interface{}
		want []utils.BacklogQuotaType
	}{
		{
			name: "remove message age",
			old: []interface{}{
				quota(utils.DestinationStorage, "100"),
				quota(utils.MessageAge, "-1"),
			},
			new:  []interface{}{quota(utils.DestinationStorage, "100")},
			want: []utils.BacklogQuotaType{utils.MessageAge},
		},
		{
			name: "remove destination storage",
			old: []interface{}{
				quota(utils.DestinationStorage, "100"),
				quota(utils.MessageAge, "-1"),
			},
			new:  []interface{}{quota(utils.MessageAge, "-1")},
			want: []utils.BacklogQuotaType{utils.DestinationStorage},
		},
		{
			name: "value change keeps same type",
			old:  []interface{}{quota(utils.DestinationStorage, "100")},
			new:  []interface{}{quota(utils.DestinationStorage, "200")},
			want: []utils.BacklogQuotaType{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := removedBacklogQuotaTypes(
				schema.NewSet(hashBacklogQuotaSubset(), tt.old),
				schema.NewSet(hashBacklogQuotaSubset(), tt.new),
			)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestBacklogQuotaPlannedSetForOwnership(t *testing.T) {
	t.Parallel()

	destination := testBacklogQuotaValue(utils.DestinationStorage, "100")
	destinationUpdated := testBacklogQuotaValue(utils.DestinationStorage, "200")
	messageAge := testBacklogQuotaValue(utils.MessageAge, "-1")
	oldQuotas := schema.NewSet(hashBacklogQuotaSubset(), []interface{}{destination, messageAge})

	tests := []struct {
		name           string
		rawState       cty.Value
		newQuotas      *schema.Set
		wantOverride   bool
		wantQuotaTypes []utils.BacklogQuotaType
		wantDestLimit  string
	}{
		{
			name:           "fresh import preserves unconfigured type",
			rawState:       testBacklogQuotaOwnershipRawState(),
			newQuotas:      schema.NewSet(hashBacklogQuotaSubset(), []interface{}{destination}),
			wantOverride:   true,
			wantQuotaTypes: []utils.BacklogQuotaType{utils.DestinationStorage, utils.MessageAge},
			wantDestLimit:  "100",
		},
		{
			name: "managed type removal remains authoritative",
			rawState: testBacklogQuotaOwnershipRawState(
				utils.DestinationStorage,
				utils.MessageAge,
			),
			newQuotas:      schema.NewSet(hashBacklogQuotaSubset(), []interface{}{destination}),
			wantOverride:   false,
			wantQuotaTypes: []utils.BacklogQuotaType{utils.DestinationStorage},
			wantDestLimit:  "100",
		},
		{
			name:           "legacy state without metadata preserves all unproven types",
			rawState:       cty.ObjectVal(map[string]cty.Value{}),
			newQuotas:      schema.NewSet(hashBacklogQuotaSubset(), []interface{}{destination}),
			wantOverride:   true,
			wantQuotaTypes: []utils.BacklogQuotaType{utils.DestinationStorage, utils.MessageAge},
			wantDestLimit:  "100",
		},
		{
			name:           "configured value replaces imported value without adopting extra type",
			rawState:       testBacklogQuotaOwnershipRawState(),
			newQuotas:      schema.NewSet(hashBacklogQuotaSubset(), []interface{}{destinationUpdated}),
			wantOverride:   true,
			wantQuotaTypes: []utils.BacklogQuotaType{utils.DestinationStorage, utils.MessageAge},
			wantDestLimit:  "200",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			configuredQuota := tt.newQuotas.List()[0].(map[string]interface{})

			planned, changed, err := backlogQuotaPlannedSetForOwnership(
				testBacklogQuotaRawConfig(configuredQuota),
				tt.rawState,
				oldQuotas,
				tt.newQuotas,
			)
			require.NoError(t, err)
			require.Equal(t, tt.wantOverride, changed)
			if !changed {
				planned = tt.newQuotas
			}

			types, err := backlogQuotaTypes(planned)
			require.NoError(t, err)
			require.Len(t, types, len(tt.wantQuotaTypes))
			for _, quotaType := range tt.wantQuotaTypes {
				require.Contains(t, types, quotaType)
			}

			for _, quota := range planned.List() {
				data := quota.(map[string]interface{})
				if data["type"] == utils.DestinationStorage.String() {
					require.Equal(t, tt.wantDestLimit, data["limit_bytes"])
				}
			}
		})
	}
}

func TestRemovedManagedBacklogQuotaTypes(t *testing.T) {
	t.Parallel()

	oldQuotas := schema.NewSet(hashBacklogQuotaSubset(), []interface{}{
		testBacklogQuotaValue(utils.DestinationStorage, "100"),
		testBacklogQuotaValue(utils.MessageAge, "-1"),
	})
	newQuotas := schema.NewSet(hashBacklogQuotaSubset(), []interface{}{
		testBacklogQuotaValue(utils.DestinationStorage, "100"),
	})

	removed, err := removedManagedBacklogQuotaTypes(
		oldQuotas,
		newQuotas,
		testBacklogQuotaOwnershipRawState(utils.MessageAge),
	)
	require.NoError(t, err)
	require.Equal(t, []utils.BacklogQuotaType{utils.MessageAge}, removed)

	removed, err = removedManagedBacklogQuotaTypes(
		oldQuotas,
		newQuotas,
		cty.ObjectVal(map[string]cty.Value{}),
	)
	require.NoError(t, err)
	require.Empty(t, removed)
}

func TestInitializeBacklogQuotaManagedTypesStateDefaultsLegacyStateToEmpty(t *testing.T) {
	t.Parallel()

	d := resourcePulsarNamespace().TestResourceData()
	d.SetId("tenant/namespace")
	require.NoError(t, initializeBacklogQuotaManagedTypesState(d))

	state := d.State()
	require.NotNil(t, state)
	require.Equal(t, "0", state.Attributes[backlogQuotaManagedTypesStateAttr+".#"])
}

func testBacklogQuotaValue(quotaType utils.BacklogQuotaType, limitBytes string) map[string]interface{} {
	limitSeconds := "-1"
	policy := utils.ProducerRequestHold
	if quotaType == utils.MessageAge {
		limitSeconds = "3600"
		policy = utils.ConsumerBacklogEviction
	}
	return map[string]interface{}{
		"limit_bytes":   limitBytes,
		"limit_seconds": limitSeconds,
		"policy":        policy.String(),
		"type":          quotaType.String(),
	}
}

func testBacklogQuotaRawConfig(quotas ...map[string]interface{}) cty.Value {
	values := make([]cty.Value, 0, len(quotas))
	for _, quota := range quotas {
		values = append(values, cty.ObjectVal(map[string]cty.Value{
			"limit_bytes":   cty.StringVal(quota["limit_bytes"].(string)),
			"limit_seconds": cty.StringVal(quota["limit_seconds"].(string)),
			"policy":        cty.StringVal(quota["policy"].(string)),
			"type":          cty.StringVal(quota["type"].(string)),
		}))
	}
	return cty.ObjectVal(map[string]cty.Value{
		"backlog_quota": cty.SetVal(values),
	})
}

func testBacklogQuotaOwnershipRawState(managed ...utils.BacklogQuotaType) cty.Value {
	values := make([]cty.Value, 0, len(managed))
	for _, quotaType := range managed {
		values = append(values, cty.StringVal(quotaType.String()))
	}
	managedTypes := cty.SetValEmpty(cty.String)
	if len(values) > 0 {
		managedTypes = cty.SetVal(values)
	}
	return cty.ObjectVal(map[string]cty.Value{
		backlogQuotaManagedTypesStateAttr: managedTypes,
	})
}
