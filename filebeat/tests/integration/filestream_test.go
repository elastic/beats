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
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
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

	// 2. Write configuration file ans start Filebeat
	filebeat.WriteConfigFile(fmt.Sprintf(filestreamCleanInactiveCfg, logFilePath, tempDir))
	filebeat.Start()

	// 3. Create the log file
	integration.GenerateLogFile(t, logFilePath, 10, false)

	// 4. Wait for Filebeat to start scanning for files
	//
	filebeat.WaitForLogs(
		fmt.Sprintf("A new file %s has been found", logFilePath),
		10*time.Second,
		"Filebeat did not start looking for files to ingest")

	filebeat.WaitForLogs(
		fmt.Sprintf("Reader was closed. Closing. Path='%s", logFilePath),
		10*time.Second, "Filebeat did not close the file")

	// 5. Now that the reader has been closed, nothing is holding the state
	// of the file, so once the TTL of its state expires and the store GC runs,
	// it will be removed from the registry.
	// Wait for the log message stating 1 entry has been removed from the registry
	filebeat.WaitForLogs("1 entries removed", 20*time.Second, "entry was not removed from registtry")

	// 6. Then assess it has been removed in the registry
	registryFile := filepath.Join(filebeat.TempDir(), "data", "registry", "filebeat", "log.json")
	filebeat.WaitFileContains(registryFile, `"op":"remove"`, time.Second)
}

func TestFilestreamValidationPreventsFilebeatStart(t *testing.T) {
	duplicatedIDs := `
filebeat.inputs:
  - type: filestream
    id: duplicated-id-1
    enabled: true
    paths:
      - /tmp/*.log
  - type: filestream
    id: duplicated-id-1
    enabled: true
    paths:
      - /var/log/*.log

output.discard.enabled: true
logging:
  level: debug
  metrics:
    enabled: false
`
	emptyID := `
filebeat.inputs:
  - type: filestream
    enabled: true
    paths:
      - /tmp/*.log
  - type: filestream
    enabled: true
    paths:
      - /var/log/*.log

output.discard.enabled: true
logging:
  level: debug
  metrics:
    enabled: false
`
	multipleDuplicatedIDs := `
filebeat.inputs:
  - type: filestream
    enabled: true
    paths:
      - /tmp/*.log
  - type: filestream
    enabled: true
    paths:
      - /var/log/*.log

  - type: filestream
    id: duplicated-id-1
    enabled: true
    paths:
      - /tmp/duplicated-id-1.log
  - type: filestream
    id: duplicated-id-1
    enabled: true
    paths:
      - /tmp/duplicated-id-1-2.log


  - type: filestream
    id: unique-id-1
    enabled: true
    paths:
      - /tmp/unique-id-1.log
  - type: filestream
    id: unique-id-2
    enabled: true
    paths:
      - /var/log/unique-id-2.log

output.discard.enabled: true
logging:
  level: debug
  metrics:
    enabled: false
`
	tcs := []struct {
		name string
		cfg  string
	}{
		{
			name: "duplicated IDs",
			cfg:  duplicatedIDs,
		},
		{
			name: "duplicated empty ID",
			cfg:  emptyID,
		},
		{
			name: "two inputs without ID and duplicated IDs",
			cfg:  multipleDuplicatedIDs,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			filebeat := integration.NewBeat(
				t,
				"filebeat",
				"../../filebeat.test",
			)

			// Write configuration file and start Filebeat
			filebeat.WriteConfigFile(tc.cfg)
			filebeat.Start()

			// Wait for error log
			filebeat.WaitForLogs(
				"filestream inputs validation error",
				10*time.Second,
				"Filebeat did not log a filestream input validation error")

			proc, err := filebeat.Process.Wait()
			require.NoError(t, err, "filebeat process.Wait returned an error")
			assert.False(t, proc.Success(), "filebeat should have failed to start")

		})
	}
}

func TestFilestreamValidationSucceeds(t *testing.T) {
	cfg := `
filebeat.inputs:
  - type: filestream
    enabled: true
    paths:
      - /var/log/*.log

  - type: filestream
    id: unique-id-1
    enabled: true
    paths:
      - /tmp/unique-id-1.log
  - type: filestream
    id: unique-id-2
    enabled: true
    paths:
      - /var/log/unique-id-2.log

output.discard.enabled: true
logging:
  level: debug
  metrics:
    enabled: false
`
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	// Write configuration file and start Filebeat
	filebeat.WriteConfigFile(cfg)
	filebeat.Start()

	// Wait for error log
	filebeat.WaitForLogs(
		"Input 'filestream' starting",
		10*time.Second,
		"Filebeat did not log a validation error")
}

