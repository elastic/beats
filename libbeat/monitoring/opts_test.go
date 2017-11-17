// +build !integration

package monitoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOptions(t *testing.T) {
	tests := []struct {
		name     string
		parent   *options
		options  []Option
		expected options
	}{
		{
			"empty parent without opts should generate defaults",
			nil,
			nil,
			defaultOptions,
		},
		{
			"non empty parent should return same options",
			&options{},
			nil,
			options{},
		},
		{
			"apply publishexpvar",
			&options{publishExpvar: false},
			[]Option{PublishExpvar},
			options{publishExpvar: true},
		},
		{
			"apply disable publishexpvar",
			&options{publishExpvar: true},
			[]Option{IgnorePublishExpvar},
			options{publishExpvar: false},
		},
	}

	for i, test := range tests {
		t.Logf("run test (%v): %v", i, test.name)

		origParent := options{}
		if test.parent != nil {
			origParent = *test.parent
		}
		actual := applyOpts(test.parent, test.options)
		assert.NotNil(t, actual)

		// test parent has not been modified by accident
		if test.parent != nil {
			assert.Equal(t, origParent, *test.parent)
		}

		// check parent and actual are same object if options is nil
		if test.parent != nil && test.options == nil {
			assert.Equal(t, test.parent, actual)
		}

		// validate output
		assert.Equal(t, test.expected, *actual)
	}
}
