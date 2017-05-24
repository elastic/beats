package harvester

import "github.com/elastic/beats/libbeat/common/match"

// Contains available prospector types
const (
	LogType   = "log"
	StdinType = "stdin"
)

// ValidType is a list of all valid input types
var ValidType = map[string]struct{}{
	StdinType: {},
	LogType:   {},
}

// MatchAny checks if the text matches any of the regular expressions
func MatchAny(matchers []match.Matcher, text string) bool {
	for _, m := range matchers {
		if m.MatchString(text) {
			return true
		}
	}
	return false
}
