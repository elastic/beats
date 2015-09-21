package crawler

import (
	"github.com/elastic/filebeat/config"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestProspectorInit(t *testing.T) {

	prospectorConfig := config.ProspectorConfig{
		ScanFrequency: "15s",
		IgnoreOlder:   "100m",
	}

	prospector := Prospector{
		ProspectorConfig: prospectorConfig,
	}

	assert.NotNil(t, prospector)

	prospector.Init()

	// Predefined values expected
	assert.Equal(t, 100*time.Minute, prospector.ProspectorConfig.IgnoreOlderDuration)
	assert.Equal(t, 15*time.Second, prospector.ProspectorConfig.ScanFrequencyDuration)
}

func TestProspectorInitEmpty(t *testing.T) {

	prospectorConfig := config.ProspectorConfig{
		ScanFrequency: "",
		IgnoreOlder:   "",
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
