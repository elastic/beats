package useragent

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserAgent(t *testing.T) {
	ua := UserAgent("FakeBeat")
	assert.Regexp(t, regexp.MustCompile("^Elastic FakeBeat"), ua)
}
