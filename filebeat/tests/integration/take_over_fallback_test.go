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
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

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
	const batchSize = 25

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

	appendLogsToFiles := func(n int) {
		for _, path := range logFiles {
			nextCounter[path] = integration.WriteLogFileFrom(
				t,
				path,
				nextCounter[path],
				n,
				true,
				"================"+filepath.Base(path),
			)
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

	// 1. Add data to the log files and run the Log input
	writeLogInputConfig(t, inputsDir, logPaths)
	appendLogsToFiles(batchSize)

	expectedEvents := batchSize * len(logFiles)
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
	}
	_, logMaxByPath := countExtremesByPath(t, events, "log")
	lastSeen["log"] = logMaxByPath
	assertPathsPresent(t, lastSeen["log"], logFiles, "log baseline")

	// 2. Disable Log input and snapshot output file for debugging
	snapshotIdx := 0
	disableActiveInput(t, inputsDir, filebeat, "log")
	snapshotIdx++
	copyOutputSnapshot(t, tempDir, snapshotIdx, "log-1")

	// 3. Enable Filestream with take_over and ingest one deterministic batch
	writeFilestreamConfig(t, inputsDir, "take-over-from-log-input", logPaths)
	filebeat.WaitLogsContains("Input 'filestream' starting", 30*time.Second, "filestream runner did not start")
	appendLogsToFiles(batchSize)

	expectedEvents += batchSize * len(logFiles)
	filebeat.WaitPublishedEvents(30*time.Second, expectedEvents)
	events = integration.GetEventsFromFileOutput[fallbackEvent](filebeat, expectedEvents, true)

	// 4. Ensure Filestream did not duplicate data already ingested by Log input
	assertNoDuplicationFromPreviousInput(t, events, logFiles, lastSeen["log"], "filestream")

	// 5. Capture Filestream baseline, disable all inputs and snapshot.
	_, filestreamMaxByPath := countExtremesByPath(t, events, "filestream")
	lastSeen["filestream"] = filestreamMaxByPath
	assertPathsPresent(t, lastSeen["filestream"], logFiles, "filestream baseline")

	disableActiveInput(t, inputsDir, filebeat, "filestream")
	snapshotIdx++
	copyOutputSnapshot(t, tempDir, snapshotIdx, "filestream-1")

	// ========================= output "status" =========================
	// At this point, each input has ingested 25 events per file, and Filestream
	// correctly continued from where the log input stopped.
	// For each file:
	//   - events 00 ~ 24: Log input
	//   - events 25 ~ 49: Filestream input

	// 6. Re-enable Log input, ingest one deterministic batch and assert continuity.
	writeLogInputConfig(t, inputsDir, logPaths)
	appendLogsToFiles(batchSize)

	prevExpectedEvents := expectedEvents
	newLogEvents := 0
	for _, path := range logFiles {
		latestWrittenCounter := nextCounter[path] - 1
		newLogEvents += latestWrittenCounter - lastSeen["log"][path]
	}
	expectedEvents += newLogEvents
	filebeat.WaitPublishedEvents(30*time.Second, expectedEvents)
	events = integration.GetEventsFromFileOutput[fallbackEvent](filebeat, expectedEvents, true)
	assertContinuesFromLast(t, events[prevExpectedEvents:], logFiles, lastSeen["log"], "log")

	// 7. Update Log baseline, disable all inputs and snapshot.
	_, updatedLogMaxByPath := countExtremesByPath(t, events, "log")
	lastSeen["log"] = updatedLogMaxByPath
	assertPathsPresent(t, lastSeen["log"], logFiles, "updated log baseline")

	disableActiveInput(t, inputsDir, filebeat, "log")
	snapshotIdx++
	copyOutputSnapshot(t, tempDir, snapshotIdx, "log-2")

	// ========================= output "status" =========================
	// At this point, there was "one fallback", for the Log input, and only
	// the log input ingested the last batch of events.
	// For each file (in the order they appear in the output):
	//   - events 00 ~ 24: Log input
	//   - events 25 ~ 49: Filestream input
	//   - events 25 ~ 74: Log input

	// 8. Re-enable Filestream input, ingest one deterministic batch and assert continuity.
	writeFilestreamConfig(t, inputsDir, "take-over-from-log-input", logPaths)
	filebeat.WaitLogsContains("Input 'filestream' starting", 30*time.Second, "filestream runner did not start")
	appendLogsToFiles(batchSize)

	prevExpectedEvents = expectedEvents
	newFilestreamEvents := 0
	for _, path := range logFiles {
		latestWrittenCounter := nextCounter[path] - 1
		newFilestreamEvents += latestWrittenCounter - lastSeen["filestream"][path]
	}
	expectedEvents += newFilestreamEvents
	filebeat.WaitPublishedEvents(30*time.Second, expectedEvents)
	events = integration.GetEventsFromFileOutput[fallbackEvent](filebeat, expectedEvents, true)
	assertContinuesFromLast(t, events[prevExpectedEvents:], logFiles, lastSeen["filestream"], "filestream")

	// Step 19+ continues from this state.
}

// writeLogInputConfig renders and writes the active log input configuration file.
// It replaces inputs.d/active.yml with a config that reads the given paths.
func writeLogInputConfig(t *testing.T, inputsDir string, paths []string) {
	content := getConfig(t, map[string]any{
		"paths": paths,
	}, "take-over-fallback", "log-input.yml")

	if err := os.WriteFile(filepath.Join(inputsDir, "active.yml"), []byte(content), 0o666); err != nil {
		t.Fatalf("failed to write log input config: %s", err)
	}
}

