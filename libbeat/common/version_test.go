package common

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
			result:  Version{Major: 1, Minor: 2, Bugfix: 3, version: "1.2.3"},
		},
		{
			version: "1.3.3",
			err:     false,
			result:  Version{Major: 1, Minor: 3, Bugfix: 3, version: "1.3.3"},
		},
		{
			version: "1.3.2-alpha1",
			err:     false,
			result:  Version{Major: 1, Minor: 3, Bugfix: 2, version: "1.3.2-alpha1", Meta: "alpha1"},
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

func TestVersionLessThan(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		version1 string
		result   bool
	}{
		{
			name:     "1.2.3 < 2.0.0",
			version:  "1.2.3",
			version1: "2.0.0",
			result:   true,
		},
		{
			name:     "1.2.3 = 1.2.3-beta1",
			version:  "1.2.3",
			version1: "1.2.3-beta1",
			result:   false,
		},
		{
			name:     "5.4.1 < 5.4.2",
			version:  "5.4.1",
			version1: "5.4.2",
			result:   true,
		},
		{
			name:     "5.5.1 > 5.4.2",
			version:  "5.5.1",
			version1: "5.4.2",
			result:   false,
		},
		{
			name:     "6.1.1-alpha3 < 6.2.0",
			version:  "6.1.1-alpha3",
			version1: "6.2.0",
			result:   true,
		},
	}

	for _, test := range tests {
		v, err := NewVersion(test.version)
		assert.NoError(t, err)
		v1, err := NewVersion(test.version1)
		assert.NoError(t, err)

		assert.Equal(t, v.LessThan(v1), test.result, test.name)
	}
}
