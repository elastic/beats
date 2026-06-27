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
