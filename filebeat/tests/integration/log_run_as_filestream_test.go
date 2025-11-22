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
	"fmt"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

func TestLogAsFilestreamRunsLogInput(t *testing.T) {
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	logfile := filepath.Join(filebeat.TempDir(), "log.log")
	integration.WriteLogFile(t, logfile, 50, false, "")

	cfg := getConfig(
		t,
		map[string]any{
			"logfile": logfile,
		},
		filepath.Join("run_as_filestream"),
		"run_as_log.yml")

	// Write configuration file and start Filebeat
	filebeat.WriteConfigFile(cfg)
	filebeat.Start()

	// Ensure the Log input is started
	filebeat.WaitLogsContains(
		"Log input (deprecated) running as Log input (deprecated)",
		10*time.Second,
		"Log input did not start",
	)

	events := integration.GetEventsFromFileOutput[BeatEvent](filebeat, 50, true)
	for i, ev := range events {
		if ev.Input.Type != "log" {
			t.Errorf("Event %d expecting type 'log', got %q", i, ev.Input.Type)
		}

		if len(ev.Log.File.Fingerprint) != 0 {
			t.Errorf("Event %d fingerprint must be empty", i)
		}
	}
}

func TestLogAsFilestreamFeatureFlag(t *testing.T) {
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
		"run_as_filestream.yml")

	// Write configuration file and start Filebeat
	filebeat.WriteConfigFile(cfg)
	filebeat.Start()

	// Ensure the Log input is started
	filebeat.WaitLogsContains(
		"Log input (deprecated) running as Filestream input",
		10*time.Second,
		"Filestream input did not start",
	)

	events := integration.GetEventsFromFileOutput[BeatEvent](filebeat, eventsCount, true)
	for i, ev := range events {
		if ev.Input.Type != "log" {
			t.Errorf("Event %d expecting type 'log', got %q", i, ev.Input.Type)
		}

		if !slices.Contains(ev.Tags, "take_over") {
			t.Errorf("Event %d: 'take_over' tag not present", i)
		}
	}
}

func TestLogAsFilestreamSupportsFingerprint(t *testing.T) {
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	logfile := filepath.Join(filebeat.TempDir(), "log.log")
	integration.WriteLogFile(t, logfile, 50, false, "")

	cfg := getConfig(
		t,
		map[string]any{
			"logfile": logfile,
		},
		filepath.Join("run_as_filestream"),
		"fingerprint.yml")

	// Write configuration file and start Filebeat
	filebeat.WriteConfigFile(cfg)
	filebeat.Start()

	// Ensure the Log input is started
	filebeat.WaitLogsContains(
		"Log input (deprecated) running as Filestream input",
		10*time.Second,
		"Filestream input did not start",
	)

	events := integration.GetEventsFromFileOutput[BeatEvent](filebeat, 50, true)
	for i, ev := range events {
		if ev.Input.Type != "log" {
			t.Errorf("Event %d expecting type 'log', got %q", i, ev.Input.Type)
		}

		if len(ev.Log.File.Fingerprint) == 0 {
			t.Errorf("Event %d fingerprint cannot be empty", i)
		}

		if !slices.Contains(ev.Tags, "take_over") {
			t.Errorf("Event %d: 'take_over' tag not present", i)
		}
	}
}

func TestLogAsFilestreamCanMigrateState(t *testing.T) {
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	logfile := filepath.Join(filebeat.TempDir(), "log.log")
	integration.WriteLogFile(t, logfile, 50, false, "")

	cfg := getConfig(
		t,
		map[string]any{
			"logfile": logfile,
		},
		filepath.Join("run_as_filestream"),
		"run_as_log.yml")

	// Write configuration file and start Filebeat
	filebeat.WriteConfigFile(cfg)
	filebeat.Start()

	// Ensure the Log input is started
	filebeat.WaitLogsContains(
		"Log input (deprecated) running as Log input (deprecated)",
		10*time.Second,
		"Log input did not start",
	)

	filebeat.WaitPublishedEvents(5*time.Second, 50)

	filebeat.Stop()

	cfg = getConfig(
		t,
		map[string]any{
			"logfile": logfile,
		},
		filepath.Join("run_as_filestream"),
		"run_as_filestream.yml")

	// Write configuration with the feature flag enabled
	filebeat.WriteConfigFile(cfg)
	filebeat.RemoveLogFiles()
	filebeat.Start()

	filebeat.WaitLogsContains(
		fmt.Sprintf("End of file reached: %s; Backoff now.", logfile),
		5*time.Second,
		"Filestream did not reach EOF")

	// Ensure we still have 50 events in the output
	filebeat.WaitPublishedEvents(time.Second, 50)

	// Write more events
	integration.WriteLogFile(t, logfile, 10, true)
	// Ensure only the new events are ingested
	filebeat.WaitPublishedEvents(15*time.Second, 60)
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
	Tags []string `json:"tags"`
}
