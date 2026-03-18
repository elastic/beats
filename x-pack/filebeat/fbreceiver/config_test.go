// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fbreceiver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap"
)

func TestUnmarshal(t *testing.T) {
	t.Run("partial path override preserves defaults", func(t *testing.T) {
		cfg := &Config{
			Beatconfig: map[string]any{
				"path": map[string]any{
					"home": "/default/home",
					"data": "/default/data",
				},
			},
		}

		userConf := confmap.NewFromStringMap(map[string]any{
			"path.home": "/custom/home",
			"filebeat":  map[string]any{"inputs": []any{}},
		})

		require.NoError(t, cfg.Unmarshal(userConf))

		pathMap, ok := cfg.Beatconfig["path"].(map[string]any)
		require.True(t, ok, "path should be a map")
		assert.Equal(t, "/custom/home", pathMap["home"], "user override should win")
		assert.Equal(t, "/default/data", pathMap["data"], "unspecified default should be preserved")
		assert.Contains(t, cfg.Beatconfig, "filebeat")
	})

	t.Run("no defaults does not error", func(t *testing.T) {
		cfg := &Config{}

		userConf := confmap.NewFromStringMap(map[string]any{
			"filebeat": map[string]any{"inputs": []any{}},
		})

		require.NoError(t, cfg.Unmarshal(userConf))
		assert.Contains(t, cfg.Beatconfig, "filebeat")
	})

	t.Run("full path override replaces both", func(t *testing.T) {
		cfg := &Config{
			Beatconfig: map[string]any{
				"path": map[string]any{
					"home": "/default/home",
					"data": "/default/data",
				},
			},
		}

		userConf := confmap.NewFromStringMap(map[string]any{
			"path": map[string]any{
				"home": "/custom/home",
				"data": "/custom/data",
			},
			"filebeat": map[string]any{"inputs": []any{}},
		})

		require.NoError(t, cfg.Unmarshal(userConf))

		pathMap, ok := cfg.Beatconfig["path"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "/custom/home", pathMap["home"])
		assert.Equal(t, "/custom/data", pathMap["data"])
	})
}

func TestValidate(t *testing.T) {
	tests := map[string]struct {
		c           *Config
		hasError    bool
		errorString string
	}{
		"Empty config": {
			c:           &Config{Beatconfig: map[string]interface{}{}},
			hasError:    true,
			errorString: "configuration is required",
		},
		"No filebeat section": {
			c:           &Config{Beatconfig: map[string]interface{}{"other": map[string]interface{}{}}},
			hasError:    true,
			errorString: "configuration key 'filebeat' is required",
		},
		"Valid config": {
			c:           &Config{Beatconfig: map[string]interface{}{"filebeat": map[string]interface{}{}}},
			hasError:    false,
			errorString: "",
		},
	}
	for name, tc := range tests {
		err := tc.c.Validate()
		if tc.hasError {
			assert.Error(t, err, name)
			assert.Equal(t, err.Error(), tc.errorString, name)
		}
		if !tc.hasError {
			assert.NoError(t, err, name)
		}
	}
}
