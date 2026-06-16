// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package instance

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap"
)

func TestConvertPaths(t *testing.T) {
	t.Run("no dotted keys is a no-op", func(t *testing.T) {
		input := map[string]any{
			"filebeat": map[string]any{"inputs": []string{"a"}},
			"output":   "elasticsearch",
		}
		conf := confmap.NewFromStringMap(input)
		require.NoError(t, DeDotKeys(conf))

		assert.Equal(t, map[string]any{"inputs": []string{"a"}}, conf.ToStringMap()["filebeat"])
		assert.Equal(t, "elasticsearch", conf.ToStringMap()["output"])
	})

	t.Run("single dotted key becomes nested", func(t *testing.T) {
		input := map[string]any{
			"path.home": "/tmp/beats",
		}
		conf := confmap.NewFromStringMap(input)
		require.NoError(t, DeDotKeys(conf))

		got := conf.ToStringMap()
		assert.NotContains(t, got, "path.home")
		pathMap, ok := got["path"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "/tmp/beats", pathMap["home"])
	})

	t.Run("multi-level dotted key", func(t *testing.T) {
		input := map[string]any{
			"management.otel.enabled": true,
		}
		conf := confmap.NewFromStringMap(input)
		require.NoError(t, DeDotKeys(conf))

		got := conf.ToStringMap()
		assert.NotContains(t, got, "management.otel.enabled")
		mgmt, ok := got["management"].(map[string]any)
		require.True(t, ok)
		otel, ok := mgmt["otel"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, true, otel["enabled"])
	})

	t.Run("multiple dotted keys sharing a prefix", func(t *testing.T) {
		input := map[string]any{
			"path.home": "/home",
			"path.data": "/data",
		}
		conf := confmap.NewFromStringMap(input)
		require.NoError(t, DeDotKeys(conf))

		got := conf.ToStringMap()
		pathMap, ok := got["path"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "/home", pathMap["home"])
		assert.Equal(t, "/data", pathMap["data"])
	})

	t.Run("dotted and non-dotted keys coexist", func(t *testing.T) {
		input := map[string]any{
			"output":    "elasticsearch",
			"path.home": "/tmp",
		}
		conf := confmap.NewFromStringMap(input)
		require.NoError(t, DeDotKeys(conf))

		got := conf.ToStringMap()
		assert.Equal(t, "elasticsearch", got["output"])
		pathMap, ok := got["path"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "/tmp", pathMap["home"])
	})

	t.Run("empty conf", func(t *testing.T) {
		conf := confmap.New()
		require.NoError(t, DeDotKeys(conf))
		assert.Empty(t, conf.ToStringMap())
	})
}

func TestSetNested(t *testing.T) {
	t.Run("single part sets value directly", func(t *testing.T) {
		m := map[string]any{}
		setNested(m, []string{"key"}, "val")
		assert.Equal(t, "val", m["key"])
	})

	t.Run("two parts creates one level of nesting", func(t *testing.T) {
		m := map[string]any{}
		setNested(m, []string{"a", "b"}, 42)
		inner, ok := m["a"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, 42, inner["b"])
	})

	t.Run("deep nesting", func(t *testing.T) {
		m := map[string]any{}
		setNested(m, []string{"a", "b", "c", "d"}, "deep")
		a, ok := m["a"].(map[string]any)
		require.True(t, ok)
		b, ok := a["b"].(map[string]any)
		require.True(t, ok)
		c, ok := b["c"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "deep", c["d"])
	})

	t.Run("merges into existing intermediate map", func(t *testing.T) {
		m := map[string]any{
			"a": map[string]any{"existing": true},
		}
		setNested(m, []string{"a", "new"}, "value")
		inner, ok := m["a"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, true, inner["existing"])
		assert.Equal(t, "value", inner["new"])
	})

	t.Run("overwrites non-map intermediate", func(t *testing.T) {
		m := map[string]any{
			"a": "scalar",
		}
		setNested(m, []string{"a", "b"}, "value")
		inner, ok := m["a"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "value", inner["b"])
	})
}
