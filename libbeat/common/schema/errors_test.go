package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrors(t *testing.T) {
	var errs Errors
	errs = append(errs, NewError("test", "Hello World"))

	assert.True(t, errs.HasRequiredErrors())
}
