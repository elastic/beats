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

// This file was contributed to by generative AI

//go:build integration

package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
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

func TestLogAsFilestreamContainerInput(t *testing.T) {
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	eventsCount := 50
	logDir := filepath.Join(filebeat.TempDir(), "containers")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatalf("cannot create container logs directory: %s", err)
	}

	stdoutFile := filepath.Join(logDir, "container-stdout.log")
	stderrFile := filepath.Join(logDir, "container-stderr.log")
	integration.WriteDockerJSONLog(t, stdoutFile, eventsCount, []string{"stdout"}, false)
	integration.WriteDockerJSONLog(t, stderrFile, eventsCount, []string{"stderr"}, false)

	cfg := getConfig(
		t,
		map[string]any{
			"logfile": filepath.Join(logDir, "*.log"),
		},
		filepath.Join("run_as_filestream"),
		"run_as_container.yml")

	// Write configuration file and start Filebeat
	filebeat.WriteConfigFile(cfg)
	filebeat.Start()

	filebeat.WaitLogsContains(
		"Log input (deprecated) running as Filestream input",
		10*time.Second,
		"Filestream input did not start",
	)

	events := integration.GetEventsFromFileOutput[BeatEvent](filebeat, eventsCount*2, true)
	streamCounts := map[string]int{
		"stdout": 0,
		"stderr": 0,
	}
	for i, ev := range events {
		if ev.Input.Type != "container" {
			t.Errorf("Event %d expecting type 'container', got %q", i, ev.Input.Type)
		}

		if !strings.HasPrefix(ev.Message, "message ") {
			t.Errorf("Event %d: unexpected message %q", i, ev.Message)
		}

		if _, ok := streamCounts[ev.Stream]; !ok {
			t.Errorf("Event %d: unexpected stream %q", i, ev.Stream)
		} else {
			streamCounts[ev.Stream]++
		}

		if !slices.Contains(ev.Tags, "take_over") {
			t.Errorf("Event %d: 'take_over' tag not present", i)
		}
	}

	if streamCounts["stdout"] != eventsCount {
		t.Errorf("expecting %d events from stdout, got %d", eventsCount, streamCounts["stdout"])
	}
	if streamCounts["stderr"] != eventsCount {
		t.Errorf("expecting %d events from stderr, got %d", eventsCount, streamCounts["stderr"])
	}
}

func TestLogAsFilestreamContainerInputMixedFile(t *testing.T) {
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	eventsCount := 50
	logDir := filepath.Join(filebeat.TempDir(), "containers")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatalf("cannot create container logs directory: %s", err)
	}

	inputFile := filepath.Join(logDir, "container-stdout.log")
	integration.WriteDockerJSONLog(t, inputFile, eventsCount, []string{"stdout", "stderr"}, false)

	cfgMap := map[string]any{
		"logfile":    filepath.Join(logDir, "*.log"),
		"filestream": false,
	}

	cfgStr := getConfig(t, cfgMap, "run_as_filestream", "run_as_container_mixed.yml")

	// Write configuration file and start Filebeat
	filebeat.WriteConfigFile(cfgStr)
	filebeat.Start()

	assertContainerEvents(t, filebeat, eventsCount/2, eventsCount/2, false)
	filebeat.Stop()

	filebeat.RemoveLogFiles()
	filebeat.RemoveOutputFile()

	cfgMap["filestream"] = true
	cfgStr = getConfig(t, cfgMap, "run_as_filestream", "run_as_container_mixed.yml")
	filebeat.WriteConfigFile(cfgStr)

	filebeat.Start()
	filebeat.WaitLogsContains(
		"Log input (deprecated) running as Filestream",
		10*time.Second,
		"Filestream input did not start",
	)

	integration.WriteDockerJSONLog(t, inputFile, eventsCount, []string{"stdout", "stderr"}, true)

	filebeat.WaitLogsContains("End of file reached", 5*time.Second, "did not read file til end")
	filebeat.WaitPublishedEvents(5*time.Second, eventsCount)

	// Wait a few extra seconds to ensure no other events have been published
	time.Sleep(2 * time.Second)
	assertContainerEvents(t, filebeat, eventsCount/2, eventsCount/2, true)
}

