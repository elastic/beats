// +build !integration

package crawler

import (
	"regexp"
	"testing"

	"github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/common"
	"github.com/stretchr/testify/assert"
)

func TestProspectorDefaultConfigs(t *testing.T) {

	prospector, err := NewProspector(common.NewConfig(), nil, nil)
	assert.NoError(t, err)

	// Default values expected
	assert.Equal(t, DefaultIgnoreOlder, prospector.config.IgnoreOlder)
	assert.Equal(t, DefaultScanFrequency, prospector.config.ScanFrequency)
	assert.Equal(t, config.DefaultHarvesterBufferSize, prospector.config.Harvester.BufferSize)
	assert.Equal(t, config.DefaultTailFiles, prospector.config.Harvester.TailFiles)
	assert.Equal(t, config.DefaultBackoff, prospector.config.Harvester.BackoffDuration)
	assert.Equal(t, config.DefaultBackoffFactor, prospector.config.Harvester.BackoffFactor)
	assert.Equal(t, config.DefaultMaxBackoff, prospector.config.Harvester.MaxBackoffDuration)
	assert.Equal(t, config.DefaultForceCloseFiles, prospector.config.Harvester.ForceCloseFiles)
	assert.Equal(t, config.DefaultMaxBytes, prospector.config.Harvester.MaxBytes)
}

func TestProspectorInitInputTypeLog(t *testing.T) {

	prospectorConfig := prospectorConfig{
		Paths: []string{"testpath1", "testpath2"},
		Harvester: harvester.HarvesterConfig{
			InputType: "log",
		},
	}

	prospector := Prospector{
		config: prospectorConfig,
		states: input.NewStates(),
	}

	err := prospector.Init()
	assert.Nil(t, err)
	assert.Equal(t, "log", prospector.config.Harvester.InputType)
}

func TestProspectorInitInputTypeLogError(t *testing.T) {

	prospectorConfig := prospectorConfig{
		Harvester: harvester.HarvesterConfig{
			InputType: "log",
		},
	}

	prospector := Prospector{
		config: prospectorConfig,
	}

	err := prospector.Init()
	// Error should be returned because no path is set
	assert.Error(t, err)
}

func TestProspectorInitInputTypeStdin(t *testing.T) {

	prospectorConfig := prospectorConfig{
		Harvester: harvester.HarvesterConfig{
			InputType: "stdin",
		},
	}

	prospector := Prospector{
		config: prospectorConfig,
	}

	err := prospector.Init()
	assert.Nil(t, err)
	assert.Equal(t, "stdin", prospector.config.Harvester.InputType)
}

func TestProspectorInitInputTypeWrong(t *testing.T) {

	prospectorConfig := prospectorConfig{
		Harvester: harvester.HarvesterConfig{
			InputType: "wrong-type",
		},
	}

	prospector := Prospector{
		config: prospectorConfig,
		states: input.NewStates(),
	}

	err := prospector.Init()
	assert.Nil(t, err)
	assert.Equal(t, "log", prospector.config.Harvester.InputType)
}

func TestProspectorFileExclude(t *testing.T) {

	prospectorConfig := prospectorConfig{
		ExcludeFiles: []*regexp.Regexp{regexp.MustCompile(`\.gz$`)},
		Harvester: harvester.HarvesterConfig{
			BufferSize: 0,
		},
	}

	prospector := Prospector{
		config: prospectorConfig,
		states: input.NewStates(),
	}

	prospector.Init()
	prospectorer := prospector.prospectorer.(*ProspectorLog)

	assert.True(t, prospectorer.isFileExcluded("/tmp/log/logw.gz"))
	assert.False(t, prospectorer.isFileExcluded("/tmp/log/logw.log"))

}
