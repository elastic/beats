// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package logstashexporter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"

	"github.com/elastic/go-ucfg"

	"github.com/elastic/elastic-agent-libs/config"
)

func TestCreateDefaultConfig(t *testing.T) {
	_, ok := createDefaultConfig().(*Config)
	require.True(t, ok)
}

func TestParseLogstashConfig(t *testing.T) {
	testingConfig := newTestConfig(t)
	expectedRawConfig, err := config.NewConfigFrom(testingConfig)
	require.NoError(t, err)

	expectedLogstashConfig := logstashOutputConfig{}
	require.NoError(t, expectedRawConfig.Unpack(&expectedLogstashConfig))

	rawConfig, logstashConfig, err := parseLogstashConfig(testingConfig)
	require.NoError(t, err)

	assert.Equal(t, expectedRawConfig, rawConfig)
	assert.Equal(t, expectedLogstashConfig, *logstashConfig)
}

func TestParseLogstashConfigInvalidConfig(t *testing.T) {
	invalidConfig := component.Config("invalid") // fail config.NewConfigFrom
	rawConfig, logstashConfig, err := parseLogstashConfig(&invalidConfig)

	assert.Error(t, err)
	var ucfgErr ucfg.Error
	require.ErrorAs(t, err, &ucfgErr)
	assert.Equal(t, ucfg.ErrTypeMismatch, ucfgErr.Reason())
	assert.Nil(t, rawConfig)
	assert.Nil(t, logstashConfig)
}

func newTestConfig(t *testing.T) *component.Config {
	defaultConfig := createDefaultConfig()
	var cfg Config
	if c, ok := defaultConfig.(*Config); ok {
		cfg = *c
	} else {
		t.Fatal("default config is not of type *Config")
	}
	cfg["hosts"] = []string{"localhost:5044"}
	cfg["workers"] = 1
	cfg["index"] = "test-index"
	cfg["loadbalance"] = true
	cfg["bulk_max_size"] = 100
	cfg["slow_start"] = true
	cfg["timeout"] = time.Second * 5
	cfg["ttl"] = time.Second * 10
	cfg["pipelining"] = 4
	cfg["compression_level"] = 5
	cfg["max_retries"] = 7
	cfg["escape_html"] = true
	cfg["backoff"] = map[string]any{
		"init": "2s",
		"max":  "1m",
	}
	cfg["ssl"] = map[string]any{
		"enabled":             false,
		"verification_mode":   "none",
		"supported_protocols": []string{"TLSv1.3"},
		"cipher_suites":       []string{"ECDHE-ECDSA-AES-128-CBC-SHA"},
		"certificate_authorities": []string{
			"/path/to/ca1",
			"/path/to/ca2",
		},
		"certificate":            "/path/to/cert",
		"key":                    "/path/to/key",
		"key_passphrase":         "passphrase",
		"key_passphrase_path":    "/path/to/key.pass",
		"curve_types":            []string{"P-256"},
		"renegotiation":          "never",
		"ca_sha256":              []string{"abc", "def"},
		"ca_trusted_fingerprint": "12:34:56:78:90:ab:cd:ef:12:34:56:78:90:ab:cd:ef:12:34:56:78",
	}
	cfg["proxy_url"] = "socks5://proxy:3128"
	cfg["proxy_use_local_resolver"] = false
	cfg["queue"] = map[string]any{}
	return &defaultConfig
}
