package beat

import (
	cfg "github.com/elastic/filebeat/config"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"testing"
)

func TestNewSpoolerDefaultConfig(t *testing.T) {

	var config cfg.FilebeatConfig
	// Read from empty yaml config
	yaml.Unmarshal([]byte(""), &config)
	fbConfig := &cfg.Config{Filebeat: config}

	fb := &Filebeat{FbConfig: fbConfig}
	spooler := NewSpooler(fb)
	spooler.Init()

	assert.Equal(t, cfg.DefaultSpoolSize, fb.FbConfig.Filebeat.SpoolSize)
	assert.Equal(t, cfg.DefaultIdleTimeout, fb.FbConfig.Filebeat.IdleTimeoutDuration)
}

func TestNewSpoolerSpoolSize(t *testing.T) {

	spoolSize := uint64(19)
	config := cfg.FilebeatConfig{SpoolSize: spoolSize}

	fbConfig := &cfg.Config{Filebeat: config}

	fb := &Filebeat{FbConfig: fbConfig}
	spooler := NewSpooler(fb)
	spooler.Init()

	assert.Equal(t, spoolSize, fb.FbConfig.Filebeat.SpoolSize)
}

func TestNewSpoolerIdleTimeout(t *testing.T) {

	idleTimoeout := "10s"
	config := cfg.FilebeatConfig{IdleTimeout: idleTimoeout}

	fbConfig := &cfg.Config{Filebeat: config}

	fb := &Filebeat{FbConfig: fbConfig}
	spooler := NewSpooler(fb)
	spooler.Init()

	assert.Equal(t, idleTimoeout, fb.FbConfig.Filebeat.IdleTimeout)
}
