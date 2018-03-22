package elasticsearch

import (
	"unicode"
	"github.com/elastic/beats/libbeat/logp"
	"strings"
)

func TryLowercaseIndex(index string) string {
	for _, u := range []rune(index) {
		if unicode.IsUpper(u) {
			newIndex := strings.ToLower(index)
			logp.Warn("Index name %s is invalid, replace it with %s", index, newIndex)
			return newIndex
		}
	}
	return index
}
