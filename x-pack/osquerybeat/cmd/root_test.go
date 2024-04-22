// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/google/go-cmp/cmp"
)

func TestOsquerybeatCfg(t *testing.T) {
	matches, err := filepath.Glob("testdata/osquerycfg/*.in.json")
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

				cfg, err := osquerybeatCfg(&rawIn, &client.AgentInfo{ID: "abc7d0a8-ce04-4663-95da-ff6d537c268f", Version: "8.13.1"})
				if err != nil {
					t.Fatal(err)
				}
				got, err := cfgToArrMap(cfg)
				if err != nil {
					t.Fatal(err)
				}

				diff := cmp.Diff(want, got)
				if diff != "" {
					t.Fatal(diff)
				}
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
