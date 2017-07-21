// +build !integration

package log

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/filebeat/harvester"
)

func TestCleanOlderError(t *testing.T) {
	config := config{
		CleanInactive: 10 * time.Hour,
	}

	err := config.Validate()
	assert.Error(t, err)
}

func TestCleanOlderIgnoreOlderError(t *testing.T) {
	config := config{
		CleanInactive: 10 * time.Hour,
		IgnoreOlder:   15 * time.Hour,
	}

	err := config.Validate()
	assert.Error(t, err)
}

func TestCleanOlderIgnoreOlderErrorEqual(t *testing.T) {
	config := config{
		CleanInactive: 10 * time.Hour,
		IgnoreOlder:   10 * time.Hour,
	}

	err := config.Validate()
	assert.Error(t, err)
}

func TestCleanOlderIgnoreOlder(t *testing.T) {
	config := config{
		CleanInactive: 10*time.Hour + defaultConfig.ScanFrequency + 1*time.Second,
		IgnoreOlder:   10 * time.Hour,
		Paths:         []string{"hello"},
		ForwarderConfig: harvester.ForwarderConfig{
			Type: "log",
		},
	}

	err := config.Validate()
	assert.NoError(t, err)
}
