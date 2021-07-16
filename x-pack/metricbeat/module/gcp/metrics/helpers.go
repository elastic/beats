package metrics

import "strings"

// withSuffix ensures a string end with the specified suffix.
func withSuffix(s, suffix string) string {
	if strings.HasSuffix(s, suffix) {
		return s
	}

	return s + suffix
}
