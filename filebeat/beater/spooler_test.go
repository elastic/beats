package beater

import (
	"testing"
	"time"

	cfg "github.com/elastic/beats/filebeat/config"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func TestNewSpoolerDefaultConfig(t *testing.T) {
	var config cfg.FilebeatConfig
	// Read from empty yaml config
	yaml.Unmarshal([]byte(""), &config)
	spooler := NewSpooler(config, nil)

	assert.Equal(t, cfg.DefaultSpoolSize, spooler.spoolSize)
	assert.Equal(t, cfg.DefaultIdleTimeout, spooler.idleTimeout)
}

func TestNewSpoolerSpoolSize(t *testing.T) {
	spoolSize := uint64(19)
	config := cfg.FilebeatConfig{SpoolSize: spoolSize}
	spooler := NewSpooler(config, nil)

	assert.Equal(t, spoolSize, spooler.spoolSize)
}

func TestNewSpoolerIdleTimeout(t *testing.T) {
	var config cfg.FilebeatConfig
	yaml.Unmarshal([]byte("idle_timeout: 10s"), &config)
	spooler := NewSpooler(config, nil)

	assert.Equal(t, time.Duration(10*time.Second), spooler.idleTimeout)
}
