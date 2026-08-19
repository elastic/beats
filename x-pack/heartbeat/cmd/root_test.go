// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
//go:build linux || synthetics

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/elastic/beats/v7/heartbeat/monitors/plugin"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// Test all required plugins are exported by this module, since it's the
// one imported by elastic-otel-collector beats bundle: https://github.com/elastic/beats/pull/39818
func TestRootCmdPlugins(t *testing.T) {
	t.Parallel()
	plugins := []string{"http", "tcp", "icmp", "browser"}
	for _, p := range plugins {
		t.Run(fmt.Sprintf("%s plugin", p), func(t *testing.T) {
			_, found := plugin.GlobalPluginsReg.Get(p)
			assert.True(t, found)
		})
	}
}

func TestHeartbeatbeatCfg(t *testing.T) {
	matches, err := filepath.Glob("testdata/*.in.json")
	if err != nil {
		t.Fatal(err)
	}

	for _, match := range matches {
		dir := filepath.Dir(match)
		key := strings.TrimSuffix(filepath.Base(match), `.in.json`)

		out := filepath.Join(dir, key+".out.json")
		t.Run(key, func(in, out string) func(t *testing.T) {
			return func(t *testing.T) {
				var rawIn proto.UnitExpectedConfig
				err := readRawIn(in, &rawIn)
				if err != nil {
					t.Fatal(err)
				}

				want, err := readOut(out)
				if err != nil {
					t.Fatal(err)
				}

				cfg, err := heartbeatCfg(&rawIn, &client.AgentInfo{ID: "abc7d0a8-ce04-4663-95da-ff6d537c268f", Version: "8.13.1"})
				if err != nil {
					t.Fatal(err)
				}
				got, err := cfgToArrMap(cfg)
				require.NoError(t, err)

				diff := cmp.Diff(want, got)
				assert.Empty(t, diff)
			}
		}(match, out))
	}
}

func readRawIn(filename string, rawIn *proto.UnitExpectedConfig) error {
	b, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, rawIn)
	return err
}

func readOut(filename string) (cfg []map[string]interface{}, err error) {
	b, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &cfg)
	if err != nil {
		return nil, err
	}
	return cfg, err
}

// TestHeartbeatCfg_BrowserParamsDottedKeys verifies that dotted keys in browser
// "params" are kept literal, while dotted keys elsewhere in the monitor config
// still expand into nested structure (see https://github.com/elastic/beats/issues/51685).
func TestHeartbeatCfg_BrowserParamsDottedKeys(t *testing.T) {
	tests := []struct {
		name         string
		streamFields map[string]interface{}
		assert       func(t *testing.T, monitor map[string]interface{})
	}{
		{
			name: "dotted param keys are preserved literally",
			streamFields: map[string]interface{}{
				"params": map[string]interface{}{
					"subdomain":             "value1",
					"subdomain.example.com": "value2",
				},
			},
			assert: func(t *testing.T, monitor map[string]interface{}) {
				params, ok := monitor["params"].(map[string]interface{})
				require.True(t, ok, "expected browser params map in monitor config")
				assert.Equal(t, "value1", params["subdomain"], "expected non-dotted param key to be preserved")
				assert.Equal(t, "value2", params["subdomain.example.com"], "expected dotted param key to be preserved literally")
			},
		},
		{
			name: "dotted keys outside params still expand",
			streamFields: map[string]interface{}{
				// A dotted key that is NOT a param must still expand to nested structure.
				"a.b.c": "nested",
				"params": map[string]interface{}{
					"subdomain.example.com": "literal",
				},
			},
			assert: func(t *testing.T, monitor map[string]interface{}) {
				// The non-param dotted key expands into nested maps.
				a, ok := monitor["a"].(map[string]interface{})
				require.True(t, ok, "expected dotted key 'a.b.c' to expand into nested maps")
				b, ok := a["b"].(map[string]interface{})
				require.True(t, ok, "expected 'a.b' to be a nested map")
				assert.Equal(t, "nested", b["c"], "expected 'a.b.c' to expand")

				// The param dotted key stays literal.
				params, ok := monitor["params"].(map[string]interface{})
				require.True(t, ok, "expected browser params map in monitor config")
				assert.Equal(t, "literal", params["subdomain.example.com"], "expected param dotted key to stay literal")
				assert.NotContains(t, params, "subdomain", "param dotted key must not expand")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := runBrowserCfg(t, tc.streamFields)
			tc.assert(t, got[0])
		})
	}
}

// runBrowserCfg loads the browser test fixture, merges the given fields into its
// first (browser) stream, runs it through heartbeatCfg, and returns the
// resulting monitor configs as maps.
func runBrowserCfg(t *testing.T, streamFields map[string]interface{}) []map[string]interface{} {
	t.Helper()

	var rawIn proto.UnitExpectedConfig
	err := readRawIn("testdata/simple-browser.in.json", &rawIn)
	require.NoError(t, err, "failed to read browser test fixture")

	sourceMap := rawIn.GetSource().AsMap()
	streams, ok := sourceMap["streams"].([]interface{})
	require.True(t, ok, "expected streams to be a slice")
	require.NotEmpty(t, streams, "expected at least one stream")

	browserStream, ok := streams[0].(map[string]interface{})
	require.True(t, ok, "expected browser stream to be an object")
	for k, v := range streamFields {
		browserStream[k] = v
	}

	rawIn.Source, err = structpb.NewStruct(sourceMap)
	require.NoError(t, err, "failed to rebuild proto source")

	cfg, err := heartbeatCfg(&rawIn, &client.AgentInfo{ID: "abc7d0a8-ce04-4663-95da-ff6d537c268f", Version: "8.13.1"})
	require.NoError(t, err, "heartbeatCfg returned an error")

	got, err := cfgToArrMap(cfg)
	require.NoError(t, err, "failed to convert configs to maps")
	require.NotEmpty(t, got, "expected at least one monitor config")

	return got
}

func cfgToArrMap(cfg []*reload.ConfigWithMeta) ([]map[string]interface{}, error) {
	res := make([]map[string]interface{}, 0, len(cfg))
	for _, c := range cfg {
		var m mapstr.M
		err := c.Config.Unpack(&m)
		if err != nil {
			return nil, err
		}
		res = append(res, map[string]interface{}(m))
	}
	return res, nil
}
