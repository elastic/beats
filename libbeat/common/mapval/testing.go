package mapval

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

// Test takes the output from a Validator invocation and runs test assertions on the result.
// If you are using this library for testing you will probably want to run Test(t, Compile(Map{...}), actual) as a pattern.
func Test(t *testing.T, v Validator, m common.MapStr) *Results {
	r := v(m)

	if !r.Valid {
		assert.Fail(
			t,
			"mapval could not validate map",
			"%d errors validating source: \n%s", len(r.Errors()), spew.Sdump(m),
		)
	}

	for _, err := range r.Errors() {
		assert.NoError(t, err)
	}
	return r
}
