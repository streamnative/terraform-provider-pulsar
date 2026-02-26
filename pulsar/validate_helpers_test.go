package pulsar

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseInactiveTopicDurationSeconds(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()

		testCases := map[string]int{
			"0s":    0,
			"60s":   60,
			"3600s": 3600,
		}

		for input, expected := range testCases {
			input := input
			expected := expected
			t.Run(input, func(t *testing.T) {
				t.Parallel()
				actual, err := parseInactiveTopicDurationSeconds(input)
				require.NoError(t, err)
				require.Equal(t, expected, actual)
			})
		}
	})

	t.Run("invalid", func(t *testing.T) {
		t.Parallel()

		testCases := []string{
			"",
			"s",
			"60",
			"-1s",
			"60m",
			"abc",
			" 60s",
			"999999999999999999999999s",
		}

		for _, input := range testCases {
			input := input
			t.Run(input, func(t *testing.T) {
				t.Parallel()
				_, err := parseInactiveTopicDurationSeconds(input)
				require.Error(t, err)
			})
		}
	})
}

func TestValidateInactiveTopicDuration(t *testing.T) {
	t.Parallel()

	_, errs := validateInactiveTopicDuration("120s", "max_inactive_duration")
	require.Empty(t, errs)

	_, errs = validateInactiveTopicDuration("120m", "max_inactive_duration")
	require.Len(t, errs, 1)
	require.Contains(t, errs[0].Error(), "max_inactive_duration")
}