// writeFilestreamConfig renders and writes the active filestream configuration
// file with takeover enabled for the provided input ID and paths.
func writeFilestreamConfig(t *testing.T, inputsDir, inputID string, paths []string) {
	content := getConfig(t, map[string]any{
		"inputID": inputID,
		"paths":   paths,
	}, "take-over-fallback", "filestream-input.yml")

	if err := os.WriteFile(filepath.Join(inputsDir, "active.yml"), []byte(content), 0o666); err != nil {
		t.Fatalf("failed to write filestream input config: %s", err)
	}
}

// counterFromMessage parses the trailing integer counter from a generated log
// message line. The helper fails the test when the message does not end with an
// integer token.
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

// disableActiveInput deactivates the current reload config by renaming
// inputs.d/active.yml to inputs.d/active.yml.disabled and waits until the
// corresponding runner stop message appears in Filebeat logs.
func disableActiveInput(t *testing.T, inputsDir string, filebeat *integration.BeatProc, input string) {
	activeCfg := filepath.Join(inputsDir, "active.yml")

	if err := os.Rename(activeCfg, activeCfg+".disabled"); err != nil {
		t.Fatalf("failed to disable active config %q: %s", activeCfg, err)
	}

	stopLogLine := fmt.Sprintf("Runner: 'input [type=%s]' has stopped", input)
	if input == "filestream" {
		stopLogLine = "Runner: 'filestream' has stopped"
	}

	filebeat.WaitLogsContains(stopLogLine, 30*time.Second, "input runner did not stop after disabling active config")
}

// copyOutputSnapshot copies the current output file to a phase snapshot name
// prefixed with a zero-padded incremental counter (for lexical sortability).
func copyOutputSnapshot(t *testing.T, tempDir string, snapshotIdx int, phase string) {
	matches, err := filepath.Glob(filepath.Join(tempDir, "output-file-*.ndjson"))
	if err != nil {
		t.Fatalf("failed to resolve output file glob: %s", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected exactly one output file, got %d", len(matches))
	}
	source := matches[0]
	snapshotPath := filepath.Join(tempDir, fmt.Sprintf("%02d-output-phase-%s.ndjson", snapshotIdx, phase))

	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("failed to read output file %q: %s", source, err)
	}

	if err := os.WriteFile(snapshotPath, data, 0o644); err != nil {
		t.Fatalf("failed to write snapshot file %q: %s", snapshotPath, err)
	}
}

// countExtremesByPath scans events from inputType and returns, per file path,
// the minimum and maximum counters found in message payloads. The two returned
// maps share the same key space and are used to validate handoff/continuation
// boundaries between test phases.
func countExtremesByPath(
	t *testing.T,
	events []fallbackEvent,
	inputType string,
) (minCounter map[string]int, maxCounter map[string]int) {
	minCounter = map[string]int{}
	maxCounter = map[string]int{}
	for _, event := range events {
		if event.Input.Type != inputType {
			continue
		}

		path := event.Log.File.Path
		counter := counterFromMessage(t, event.Message)
		if prev, ok := minCounter[path]; !ok || counter < prev {
			minCounter[path] = counter
		}
		if prev, ok := maxCounter[path]; !ok || counter > prev {
			maxCounter[path] = counter
		}
	}

	return minCounter, maxCounter
}

// assertPathsPresent verifies that valuesByPath has an entry for every file in
// logFiles. It is used to ensure each expected source file produced at least one
// event for the evaluated phase/input before deeper counter assertions run.
func assertPathsPresent(t *testing.T, valuesByPath map[string]int, logFiles []string, context string) {
	t.Helper()

	for _, path := range logFiles {
		if _, exists := valuesByPath[path]; !exists {
			t.Fatalf("missing %s value for path %q", context, path)
		}
	}
}

// assertNoDuplicationFromPreviousInput verifies that the first counter observed
// for each file in inputType is strictly greater than the previously recorded
// boundary in lastSeen, ensuring no duplication from prior phases.
func assertNoDuplicationFromPreviousInput(
	t *testing.T,
	events []fallbackEvent,
	logFiles []string,
	lastSeen map[string]int,
	inputType string,
) {
	t.Helper()

	minCounterByPath, _ := countExtremesByPath(t, events, inputType)
	assertPathsPresent(t, minCounterByPath, logFiles, inputType)

	for _, path := range logFiles {
		boundary, exists := lastSeen[path]
		if !exists {
			t.Fatalf("missing previous input boundary for %q", path)
		}

		if minCounterByPath[path] <= boundary {
			t.Fatalf(
				"%s duplicated data for %q: first counter=%d previous max=%d",
				inputType,
				path,
				minCounterByPath[path],
				boundary,
			)
		}
	}
}

// assertContinuesFromLast verifies that, for each file, events produced by
// inputType in the current phase continue exactly after the previous boundary.
// It expects the first observed counter in this phase to be lastSeen[path] + 1.
func assertContinuesFromLast(
	t *testing.T,
	events []fallbackEvent,
	logFiles []string,
	lastSeen map[string]int,
	inputType string,
) {
	t.Helper()

	// Pass 1: inspect only events from the target input type and capture
	// the first counter value observed per file in this phase.
	minCounterByPath, _ := countExtremesByPath(t, events, inputType)
	assertPathsPresent(t, minCounterByPath, logFiles, inputType)

	// Pass 2: each file must have events and must continue exactly from the
	// previous boundary (first counter == lastSeen + 1).
	for _, path := range logFiles {
		boundary, exists := lastSeen[path]
		if !exists {
			t.Fatalf("missing previous %s boundary for %q", inputType, path)
		}

		if minCounterByPath[path] != boundary+1 {
			t.Fatalf(
				"%s did not continue from previous boundary for %q: first counter=%d expected=%d",
				inputType,
				path,
				minCounterByPath[path],
				boundary+1,
			)
		}
	}
}
