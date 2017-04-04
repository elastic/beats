package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrors(t *testing.T) {
	errs := NewErrors()
	err := NewError("test", "Hello World")
	errs.AddError(err)

	assert.True(t, errs.HasRequiredErrors())
}
