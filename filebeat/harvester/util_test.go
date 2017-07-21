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

func TestExcludeLine(t *testing.T) {
	regexp, err := InitMatchers("^DBG")
	assert.Nil(t, err)
	assert.True(t, MatchAny(regexp, "DBG: a debug message"))
	assert.False(t, MatchAny(regexp, "ERR: an error message"))
}

func TestIncludeLine(t *testing.T) {
	regexp, err := InitMatchers("^ERR", "^WARN")

	assert.Nil(t, err)
	assert.False(t, MatchAny(regexp, "DBG: a debug message"))
	assert.True(t, MatchAny(regexp, "ERR: an error message"))
	assert.True(t, MatchAny(regexp, "WARNING: a simple warning message"))
}

func TestInitRegexp(t *testing.T) {
	_, err := InitMatchers("(((((")
	assert.NotNil(t, err)
}
