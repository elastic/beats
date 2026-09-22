// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package hbreceiver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/elastic-agent-libs/config"
	ucfg "github.com/elastic/go-ucfg"
)

func TestExtractBrowserMonitorParams(t *testing.T) {
	t.Run("extracts dotted params from browser monitor", func(t *testing.T) {
		beatconfig := map[string]any{
			"heartbeat": map[string]any{
				"monitors": []any{
					map[string]any{
						"type":     "browser",
						"id":       "test-browser",
						"schedule": "@every 10m",
						"params": map[string]any{
							"subdomain.example.com": "literal-value",
							"plain":                 "also-fine",
						},
					},
				},
			},
		}

		extracted, err := extractBrowserMonitorParams(beatconfig)
		require.NoError(t, err)
		require.Len(t, extracted, 1)

		hb, ok := beatconfig["heartbeat"].(map[string]any)
		require.True(t, ok)
		monitorsSlice, ok := hb["monitors"].([]any)
		require.True(t, ok)
		monitor, ok := monitorsSlice[0].(map[string]any)
		require.True(t, ok)
		_, hasParams := monitor["params"]
		assert.False(t, hasParams, "params should be removed from beatconfig before ucfg parse")

		var got map[string]any
		require.NoError(t, extracted[0].Unpack(&got))
		assert.Equal(t, "literal-value", got["subdomain.example.com"])
		assert.Equal(t, "also-fine", got["plain"])
	})

	t.Run("does not touch non-browser monitors", func(t *testing.T) {
		beatconfig := map[string]any{
			"heartbeat": map[string]any{
				"monitors": []any{
					map[string]any{
						"type":   "http",
						"id":     "test-http",
						"params": map[string]any{"some.key": "value"},
					},
				},
			},
		}

		extracted, err := extractBrowserMonitorParams(beatconfig)
		require.NoError(t, err)
		assert.Empty(t, extracted)

		hb, ok := beatconfig["heartbeat"].(map[string]any)
		require.True(t, ok)
		monitorsSlice, ok := hb["monitors"].([]any)
		require.True(t, ok)
		monitor, ok := monitorsSlice[0].(map[string]any)
		require.True(t, ok)
		_, hasParams := monitor["params"]
		assert.True(t, hasParams, "params on non-browser monitors should be left alone")
	})

	t.Run("skips browser monitor without params", func(t *testing.T) {
		beatconfig := map[string]any{
			"heartbeat": map[string]any{
				"monitors": []any{
					map[string]any{"type": "browser", "id": "no-params"},
				},
			},
		}

		extracted, err := extractBrowserMonitorParams(beatconfig)
		require.NoError(t, err)
		assert.Empty(t, extracted)
	})

	t.Run("extracts multiple browser monitors at correct indices", func(t *testing.T) {
		beatconfig := map[string]any{
			"heartbeat": map[string]any{
				"monitors": []any{
					map[string]any{"type": "http", "id": "http-0"},
					map[string]any{
						"type":   "browser",
						"id":     "browser-1",
						"params": map[string]any{"a.b": "v1"},
					},
					map[string]any{
						"type":   "browser",
						"id":     "browser-2",
						"params": map[string]any{"c.d": "v2"},
					},
				},
			},
		}

		extracted, err := extractBrowserMonitorParams(beatconfig)
		require.NoError(t, err)
		require.Len(t, extracted, 2)
		assert.Nil(t, extracted[0], "http monitor must not be extracted")
		require.NotNil(t, extracted[1])
		require.NotNil(t, extracted[2])

		var got1, got2 map[string]any
		require.NoError(t, extracted[1].Unpack(&got1))
		require.NoError(t, extracted[2].Unpack(&got2))
		assert.Equal(t, "v1", got1["a.b"])
		assert.Equal(t, "v2", got2["c.d"])

		hb, ok := beatconfig["heartbeat"].(map[string]any)
		require.True(t, ok)
		monitorsSlice, ok := hb["monitors"].([]any)
		require.True(t, ok)
		monitor1, ok := monitorsSlice[1].(map[string]any)
		require.True(t, ok)
		monitor2, ok := monitorsSlice[2].(map[string]any)
		require.True(t, ok)
		_, hasParams1 := monitor1["params"]
		_, hasParams2 := monitor2["params"]
		assert.False(t, hasParams1, "params must be removed from browser monitor 1")
		assert.False(t, hasParams2, "params must be removed from browser monitor 2")
	})
}

// TestRestoreBrowserMonitorParams is the core regression test: it shows that
// dotted params keys survive the ucfg.NewFrom(..., PathSep(".")) call that
// NewBeatForReceiver performs.
func TestRestoreBrowserMonitorParams(t *testing.T) {
	rawParams := map[string]any{
		"subdomain.example.com": "literal-value",
		"plain":                 "also-fine",
	}

	beatconfig := map[string]any{
		"heartbeat": map[string]any{
			"monitors": []any{
				map[string]any{
					"type":     "browser",
					"id":       "test-browser",
					"schedule": "@every 10m",
					// params intentionally absent — simulates post-extract state
				},
			},
		},
	}

	// Pre-parse with PathSep("") to keep dotted keys literal.
	parsed, err := ucfg.NewFrom(rawParams, ucfg.PathSep(""))
	require.NoError(t, err)
	params := map[int]*conf.C{0: (*conf.C)(parsed)}

	// Simulate what NewBeatForReceiver does internally.
	tmp, err := ucfg.NewFrom(beatconfig, ucfg.PathSep("."))
	require.NoError(t, err)
	rawConfig := (*conf.C)(tmp)

	require.NoError(t, restoreBrowserMonitorParams(rawConfig, params))

	// Navigate to params and unpack.
	hbCfg, err := rawConfig.Child("heartbeat", -1)
	require.NoError(t, err)
	monitorCfg, err := hbCfg.Child("monitors", 0)
	require.NoError(t, err)
	paramsCfg, err := monitorCfg.Child("params", -1)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, paramsCfg.Unpack(&got))

	assert.Equal(t, "literal-value", got["subdomain.example.com"],
		"dotted key must be preserved as literal, not expanded to nested map")
	assert.Equal(t, "also-fine", got["plain"])

	// Confirm the bug: without the fix, ucfg would have produced a nested map.
	buggy, err := ucfg.NewFrom(map[string]any{"heartbeat": map[string]any{
		"monitors": []any{map[string]any{"type": "browser", "params": rawParams}},
	}}, ucfg.PathSep("."))
	require.NoError(t, err)

	buggyHb, err := (*conf.C)(buggy).Child("heartbeat", -1)
	require.NoError(t, err)
	buggyMon, err := buggyHb.Child("monitors", 0)
	require.NoError(t, err)
	buggyParams, err := buggyMon.Child("params", -1)
	require.NoError(t, err)

	var buggyGot map[string]any
	require.NoError(t, buggyParams.Unpack(&buggyGot))
	assert.Nil(t, buggyGot["subdomain.example.com"],
		"without the fix, dotted key is gone (expanded to nested map)")
	assert.NotNil(t, buggyGot["subdomain"],
		"without the fix, first segment of expanded key appears instead")
}
