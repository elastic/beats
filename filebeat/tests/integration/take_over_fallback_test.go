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
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

type fallbackEvent struct {
	Input struct {
		Type string `json:"type"`
	} `json:"input"`
	Message string `json:"message"`
	Log     struct {
		File struct {
			Path string `json:"path"`
		} `json:"file"`
	} `json:"log"`
}

func TestFilebeatTakeOverFallbackWithInputReload(t *testing.T) {
	const (
		initialLinesPerFile = 15
		logPhaseBatchSize   = 20
	)

	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	tempDir := filebeat.TempDir()
	inputsDir := filepath.Join(tempDir, "inputs.d")
	if err := os.MkdirAll(inputsDir, 0o777); err != nil {
		t.Fatalf("failed to create inputs directory: %s", err)
	}

	logFile1 := filepath.Join(tempDir, "log-1.log")
	logFile2 := filepath.Join(tempDir, "log-2.log")
	logFiles := []string{logFile1, logFile2}
	logPaths := []string{filepath.Join(tempDir, "*.log")}

	nextCounter := map[string]int{}
	nextCounter[logFile1] = integration.WriteLogFileFrom(t, logFile1, 0, initialLinesPerFile, false)
	nextCounter[logFile2] = integration.WriteLogFileFrom(t, logFile2, 0, initialLinesPerFile, false)

	appendLogsToFiles := func(n int) {
		for _, path := range logFiles {
			nextCounter[path] = integration.WriteLogFileFrom(t, path, nextCounter[path], n, true)
		}
	}

	cfg := getConfig(
		t,
		map[string]any{
			"inputsDir": inputsDir,
			"homePath":  tempDir,
		},
		"take-over-fallback", "base.yml")

	filebeat.WriteConfigFile(cfg)
	filebeat.Start()

	writeLogInputConfig(t, inputsDir, logPaths)

	appendLogsToFiles(logPhaseBatchSize)

	expectedEvents := (initialLinesPerFile + logPhaseBatchSize) * len(logFiles)
	events := integration.GetEventsFromFileOutput[fallbackEvent](filebeat, expectedEvents, true)

	lastSeen := map[string]map[string]int{
		"log": {},
	}
	for _, event := range events {
		if event.Input.Type != "log" {
			t.Fatalf(
				"at this point there can only be events from the Log "+
					"input, got events from %q",
				event.Input.Type,
			)
		}

		counter := counterFromMessage(t, event.Message)
		prev, exists := lastSeen["log"][event.Log.File.Path]
		if !exists || counter > prev {
			lastSeen["log"][event.Log.File.Path] = counter
		}
	}

	for _, path := range logFiles {
		if _, exists := lastSeen["log"][path]; !exists {
			t.Fatalf("no baseline lastSeen value captured for path %q", path)
		}
	}

	// Steps 9+ will continue from this baseline.
}

func writeLogInputConfig(t *testing.T, inputsDir string, paths []string) {
	content := getConfig(t, map[string]any{
		"paths": paths,
	}, "take-over-fallback", "log-input.yml")

	if err := os.WriteFile(filepath.Join(inputsDir, "active.yml"), []byte(content), 0o666); err != nil {
		t.Fatalf("failed to write log input config: %s", err)
	}
}

func counterFromMessage(t *testing.T, msg string) int {
	fields := strings.Fields(msg)
	if len(fields) == 0 {
		t.Fatalf("cannot parse counter from empty message %q", msg)
	}

	counter, err := strconv.Atoi(fields[len(fields)-1])
	if err != nil {
		t.Fatalf("cannot parse counter from message %q: %s", msg, err)
	}

	return counter
}
