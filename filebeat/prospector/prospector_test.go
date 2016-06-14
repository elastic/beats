// +build !integration

package prospector

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProspectorInitInputTypeLogError(t *testing.T) {

	prospector := Prospector{
		config: prospectorConfig{},
	}

	err := prospector.Init()
	// Error should be returned because no path is set
	assert.Error(t, err)
}

func TestProspectorFileExclude(t *testing.T) {

	prospector := Prospector{
		config: prospectorConfig{
			ExcludeFiles: []*regexp.Regexp{regexp.MustCompile(`\.gz$`)},
		},
	}

	p, err := NewProspectorLog(&prospector)
	assert.NoError(t, err)

	assert.True(t, p.isFileExcluded("/tmp/log/logw.gz"))
	assert.False(t, p.isFileExcluded("/tmp/log/logw.log"))
}
