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
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

const checkpointTestCfgTemplate = `
filebeat.inputs:
  - type: filestream
    id: checkpoint-test
    paths:
      - %s
    prospector.scanner.check_interval: 100ms
    close.on_state_change.inactive: 500ms
    clean_removed: true

filebeat.registry:
  cleanup_interval: 1s
  flush: 100ms
  %s

path.home: %s

output.file:
  path: ${path.home}
  filename: output
  rotate_every_kb: 100000

logging:
  level: info
  metrics:
    enabled: false
`

// TestFilestreamCheckpointSize verifies that the memlog registry checkpoint
// triggers at the configured threshold when filebeat is running end-to-end.
//
// The test creates and deletes batches of log files to generate registry
// operations that grow the WAL (log.json). A background goroutine monitors
// the WAL file size and detects the checkpoint (any size drop). The maximum
// WAL size observed before the drop should approximate the configured
// checkpoint threshold.
func TestFilestreamCheckpointSize(t *testing.T) {
	testCases := []struct {
		name           string
		registryCfg    string // extra YAML under filebeat.registry
		checkpointSize int64
		fileBatchSize  int
		maxBatches     int
		deltaPercent   int64 // allowed overshoot percentage (e.g. 10 means 10%)
	}{
		{
			name:           "default 10MB",
			registryCfg:    "",
			checkpointSize: 10 * 1024 * 1024,
			fileBatchSize:  500,
			maxBatches:     200,
			deltaPercent:   10,
		},
		{
			name:           "custom 15MB",
			registryCfg:    "memlog.checkpoint_size: 15728640",
			checkpointSize: 15 * 1024 * 1024,
			fileBatchSize:  500,
			maxBatches:     300,
			deltaPercent:   10,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filebeat := integration.NewBeat(t, "filebeat", "../../filebeat.test")
			homeDir := filebeat.TempDir()
			logDir := filepath.Join(homeDir, "logs")
			require.NoError(t, os.MkdirAll(logDir, 0o755))

			glob := filepath.Join(logDir, "*.log")
			filebeat.WriteConfigFile(fmt.Sprintf(
				checkpointTestCfgTemplate, glob, tc.registryCfg, homeDir))
			filebeat.Start()

			t.Logf("path.home: %s", homeDir)
			registryLogFile := filepath.Join(
				homeDir, "data", "registry", "filebeat", "log.json")

			// Wait for filebeat to initialise and create the registry.
			require.Eventually(t, func() bool {
				_, err := os.Stat(registryLogFile)
				return err == nil
			}, 30*time.Second, 200*time.Millisecond,
				"registry log file was not created")

			// walWatcher polls the WAL file and records the maximum size
			// it sees. When the size drops (checkpoint truncated the WAL),
			// it stops and reports back.
			type walResult struct {
				maxSize int64
			}
			resultCh := make(chan walResult, 1)
			var walMaxSize atomic.Int64
			var wg sync.WaitGroup

			wg.Add(1)
			go func() {
				defer wg.Done()
				ticker := time.NewTicker(200 * time.Millisecond)
				defer ticker.Stop()

				var prevSize int64
				var maxSeen int64
				for {
					select {
					case <-t.Context().Done():
						return
					case <-ticker.C:
					}

					info, err := os.Stat(registryLogFile)
					if err != nil {
						continue
					}
					size := info.Size()

					if size > maxSeen {
						maxSeen = size
						walMaxSize.Store(maxSeen)
					}

					// Any drop in size means a checkpoint occurred.
					if prevSize > 0 && size < prevSize {
						resultCh <- walResult{maxSize: maxSeen}
						return
					}
					prevSize = size
				}
			}()

			// Create and delete batches of files until the checkpoint
			// fires or we exceed maxBatches.
			checkInterval := 100 * time.Millisecond
			batchCycle := 2 * checkInterval

			checkpointDetected := false
			for batch := range tc.maxBatches {
				// Create a batch of files, each > 1024 bytes for
				// fingerprint identity.
				for i := range tc.fileBatchSize {
					name := filepath.Join(logDir,
						fmt.Sprintf("batch%04d-file%04d.log", batch, i))
					writeTestLogFile(t, name, batch, i)
				}

				time.Sleep(batchCycle)

				// Delete the batch so cleanup generates more WAL ops.
				for i := range tc.fileBatchSize {
					name := filepath.Join(logDir,
						fmt.Sprintf("batch%04d-file%04d.log", batch, i))
					_ = os.Remove(name)
				}

				// Wait for cleanup to process the deletions.
				time.Sleep(2 * time.Second)
				// Check if the watcher detected a checkpoint.
				select {
				case res := <-resultCh:
					t.Logf("checkpoint detected after %d batches, max WAL size: %d bytes (%.2f MB)",
						batch+1, res.maxSize, float64(res.maxSize)/(1024*1024))
					checkpointDetected = true

					maxAllowed := tc.checkpointSize + tc.checkpointSize*tc.deltaPercent/100
					assert.GreaterOrEqual(t, res.maxSize, tc.checkpointSize,
						"WAL max size (%d) should be >= checkpoint threshold (%d)",
						res.maxSize, tc.checkpointSize)
					assert.LessOrEqual(t, res.maxSize, maxAllowed,
						"WAL max size (%d) should be <= checkpoint threshold + %d%% (%d)",
						res.maxSize, tc.deltaPercent, maxAllowed)
				default:
					t.Logf("batch %d done, WAL size: %.2f MB",
						batch+1, float64(walMaxSize.Load())/(1024*1024))
					continue
				}
				break
			}

			wg.Wait()
			require.True(t, checkpointDetected,
				"checkpoint was not triggered after %d batches", tc.maxBatches)
		})
	}
}

// writeTestLogFile writes a log file with content > 1024 bytes so the
// fingerprint identity can track it.
func writeTestLogFile(t *testing.T, path string, batch, fileIdx int) {
	t.Helper()

	// Each line: "batch0001-file0002: line 003 <padding>\n"
	// Pad to ensure total file size > 1024 bytes.
	var b strings.Builder
	for line := range 20 {
		fmt.Fprintf(&b, "batch%04d-file%04d: line %03d %s\n",
			batch, fileIdx, line, strings.Repeat("x", 30))
	}
	require.NoError(t, os.WriteFile(path, []byte(b.String()), 0o644))
}
