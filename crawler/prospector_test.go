package crawler

import (
	"github.com/elastic/filebeat/config"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestProspectorInit(t *testing.T) {

	fileConfig := config.FileConfig{
		ScanFrequency:       "15s",
		IgnoreOlder:         "100m",
		HarvesterBufferSize: 100,
		TailOnRotate:        true,
	}

	prospector := Prospector{
		FileConfig: fileConfig,
	}

	assert.NotNil(t, prospector)

	prospector.Init()

	// Predefined values expected
	assert.Equal(t, 100*time.Minute, prospector.FileConfig.IgnoreOlderDuration)
	assert.Equal(t, 15*time.Second, prospector.FileConfig.ScanFrequencyDuration)
	assert.Equal(t, 100, prospector.FileConfig.HarvesterBufferSize)
	assert.Equal(t, true, prospector.FileConfig.TailOnRotate)
}

func TestProspectorInitEmpty(t *testing.T) {

	fileConfig := config.FileConfig{
		ScanFrequency:       "",
		IgnoreOlder:         "",
		HarvesterBufferSize: 0,
	}

	prospector := Prospector{
		FileConfig: fileConfig,
	}

	prospector.Init()

	// Default values expected
	assert.Equal(t, config.DefaultIgnoreOlderDuration, prospector.FileConfig.IgnoreOlderDuration)
	assert.Equal(t, config.DefaultScanFrequency, prospector.FileConfig.ScanFrequencyDuration)
	assert.Equal(t, config.DefaultHarvesterBufferSize, prospector.FileConfig.HarvesterBufferSize)
}

func TestProspectorInitNotSet(t *testing.T) {

	fileConfig := config.FileConfig{}

	prospector := Prospector{
		FileConfig: fileConfig,
	}

	prospector.Init()

	// Default values expected
	assert.Equal(t, config.DefaultIgnoreOlderDuration, prospector.FileConfig.IgnoreOlderDuration)
	assert.Equal(t, config.DefaultScanFrequency, prospector.FileConfig.ScanFrequencyDuration)
	assert.Equal(t, config.DefaultHarvesterBufferSize, prospector.FileConfig.HarvesterBufferSize)
	assert.Equal(t, config.DefaultTailOnRotate, prospector.FileConfig.TailOnRotate)
}

func TestProspectorInitScanFrequency0(t *testing.T) {

	fileConfig := config.FileConfig{
		ScanFrequency: "0s",
	}

	prospector := Prospector{
		FileConfig: fileConfig,
	}

	prospector.Init()

	var zero time.Duration = 0
	// 0 expected
	assert.Equal(t, zero, prospector.FileConfig.ScanFrequencyDuration)
}

func TestProspectorInitInvalidScanFrequency(t *testing.T) {

	fileConfig := config.FileConfig{
		ScanFrequency: "abc",
	}

	prospector := Prospector{
		FileConfig: fileConfig,
	}

	err := prospector.Init()
	assert.NotNil(t, err)
}

func TestProspectorInitInvalidIgnoreOlder(t *testing.T) {

	fileConfig := config.FileConfig{
		IgnoreOlder: "abc",
	}

	prospector := Prospector{
		FileConfig: fileConfig,
	}

	err := prospector.Init()
	assert.NotNil(t, err)
}
