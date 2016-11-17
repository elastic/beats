// +build !integration

package prospector

import (
	"regexp"
	"testing"

	"github.com/elastic/beats/filebeat/input/file"

	"github.com/stretchr/testify/assert"
)

func TestProspectorInitInputTypeLogError(t *testing.T) {

	prospector := Prospector{
		config: prospectorConfig{},
	}

	states := file.NewStates()
	states.SetStates([]file.State{})
	err := prospector.Init(*states)
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