func TestFilestreamCanMigrateIdentity(t *testing.T) {
	cfgTemplate := `
filebeat.inputs:
  - type: filestream
    id: "test-migrate-ID"
    paths:
      - %s
%s

queue.mem:
  flush.timeout: 0s

path.home: %s

output.file:
  path: ${path.home}
  filename: "output-file"
  rotate_on_startup: false

logging:
  level: debug
  selectors:
    - input
    - input.filestream
    - input.filestream.prospector
  metrics:
    enabled: false
`
	nativeCfg := `
    file_identity.native: ~
`
	pathCfg := `
    file_identity.path: ~
`
	fingerprintCfg := `
    file_identity.fingerprint: ~
    prospector:
      scanner:
        fingerprint.enabled: true
        check_interval: 0.1s
`

	testCases := map[string]struct {
		oldIdentityCfg  string
		oldIdentityName string
		newIdentityCfg  string
		notMigrateMsg   string
		expectMigration bool
	}{
		"native to fingerprint": {
			oldIdentityCfg:  nativeCfg,
			oldIdentityName: "native",
			newIdentityCfg:  fingerprintCfg,
			expectMigration: true,
		},

		"path to fingerprint": {
			oldIdentityCfg:  pathCfg,
			oldIdentityName: "path",
			newIdentityCfg:  fingerprintCfg,
			expectMigration: true,
		},

		"path to native": {
			oldIdentityCfg:  pathCfg,
			newIdentityCfg:  nativeCfg,
			oldIdentityName: "path",
			expectMigration: false,
			notMigrateMsg:   "file identity is 'native', will not migrate registry",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			filebeat := integration.NewBeat(
				t,
				"filebeat",
				"../../filebeat.test",
			)
			workDir := filebeat.TempDir()
			outputFile := filepath.Join(workDir, "output-file*")
			logFilepath := filepath.Join(workDir, "log.log")
			integration.GenerateLogFile(t, logFilepath, 25, false)

			cfgYAML := fmt.Sprintf(cfgTemplate, logFilepath, tc.oldIdentityCfg, workDir)
			filebeat.WriteConfigFile(cfgYAML)
			filebeat.Start()

			// Wait for the file to be fully ingested
			eofMsg := fmt.Sprintf("End of file reached: %s; Backoff now.", logFilepath)
			filebeat.WaitForLogs(eofMsg, time.Second*10, "EOF was not reached")
			requirePublishedEvents(t, filebeat, 25, outputFile)
			filebeat.Stop()

			newCfg := fmt.Sprintf(cfgTemplate, logFilepath, tc.newIdentityCfg, workDir)
			if err := os.WriteFile(filebeat.ConfigFilePath(), []byte(newCfg), 0o644); err != nil {
				t.Fatalf("cannot write new configuration file: %s", err)
			}

			filebeat.Start()

			// The happy path is to migrate keys, so we assert it first
			if tc.expectMigration {
				// Test the case where the registry migration happens
				migratingMsg := fmt.Sprintf("are the same, migrating. Source: '%s'", logFilepath)
				filebeat.WaitForLogs(migratingMsg, time.Second*5, "prospector did not migrate registry entry")
				filebeat.WaitForLogs("migrated entry in registry from", time.Second*10, "store did not update registry key")
				filebeat.WaitForLogs(eofMsg, time.Second*10, "EOF was not reached the second time")
				requirePublishedEvents(t, filebeat, 25, outputFile)

				// Ingest more data to ensure the offset was migrated
				integration.GenerateLogFile(t, logFilepath, 17, true)
				filebeat.WaitForLogs(eofMsg, time.Second*5, "EOF was not reached the third time")

				requirePublishedEvents(t, filebeat, 42, outputFile)
				requireRegistryEntryRemoved(t, workDir, tc.oldIdentityName)
				return
			}

			// Another option is for no keys to be migrated because the current
			// file identity is not fingerprint
			if tc.notMigrateMsg != "" {
				filebeat.WaitForLogs(tc.notMigrateMsg, time.Second*5, "the registry should not have been migrated")
			}

			// The last thing to test when there is no migration is to assert
			// the file has been fully re-ingested because the file identity
			// changed
			filebeat.WaitForLogs(eofMsg, time.Second*10, "EOF was not reached the second time")
			requirePublishedEvents(t, filebeat, 50, outputFile)

			// Ingest more data to ensure the offset is correctly tracked
			integration.GenerateLogFile(t, logFilepath, 10, true)
			filebeat.WaitForLogs(eofMsg, time.Second*5, "EOF was not reached the third time")
			requirePublishedEvents(t, filebeat, 60, outputFile)
		})
	}
}

