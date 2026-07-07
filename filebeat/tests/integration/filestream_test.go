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
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/elastic/beats/v7/libbeat/tests/integration"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var filestreamCleanInactiveCfg = `
filebeat.inputs:
  - type: filestream
    id: "test-clean-inactive"
    paths:
      - %s

    file_identity.native: ~
    prospector.scanner.fingerprint.enabled: false
    clean_inactive: 3s
    ignore_older: 2s
    close.on_state_change.inactive: 1s
    prospector.scanner.check_interval: 1s

filebeat.registry:
  cleanup_interval: 5s
  flush: 1s

queue.mem:
  events: 32
  flush.min_events: 8
  flush.timeout: 0.1s

path.home: %s

output.file:
  path: ${path.home}
  filename: "output-file"
  rotate_every_kb: 10000

logging:
  level: debug
  selectors:
    - input
    - input.filestream
    - input.filestream.prospector
  metrics:
    enabled: false
`

func TestFilestreamCleanInactive(t *testing.T) {
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	tempDir := filebeat.TempDir()

	// 1. Generate the log file path, but do not write data to it
	logFilePath := path.Join(tempDir, "log.log")

	// 2. Write configuration file and start Filebeat
	filebeat.WriteConfigFile(fmt.Sprintf(filestreamCleanInactiveCfg, logFilePath, tempDir))
	filebeat.Start()

	// 3. Create the log file
	integration.WriteLogFile(t, logFilePath, 10, false)

	// 4. Wait for Filebeat to start scanning for files
	filebeat.WaitLogsContains(
		fmt.Sprintf("A new file %s has been found", logFilePath),
		10*time.Second,
		"Filebeat did not start looking for files to ingest")

	filebeat.WaitLogsContains(
		fmt.Sprintf("Reader was closed. Closing. Path='%s", logFilePath),
		10*time.Second, "Filebeat did not close the file")

	// 5. Now that the reader has been closed, nothing is holding the state
	// of the file, so once the TTL of its state expires and the store GC runs,
	// it will be removed from the registry.
	// Wait for the log message stating 1 entry has been removed from the registry
	filebeat.WaitLogsContains("1 entries removed", 20*time.Second, "entry was not removed from registry")

	// 6. Then assess it has been removed in the registry
	registryFile := filepath.Join(filebeat.TempDir(), "data", "registry", "filebeat", "log.json")
	filebeat.WaitFileContains(registryFile, `"op":"remove"`, time.Second)
}

// filestreamCleanInactiveRenamedCfg is the config for
// TestFilestreamCleanInactiveRenamed. clean_removed is disabled so that
// clean_inactive is the only mechanism that can remove the registry entry,
// which isolates the rename code path under test.
var filestreamCleanInactiveRenamedCfg = `
filebeat.inputs:
  - type: filestream
    id: "test-clean-inactive-renamed"
    paths:
      - %s

    file_identity.native: ~
    prospector.scanner.fingerprint.enabled: false
    clean_inactive: 3.1s
    ignore_older: 2s
    clean_removed: false
    close.on_state_change.inactive: 1s
    prospector.scanner.check_interval: 1s

filebeat.registry:
  cleanup_interval: 5s
  flush: 1s

queue.mem:
  events: 32
  flush.min_events: 8
  flush.timeout: 0.1s

path.home: %s

output.file:
  path: ${path.home}
  filename: "output-file"
  rotate_every_kb: 10000

logging:
  level: debug
  selectors:
    - "input"
    - "input.filestream"
    - "input.filestream.file_watcher"
    - "input.filestream.prospector"
    - "input.filestream.scanner"
  metrics:
    enabled: false
