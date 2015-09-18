package beat

import (
	"github.com/stretchr/testify/assert"
	cfg "github.com/elastic/filebeat/config"
	"testing"
)


func TestNewSpoolerDefaultConfig(t *testing.T) {

	config := cfg.FilebeatConfig{}
	fbConfig := &cfg.Config{Filebeat: config}

	fb := &Filebeat{FbConfig:fbConfig}
	NewSpooler(fb)


	assert.Equal(t, cfg.DefaultSpoolSize, fb.FbConfig.Filebeat.SpoolSize)
}

func TestNewSpoolerSpoolSize(t *testing.T) {

	spoolSize := uint64(19)
	config := cfg.FilebeatConfig{SpoolSize: spoolSize}

	fbConfig := &cfg.Config{Filebeat: config}

	fb := &Filebeat{FbConfig:fbConfig}
	NewSpooler(fb)


	assert.Equal(t, spoolSize, fb.FbConfig.Filebeat.SpoolSize)
}
