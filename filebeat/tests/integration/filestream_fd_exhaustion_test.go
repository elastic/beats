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

//go:build integration && linux

package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

// TestFilestreamFDExhaustionNoReingest reproduces when the number
// of harvested files exceeds the process file-descriptor limit, the scanner used
// to drop the unreadable files from its result, the watcher reported them deleted,
// clean_removed wiped their registry state, and once descriptors freed up they
// were rediscovered and re-ingested from offset 0 — duplicating data every cycle.
//
// Now, files under paths the scan could not observe are not treated as
// deleted, so each line is published exactly once.
func TestFilestreamFDExhaustionNoReingest(t *testing.T) {
	const (
		numFiles = 200
		fdLimit  = 100
	)

	filebeatBinary, err := filepath.Abs("../../filebeat.test")
	require.NoError(t, err, "resolving filebeat test binary path")

	wrapper := filepath.Join(t.TempDir(), "filebeat-low-nofile")
	wrapperScript := fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail
ulimit -Sn %[1]d
ulimit -Hn %[1]d
exec %[2]q "$@"
`, fdLimit, filebeatBinary)
	require.NoError(t, os.WriteFile(wrapper, []byte(wrapperScript), 0o750))

	filebeat := integration.NewBeat(t, "filebeat", wrapper)
	tempDir := filebeat.TempDir()
	logsDir := filepath.Join(tempDir, "logs")
	require.NoError(t, os.MkdirAll(logsDir, 0o770))

	// One unique line per file, written before we lower the fd limit.
	wantLines := make(map[string]int, numFiles)
	for i := range numFiles {
		line := fmt.Sprintf("fd-exhaustion-line-%03d", i)
		wantLines[line] = 0
		require.NoError(t,
			os.WriteFile(filepath.Join(logsDir, fmt.Sprintf("f-%03d.log", i)), []byte(line+"\n"), 0o640))
	}

	cfg := fmt.Sprintf(`
filebeat.inputs:
  - type: filestream
    id: fd-exhaustion
    paths:
      - %s
    prospector.scanner.check_interval: 1s
    prospector.scanner.fingerprint.enabled: false
    file_identity.native: ~
    close.on_state_change.inactive: 1s
    clean_removed: true
queue.mem:
  flush.timeout: 0s
output.file:
  enabled: true
  path: %s
  filename: output
logging.level: debug
`, filepath.Join(logsDir, "*.log"), tempDir)
	filebeat.WriteConfigFile(cfg)

	filebeat.Start()

	type msgEvent struct {
		Message string `json:"message"`
	}

	// Wait until every line has been published at least once.
	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		seen := map[string]struct{}{}
		for _, e := range integration.GetEventsFromFileOutput[msgEvent](filebeat, 0, false) {
			seen[e.Message] = struct{}{}
		}
		require.GreaterOrEqual(collect, len(seen), numFiles, "not all files were ingested")
	}, 90*time.Second, time.Second, "not all files were ingested")

	// Exhaustion must actually happen, otherwise the test does not reproduce the
	// issue. The watcher logs this (throttled) warning the first time it postpones
	// deletes because the scan could not observe some paths.
	filebeat.WaitLogsContains("postponing their delete detection", 30*time.Second,
		"test must exercise scanner fd-exhaustion, otherwise it does not reproduce the issue")

	// Then wait for two fully-observed scans after the exhaustion above — the window
	// in which the buggy version wiped registry state and re-ingested from offset 0.
	filebeat.WaitLogsContains(`"postponed":0`, 30*time.Second,
		"no fully-observed scan #1 after fd exhaustion")
	filebeat.WaitLogsContains(`"postponed":0`, 30*time.Second,
		"no fully-observed scan #2 after fd exhaustion")
	filebeat.Stop()

	counts := make(map[string]int, numFiles)
	for _, e := range integration.GetEventsFromFileOutput[msgEvent](filebeat, 0, false) {
		if _, ok := wantLines[e.Message]; ok {
			counts[e.Message]++
		}
	}

	for line := range wantLines {
		assert.Equalf(t, 1, counts[line],
			"each line must be published exactly once; %q was published %d times", line, counts[line])
	}

	// The pathology manifests as a storm of registry removals for files that were
	// never actually deleted.
	assert.Empty(t, filebeat.GetLogLine("Remove state for file as file removed"),
		"files under fd-exhausted paths must not have their registry state removed")
}