`

// TestFilestreamCleanInactiveRenamed is a regression test for a registry
// resource leak in store.findCursorMeta.
//
// When a file tracked by a file_identity that supports rename tracking (e.g.
// native) is renamed, fileProspector.onRename calls FindCursorMeta to read the
// previous metadata. FindCursorMeta acquires a reference on the in-memory
// resource through states.Find. Before the fix that reference was never
// released, so the resource stayed permanently active (pending > 0), the
// registry GC could never collect it and the entry leaked forever, even after
// clean_inactive expired.
//
// This test renames a harvested file, lets it become inactive and asserts the
// registry entry is eventually garbage collected. Without the fix the entry is
// never removed and the test times out.
func TestFilestreamCleanInactiveRenamed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("renaming files while Filebeat is running is not supported on Windows")
	}

	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	tempDir := filebeat.TempDir()

	// Harvest from a sub-folder so the configured glob keeps matching the file
	// after it is renamed. path.home (registry + output) lives in tempDir, the
	// glob only matches the logs sub-folder.
	logsDir := filepath.Join(tempDir, "logs")
	if err := os.MkdirAll(logsDir, 0o700); err != nil {
		t.Fatalf("cannot create logs directory: %s", err)
	}
	logFilePath := filepath.Join(logsDir, "log.log")
	renamedLogFilePath := filepath.Join(logsDir, "log.log.renamed")

	filebeat.WriteConfigFile(
		fmt.Sprintf(filestreamCleanInactiveRenamedCfg, filepath.Join(logsDir, "*"), tempDir))
	filebeat.Start()

	integration.WriteLogFile(t, logFilePath, 10, false)
	filebeat.WaitLogsContains(
		fmt.Sprintf("A new file %s has been found", logFilePath),
		10*time.Second,
		"Filebeat did not start harvesting the log file")

	// Renaming is what triggers fileProspector.onRename -> FindCursorMeta.
	if err := os.Rename(logFilePath, renamedLogFilePath); err != nil {
		t.Fatalf("cannot rename log file: %s", err)
	}

	// Guard: confirm the change was detected as a rename (not a removal),
	// otherwise onRename/FindCursorMeta never ran and the test would silently
	// stop exercising the leak.
	filebeat.WaitLogsContains(
		fmt.Sprintf("has been renamed to %s", renamedLogFilePath),
		10*time.Second,
		"Filebeat did not detect the rename")

	// After clean_inactive expires the GC must collect the entry. With the leak
	// the resource is never "finished", so this never happens.
	filebeat.WaitLogsContains(
		"1 entries removed",
		30*time.Second,
		"registry entry for the renamed file was not garbage collected")

	registryFile := filepath.Join(tempDir, "data", "registry", "filebeat", "log.json")
	filebeat.WaitFileContains(registryFile, `"op":"remove"`, time.Second)
}

func TestFilestreamDefaultRegistryTTL(t *testing.T) {
	cfg := `
filebeat.inputs:
  - type: filestream
    id: filestream-id
    paths:
      - %s

queue.mem:
  flush.timeout: 0s

path.home: %s

output.file:
  path: ${path.home}
  filename: "output-file"
  rotate_on_startup: false

logging:
  level: debug
