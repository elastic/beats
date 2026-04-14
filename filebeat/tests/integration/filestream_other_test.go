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

//go:build integration && !windows

package integration

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

func TestFilestreamHasOwnerAndGroup(t *testing.T) {
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	tempDir := filebeat.TempDir()
	logFilePath := filepath.Join(tempDir, "input.log")

	integration.WriteLogFile(t, logFilePath, 25, false)

	cfg := fmt.Sprintf(`
filebeat.inputs:
  - type: filestream
    id: TestFilestreamHasOwnerAndGroup
    paths:
      - %s
    include_file_owner_name: true
    include_file_owner_group_name: true

logging:
  level: debug
  metrics:
    enabled: false

output:
  file:
    path: ${path.home}
    filename: "output"
    rotate_on_startup: false
`, logFilePath)

	filebeat.WriteConfigFile(cfg)
	filebeat.Start()

	// Get logFilePath owner and group
	logFileInfo, err := os.Stat(logFilePath)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	stat, ok := logFileInfo.Sys().(*syscall.Stat_t)
	if !ok {
		t.Fatalf("Failed to stat file")
	}

	logFileOwner, err := user.LookupId(strconv.FormatUint(uint64(stat.Uid), 10))
	if err != nil {
		t.Fatalf("Failed to lookup uid %v", err)
	}
	logFileGroup, err := user.LookupGroupId(strconv.FormatUint(uint64(stat.Gid), 10))
	if err != nil {
		t.Fatalf("Failed to lookup gid %v", err)
	}

	filebeat.WaitPublishedEvents(30*time.Second, 25)

	type evt struct {
		Log struct {
			File struct {
				Owner string `json:"owner"`
				Group string `json:"group"`
			} `json:"file"`
		} `json:"log"`
	}
	evts := integration.GetEventsFromFileOutput[evt](filebeat, 5, false)
	for _, e := range evts {
		assert.Equal(t, logFileOwner.Username, e.Log.File.Owner)
		assert.Equal(t, logFileGroup.Name, e.Log.File.Group)
	}
}

func TestFilestreamIncludeFileIdentity(t *testing.T) {
	type fileIdentityEvent struct {
		Log struct {
			File struct {
				Path        string  `json:"path"`
				Fingerprint *string `json:"fingerprint,omitempty"`
			} `json:"file"`
		} `json:"log"`
	}

	tests := []struct {
		name                 string
		identityConfig       string
		includeFileIdentity  bool
		expectFingerprint    bool // fingerprint present in events
		expectLogFingerprint bool // "fingerprint" field in logger context
	}{
		{
			name:                 "fingerprint_identity_enabled",
			identityConfig:       "file_identity.fingerprint: ~\n    prospector.scanner:\n      fingerprint.enabled: true",
			includeFileIdentity:  true,
			expectFingerprint:    true,
			expectLogFingerprint: true,
		},
		{
			name:                 "fingerprint_identity_disabled_by_default",
			identityConfig:       "file_identity.fingerprint: ~\n    prospector.scanner:\n      fingerprint.enabled: true",
			includeFileIdentity:  false,
			expectFingerprint:    false,
			expectLogFingerprint: false,
		},
		{
			name:                 "native_identity_enabled",
			identityConfig:       "file_identity.native: ~",
			includeFileIdentity:  true,
			expectFingerprint:    false,
			expectLogFingerprint: false,
		},
		{
			name:                 "native_identity_disabled_by_default",
			identityConfig:       "file_identity.native: ~",
			includeFileIdentity:  false,
			expectFingerprint:    false,
			expectLogFingerprint: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			filebeat := integration.NewBeat(
				t,
				"filebeat",
				"../../filebeat.test",
			)

			logFilePath := filepath.Join(filebeat.TempDir(), "input.log")
			integration.WriteLogFile(t, logFilePath, 25, false)

			includeIdentityCfgLine := ""
			if tc.includeFileIdentity {
				includeIdentityCfgLine = "include_file_identity: true"
			}

			cfg := fmt.Sprintf(`
filebeat.inputs:
  - type: filestream
    id: filestream-include-file-identity-%s
    paths:
      - %s
    %s
    %s

logging:
  level: debug
  metrics:
    enabled: false

output:
  file:
    path: ${path.home}
    filename: "output"
    rotate_on_startup: false
`, tc.name, logFilePath, tc.identityConfig, includeIdentityCfgLine)

			filebeat.WriteConfigFile(cfg)
			filebeat.Start()
			filebeat.WaitPublishedEvents(30*time.Second, 25)

			// --- Event assertions ---
			events := integration.GetEventsFromFileOutput[fileIdentityEvent](filebeat, 25, false)
			require.NotEmpty(t, events, "expected published events")

			var missingPath []int
			var failedFingerprint []int
			for i, event := range events {
				if event.Log.File.Path == "" {
					missingPath = append(missingPath, i)
				}
				if tc.expectFingerprint && event.Log.File.Fingerprint == nil {
					failedFingerprint = append(failedFingerprint, i)
				}
				if !tc.expectFingerprint && event.Log.File.Fingerprint != nil {
					failedFingerprint = append(failedFingerprint, i)
				}
			}
			assert.Empty(t, missingPath,
				"log.file.path must always be present, missing on %d/%d events: %v",
				len(missingPath), len(events), missingPath)
			assert.Empty(t, failedFingerprint,
				"log.file.fingerprint expectation failed (expect_present=%v) on %d/%d events: %v",
				tc.expectFingerprint, len(failedFingerprint), len(events), failedFingerprint)

			// --- Logger assertions ---
			// Find the "A new file" log line (emitted on OpCreate in onFSEvent)
			// and check whether identity fields are present in its structured context.
			newFileLine := filebeat.GetLogLine("A new file")
			require.NotEmpty(t, newFileLine,
				"expected 'A new file' log message from prospector")

			if tc.includeFileIdentity {
				assert.Contains(t, newFileLine, `"source_name":`,
					"'A new file' log line must contain source_name when include_file_identity is true")
				assert.Contains(t, newFileLine, `"source_file":`,
					"'A new file' log line must contain source_file when include_file_identity is true")
			} else {
				assert.NotContains(t, newFileLine, `"source_name":`,
					"'A new file' log line must not contain source_name when include_file_identity is false")
				assert.NotContains(t, newFileLine, `"source_file":`,
					"'A new file' log line must not contain source_file when include_file_identity is false")
			}

			if tc.expectLogFingerprint {
				assert.Contains(t, newFileLine, `"fingerprint":`,
					"'A new file' log line must contain fingerprint when using fingerprint identity with include_file_identity true")
			} else {
				assert.NotContains(t, newFileLine, `"fingerprint":`,
					"'A new file' log line must not contain fingerprint")
			}

			// The harvester log always includes the source ID in the message
			// text (not as a structured field), regardless of the flag.
			harvesterLine := filebeat.GetLogLine("Starting harvester for file")
			require.NotEmpty(t, harvesterLine,
				"expected 'Starting harvester for file' log message from harvester")
			assert.Contains(t, harvesterLine, "filestream::filestream-include-file-identity",
				"'Starting harvester' log line must contain the source ID in the message")
		})
	}
}
