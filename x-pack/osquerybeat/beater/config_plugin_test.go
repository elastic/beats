// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/testutil"
)

func buildConfigFilePath(dataPath string) string {
	return filepath.Join(dataPath, "osquery.conf")
}

func renderFullConfig(schedule map[string]osqueryConfigInfo) (map[string]string, error) {
	raw, err := newOsqueryConfig(schedule).render()
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
				testutil.AssertPanic(t, func() { NewConfigPlugin(tc.log, tc.dataPath) })
				return
			}

			p := NewConfigPlugin(tc.log, tc.dataPath)

			diff := cmp.Diff(tc.dataPath, p.dataPath)
			if diff != "" {
				t.Error(diff)
			}
			diff = cmp.Diff(buildConfigFilePath(tc.dataPath), p.getConfigFilePath())
			if diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestConfigPluginNoConfigFile(t *testing.T) {
	validLogger := logp.NewLogger("config_test")

	tempDirPath, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		os.RemoveAll(tempDirPath)
	}()

	p := NewConfigPlugin(validLogger, tempDirPath)
	diff := cmp.Diff(buildConfigFilePath(tempDirPath), p.getConfigFilePath())
	if diff != "" {
		t.Error(diff)
	}

	diff = cmp.Diff(0, p.Count())
	if diff != "" {
		t.Error(diff)
	}

	generatedConfig, err := p.GenerateConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// Expecting empty config with non-existent file
	expectedConfig, err := renderFullConfig(nil)
	if err != nil {
		t.Fatal(err)
	}

	diff = cmp.Diff(expectedConfig, generatedConfig)
	if diff != "" {
		t.Error(diff)
	}
}

var testQueries = []QueryConfig{
	{
		Name:     "users",
		Query:    "select * from users",
		Interval: 60,
	},
	{
		Name:     "uptime",
		Query:    "select * from uptime",
		Interval: 30,
	},
	{
		Name:     "processes",
		Query:    "select * from processes",
		Interval: 45,
	},
}

func convertQueriesToSchedule(queryConfigs []QueryConfig) map[string]osqueryConfigInfo {
	schedule := make(map[string]osqueryConfigInfo)

	for _, qc := range queryConfigs {
		schedule[qc.Name] = osqueryConfigInfo{
			Query:    qc.Query,
			Interval: qc.Interval,
			Platform: qc.Platform,
			Version:  qc.Version,
			Snapshot: true, // enforce snapshot for all queries
		}
	}
	return schedule
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

	p := NewConfigPlugin(validLogger, tempDirPath)
	diff := cmp.Diff(buildConfigFilePath(tempDirPath), p.getConfigFilePath())
	if diff != "" {
		t.Error(diff)
	}

	p.Set(testQueries)

	generatedConfig, err := p.GenerateConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// Test the expected configuration
	expectedConfig, err := renderFullConfig(convertQueriesToSchedule(testQueries))
	if err != nil {
		t.Fatal(err)
	}
	diff = cmp.Diff(expectedConfig, generatedConfig)
	if diff != "" {
		t.Error(diff)
	}

	// Create a new configuration plugin, test the configuration read from the file is correct
	p2 := NewConfigPlugin(validLogger, tempDirPath)
	generatedConfig2, err := p2.GenerateConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	diff = cmp.Diff(generatedConfig, generatedConfig2)
	if diff != "" {
		t.Error(diff)
	}
}
