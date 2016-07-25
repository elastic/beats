package harvester

import (
	"regexp"
	"time"

	"github.com/elastic/beats/filebeat/harvester/reader"
	"github.com/elastic/beats/libbeat/common"
)

// readLine reads a full line into buffer and returns it.
// In case of partial lines, readLine does return and error and en empty string
// This could potentialy be improved / replaced by https://github.com/elastic/beats/libbeat/tree/master/common/streambuf
func readLine(reader reader.Reader) (time.Time, string, int, common.MapStr, error) {
	l, err := reader.Next()

	// Full line read to be returned
	if l.Bytes != 0 && err == nil {
		return l.Ts, string(l.Content), l.Bytes, l.Fields, err
	}

	return time.Time{}, "", 0, nil, err
}

// MatchAnyRegexps checks if the text matches any of the regular expressions
func MatchAnyRegexps(regexps []*regexp.Regexp, text string) bool {

	for _, rexp := range regexps {
		if rexp.MatchString(text) {
			return true
		}
	}

	return false
}
