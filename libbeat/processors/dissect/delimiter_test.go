package dissect

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMultiByte(t *testing.T) {
	m := newDelimiter("needle")
	assert.Equal(t, 3, m.IndexOf("   needle", 1))
}

func TestSingleByte(t *testing.T) {
	m := newDelimiter("")
	assert.Equal(t, 5, m.IndexOf("  needle", 5))
}