// assertContainerEvents waits until the desired number of events is published,
// then checks the events for the stream key.
func assertContainerEvents(
	t *testing.T,
	filebeat *integration.BeatProc,
	stderrEvents, stdoutEvents int,
	containsTakeOverTag bool,
) {
	t.Helper()
	eventsCount := stderrEvents + stdoutEvents

	filebeat.WaitPublishedEvents(5*time.Second, eventsCount)
	events := integration.GetEventsFromFileOutput[BeatEvent](filebeat, eventsCount, true)
	streamCounts := map[string]int{
		"stdout": 0,
		"stderr": 0,
	}
	for i, ev := range events {
		if ev.Input.Type != "container" {
			t.Errorf("Event %d expecting type 'container', got %q", i, ev.Input.Type)
		}

		if !strings.HasPrefix(ev.Message, "message ") {
			t.Errorf("Event %d: unexpected message %q", i, ev.Message)
		}

		if _, ok := streamCounts[ev.Stream]; !ok {
			t.Errorf("Event %d: unexpected stream %q", i, ev.Stream)
		} else {
			streamCounts[ev.Stream]++
		}

		if slices.Contains(ev.Tags, "take_over") != containsTakeOverTag {
			t.Errorf(
				"Event %d: take_over tag present = %t, expected %t. Tags: %v",
				i,
				slices.Contains(ev.Tags, "take_over"),
				containsTakeOverTag,
				ev.Tags,
			)
		}
	}

	if streamCounts["stdout"] != stdoutEvents {
		t.Errorf("expecting %d events from stdout, got %d", stdoutEvents, streamCounts["stdout"])
	}
	if streamCounts["stderr"] != stderrEvents {
		t.Errorf("expecting %d events from stderr, got %d", stderrEvents, streamCounts["stderr"])
	}
}

func TestLogAsFilestreamContainerInputNoFeatureFlag(t *testing.T) {
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	eventsCount := 50
	logDir := filepath.Join(filebeat.TempDir(), "containers")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatalf("cannot create container logs directory: %s", err)
	}

	stdoutFile := filepath.Join(logDir, "container-stdout.log")
	stderrFile := filepath.Join(logDir, "container-stderr.log")
	integration.WriteDockerJSONLog(t, stdoutFile, eventsCount, []string{"stdout"}, false)
	integration.WriteDockerJSONLog(t, stderrFile, eventsCount, []string{"stderr"}, false)

	cfg := getConfig(
		t,
		map[string]any{
			"logfile": filepath.Join(logDir, "*.log"),
		},
		filepath.Join("run_as_filestream"),
		"run_as_container_no_feature_flag.yml")

	// Write configuration file and start Filebeat
	filebeat.WriteConfigFile(cfg)
	filebeat.Start()

	events := integration.GetEventsFromFileOutput[BeatEvent](filebeat, eventsCount*2, true)
	streamCounts := map[string]int{
		"stdout": 0,
		"stderr": 0,
	}
	for i, ev := range events {
		if ev.Input.Type != "container" {
			t.Errorf("Event %d expecting type 'container', got %q", i, ev.Input.Type)
		}

		if !strings.HasPrefix(ev.Message, "message ") {
			t.Errorf("Event %d: unexpected message %q", i, ev.Message)
		}

		if _, ok := streamCounts[ev.Stream]; !ok {
			t.Errorf("Event %d: unexpected stream %q", i, ev.Stream)
		} else {
			streamCounts[ev.Stream]++
		}

		if slices.Contains(ev.Tags, "take_over") {
			t.Errorf("Event %d: 'take_over' tag must not be present", i)
		}
	}

	if streamCounts["stdout"] != eventsCount {
		t.Errorf("expecting %d events from stdout, got %d", eventsCount, streamCounts["stdout"])
	}
	if streamCounts["stderr"] != eventsCount {
		t.Errorf("expecting %d events from stderr, got %d", eventsCount, streamCounts["stderr"])
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
	Message string `json:"message"`
	Stream  string `json:"stream"`
	Log     struct {
		File struct {
			Fingerprint string `json:"fingerprint"`
		} `json:"file"`
	} `json:"log"`
	Tags []string `json:"tags"`
}
