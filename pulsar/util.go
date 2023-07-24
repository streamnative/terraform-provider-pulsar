package pulsar

import (
	"fmt"
	"strings"
)
import "encoding/json"

const (
	ProcessingGuaranteesAtLeastOnce     = "ATLEAST_ONCE"
	ProcessingGuaranteesAtMostOnce      = "ATMOST_ONCE"
	ProcessingGuaranteesEffectivelyOnce = "EFFECTIVELY_ONCE"
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
