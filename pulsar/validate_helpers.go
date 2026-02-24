package pulsar

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
)

var inactiveTopicDurationPattern = regexp.MustCompile(`^\d+s$`)

func parseInactiveTopicDurationSeconds(raw string) (int, error) {
	if !inactiveTopicDurationPattern.MatchString(raw) {
		return 0, fmt.Errorf("must use seconds format like 60s (got: %s)", raw)
	}

	seconds, err := strconv.Atoi(strings.TrimSuffix(raw, "s"))
	if err != nil {
		return 0, fmt.Errorf("invalid seconds value in duration %q: %w", raw, err)
	}

	return seconds, nil
}

func validateNotBlank(val interface{}, key string) (warns []string, errs []error) {
	v := val.(string)
	if len(strings.Trim(strings.TrimSpace(v), "\"")) == 0 {
		errs = append(errs, fmt.Errorf("%q must not be empty", key))
	}
	return
}

func validateURL(val interface{}, key string) (warns []string, errs []error) {
	v := val.(string)
	_, err := url.ParseRequestURI(strings.Trim(strings.TrimSpace(v), "\""))
	if err != nil {
		errs = append(errs, fmt.Errorf("%q must be a valid url: %w", key, err))
	}
	return
}

func validateGtEq0(val interface{}, key string) (warns []string, errs []error) {
	v := val.(int)
	if v < 0 {
		errs = append(errs, fmt.Errorf("%q must be 0 or more, got: %d", key, v))
	}
	return
}

func validateTopicType(val interface{}, key string) (warns []string, errs []error) {
	v := val.(string)
	_, err := utils.ParseTopicDomain(v)
	if err != nil {
		errs = append(errs, fmt.Errorf("%q must be a valid topic name (got: %s): %w", key, v, err))
	}
	return
}

func validateAuthAction(val interface{}, key string) (warns []string, errs []error) {
	v := val.(string)
	_, err := utils.ParseAuthAction(v)
	if err != nil {
		errs = append(errs, fmt.Errorf("%q must be a valid auth action (got: %s): %w", key, v, err))
	}
	return
}

func validatePartitionedTopicType(val interface{}, key string) (warns []string, errs []error) {
	v := val.(string)
	_, err := utils.ParseTopicType(v)
	if err != nil {
		errs = append(errs, fmt.Errorf("%q must be a valid topic type (got: %s): %w", key, v, err))
	}
	return
}

func validiateDeleteMode(val interface{}, key string) (warns []string, errs []error) {
	v := val.(string)
	if v != "delete_when_no_subscriptions" && v != "delete_when_subscriptions_caught_up" {
		errs = append(errs, fmt.Errorf("%q must be one of delete_when_no_subscriptions"+
			" or delete_when_subscriptions_caught_up (got: %s)", key, v))
	}
	return
}

func validateInactiveTopicDuration(val interface{}, key string) (warns []string, errs []error) {
	v := val.(string)
	if _, err := parseInactiveTopicDurationSeconds(v); err != nil {
		errs = append(errs, fmt.Errorf("%q %v", key, err))
	}
	return
}
