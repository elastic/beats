// +build !integration

package crawler

import (
	"regexp"
	"testing"

	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/common"
	"github.com/stretchr/testify/assert"
)

func TestProspectorDefaultConfigs(t *testing.T) {

	prospector, err := NewProspector(common.NewConfig(), *input.NewStates(), nil)
	assert.NoError(t, err)

	// Default values expected
	assert.Equal(t, DefaultIgnoreOlder, prospector.config.IgnoreOlder)
	assert.Equal(t, DefaultScanFrequency, prospector.config.ScanFrequency)
}

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
