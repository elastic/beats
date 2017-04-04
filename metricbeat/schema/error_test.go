package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsError(t *testing.T) {
	err := NewError("test", "Hello World")
	assert.Error(t, err)
}

func TestType(t *testing.T) {
	err := NewError("test", "Hello World")
	assert.True(t, err.IsType(RequiredType))

	err.SetType(OptionalType)
	assert.True(t, err.IsType(OptionalType))
}
