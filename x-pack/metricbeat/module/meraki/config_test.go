package meraki

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, "https://api.meraki.com", cfg.BaseURL)
	assert.Equal(t, "false", cfg.DebugMode)
	assert.Equal(t, time.Second*300, cfg.Period)
}

func TestConfigValidate(t *testing.T) {
	// Missing BaseURL
	cfg := &config{ApiKey: "key", Organizations: []string{"org"}, Period: 10 * time.Second}
	cfg.BaseURL = ""
	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "apiBaseURL is required")

	// Missing ApiKey
	cfg = &config{BaseURL: "url", Organizations: []string{"org"}, Period: 10 * time.Second}
	cfg.ApiKey = ""
	err = cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "apiKey is required")

	// Missing Organizations
	cfg = &config{BaseURL: "url", ApiKey: "key", Organizations: []string{}, Period: 10 * time.Second}
	err = cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "organizations is required")

	// Period too long
	cfg = &config{BaseURL: "url", ApiKey: "key", Organizations: []string{"org"}, Period: 301 * time.Second}
	err = cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "maximum allowed collection period")

	// Valid config
	cfg = &config{BaseURL: "url", ApiKey: "key", Organizations: []string{"org"}, Period: 10 * time.Second}
	err = cfg.Validate()
	assert.NoError(t, err)
}
