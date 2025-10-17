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

//go:build integration

package integration

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

func TestStandaloneLogInputIgnoresRunAsFilestream(t *testing.T) {
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	eventsCount := 50
	logfile := filepath.Join(filebeat.TempDir(), "log.log")
	integration.WriteLogFile(t, logfile, eventsCount, false, "")

	cfg := getConfig(
		t,
		map[string]any{
			"logfile": logfile,
		},
		filepath.Join("run_as_filestream"),
		"happy_path.yml")

	// Write configuration file and start Filebeat
	filebeat.WriteConfigFile(cfg)
	filebeat.Start()

	// Ensure the Filestream input is started
	filebeat.WaitLogsContains(
		"Log input running as Log input",
		10*time.Second,
		"Log input did not start",
	)

	events := integration.GetEventsFromFileOutput[BeatEvent](filebeat, eventsCount, true)
	for i, ev := range events {
		if ev.Input.Type != "log" {
			t.Errorf("Event %d expecting type 'log', got %q", i, ev.Input.Type)
		}

		if len(ev.Log.File.Fingerprint) != 0 {
			t.Errorf("Event %d fingerprint must be empty", i)
		}
	}
}

type BeatEvent struct {
	Input struct {
		Type string `json:"type"`
	} `json:"input"`
	Log struct {
		File struct {
			Fingerprint string `json:"fingerprint"`
		} `json:"file"`
	} `json:"log"`
}
