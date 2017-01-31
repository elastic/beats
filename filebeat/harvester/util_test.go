// +build !integration

package harvester

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common/match"
	"github.com/elastic/beats/libbeat/logp"
)

// InitMatchers initializes a list of compiled regular expressions.
func InitMatchers(exprs ...string) ([]match.Matcher, error) {
	result := []match.Matcher{}

	for _, exp := range exprs {
		rexp, err := match.Compile(exp)
		if err != nil {
			logp.Err("Fail to compile the regexp %s: %s", exp, err)
			return nil, err
		}
		result = append(result, rexp)
	}
	return result, nil
}

func TestMatchAnyRegexps(t *testing.T) {
	matchers, err := InitMatchers("\\.gz$")
	assert.Nil(t, err)
	assert.Equal(t, MatchAny(matchers, "/var/log/log.gz"), true)

}
