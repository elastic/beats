// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elasticsearchclient

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap"

	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

func TestConfigDecodeConnectionSettings(t *testing.T) {
	cfg := decodeConfig(t, map[string]any{
		"hosts":      []string{"es-a:9201", "https://es-b:9443"},
		"protocol":   "https",
		"path":       "/elastic",
		"parameters": map[string]any{"pretty": "true"},
		"headers":    map[string]any{"X-Elastic-Product-Origin": "elasticsearchclient"},
		"username":   "elastic",
		"password":   "secret",
		"ssl": map[string]any{
			"verification_mode": "none",
		},
		"timeout":                 "11s",
		"idle_connection_timeout": "4s",
	})

	require.NoError(t, cfg.Validate(), "complete connection settings must validate")
	assert.Equal(t, []string{"es-a:9201", "https://es-b:9443"}, cfg.Hosts, "hosts must be decoded explicitly")
	assert.Equal(t, "https", cfg.Protocol, "protocol must be decoded explicitly")
	assert.Equal(t, "/elastic", cfg.Path, "path must be decoded explicitly")
	assert.Equal(t, map[string]string{"pretty": "true"}, cfg.Parameters, "request parameters must be decoded")
	assert.Equal(t, map[string]string{"X-Elastic-Product-Origin": "elasticsearchclient"}, cfg.Headers, "headers must be decoded")
	assert.Equal(t, "elastic", cfg.Username, "basic auth username must be decoded")
	assert.Equal(t, "secret", cfg.Password, "basic auth password must be decoded")
	assert.Equal(t, 11*time.Second, cfg.Transport.Timeout, "request timeout must be decoded")
	assert.Equal(t, 4*time.Second, cfg.Transport.IdleConnTimeout, "idle timeout must be decoded")
	require.NotNil(t, cfg.Transport.TLS, "TLS settings must be decoded")
	assert.Equal(t, tlscommon.VerifyNone, cfg.Transport.TLS.VerificationMode, "TLS verification mode must use common TLS type")
}

func TestConfigDecodeRejectsOutputOnlySettings(t *testing.T) {
	config := defaultTestConfig(t)
	err := confmap.NewFromStringMap(map[string]any{
		"hosts":         []string{"localhost:9200"},
		"bulk_max_size": 1000,
	}).Unmarshal(config)
	require.Error(t, err, "output-only settings must not be accepted by the connection configuration")
	assert.Contains(t, err.Error(), "bulk_max_size", "decode error must name the rejected output-only setting")
}

func TestConfigAuthenticationValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr string
	}{
		{
			name:    "missing hosts",
			config:  Config{},
			wantErr: "at least one host",
		},
		{
			name:    "API key and basic auth",
			config:  Config{Hosts: []string{"localhost:9200"}, Username: "elastic", Password: "secret", APIKey: "id:key"},
			wantErr: "cannot set both",
		},
		{
			name:   "API key",
			config: Config{Hosts: []string{"localhost:9200"}, APIKey: "id:key"},
		},
		{
			name:   "basic auth",
			config: Config{Hosts: []string{"localhost:9200"}, Username: "elastic", Password: "secret"},
		},
		{
			name:   "partial basic auth is accepted",
			config: Config{Hosts: []string{"localhost:9200"}, Username: "elastic"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err, "valid authentication configuration must be accepted")
				return
			}
			require.Error(t, err, "invalid authentication configuration must be rejected")
			assert.Contains(t, err.Error(), tt.wantErr, "validation error must describe the invalid setting")
		})
	}
}

func TestTransportSettingsProxyAndDefaults(t *testing.T) {
	cfg := decodeConfig(t, map[string]any{
		"hosts":                   []string{"localhost:9200"},
		"proxy_url":               "http://proxy.example:8080",
		"proxy_headers":           map[string]any{"X-Proxy-Token": "token"},
		"timeout":                 "7s",
		"idle_connection_timeout": "9s",
	})
	transport := cfg.Transport
	require.NotNil(t, transport.Proxy.URL, "configured proxy URL must be retained")
	assert.Equal(t, "http://proxy.example:8080", transport.Proxy.URL.String(), "configured proxy URL must be retained")
	assert.Equal(t, "token", transport.Proxy.Headers["X-Proxy-Token"], "configured proxy headers must be retained")
	assert.False(t, transport.Proxy.Disable, "configured proxy must remain enabled")
	assert.Equal(t, 7*time.Second, transport.Timeout, "request timeout must be forwarded")
	assert.Equal(t, 9*time.Second, transport.IdleConnTimeout, "idle timeout must be forwarded")

	defaults := defaultTestConfig(t).Transport
	assert.Nil(t, defaults.Proxy.URL, "absent proxy URL must defer to environment proxy settings")
	assert.False(t, defaults.Proxy.Disable, "absent proxy settings must not disable environment proxy settings")

	disabled := decodeConfig(t, map[string]any{
		"hosts":         []string{"localhost:9200"},
		"proxy_disable": true,
	})
	assert.True(t, disabled.Transport.Proxy.Disable, "proxy_disable must prevent configured and environment proxy use")
}

func TestConfigValidateRejectsMalformedTransportSettings(t *testing.T) {
	config := defaultTestConfig(t)
	err := confmap.NewFromStringMap(map[string]any{
		"hosts":     []string{"localhost:9200"},
		"proxy_url": "http://%41:8080",
	}).Unmarshal(config)
	require.Error(t, err, "malformed proxy URL must be rejected")
	assert.Contains(t, err.Error(), "invalid URL escape", "proxy parsing error must identify the malformed URL")

	tests := []struct {
		name    string
		config  Config
		wantErr string
	}{
		{
			name:    "unsupported TLS verification mode",
			config:  Config{Hosts: []string{"localhost:9200"}, Transport: httpcommon.HTTPTransportSettings{TLS: &tlscommon.Config{VerificationMode: 99}}},
			wantErr: "unsupported verification mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			require.Error(t, err, "malformed transport settings must fail Collector configuration validation")
			assert.Contains(t, err.Error(), tt.wantErr, "validation error must identify the malformed transport setting")
		})
	}
}

func decodeConfig(t *testing.T, values map[string]any) *Config {
	t.Helper()
	config := defaultTestConfig(t)
	require.NoError(t, confmap.NewFromStringMap(values).Unmarshal(config), "connection configuration must decode")
	require.NoError(t, config.Validate(), "decoded connection configuration must validate")
	return config
}

func defaultTestConfig(t *testing.T) *Config {
	t.Helper()
	config, ok := createDefaultConfig().(*Config)
	require.True(t, ok, "default configuration must use the elasticsearchclient Config type")
	return config
}
