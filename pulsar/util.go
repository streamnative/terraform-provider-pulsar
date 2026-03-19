package pulsar

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	ProcessingGuaranteesAtLeastOnce     = "ATLEAST_ONCE"
	ProcessingGuaranteesAtMostOnce      = "ATMOST_ONCE"
	ProcessingGuaranteesEffectivelyOnce = "EFFECTIVELY_ONCE"
)

const (
	SubscriptionPositionEarliest = "Earliest"
	SubscriptionPositionLatest   = "Latest"
)

func isPackageURLSupported(functionPkgURL string) bool {
	return strings.HasPrefix(functionPkgURL, "http://") ||
		strings.HasPrefix(functionPkgURL, "https://") ||
		strings.HasPrefix(functionPkgURL, "file://") ||
		strings.HasPrefix(functionPkgURL, "function://") ||
		strings.HasPrefix(functionPkgURL, "sink://") ||
		strings.HasPrefix(functionPkgURL, "source://")
}

func jsonValidateFunc(i interface{}, s string) ([]string, []error) {
	v := i.(string)
	_, err := json.Marshal(v)
	if err != nil {
		return nil, []error{
			fmt.Errorf("cannot marshal %s: %s", v, err.Error()),
		}
	}
	return nil, nil
}

func ignoreServerSetCustomRuntimeOptions(tfGenString string, readString string) (string, error) {
	tfGenMap := make(map[string]interface{})
	err := json.Unmarshal([]byte(tfGenString), &tfGenMap)
	if err != nil {
		return "", err
	}
	readMap := make(map[string]interface{})
	computedMap := make(map[string]interface{})
	err = json.Unmarshal([]byte(readString), &readMap)
	if err != nil {
		return "", err
	}
	for k := range readMap {
		if _, has := tfGenMap[k]; has {
			computedMap[k] = readMap[k]
		}
	}
	s, err := json.Marshal(computedMap)
	if err != nil {
		return "", err
	}
	return string(s), nil
}

func ignoreServerSetTopicProperties(managedKeys map[string]struct{}, readProperties map[string]string) map[string]string {
	filteredProperties := make(map[string]string)

	if len(managedKeys) == 0 || len(readProperties) == 0 {
		return filteredProperties
	}

	for key, value := range readProperties {
		if _, ok := managedKeys[key]; ok {
			filteredProperties[key] = value
		}
	}

	return filteredProperties
}
