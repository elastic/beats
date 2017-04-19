// +build !integration

package prospector

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/libbeat/common/match"
)

func TestProspectorInitInputTypeLogError(t *testing.T) {

	prospector := Prospector{
		config: prospectorConfig{},
	}

	states := file.NewStates()
	states.SetStates([]file.State{})
	err := prospector.LoadStates(states.GetStates())
	// Error should be returned because no path is set
	assert.Error(t, err)
}

func TestProspectorFileExclude(t *testing.T) {

	prospector := Prospector{
		config: prospectorConfig{
			Paths:        []string{"test.log"},
			ExcludeFiles: []match.Matcher{match.MustCompile(`\.gz$`)},
		},
	}

	p, err := NewProspectorLog(&prospector)
	assert.NoError(t, err)

	assert.True(t, p.isFileExcluded("/tmp/log/logw.gz"))
	assert.False(t, p.isFileExcluded("/tmp/log/logw.log"))
}
