package harvester

import (
	"regexp"
	"time"

	"github.com/elastic/beats/filebeat/harvester/processor"
	"github.com/elastic/beats/libbeat/logp"
)

// readLine reads a full line into buffer and returns it.
// In case of partial lines, readLine does return and error and en empty string
// This could potentialy be improved / replaced by https://github.com/elastic/beats/libbeat/tree/master/common/streambuf
func readLine(reader processor.LineProcessor) (time.Time, string, int, error) {
	for {
		l, err := reader.Next()

		// Full line read to be returned
		if l.Bytes != 0 && err == nil {
			logp.Debug("harvester", "full line read")
			return l.Ts, string(l.Content), l.Bytes, err
		}

		return time.Time{}, "", 0, err
	}
}

// InitRegexps initializes a list of compiled regular expressions.
func InitRegexps(exprs []string) ([]*regexp.Regexp, error) {

	result := []*regexp.Regexp{}

	for _, exp := range exprs {

		rexp, err := regexp.CompilePOSIX(exp)
		if err != nil {
			logp.Err("Fail to compile the regexp %s: %s", exp, err)
			return nil, err
		}
		result = append(result, rexp)
	}
	return result, nil
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