`

	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	tempDir := filebeat.TempDir()
	logFilePath := path.Join(tempDir, "input_log.log")
	outputFile := filepath.Join(tempDir, "output-file*")

	// > 1kb in total to trigger default fingerprinting
	numEvents := 30

	integration.WriteLogFile(t, logFilePath, numEvents, false)

	filebeat.WriteConfigFile(fmt.Sprintf(cfg, logFilePath, tempDir))
	filebeat.Start()

	filebeat.WaitLogsContains(
		fmt.Sprintf("A new file %s has been found", logFilePath),
		10*time.Second,
		"Filebeat did not start looking for files to ingest")

	eofMsg := fmt.Sprintf("End of file reached: %s; Backoff now.", logFilePath)
	filebeat.WaitLogsContains(eofMsg, 10*time.Second, "EOF was not reached")

	requirePublishedEvents(t, filebeat, numEvents, outputFile)

	// Read the registry log file and check the TTL
	registryLogFile := filepath.Join(tempDir, "data", "registry", "filebeat", "log.json")
	entries, _ := readFilestreamRegistryLog(t, registryLogFile)
	require.GreaterOrEqual(t, len(entries), 1, "No registry entries found")
	firstEntry := entries[0]

	expectedTTL := time.Duration(-1)
	assert.Equal(t, expectedTTL, firstEntry.TTL,
		"Registry entry TTL should be -1 by default, but got %v", firstEntry.TTL)
}

// migrated from test_fixup_registry_entries_with_global_id in test_input.py
func TestFixupRegistryEntriesWithGlobalID(t *testing.T) {
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	workDir := filebeat.TempDir()
	outputFile := filepath.Join(workDir, "output-file*")
	logFilepath := filepath.Join(workDir, "log.log")
	msgLogFilepath := logFilepath
	if runtime.GOOS == "windows" {
		msgLogFilepath = strings.ReplaceAll(logFilepath, `\`, `\\`)
	}

	integration.WriteLogFile(t, logFilepath, 50, false)

	// First run: no explicit ID, Filestream stores state under `.global`.
	cfgYAML := getConfig(t, map[string]any{
		"homePath":    workDir,
		"logFilePath": logFilepath,
		"inputID":     "",
	}, "", "filestream_fixup_registry_global_id.yml")
	filebeat.WriteConfigFile(cfgYAML)
	filebeat.Start()

	eofMsg := fmt.Sprintf("End of file reached: %s; Backoff now.", msgLogFilepath)
	filebeat.WaitLogsContains(eofMsg, 10*time.Second, "EOF was not reached on first run")
	requirePublishedEvents(t, filebeat, 50, outputFile)
	filebeat.Stop()

	// Second run: add explicit ID and verify previous state is migrated.
	cfgYAML = getConfig(t, map[string]any{
		"homePath":    workDir,
		"logFilePath": logFilepath,
		"inputID":     "test-fix-global-id",
	}, "", "filestream_fixup_registry_global_id.yml")
	filebeat.WriteConfigFile(cfgYAML)
	filebeat.RemoveLogFiles()
	filebeat.Start()

	// Ensure no duplicate ingestion after state migration.
	filebeat.WaitLogsContains(eofMsg, 10*time.Second, "EOF was not reached on second run")
	requirePublishedEvents(t, filebeat, 50, outputFile)

	// Add new data and assert only new lines are ingested.
	integration.WriteLogFile(t, logFilepath, 2, true)
	filebeat.WaitLogsContains(eofMsg, 10*time.Second, "EOF was not reached after appending lines")
	filebeat.Stop()
	requirePublishedEvents(t, filebeat, 52, outputFile)

	registryFile := filepath.Join(workDir, "data", "registry", "filebeat", "log.json")
	entries, _ := readFilestreamRegistryLog(t, registryFile)
	registry := parseRegistry(entries)

	requireRegistryEntryRemoved(t, workDir, ".global")

	// Assert old registry entry was removed
	for key, entry := range registry {
		if strings.Contains(key, "filestream::.global::") && !entry.Removed {
			t.Error("entry from input without ID was not removed from registry")
		}
	}
}

func requirePublishedEvents(
	t *testing.T,
	filebeat *integration.BeatProc,
	expected int,
	outputFile string) {

	t.Helper()
	publishedEvents := filebeat.CountFileLines(outputFile)
	if publishedEvents != expected {
		t.Fatalf("expecting %d published events after file migration, got %d instead", expected, publishedEvents)
	}
}

// getConfig renders the template in testdata/<folder>/<tmplPath> using vars.
func getConfig(t *testing.T, vars map[string]any, folder, tmplPath string) string {
	t.Helper()
	tmpl := template.Must(
		template.ParseFiles(
			filepath.Join("testdata", folder, tmplPath)))

	str := strings.Builder{}
	if err := tmpl.Execute(&str, vars); err != nil {
		t.Fatalf("cannot execute template: %s", err)
	}

	return str.String()
}

func requireRegistryEntryRemoved(t *testing.T, workDir, identity string) {
	t.Helper()

	registryFile := filepath.Join(workDir, "data", "registry", "filebeat", "log.json")
	entries, _ := readFilestreamRegistryLog(t, registryFile)
	for _, entry := range entries {
		if strings.Contains(entry.Key, "filestream::"+identity+"::") && entry.Removed {
			return
		}
	}

	t.Fatalf("expected registry entry for identity %q to be removed", identity)
}

func parseRegistry(entries []registryEntry) map[string]registryEntry {
	registry := map[string]registryEntry{}
	for _, entry := range entries {
		registry[entry.Key] = entry
	}
	return registry
}
