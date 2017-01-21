package socket

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashSet(t *testing.T) {
	set := hashSet{}

	set.Add(10)
	assert.True(t, set.Contains(10))
	assert.False(t, set.Contains(0))

	set.Reset()
	assert.Len(t, set, 0)
}