func TestFilestreamMigrateIdentityCornerCases(t *testing.T) {
	cfgTemplate := `
filebeat.inputs:
  - type: filestream
    id: "test-migrate-ID"
    paths:
      - %s
%s

queue.mem:
  flush.timeout: 0s

path.home: %s

output.file:
  path: ${path.home}
  filename: "output-file"
  rotate_on_startup: false

logging:
  level: debug
  selectors:
    - input
    - input.filestream
    - input.filestream.prospector
  metrics:
    enabled: false
`
	nativeCfg := `
    file_identity.native: ~
    prospector:
      scanner:
        fingerprint.enabled: false
        check_interval: 0.1s
`
	fingerprintCfg := `
    file_identity.fingerprint: ~
    prospector:
      scanner:
        fingerprint.enabled: true
        check_interval: 0.1s
`

	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	workDir := filebeat.TempDir()

	logFilepath := filepath.Join(workDir, "log.log")
	outputFile := filepath.Join(workDir, "output-file*")

	cfgYAML := fmt.Sprintf(cfgTemplate, logFilepath, nativeCfg, workDir)
	filebeat.WriteConfigFile(cfgYAML)
	filebeat.Start()

	// Create and ingest 4 different files, all with the same path
	// to simulate log rotation
	createFileAndWaitIngestion(t, logFilepath, outputFile, filebeat, 50, 50)
	createFileAndWaitIngestion(t, logFilepath, outputFile, filebeat, 50, 100)
	createFileAndWaitIngestion(t, logFilepath, outputFile, filebeat, 50, 150)
	createFileAndWaitIngestion(t, logFilepath, outputFile, filebeat, 50, 200)

	filebeat.Stop()
	cfgYAML = fmt.Sprintf(cfgTemplate, logFilepath, fingerprintCfg, workDir)
	if err := os.WriteFile(filebeat.ConfigFilePath(), []byte(cfgYAML), 0666); err != nil {
		t.Fatalf("cannot write config file: %s", err)
	}

	filebeat.Start()

	migratingMsg := fmt.Sprintf("are the same, migrating. Source: '%s'", logFilepath)
	eofMsg := fmt.Sprintf("End of file reached: %s; Backoff now.", logFilepath)

	filebeat.WaitForLogs(migratingMsg, time.Second*10, "prospector did not migrate registry entry")
	filebeat.WaitForLogs("migrated entry in registry from", time.Second*10, "store did not update registry key")
	// Filebeat logs the EOF message when it starts and the file had already been fully ingested.
	filebeat.WaitForLogs(eofMsg, time.Second*10, "EOF was not reached after restart")

	requirePublishedEvents(t, filebeat, 200, outputFile)
	// Ingest more data to ensure the offset was migrated
	integration.GenerateLogFile(t, logFilepath, 20, true)
	filebeat.WaitForLogs(eofMsg, time.Second*5, "EOF was not reached after adding data")

	requirePublishedEvents(t, filebeat, 220, outputFile)
	requireRegistryEntryRemoved(t, workDir, "native")
}

func requireRegistryEntryRemoved(t *testing.T, workDir, identity string) {
	t.Helper()

	registryLogFile := filepath.Join(workDir, "data", "registry", "filebeat", "log.json")
	entries := readFilestreamRegistryLog(t, registryLogFile)
	inputEntries := []registryEntry{}
	for _, currentEntry := range entries {
		if strings.Contains(currentEntry.Key, identity) {
			inputEntries = append(inputEntries, currentEntry)
		}
	}

	lastNativeEntry := inputEntries[len(inputEntries)-1]
	if lastNativeEntry.TTL != 0 {
		t.Errorf("'%s' has not been removed from the registry", lastNativeEntry.Key)
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

func createFileAndWaitIngestion(
	t *testing.T,
	logFilepath, outputFilepath string,
	fb *integration.BeatProc,
	n, outputTotal int) {

	t.Helper()
	_, err := os.Stat(logFilepath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cannot stat log file: %s", err)
	}
	// Remove the file if it exists
	if err == nil {
		if err := os.Remove(logFilepath); err != nil {
			t.Fatalf("cannot remove log file: %s", err)
		}
	}

	integration.GenerateLogFile(t, logFilepath, n, false)

	eofMsg := fmt.Sprintf("End of file reached: %s; Backoff now.", logFilepath)
	fb.WaitForLogs(eofMsg, time.Second*10, "EOF was not reached")
	requirePublishedEvents(t, fb, outputTotal, outputFilepath)
}
