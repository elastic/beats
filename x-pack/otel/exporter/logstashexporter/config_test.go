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
	t.Run("ReturnsValidConfig", func(t *testing.T) {
		cfg := createDefaultConfig()
		require.NotNil(t, cfg)

		_, ok := cfg.(Config)
		require.True(t, ok, "createDefaultConfig should return Config type")
	})

	t.Run("ContainsExpectedDefaults", func(t *testing.T) {
		cfg := createDefaultConfig()
		config, ok := cfg.(Config)
		require.True(t, ok)

		configMap := map[string]any(config)

		// Verify some key default values from logstash.DefaultConfig()
		assert.Equal(t, false, configMap["loadbalance"])
		assert.Equal(t, uint64(2), configMap["pipelining"])
		assert.Equal(t, uint64(2048), configMap["bulk_max_size"])
		assert.Equal(t, false, configMap["slow_start"])
		assert.Equal(t, false, configMap["escape_html"])
	})

	t.Run("ReturnsNonEmptyConfig", func(t *testing.T) {
		cfg := createDefaultConfig()
		config, ok := cfg.(Config)
		require.True(t, ok)

		// Config should not be empty
		assert.NotEmpty(t, config)
	})
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

func TestUnpackLogstashConfig(t *testing.T) {
	t.Run("ValidConfig", func(t *testing.T) {
		// Create a valid configuration
		validConfigMap := map[string]any{
			"hosts":         []string{"localhost:5044"},
			"workers":       1,
			"loadbalance":   true,
			"bulk_max_size": 100,
			"slow_start":    true,
			"timeout":       "5s",
			"ttl":           "10s",
			"pipelining":    4,
			"max_retries":   3,
			"escape_html":   false,
		}

		cfg, err := config.NewConfigFrom(validConfigMap)
		require.NoError(t, err)

		result, err := unpackLogstashConfig(cfg)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify some key fields were unpacked correctly
		resultConfig := result.Config
		hostWorkerCfg := result.HostWorkerCfg
		assert.Equal(t, []string{"localhost:5044"}, hostWorkerCfg.Hosts)
		assert.Equal(t, 1, hostWorkerCfg.Workers)
		assert.True(t, resultConfig.LoadBalance)
		assert.Equal(t, 100, resultConfig.BulkMaxSize)
		assert.True(t, resultConfig.SlowStart)
	})

	t.Run("EmptyConfig", func(t *testing.T) {
		cfg, err := config.NewConfigFrom(map[string]any{})
		require.NoError(t, err)

		result, err := unpackLogstashConfig(cfg)
		if err != nil {
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "hosts", "error should mention missing hosts")
		} else {
			assert.NotNil(t, result)
		}
	})

	t.Run("ConfigWithRemovedPort", func(t *testing.T) {
		configMapWithPort := map[string]any{
			"hosts": []string{"localhost:5044"},
			"port":  5044,
		}

		cfg, err := config.NewConfigFrom(configMapWithPort)
		require.NoError(t, err)

		result, err := unpackLogstashConfig(cfg)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("InvalidConfigUnpackError", func(t *testing.T) {
		invalidConfigMap := map[string]any{
			"workers": "invalid_worker_count",
		}

		cfg, err := config.NewConfigFrom(invalidConfigMap)
		require.NoError(t, err)

		result, err := unpackLogstashConfig(cfg)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestParseLogstashConfigEdgeCases(t *testing.T) {
	t.Run("NilConfig", func(t *testing.T) {
		rawConfig, logstashConfig, err := parseLogstashConfig(nil)
		assert.Error(t, err)
		assert.Nil(t, rawConfig)
		assert.Nil(t, logstashConfig)
	})

	t.Run("ValidMinimalConfig", func(t *testing.T) {
		minimalConfig := Config(map[string]any{
			"hosts": []string{"localhost:5044"},
		})

		var componentConfig component.Config = minimalConfig
		rawConfig, logstashConfig, err := parseLogstashConfig(&componentConfig)
		require.NoError(t, err)
		assert.NotNil(t, rawConfig)
		assert.NotNil(t, logstashConfig)
		hostWorkerCfg := logstashConfig.HostWorkerCfg
		assert.Equal(t, []string{"localhost:5044"}, hostWorkerCfg.Hosts)
	})

	t.Run("ConfigWithAllFieldTypes", func(t *testing.T) {
		complexConfig := Config(map[string]any{
			"hosts":             []string{"host1:5044", "host2:5044"},
			"workers":           2,
			"loadbalance":       true,
			"bulk_max_size":     500,
			"slow_start":        false,
			"timeout":           "30s",
			"ttl":               "60s",
			"pipelining":        5,
			"compression_level": 6,
			"max_retries":       5,
			"escape_html":       true,
			"backoff": map[string]any{
				"init": "1s",
				"max":  "30s",
			},
			"ssl": map[string]any{
				"enabled":           true,
				"verification_mode": "full",
			},
		})

		var componentConfig component.Config = complexConfig
		rawConfig, logstashConfig, err := parseLogstashConfig(&componentConfig)
		require.NoError(t, err)
		assert.NotNil(t, rawConfig)
		assert.NotNil(t, logstashConfig)

		// Verify complex nested structures are parsed correctly
		resultConfig := logstashConfig.Config
		hostWorkerCfg := logstashConfig.HostWorkerCfg
		assert.Equal(t, []string{"host1:5044", "host2:5044"}, hostWorkerCfg.Hosts)
		assert.Equal(t, 2, hostWorkerCfg.Workers)
		assert.True(t, resultConfig.LoadBalance)
		assert.Equal(t, 500, resultConfig.BulkMaxSize)
	})
}

func TestConfigType(t *testing.T) {
	t.Run("ConfigTypeAssignment", func(t *testing.T) {
		configMap := map[string]any{
			"hosts":   []string{"localhost:5044"},
			"workers": 1,
		}

		config := Config(configMap)
		assert.Equal(t, []string{"localhost:5044"}, config["hosts"])
		assert.Equal(t, 1, config["workers"])
	})

	t.Run("ConfigTypeConversion", func(t *testing.T) {
		// Test conversion between Config and map[string]any
		originalMap := map[string]any{
			"key1": "value1",
			"key2": 42,
			"key3": true,
		}

		config := Config(originalMap)
		convertedMap := map[string]any(config)

		assert.Equal(t, originalMap, convertedMap)
	})
}

func newTestConfig(t *testing.T) *component.Config {
	defaultConfig := createDefaultConfig()
	var cfg Config
	if c, ok := defaultConfig.(Config); ok {
		cfg = c
	} else {
		t.Fatal("default config is not of type Config")
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
