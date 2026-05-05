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

// TestFilebeatTakeOverFallbackWithInputReload performs an extensive test
// of the Take Over fallback. This test ensures that once the Take Over
// from the Log input takes place, the Log input states are left untouched
// and if Filestream is disabled and the Log input re-enabled, it continues
// from where it left off.
//
// The steps of the test:
//  1. Add 25 lines to the target files
//  2. Start the Log input
//  3. Assert the files are in the output
//  4. Stop the Log input
//  5. Start the Filestream input with Take Over enabled
//  6. Append 25 lines to the files
//  7. Assert Filestream continues reading from where the Log input stopped
//  8. Stop the Filestream input
//  9. Start the Log input
//  10. Append 25 lines to the files
//  11. Assert the Log input continues from where it stopped
//  12. Stop the Log input
//  13. Append 25 lines to the files
//  14. Start the Filestream input
//  15. Assert the Filestream input continues from where it left off
//
// Before a new input is started, a snapshot of the output is taken for
// debugging purposes.
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

	// nextCounter holds the next counter that should go into the log file.
	// Which matches the number of lines in the file.
	nextCounter := map[string]int{}

	// lastSeen holds the last event count seen by each input for each file
	lastSeen := map[string]map[string]int{}

	// newEventsCount calculates the number of new events ingested by an input
	// since it last run.
	newEventsCount := func(input string) int {
		newEvents := 0
		for path, counter := range nextCounter {
			newEvents += counter - 1 - lastSeen[input][path]
		}

		return newEvents
	}

	// Create a helper to add data to the log files.
	// Each line contains: padding, filename and a counter.
	// Lines are 50 bytes long.
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

	_, logMaxByPath := countExtremesByPath(t, events, "log")
	lastSeen["log"] = logMaxByPath

	// 2. Disable Log input and snapshot output file for debugging
	snapshotIdx := 0
	disableActiveInput(t, inputsDir, filebeat, "log")
	copyOutputSnapshot(t, tempDir, snapshotIdx, "log-1")

	// ========================= output "status" =========================
	// At this point only the Log input has run and it should have ingested
	// all the files.
	// For each file:
	//   - events: 00 ~ 24: Log input

	// 3. Enable Filestream with take_over and ingest one deterministic batch
	writeFilestreamConfig(t, inputsDir, logPaths)
	filebeat.WaitLogsContains("Input 'filestream' starting", 30*time.Second, "filestream runner did not start")
	appendLogsToFiles(batchSize)

	expectedEvents += batchSize * len(logFiles)
	filebeat.WaitPublishedEvents(30*time.Second, expectedEvents)
	events = integration.GetEventsFromFileOutput[fallbackEvent](filebeat, expectedEvents, true)

	// 4. Ensure Filestream did not duplicate data already ingested by Log input
	assertNoDuplicationFromPreviousInput(t, events, logFiles, lastSeen["log"], "filestream")

	// 5. Stop Filestream input and snapshot output file for debugging
	_, filestreamMaxByPath := countExtremesByPath(t, events, "filestream")
	lastSeen["filestream"] = filestreamMaxByPath

	snapshotIdx++
	disableActiveInput(t, inputsDir, filebeat, "filestream")
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
	expectedEvents += newEventsCount("log")
	filebeat.WaitPublishedEvents(30*time.Second, expectedEvents)
	events = integration.GetEventsFromFileOutput[fallbackEvent](filebeat, expectedEvents, true)
	assertContinuesFromLast(t, events[prevExpectedEvents:], logFiles, lastSeen["log"], "log")

	// 7. Update Log baseline, disable all inputs and snapshot.
	_, updatedLogMaxByPath := countExtremesByPath(t, events, "log")
	lastSeen["log"] = updatedLogMaxByPath

	snapshotIdx++
	disableActiveInput(t, inputsDir, filebeat, "log")
	copyOutputSnapshot(t, tempDir, snapshotIdx, "log-2")

	// ========================= output "status" =========================
	// At this point, there was "one fallback", for the Log input, and only
	// the log input ingested the last batch of events.
	// For each file (in the order they appear in the output):
	//   - events 00 ~ 24: Log input
	//   - events 25 ~ 49: Filestream input
	//   - events 25 ~ 74: Log input

	// 8. Re-enable Filestream input, ingest one deterministic batch and assert continuity.
	writeFilestreamConfig(t, inputsDir, logPaths)
	filebeat.WaitLogsContains("Input 'filestream' starting", 30*time.Second, "filestream runner did not start")
	appendLogsToFiles(batchSize)

	prevExpectedEvents = expectedEvents
	expectedEvents += newEventsCount("filestream")
	filebeat.WaitPublishedEvents(30*time.Second, expectedEvents)
	events = integration.GetEventsFromFileOutput[fallbackEvent](filebeat, expectedEvents, true)
	assertContinuesFromLast(t, events[prevExpectedEvents:], logFiles, lastSeen["filestream"], "filestream")

	// ========================= output "status" =========================
	// At this point, there was "one fallback", for the each input, and only
	// the Filestream input ingested the last batch of events.
	// For each file (in the order they appear in the output):
	//   - events 00 ~ 24: Log input
	//   - events 25 ~ 49: Filestream input
	//   - events 25 ~ 74: Log input
	//   - events 75 ~ 99: Filestream input
	// For a total of:
	// - 75 events ingested by Filestream (count: 25 ~ 99)
	// - 75 events ingested by the Log input (count: 0 ~ 74)

	// 9. Final check: ensure each input never duplicated data
	assertPerInputCountersStrictlyIncrease(t, events, []string{"log", "filestream"})
}

