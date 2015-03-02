package filters

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterNames(t *testing.T) {
	assert.Equal(t, "nop", NopFilter.String())
	assert.Equal(t, "sample", SampleFilter.String())
	assert.Equal(t, "impossible", Filter(2).String())
	assert.Equal(t, "impossible", Filter(-2).String())
}
