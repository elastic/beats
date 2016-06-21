// +build !integration

package spooler

import (
	"testing"
	"time"

	cfg "github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/libbeat/common"
	"github.com/stretchr/testify/assert"
)

func load(t *testing.T, in string) cfg.FilebeatConfig {
	yaml, err := common.NewConfigWithYAML([]byte(in), "")
	if err != nil {
		t.Fatalf("Failed to parse config input: %v", err)
	}

	var config cfg.FilebeatConfig
	err = yaml.Unpack(&config)
	if err != nil {
		t.Fatalf("Failed to unpack config: %v", err)
	}

	return config
}

func TestNewSpoolerDefaultConfig(t *testing.T) {
	config := load(t, "")

	// Read from empty yaml config
	spooler, err := New(config, nil)

	assert.NoError(t, err)
	assert.Equal(t, cfg.DefaultSpoolSize, spooler.spoolSize)
	assert.Equal(t, cfg.DefaultIdleTimeout, spooler.idleTimeout)
}

func TestNewSpoolerSpoolSize(t *testing.T) {
	spoolSize := uint64(19)
	config := cfg.FilebeatConfig{SpoolSize: spoolSize}
	spooler, err := New(config, nil)

	assert.NoError(t, err)
	assert.Equal(t, spoolSize, spooler.spoolSize)
}

func TestNewSpoolerIdleTimeout(t *testing.T) {
	config := load(t, "idle_timeout: 10s")
	spooler, err := New(config, nil)

	assert.NoError(t, err)
	assert.Equal(t, time.Duration(10*time.Second), spooler.idleTimeout)
}
