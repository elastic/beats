// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/testutil"
)

func renderFullConfig(inputs []config.InputConfig) (map[string]string, error) {
	packs := make(map[string]pack)
	for _, input := range inputs {
		pack := pack{
			Queries: make(map[string]query),
		}
		for _, stream := range input.Streams {
			query := query{
				Query:    stream.Query,
				Interval: stream.Interval,
				Platform: stream.Platform,
				Version:  stream.Version,
				Snapshot: true, // enforce snapshot for all queries
			}
			pack.Queries[stream.ID] = query
		}
		packs[input.Name] = pack
	}
	raw, err := newOsqueryConfig(packs).render()
	if err != nil {
		return nil, err
	}

	return map[string]string{
		configName: string(raw),
	}, nil
}

func TestConfigPluginNew(t *testing.T) {
	validLogger := logp.NewLogger("config_test")

	tests := []struct {
		name        string
		log         *logp.Logger
		dataPath    string
		shouldPanic bool
	}{
		{
			name:        "invalid",
			log:         nil,
			dataPath:    "",
			shouldPanic: true,
		},
		{
			name:     "empty",
			log:      validLogger,
			dataPath: "",
		},
		{
			name:     "nonempty",
			log:      validLogger,
			dataPath: "data",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.shouldPanic {
				testutil.AssertPanic(t, func() { NewConfigPlugin(tc.log) })
				return
			}

			p := NewConfigPlugin(tc.log)
			if p == nil {
				t.Fatal("nil config plugin")
			}
		})
	}
}

var testInputConfigs = []config.InputConfig{
	{
		Name: "osquery_manager-1",
		Type: "osquery",
		Streams: []config.StreamConfig{
			{
				ID:       "users",
				Query:    "select * from users",
				Interval: 60,
			},
		},
	},
	{
		Name: "osquery_manager-2",
		Type: "osquery",
		Streams: []config.StreamConfig{
			{
				ID:       "uptime",
				Query:    "select * from uptime",
				Interval: 30,
			},
			{
				ID:       "processes",
				Query:    "select * from processes",
				Interval: 45,
			},
		},
	},
}

func TestConfigPluginWithConfig(t *testing.T) {
	validLogger := logp.NewLogger("config_test")
	tempDirPath, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		os.RemoveAll(tempDirPath)
	}()

	p := NewConfigPlugin(validLogger)

	p.Set(testInputConfigs)

	generatedConfig, err := p.GenerateConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// Test the expected configuration
	expectedConfig, err := renderFullConfig(testInputConfigs)
	if err != nil {
		t.Fatal(err)
	}
	diff := cmp.Diff(expectedConfig, generatedConfig)
	if diff != "" {
		t.Error(diff)
	}
}