// writeLogInputConfig renders and writes the active log input configuration file.
// It replaces inputs.d/active.yml with a config that reads the given paths.
func writeLogInputConfig(t *testing.T, inputsDir string, paths []string) {
	writeInputConfigFromTemplate(
		t,
		inputsDir,
		"take-over-fallback",
		"log-input.yml",
		map[string]any{
			"paths": paths,
		},
	)
}

// writeFilestreamConfig renders and writes the active filestream configuration
// file with takeover enabled for the provided input ID and paths.
func writeFilestreamConfig(t *testing.T, inputsDir string, paths []string) {
	writeInputConfigFromTemplate(
		t,
		inputsDir,
		"take-over-fallback",
		"filestream-input.yml",
		map[string]any{
			"paths": paths,
		})
}

// writeInputConfigFromTemplate renders an input template and writes it as the
// currently active reload configuration at inputs.d/active.yml.
func writeInputConfigFromTemplate(t *testing.T, inputsDir, folder, templateName string, vars map[string]any) {
	content := getConfig(t, vars, folder, templateName)
	if err := os.WriteFile(filepath.Join(inputsDir, "active.yml"), []byte(content), 0o666); err != nil {
		t.Fatalf("failed to write active input config from %q: %s", templateName, err)
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

// assertPerInputCountersStrictlyIncrease verifies counters never regress or
// duplicate for the same (input.type, log.file.path) stream.
func assertPerInputCountersStrictlyIncrease(t *testing.T, events []fallbackEvent, inputTypes []string) {
	t.Helper()

	allowed := map[string]bool{}
	lastCounter := map[string]map[string]int{}
	for _, inputType := range inputTypes {
		allowed[inputType] = true
		lastCounter[inputType] = map[string]int{}
	}

	for _, event := range events {
		inputType := event.Input.Type
		if !allowed[inputType] {
			continue
		}

		path := event.Log.File.Path
		counter := counterFromMessage(t, event.Message)
		if prev, ok := lastCounter[inputType][path]; ok && counter <= prev {
			t.Fatalf(
				"counter duplication for input=%q path=%q: prev=%d current=%d",
				inputType,
				path,
				prev,
				counter,
			)
		}
		lastCounter[inputType][path] = counter
	}
}
