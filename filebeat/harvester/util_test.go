// +build !integration

package harvester

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchAnyRegexps(t *testing.T) {

	patterns := []string{"\\.gz$"}

	regexps, err := InitRegexps(patterns)

	assert.Nil(t, err)

	assert.Equal(t, MatchAnyRegexps(regexps, "/var/log/log.gz"), true)

}
