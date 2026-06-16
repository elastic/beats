// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build !integration

package config

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/cfgfile"
	conf "github.com/elastic/elastic-agent-libs/config"
)

func TestLoadConfig2(t *testing.T) {
	// Tests with different params from config file
	absPath, err := filepath.Abs("../tests/files/")

	assert.NotNil(t, absPath)
	assert.NoError(t, err)

	config := &Config{}

	// Reads second config file
	cfg, err := cfgfile.Load(absPath+"/config2.yml", nil)
	assert.NoError(t, err)
	err = cfg.Unpack(config)
	assert.NoError(t, err)
}

func TestEnabledInputs(t *testing.T) {
	stdinEnabled, err := conf.NewConfigFrom(map[string]interface{}{
		"type":    "stdin",
		"enabled": true,
	})
	if !assert.NoError(t, err) {
		return
	}

	udpDisabled, err := conf.NewConfigFrom(map[string]interface{}{
		"type":    "udp",
		"enabled": false,
	})
	if !assert.NoError(t, err) {
		return
	}

	logDisabled, err := conf.NewConfigFrom(map[string]interface{}{
		"type":    "log",
		"enabled": false,
	})
	if !assert.NoError(t, err) {
		return
	}

	t.Run("ListEnabledInputs", func(t *testing.T) {
		tests := []struct {
			name     string
			config   *Config
			expected []string
		}{
			{
				name:     "all inputs disabled",
				config:   &Config{Inputs: []*conf.C{udpDisabled, logDisabled}},
				expected: []string{},
			},
			{
				name:     "all inputs enabled",
				config:   &Config{Inputs: []*conf.C{stdinEnabled}},
				expected: []string{"stdin"},
			},
			{
				name:     "disabled and enabled inputs",
				config:   &Config{Inputs: []*conf.C{stdinEnabled, udpDisabled, logDisabled}},
				expected: []string{"stdin"},
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				assert.ElementsMatch(t, test.expected, test.config.ListEnabledInputs())
			})
		}
	})

	t.Run("IsInputEnabled", func(t *testing.T) {
		config := &Config{Inputs: []*conf.C{stdinEnabled, udpDisabled, logDisabled}}

		tests := []struct {
			name     string
			input    string
			expected bool
			config   *Config
		}{
			{name: "input exists and enabled", input: "stdin", expected: true, config: config},
			{name: "input exists and disabled", input: "udp", expected: false, config: config},
			{name: "input doesn't exist", input: "redis", expected: false, config: config},
			{name: "no inputs are enabled", input: "redis", expected: false, config: &Config{}},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				assert.Equal(t, test.expected, config.IsInputEnabled(test.input))
			})
		}
	})
}

func TestRegistryFileStorageUnpack(t *testing.T) {
	t.Run("no otel_file_storage section leaves FileStorage nil", func(t *testing.T) {
		raw, err := conf.NewConfigFrom(map[string]any{
			"registry": map[string]any{
				"backend": "otel_file_storage",
			},
		})
		require.NoError(t, err)

		cfg := DefaultConfig
		require.NoError(t, raw.Unpack(&cfg))

		assert.Equal(t, "otel_file_storage", cfg.Registry.Backend)
		assert.Nil(t, cfg.Registry.FileStorage)
	})

	t.Run("explicit otel_file_storage section unpacks as map", func(t *testing.T) {
		raw, err := conf.NewConfigFrom(map[string]any{
			"registry": map[string]any{
				"backend": "otel_file_storage",
				"otel_file_storage": map[string]any{
					"timeout":               "5s",
					"create_directory":      true,
					"directory_permissions": "0750",
					"fsync":                 true,
					"recreate":              true,
				},
			},
		})
		require.NoError(t, err)

		cfg := DefaultConfig
		require.NoError(t, raw.Unpack(&cfg))

		require.NotNil(t, cfg.Registry.FileStorage)
		assert.Equal(t, "5s", cfg.Registry.FileStorage["timeout"])
		assert.Equal(t, true, cfg.Registry.FileStorage["create_directory"])
		assert.Equal(t, "0750", cfg.Registry.FileStorage["directory_permissions"])
		assert.Equal(t, true, cfg.Registry.FileStorage["fsync"])
		assert.Equal(t, true, cfg.Registry.FileStorage["recreate"])
	})

	t.Run("partial otel_file_storage section contains only provided keys", func(t *testing.T) {
		raw, err := conf.NewConfigFrom(map[string]any{
			"registry": map[string]any{
				"backend": "otel_file_storage",
				"otel_file_storage": map[string]any{
					"create_directory":      true,
					"directory_permissions": "0700",
				},
			},
		})
		require.NoError(t, err)

		cfg := DefaultConfig
		require.NoError(t, raw.Unpack(&cfg))

		require.NotNil(t, cfg.Registry.FileStorage)
		assert.Len(t, cfg.Registry.FileStorage, 2)
		assert.Equal(t, true, cfg.Registry.FileStorage["create_directory"])
		assert.Equal(t, "0700", cfg.Registry.FileStorage["directory_permissions"])
	})

	t.Run("other registry fields preserved alongside otel_file_storage", func(t *testing.T) {
		raw, err := conf.NewConfigFrom(map[string]any{
			"registry": map[string]any{
				"path":             "custom/path",
				"file_permissions": 0o640,
				"flush":            "2s",
				"cleanup_interval": "10m",
				"backend":          "otel_file_storage",
				"otel_file_storage": map[string]any{
					"create_directory":      true,
					"directory_permissions": "0700",
				},
			},
		})
		require.NoError(t, err)

		cfg := DefaultConfig
		require.NoError(t, raw.Unpack(&cfg))

		assert.Equal(t, "custom/path", cfg.Registry.Path)
		assert.Equal(t, 0o640, int(cfg.Registry.Permissions))
		assert.Equal(t, 2*time.Second, cfg.Registry.FlushTimeout)
		assert.Equal(t, 10*time.Minute, cfg.Registry.CleanInterval)
		assert.Equal(t, "otel_file_storage", cfg.Registry.Backend)
		require.NotNil(t, cfg.Registry.FileStorage)
		assert.Equal(t, true, cfg.Registry.FileStorage["create_directory"])
	})
}
