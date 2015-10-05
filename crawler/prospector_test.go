package crawler

import (
	"testing"
	"time"

	"github.com/elastic/filebeat/config"
	"github.com/stretchr/testify/assert"
)

func TestProspectorInit(t *testing.T) {

	prospectorConfig := config.ProspectorConfig{
		ScanFrequency:       "15s",
		IgnoreOlder:         "100m",
		HarvesterBufferSize: 100,
		TailOnRotate:        true,
	}

	prospector := Prospector{
		ProspectorConfig: prospectorConfig,
	}

	assert.NotNil(t, prospector)

	prospector.Init()

	// Predefined values expected
	assert.Equal(t, 100*time.Minute, prospector.ProspectorConfig.IgnoreOlderDuration)
	assert.Equal(t, 15*time.Second, prospector.ProspectorConfig.ScanFrequencyDuration)
	assert.Equal(t, 100, prospector.ProspectorConfig.HarvesterBufferSize)
	assert.Equal(t, true, prospector.ProspectorConfig.TailOnRotate)
}

func TestProspectorInitEmpty(t *testing.T) {

	prospectorConfig := config.ProspectorConfig{
		ScanFrequency:       "",
		IgnoreOlder:         "",
		HarvesterBufferSize: 0,
	}

	prospector := Prospector{
		ProspectorConfig: prospectorConfig,
	}

	prospector.Init()

	// Default values expected
	assert.Equal(t, config.DefaultIgnoreOlderDuration, prospector.ProspectorConfig.IgnoreOlderDuration)
	assert.Equal(t, config.DefaultScanFrequency, prospector.ProspectorConfig.ScanFrequencyDuration)
}

func TestProspectorInitNotSet(t *testing.T) {

	prospectorConfig := config.ProspectorConfig{}

	prospector := Prospector{
		ProspectorConfig: prospectorConfig,
	}

	prospector.Init()

	// Default values expected
	assert.Equal(t, config.DefaultIgnoreOlderDuration, prospector.ProspectorConfig.IgnoreOlderDuration)
	assert.Equal(t, config.DefaultScanFrequency, prospector.ProspectorConfig.ScanFrequencyDuration)
	assert.Equal(t, config.DefaultHarvesterBufferSize, prospector.ProspectorConfig.HarvesterBufferSize)
	assert.Equal(t, config.DefaultTailOnRotate, prospector.ProspectorConfig.TailOnRotate)
}

func TestProspectorInitScanFrequency0(t *testing.T) {

	prospectorConfig := config.ProspectorConfig{
		ScanFrequency: "0s",
	}

	prospector := Prospector{
		ProspectorConfig: prospectorConfig,
	}

	prospector.Init()

	var zero time.Duration = 0
	// 0 expected
	assert.Equal(t, zero, prospector.ProspectorConfig.ScanFrequencyDuration)
}

func TestProspectorInitInvalidScanFrequency(t *testing.T) {

	prospectorConfig := config.ProspectorConfig{
		ScanFrequency: "abc",
	}

	prospector := Prospector{
		ProspectorConfig: prospectorConfig,
	}

	err := prospector.Init()
	assert.NotNil(t, err)
}

func TestProspectorInitInvalidIgnoreOlder(t *testing.T) {

	prospectorConfig := config.ProspectorConfig{
		IgnoreOlder: "abc",
	}

	prospector := Prospector{
		ProspectorConfig: prospectorConfig,
	}

	err := prospector.Init()
	assert.NotNil(t, err)
}
