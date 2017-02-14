package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersion(t *testing.T) {

	tests := []struct {
		version string
		err     bool
		result  Version
	}{
		{
			version: "1.2.3",
			err:     false,
			result:  Version{major: 1, minor: 2, bugfix: 3, version: "1.2.3"},
		},
		{
			version: "1.3.3",
			err:     false,
			result:  Version{major: 1, minor: 3, bugfix: 3, version: "1.3.3"},
		},
		{
			version: "1.3.2-alpha1",
			err:     false,
			result:  Version{major: 1, minor: 3, bugfix: 2, version: "1.3.2-alpha1", meta: "alpha1"},
		},
		{
			version: "alpha1",
			err:     true,
		},
	}

	for _, test := range tests {
		v, err := NewVersion(test.version)
		if test.err {
			assert.Error(t, err)
			continue
		} else {
			assert.NoError(t, err)
		}
		assert.Equal(t, *v, test.result)
	}
}
