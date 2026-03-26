package pulsar

import (
	"fmt"
	"testing"

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/rest"
	"github.com/stretchr/testify/require"
)

func TestIsDisabledTopicPolicyError(t *testing.T) {
	t.Parallel()

	disabledErr := rest.Error{Code: 405, Reason: topicLevelPoliciesDisabledReason}

	require.True(t, isDisabledTopicPolicyError(disabledErr))
	require.True(t, isDisabledTopicPolicyError(fmt.Errorf("wrapped: %w", disabledErr)))
	require.False(t, isDisabledTopicPolicyError(rest.Error{Code: 404, Reason: "Not Found"}))
	require.False(t, isDisabledTopicPolicyError(rest.Error{Code: 405, Reason: "Method Not Allowed"}))
}

func TestIsIgnorableTopicPolicyError(t *testing.T) {
	t.Parallel()

	require.True(t, isIgnorableTopicPolicyError(rest.Error{Code: 404, Reason: "Not Found"}))
	require.True(t, isIgnorableTopicPolicyError(rest.Error{Code: 405, Reason: topicLevelPoliciesDisabledReason}))
	require.True(
		t,
		isIgnorableTopicPolicyError(
			fmt.Errorf(
				"wrapped: %w",
				rest.Error{Code: 405, Reason: topicLevelPoliciesDisabledReason},
			),
		),
	)
	require.False(t, isIgnorableTopicPolicyError(rest.Error{Code: 500, Reason: "Internal Server Error"}))
}

func TestWrapTopicPolicyWriteError(t *testing.T) {
	t.Parallel()

	disabledErr := wrapTopicPolicyWriteError(
		"ERROR_UPDATE_REPLICATION_CLUSTERS: SetReplicationClusters",
		rest.Error{Code: 405, Reason: topicLevelPoliciesDisabledReason},
	)
	require.True(t, isPermanentBackoffError(disabledErr))
	require.ErrorContains(t, disabledErr, "topic-level policies are disabled in the cluster")

	notFoundErr := wrapTopicPolicyWriteError(
		"ERROR_UPDATE_REPLICATION_CLUSTERS: SetReplicationClusters",
		rest.Error{Code: 404, Reason: "Not Found"},
	)
	require.False(t, isPermanentBackoffError(notFoundErr))
	require.ErrorContains(t, notFoundErr, "ERROR_UPDATE_REPLICATION_CLUSTERS: SetReplicationClusters")

	internalErr := wrapTopicPolicyWriteError(
		"ERROR_UPDATE_REPLICATION_CLUSTERS: SetReplicationClusters",
		rest.Error{Code: 500, Reason: "Internal Server Error"},
	)
	require.True(t, isPermanentBackoffError(internalErr))
}

func TestWrapTopicPolicyDeleteError(t *testing.T) {
	t.Parallel()

	disabledErr := wrapTopicPolicyDeleteError(
		rest.Error{Code: 405, Reason: topicLevelPoliciesDisabledReason},
	)
	require.True(t, isPermanentBackoffError(disabledErr))
	require.ErrorContains(t, disabledErr, "topic-level policies are disabled in the cluster")

	notFoundErr := wrapTopicPolicyDeleteError(
		rest.Error{Code: 404, Reason: "Not Found"},
	)
	require.NoError(t, notFoundErr)

	internalErr := wrapTopicPolicyDeleteError(
		rest.Error{Code: 500, Reason: "Internal Server Error"},
	)
	require.True(t, isPermanentBackoffError(internalErr))
}

func TestRetryFastFailsOnDisabledTopicPolicyError(t *testing.T) {
	t.Parallel()

	calls := 0
	err := retry(func() error {
		calls++
		return wrapTopicPolicyWriteError(
			"ERROR_UPDATE_REPLICATION_CLUSTERS: SetReplicationClusters",
			rest.Error{Code: 405, Reason: topicLevelPoliciesDisabledReason},
		)
	})

	require.Error(t, err)
	require.Equal(t, 1, calls)
	require.ErrorContains(t, err, "topic-level policies are disabled in the cluster")
}
