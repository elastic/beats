// +build !integration

package reader

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsLine(t *testing.T) {
	notLine := []byte("This is not a line")
	assert.False(t, isLine(notLine))

	notLine = []byte("This is not a line\n\r")
	assert.False(t, isLine(notLine))

	notLine = []byte("This is \n not a line")
	assert.False(t, isLine(notLine))

	line := []byte("This is a line \n")
	assert.True(t, isLine(line))

	line = []byte("This is a line\r\n")
	assert.True(t, isLine(line))
}

func TestLineEndingChars(t *testing.T) {
	line := []byte("Not ending line")
	assert.Equal(t, 0, lineEndingChars(line))

	line = []byte("N ending \n")
	assert.Equal(t, 1, lineEndingChars(line))

	line = []byte("RN ending \r\n")
	assert.Equal(t, 2, lineEndingChars(line))

	// This is an invalid option
	line = []byte("NR ending \n\r")
	assert.Equal(t, 0, lineEndingChars(line))
}
