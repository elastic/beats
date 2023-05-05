// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package artifact

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

func TestReload(t *testing.T) {
	type testCase struct {
		input                    string
		initialConfig            *Config
		expectedSourceURI        string
		expectedTargetDirectory  string
		expectedInstallDirectory string
		expectedDropDirectory    string
		expectedFingerprint      string
		expectedTLS              bool
		expectedTLSEnabled       bool
		expectedDisableProxy     bool
		expectedTimeout          time.Duration
	}
	defaultValues := DefaultConfig()
	testCases := []testCase{
		{
			input: `agent.download:
  sourceURI: "testing.uri"
  target_directory: "a/b/c"
  install_path: "i/p"
  drop_path: "d/p"
  proxy_disable: true
  timeout: 33s
  ssl.enabled: true
  ssl.ca_trusted_fingerprint: "my_finger_print"
`,
			initialConfig:            DefaultConfig(),
			expectedSourceURI:        "testing.uri",
			expectedTargetDirectory:  "a/b/c",
			expectedInstallDirectory: "i/p",
			expectedDropDirectory:    "d/p",
			expectedFingerprint:      "my_finger_print",
			expectedTLS:              true,
			expectedTLSEnabled:       true,
			expectedDisableProxy:     true,
			expectedTimeout:          33 * time.Second,
		},
		{
			input: `agent.download:
  sourceURI: "testing.uri"
`,
			initialConfig:            DefaultConfig(),
			expectedSourceURI:        "testing.uri",
			expectedTargetDirectory:  defaultValues.TargetDirectory,
			expectedInstallDirectory: defaultValues.InstallPath,
			expectedDropDirectory:    defaultValues.DropPath,
			expectedFingerprint:      "",
			expectedTLS:              defaultValues.TLS != nil,
			expectedTLSEnabled:       false,
			expectedDisableProxy:     defaultValues.Proxy.Disable,
			expectedTimeout:          defaultValues.Timeout,
		},
		{
			input: `agent.download:
  sourceURI: ""
`,
			initialConfig: &Config{
				SourceURI:             "testing.uri",
				HTTPTransportSettings: defaultValues.HTTPTransportSettings,
			},
			expectedSourceURI:        defaultValues.SourceURI, // fallback to default when set to empty
			expectedTargetDirectory:  defaultValues.TargetDirectory,
			expectedInstallDirectory: defaultValues.InstallPath,
			expectedDropDirectory:    defaultValues.DropPath,
			expectedFingerprint:      "",
			expectedTLS:              defaultValues.TLS != nil,
			expectedTLSEnabled:       false,
			expectedDisableProxy:     defaultValues.Proxy.Disable,
			expectedTimeout:          defaultValues.Timeout,
		},
		{
			input: ``,
			initialConfig: &Config{
				SourceURI:             "testing.uri",
				HTTPTransportSettings: defaultValues.HTTPTransportSettings,
			},
			expectedSourceURI:        defaultValues.SourceURI, // fallback to default when not set
			expectedTargetDirectory:  defaultValues.TargetDirectory,
			expectedInstallDirectory: defaultValues.InstallPath,
			expectedDropDirectory:    defaultValues.DropPath,
			expectedFingerprint:      "",
			expectedTLS:              defaultValues.TLS != nil,
			expectedTLSEnabled:       false,
			expectedDisableProxy:     defaultValues.Proxy.Disable,
			expectedTimeout:          defaultValues.Timeout,
		},
		{
			input: `agent.download:
  sourceURI: " "
`,
			initialConfig: &Config{
				SourceURI:             "testing.uri",
				HTTPTransportSettings: defaultValues.HTTPTransportSettings,
			},
			expectedSourceURI:        defaultValues.SourceURI, // fallback to default when set to whitespace
			expectedTargetDirectory:  defaultValues.TargetDirectory,
			expectedInstallDirectory: defaultValues.InstallPath,
			expectedDropDirectory:    defaultValues.DropPath,
			expectedFingerprint:      "",
			expectedTLS:              defaultValues.TLS != nil,
			expectedTLSEnabled:       false,
			expectedDisableProxy:     defaultValues.Proxy.Disable,
			expectedTimeout:          defaultValues.Timeout,
		},
		{
			input: `agent.download:
  source_uri: " "
`,
			initialConfig: &Config{
				SourceURI:             "testing.uri",
				HTTPTransportSettings: defaultValues.HTTPTransportSettings,
			},
			expectedSourceURI:        defaultValues.SourceURI, // fallback to default when set to whitespace
			expectedTargetDirectory:  defaultValues.TargetDirectory,
			expectedInstallDirectory: defaultValues.InstallPath,
			expectedDropDirectory:    defaultValues.DropPath,
			expectedFingerprint:      "",
			expectedTLS:              defaultValues.TLS != nil,
			expectedTLSEnabled:       false,
			expectedDisableProxy:     defaultValues.Proxy.Disable,
			expectedTimeout:          defaultValues.Timeout,
		},
		{
			input: `agent.download:
  source_uri: " "
  sourceURI: " "
`,
			initialConfig:            DefaultConfig(),
			expectedSourceURI:        defaultValues.SourceURI, // fallback to default when set to whitespace
			expectedTargetDirectory:  defaultValues.TargetDirectory,
			expectedInstallDirectory: defaultValues.InstallPath,
			expectedDropDirectory:    defaultValues.DropPath,
			expectedFingerprint:      "",
			expectedTLS:              defaultValues.TLS != nil,
			expectedTLSEnabled:       false,
			expectedDisableProxy:     defaultValues.Proxy.Disable,
			expectedTimeout:          defaultValues.Timeout,
		},
		{
			input: ``,
			initialConfig: &Config{
				SourceURI:             "testing.uri",
				HTTPTransportSettings: defaultValues.HTTPTransportSettings,
			},
			expectedSourceURI:        defaultValues.SourceURI,
			expectedTargetDirectory:  defaultValues.TargetDirectory,
			expectedInstallDirectory: defaultValues.InstallPath,
			expectedDropDirectory:    defaultValues.DropPath,
			expectedFingerprint:      "",
			expectedTLS:              defaultValues.TLS != nil,
			expectedTLSEnabled:       false,
			expectedDisableProxy:     defaultValues.Proxy.Disable,
			expectedTimeout:          defaultValues.Timeout,
		},
		{
			input: `agent.download:
  source_uri: " "
  sourceURI: "testing.uri"
`,
			initialConfig:            DefaultConfig(),
			expectedSourceURI:        "testing.uri",
			expectedTargetDirectory:  defaultValues.TargetDirectory,
			expectedInstallDirectory: defaultValues.InstallPath,
			expectedDropDirectory:    defaultValues.DropPath,
			expectedFingerprint:      "",
			expectedTLS:              defaultValues.TLS != nil,
			expectedTLSEnabled:       false,
			expectedDisableProxy:     defaultValues.Proxy.Disable,
			expectedTimeout:          defaultValues.Timeout,
		},
		{
			input: `agent.download:
  source_uri: "testing.uri"
  sourceURI: " "
`,
			initialConfig:            DefaultConfig(),
			expectedSourceURI:        "testing.uri",
			expectedTargetDirectory:  defaultValues.TargetDirectory,
			expectedInstallDirectory: defaultValues.InstallPath,
			expectedDropDirectory:    defaultValues.DropPath,
			expectedFingerprint:      "",
			expectedTLS:              defaultValues.TLS != nil,
			expectedTLSEnabled:       false,
			expectedDisableProxy:     defaultValues.Proxy.Disable,
			expectedTimeout:          defaultValues.Timeout,
		},
		{
			input: `agent.download:
  source_uri: "testing.uri"
  sourceURI: "another.uri"
`,
			initialConfig:            DefaultConfig(),
			expectedSourceURI:        "testing.uri",
			expectedTargetDirectory:  defaultValues.TargetDirectory,
			expectedInstallDirectory: defaultValues.InstallPath,
			expectedDropDirectory:    defaultValues.DropPath,
			expectedFingerprint:      "",
			expectedTLS:              defaultValues.TLS != nil,
			expectedTLSEnabled:       false,
			expectedDisableProxy:     defaultValues.Proxy.Disable,
			expectedTimeout:          defaultValues.Timeout,
		},
	}

	l, _ := logger.NewWithLogpLevel("t", logp.ErrorLevel, false)
	for _, tc := range testCases {
		cfg := tc.initialConfig
		reloader := NewReloader(cfg, l)

		c, err := config.NewConfigFrom(tc.input)
		require.NoError(t, err)

		require.NoError(t, reloader.Reload(c))

		require.Equal(t, tc.expectedSourceURI, cfg.SourceURI)
		require.Equal(t, tc.expectedTargetDirectory, cfg.TargetDirectory)
		require.Equal(t, tc.expectedInstallDirectory, cfg.InstallPath)
		require.Equal(t, tc.expectedDropDirectory, cfg.DropPath)
		require.Equal(t, tc.expectedTimeout, cfg.Timeout)

		require.Equal(t, tc.expectedDisableProxy, cfg.Proxy.Disable)

		if tc.expectedTLS {
			require.NotNil(t, cfg.TLS)
			require.Equal(t, tc.expectedTLSEnabled, *cfg.TLS.Enabled)
		} else {
			require.Nil(t, cfg.TLS)
		}
	}
}
