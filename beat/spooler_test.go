package beat

import (
	cfg "github.com/elastic/filebeat/config"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewSpoolerDefaultConfig(t *testing.T) {

	config := cfg.FilebeatConfig{}
	fbConfig := &cfg.Config{Filebeat: config}

	fb := &Filebeat{FbConfig: fbConfig}
	spooler := NewSpooler(fb)
	spooler.Init()

	assert.Equal(t, cfg.DefaultSpoolSize, fb.FbConfig.Filebeat.SpoolSize)
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
