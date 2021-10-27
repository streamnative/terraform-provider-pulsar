package vault

import "strings"

func resolvePath(parts []string) string {
	trimmedParts := make([]string, len(parts))

	for i, v := range parts {
		trimmedParts[i] = strings.Trim(v, "/")
	}

	return "/" + strings.Join(trimmedParts, "/")
}

func BoolPtr(input bool) *bool {
	b := input
	return &b
}

func IntPtr(input int) *int {
	i := input
	return &i
}

func StringPtr(input string) *string {
	s := input
	return &s
}
