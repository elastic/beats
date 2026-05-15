// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package abreceiver

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
			"auditbeat": map[string]any{"modules": []any{}},
		})

		require.NoError(t, cfg.Unmarshal(userConf))

		pathMap, ok := cfg.Beatconfig["path"].(map[string]any)
		require.True(t, ok, "path should be a map")
		assert.Equal(t, "/custom/home", pathMap["home"], "user override should win")
		assert.Equal(t, "/default/data", pathMap["data"], "unspecified default should be preserved")
		assert.Contains(t, cfg.Beatconfig, "auditbeat")
	})

	t.Run("no defaults does not error", func(t *testing.T) {
		cfg := &Config{}

		userConf := confmap.NewFromStringMap(map[string]any{
			"auditbeat": map[string]any{"modules": []any{}},
		})

		require.NoError(t, cfg.Unmarshal(userConf))
		assert.Contains(t, cfg.Beatconfig, "auditbeat")
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
			"auditbeat": map[string]any{"modules": []any{}},
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
		"No auditbeat section": {
			c:           &Config{Beatconfig: map[string]interface{}{"other": map[string]interface{}{}}},
			hasError:    true,
			errorString: "configuration key 'auditbeat' is required",
		},
		"Valid config": {
			c:           &Config{Beatconfig: map[string]interface{}{"auditbeat": map[string]interface{}{}}},
			hasError:    false,
			errorString: "",
		},
	}
	for name, tc := range tests {
		err := tc.c.Validate()
		if tc.hasError {
			assert.Errorf(t, err, "%s failed, should have had error", name)
			assert.Equalf(t, err.Error(), tc.errorString, "%s failed, error not equal", name)
		} else {
			assert.NoErrorf(t, err, "%s failed, should not have error", name)
		}
	}
}
