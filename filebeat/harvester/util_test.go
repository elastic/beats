// +build !integration

package harvester

import (
	"regexp"
	"testing"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/stretchr/testify/assert"
)

// InitRegexps initializes a list of compiled regular expressions.
func InitRegexps(exprs []string) ([]*regexp.Regexp, error) {

	result := []*regexp.Regexp{}

	for _, exp := range exprs {
		rexp, err := regexp.Compile(exp)
		if err != nil {
			logp.Err("Fail to compile the regexp %s: %s", exp, err)
			return nil, err
		}
		result = append(result, rexp)
	}
	return result, nil
}

func TestMatchAnyRegexps(t *testing.T) {

	patterns := []string{"\\.gz$"}

	regexps, err := InitRegexps(patterns)

	assert.Nil(t, err)

	assert.Equal(t, MatchAnyRegexps(regexps, "/var/log/log.gz"), true)

}
