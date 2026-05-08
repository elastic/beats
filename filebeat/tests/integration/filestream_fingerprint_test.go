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
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/filebeat/testing/gziptest"
	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

// Test configuration for fingerprint file identity with small files.
// This test documents the current behavior where files smaller than the
// fingerprint size (default 1024 bytes) are not ingested until they grow
// large enough.
//
// When growing_fingerprint is implemented and this test is updated to use it,
// the assertions should change to verify that small files ARE ingested
// immediately.
var fingerprintSmallFilesCfg = `
filebeat.inputs:
  - type: filestream
    id: test-fingerprint-small-files
    enabled: true
    paths:
      - %s/*.log
    prospector.scanner:
      check_interval: 1s
      fingerprint.enabled: true
    file_identity.fingerprint: ~

queue.mem:
  flush.timeout: 0s

path.home: %s

output.file:
  path: ${path.home}
  filename: "output"
  rotate_on_startup: false

logging:
  level: debug
  metrics:
    enabled: false
`

// TestFilestreamFingerprintSmallFiles tests that files smaller than the
// fingerprint size (default 1024 bytes) are not ingested until they grow
// large enough.
//
// This test documents the current behavior. When growing_fingerprint is
// implemented, the assertions should be updated to verify that small files
// ARE ingested immediately.
func TestFilestreamFingerprintSmallFiles(t *testing.T) {
	t.Skip("the way we log small files changed. needs to update this test")
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	tempDir := filebeat.TempDir()
	logDir := filepath.Join(tempDir, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("failed to create log directory: %s", err)
	}

	file1 := filepath.Join(logDir, "file1.log")
	file2 := filepath.Join(logDir, "file2.log")
	file3 := filepath.Join(logDir, "file3.log")

	filebeat.WriteConfigFile(fmt.Sprintf(fingerprintSmallFilesCfg, logDir, tempDir))
	filebeat.Start()

	filebeat.WaitLogsContains("Input 'filestream' starting", 10*time.Second, "filestream did not start")

	// ===== Phase 1: Create 3 small files with same content =====
	// Each file has ~250 bytes (5 lines * 50 bytes)
	// All files have identical content (simulating header lines)
	headerContent := generateLines("header line", 5)

	appendToFile(t, file1, headerContent)
	appendToFile(t, file2, headerContent)
	appendToFile(t, file3, headerContent)

	filebeat.WaitLogsContains(
		"3 files are too small to be ingested, files need to be at least 1024 in size for ingestion to start",
		5*time.Second,
		"expected log about file size for fingerprinting",
	)

	// output file isn't created yet as no event has been published. Thus, check
	// it manually
	path := filepath.Join(filebeat.TempDir(), "output-*.ndjson")
	files, err := filepath.Glob(path)
	require.NoError(t, err, "failed to glob output files")
	require.Len(t, files, 0, "expected no output file to be created yet as no event should have been published")

	// ===== Phase 2: Grow file1 past 1024 bytes =====
	// Add enough lines to exceed 1024 bytes (need ~16 more lines of 50 bytes)
	file1Content := generateLines("file1 data line", 20)
	appendToFile(t, file1, file1Content)

	// Wait for file1 to be ingested
	filebeat.WaitLogsContains(
		fmt.Sprintf("End of file reached: %s; Backoff now.", file1),
		10*time.Second,
		"file1 was not fully read",
	)

	// only file1's content is published (5 header + 20 data = 25 lines)
	filebeat.WaitPublishedEvents(time.Second, 25)

	// ===== Phase 3: Grow file2 and file3 but still below threshold =====

	// Add some lines but keep them under 1024 bytes
	file2SmallContent := generateLines("file2 small line", 5)
	file3SmallContent := generateLines("file3 small line", 5)
	appendToFile(t, file2, file2SmallContent)
	appendToFile(t, file3, file3SmallContent)
	filebeat.WaitLogsContains(
		"2 files are too small to be ingested, files need to be at least 1024 in size for ingestion to start",
		5*time.Second, "wrong number os small files",
	)

	// still only file1's events (file2 and file3 still too small)
	filebeat.WaitPublishedEvents(2*time.Second, 25)

	// ===== Phase 4: Stop Filebeat =====
	filebeat.Stop()

	// ===== Phase 5: Grow file2 and file3 past threshold (while stopped) =====
	// Add different content to each so they get different fingerprints
	file2LargeContent := generateLines("file2 unique data line", 20)
	file3LargeContent := generateLines("file3 unique data line", 20)
	appendToFile(t, file2, file2LargeContent)
	appendToFile(t, file3, file3LargeContent)

	// ===== Phase 6: Restart Filebeat =====
	filebeat.Start()

	// Wait for file2 and file3 to be detected and ingested
	filebeat.WaitLogsContainsAnyOrder(
		[]string{
			fmt.Sprintf("End of file reached: %s; Backoff now.", file2),
			fmt.Sprintf("End of file reached: %s; Backoff now.", file3),
		},
		10*time.Second,
		"file 2 and 3 were not read until EOF",
	)

	// all files fully ingested
	// file1: 5 header + 20 data = 25 lines (already ingested, should not duplicate)
	// file2: 5 header + 5 small + 20 large = 30 lines
	// file3: 5 header + 5 small + 20 large = 30 lines
	// Total: 25 + 30 + 30 = 85 lines
	filebeat.WaitPublishedEvents(10*time.Second, 85)
}

// growingFingerprintCfg is the test configuration for growing_fingerprint
// file identity. Format args: logDir, checkInterval, tempDir.
var growingFingerprintCfg = `
filebeat.inputs:
  - type: filestream
    id: test-growing-fingerprint
    enabled: true
    compression: auto
    paths:
      - %s/*.log*
    prospector.scanner:
      check_interval: %s
      fingerprint:
        growing: true
        max_length: 100
    file_identity.growing_fingerprint: ~

queue.mem:
  flush.timeout: 0s

path.home: %s

output.file:
  path: ${path.home}
  filename: "output"
  rotate_on_startup: false

logging:
  level: debug
  metrics:
    enabled: false
`

// TestFilestreamGrowingFingerprint tests the growing_fingerprint file identity
// which allows files of any size to be ingested immediately. The fingerprint
// grows as the file grows, and the registry entry is migrated to the new key.
//
// This test includes both plain text and gzipped files to verify that growing
// fingerprint works correctly with compressed files.
//
// This is the counterpart to TestFilestreamFingerprintSmallFiles which tests
// the current fingerprint behavior where small files are not ingested.
func TestFilestreamGrowingFingerprint(t *testing.T) {
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	tempDir := filebeat.TempDir()
	printOutputOnFailure(t, tempDir)
	logDir := filepath.Join(tempDir, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("failed to create log directory: %s", err)
	}

	file1 := filepath.Join(logDir, "file1.log")
	file2 := filepath.Join(logDir, "file2.log")
	file3 := filepath.Join(logDir, "file3.log")
	file4 := filepath.Join(logDir, "file4.log.gz")
	file5 := filepath.Join(logDir, "file5.log.gz")

	filebeat.WriteConfigFile(fmt.Sprintf(growingFingerprintCfg, logDir, "1s", tempDir))
	filebeat.Start()

	filebeat.WaitLogsContains("Input 'filestream' starting",
		10*time.Second, "filestream did not start")

	// ===== Phase 1: Create 4 small files with same content =====
	// All files have identical content - this creates a COLLISION scenario
	// where all 4 files have the same fingerprint (fingerprint is calculated
	// on decompressed data for gzip files)
	headerContent := generateLines("header line", 1)

	appendToFile(t, file1, headerContent)
	appendToFile(t, file2, headerContent)
	appendToFile(t, file3, headerContent)

	// Create gzipped file with same header content
	headerGZ := gziptest.Compress(t,
		[]byte(generateLines("gzip header line", 1)), gziptest.CorruptNone)
	require.NoError(t, os.WriteFile(file4, headerGZ, 0644), "failed to write gzipped file")

	// With growing_fingerprint, small files ARE ingested immediately (unlike regular fingerprint)
	// Due to collision (same content = same fingerprint), only ONE file's entry is created
	// but events ARE published. We wait for EOF on the first detected file.
	filebeat.WaitLogsContains(
		"End of file reached", // any of the 4 files might be the one ingested first
		10*time.Second,
		"file was not read to EOF",
	)

	// Only one event from whichever file was processed first
	filebeat.WaitPublishedEvents(5*time.Second, 2)

	// ===== Phase 2: Grow all 4 files to make them diverge =====
	// Each file gets unique content so they each get a unique fingerprint.
	// Due to collision handling:
	// - The file that created the collision entry (first detected) will get migration
	// - The other 3 files will be treated as NEW files (path doesn't match)
	file1Content := generateLines("file1 unique line", 4)
	file2Content := generateLines("file2 unique line", 4)
	file3Content := generateLines("file3 unique line", 4)
	file5Content := headerContent + generateLines("file5 unique line", 4)
	appendToFile(t, file1, file1Content)
	appendToFile(t, file2, file2Content)
	appendToFile(t, file3, file3Content)

	// GZIP files should not grow. Thus create another file
	file5ContentGZ := gziptest.Compress(t, []byte(file5Content), gziptest.CorruptNone)
	require.NoError(t, os.WriteFile(file5, []byte(file5ContentGZ), 0644),
		"failed to write gzipped file")

	// Wait for migration to occur (only ONE file will have migration - the collision owner)
	filebeat.WaitLogsContains(
		"migrated growing fingerprint entry",
		10*time.Second,
		"no migration occurred",
	)

	// Wait for all 4 files to be read to EOF
	// Note: gzipped files show "EOF has been reached. Closing." instead of "End of file reached"
	filebeat.WaitLogsContainsAnyOrder(
		[]string{
			fmt.Sprintf("End of file reached: %s; Backoff now.", file1),
			fmt.Sprintf("End of file reached: %s; Backoff now.", file2),
			fmt.Sprintf("End of file reached: %s; Backoff now.", file3),
		},
		10*time.Second,
		"plain files were not fully read after growth",
	)

	// Wait for gzipped file separately as it has a different EOF log message
	filebeat.WaitLogsContains(
		fmt.Sprintf("EOF has been reached. Closing. Path='%s'", file5),
		10*time.Second,
		"gzipped file was not fully read after growth",
	)

	// Total events: 4 files × 5 lines each = 20 events + 1 GZIP small file (1 line)
	filebeat.WaitPublishedEvents(10*time.Second, 21)

	// ===== Phase 3: Stop Filebeat =====
	filebeat.Stop()
}

// TestFilestreamGrowingFingerprint_update_while_stopped tests the
// growing_fingerprint file identity which allows files of any size to be
// ingested immediately. The fingerprint grows as the file grows, and the
// registry entry is migrated to the new key.
//
// This is the counterpart to TestFilestreamFingerprintSmallFiles which tests
// the current fingerprint behavior where small files are not ingested.
func TestFilestreamGrowingFingerprint_update_while_stopped(t *testing.T) {
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	tempDir := filebeat.TempDir()
	printOutputOnFailure(t, tempDir)
	logDir := filepath.Join(tempDir, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("failed to create log directory: %s", err)
	}

	file1 := filepath.Join(logDir, "file1.log")
	file2 := filepath.Join(logDir, "file2.log")
	file3 := filepath.Join(logDir, "file3.log")

	filebeat.WriteConfigFile(fmt.Sprintf(growingFingerprintCfg, logDir, "1s", tempDir))
	filebeat.Start()

	filebeat.WaitLogsContains("Input 'filestream' starting",
		10*time.Second, "filestream did not start")

	// ===== Phase 1: Create 3 small files with same content =====
	// All files have identical content - this creates a COLLISION scenario
	// where all 3 files have the same fingerprint
	headerContent := generateLines("header line", 1)

	appendToFile(t, file1, headerContent)
	appendToFile(t, file2, headerContent)
	appendToFile(t, file3, headerContent)

	// With growing_fingerprint, small files are ingested immediately (unlike regular fingerprint)
	// Due to collision (same content = same fingerprint), only one file's entry is created
	// but events are published. We wait for EOF on the first detected file.
	filebeat.WaitLogsContains(
		"End of file reached",
		10*time.Second,
		"file was not read to EOF",
	)

	// With collision, we get 1 event (from whichever file was processed first)
	filebeat.WaitPublishedEvents(5*time.Second, 1)

	// ===== Phase 2: Grow all 3 files to make them diverge =====
	// Each file gets unique content so they each get a unique fingerprint.
	// Due to collision handling:
	// - The file that created the collision entry (first detected) will get migration
	// - The other 2 files will be treated as NEW files (path doesn't match)
	appendToFile(t, file1, generateLines("file1 unique line", 4))

	filebeat.WaitPublishedEvents(5*time.Second, 5)
	filebeat.WaitLogsContains(
		fmt.Sprintf("End of file reached: %s; Backoff now.", file1),
		10*time.Second, "file was not read to EOF")

	// ===== Phase 3: Stop Filebeat =====
	filebeat.Stop()

	// ==== Phase 4: While Filebeat is stopped, grow all 3 files further =====
	file1Content := generateLines("file1 2nd unique line", 5)
	file2Content := generateLines("file2 unique line", 4)
	file3Content := generateLines("file3 unique line", 4)
	appendToFile(t, file1, file1Content)
	appendToFile(t, file2, file2Content)
	appendToFile(t, file3, file3Content)

	// ===== Phase 5: Restart Filebeat =====
	filebeat.Start()

	// Wait for all 3 files to be read to EOF
	filebeat.WaitLogsContainsAnyOrder(
		[]string{
			fmt.Sprintf("End of file reached: %s; Backoff now.", file1),
			fmt.Sprintf("End of file reached: %s; Backoff now.", file2),
			fmt.Sprintf("End of file reached: %s; Backoff now.", file3),
		},
		10*time.Second,
		"files were not fully read after growth",
	)

	filebeat.WaitPublishedEvents(10*time.Second, 20)
}

// TestFilestreamGrowingFingerprint_do_not_mix_up_files tests that growing
// fingerprint correctly distinguishes between files that start with identical
// content but later diverge. This verifies that when multiple files have the
// same initial content (causing a fingerprint collision), each file is tracked
// independently once they grow with different content, even across Filebeat
// restarts.
func TestFilestreamGrowingFingerprint_do_not_mix_up_files(t *testing.T) {
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	tempDir := filebeat.TempDir()
	printOutputOnFailure(t, tempDir)
	logDir := filepath.Join(tempDir, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("failed to create log directory: %s", err)
	}

	file1 := filepath.Join(logDir, "file1.log")
	file2 := filepath.Join(logDir, "file2.log")

	filebeat.WriteConfigFile(fmt.Sprintf(growingFingerprintCfg, logDir, "1s", tempDir))
	filebeat.Start()

	filebeat.WaitLogsContains("Input 'filestream' starting",
		10*time.Second, "filestream did not start")

	// ===== Phase 1: Create 2 files with identical content =====
	// Both files have the same content, creating a fingerprint collision.
	// Only one file will be tracked initially.
	headerContent := generateLines("header line", 1)
	appendToFile(t, file1, headerContent)
	appendToFile(t, file2, headerContent)

	// file1 is ingested (first detected wins the collision)
	filebeat.WaitLogsContains(
		fmt.Sprintf("End of file reached: %s; Backoff now.", file1),
		10*time.Second,
		"file was not read to EOF",
	)
	filebeat.WaitPublishedEvents(5*time.Second, 1)

	// ===== Phase 2: file2 grows with unique content =====
	// file2 diverges from file1, getting its own fingerprint.
	appendToFile(t, file2, generateLines("file2 unique line", 4))

	// 6 Events: 1 (file1) + 5 (file2: 1 header + 4 unique) = 6 total
	filebeat.WaitPublishedEvents(5*time.Second, 6)
	filebeat.WaitLogsContains(
		fmt.Sprintf("End of file reached: %s; Backoff now.", file2),
		10*time.Second, "file was not read to EOF")

	// ===== Phase 3: Stop Filebeat =====
	filebeat.Stop()

	// ===== Phase 4: Both files grow while Filebeat is stopped =====
	// file1 gets unique content (4 lines), file2 gets more unique content (5 lines).
	// This tests that both files are correctly identified after restart.
	file1Content := generateLines("file1 unique line", 4)
	file2Content := generateLines("file2 2nd unique line", 5)
	appendToFile(t, file1, file1Content)
	appendToFile(t, file2, file2Content)

	// ===== Phase 5: Restart Filebeat and verify all content is ingested =====
	// 15 Events: 6 (previous) + 4 (file1 new) + 5 (file2 new) = 15 total
	filebeat.Start()
	filebeat.WaitLogsContainsAnyOrder(
		[]string{
			fmt.Sprintf("End of file reached: %s; Backoff now.", file1),
			fmt.Sprintf("End of file reached: %s; Backoff now.", file2),
		},
		10*time.Second,
		"files were not fully read after growth",
	)

	filebeat.WaitPublishedEvents(10*time.Second, 15)

	// Verify events match the actual file contents, in order
	events := readOutputEvents(t, tempDir)
	assertFileEvents(t, events, file1)
	assertFileEvents(t, events, file2)
}

// TestFilestreamGrowingFingerprint_do_not_mix_up_files_with_shutdown_and_deletion
// tests that growing fingerprint correctly handles the scenario where one of
// two files with identical initial content is deleted during shutdown. This
// verifies that when file1 is deleted while Filebeat is stopped, file2 (which
// started with the same content) is correctly identified and fully ingested
// without being confused with file1's registry entry.
func TestFilestreamGrowingFingerprint_do_not_mix_up_files_with_shutdown_and_deletion(t *testing.T) {
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	tempDir := filebeat.TempDir()
	printOutputOnFailure(t, tempDir)
	logDir := filepath.Join(tempDir, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("failed to create log directory: %s", err)
	}

	file1 := filepath.Join(logDir, "file1.log")
	file2 := filepath.Join(logDir, "file2.log")

	filebeat.WriteConfigFile(fmt.Sprintf(growingFingerprintCfg, logDir, "1s", tempDir))
	filebeat.Start()

	filebeat.WaitLogsContains("Input 'filestream' starting",
		10*time.Second, "filestream did not start")

	// ===== Phase 1: Create 2 files with identical content =====
	// Both files have the same content, creating a fingerprint collision.
	// Only one file will be tracked initially.
	headerContent := generateLines("header line", 1)
	appendToFile(t, file1, headerContent)
	appendToFile(t, file2, headerContent)

	// file1 is ingested (first detected wins the collision)
	// TODO: could this assertion be flaky?
	filebeat.WaitLogsContains(
		fmt.Sprintf("End of file reached: %s; Backoff now.", file1),
		10*time.Second,
		"file was not read to EOF",
	)
	filebeat.WaitPublishedEvents(5*time.Second, 1)

	// ===== Phase 2: Stop Filebeat =====
	filebeat.Stop()

	// ===== Phase 3: Delete file1 and grow file2 while Filebeat is stopped =====
	// file1 is removed, and file2 grows with unique content.
	// This tests that file2 is correctly identified as a different file
	// and not confused with file1's registry entry.
	require.NoError(t, os.Remove(file1), "failed to remove file 1")
	appendToFile(t, file2, generateLines("file2 unique line", 4))

	// ===== Phase 4: Restart Filebeat and verify file2 is fully ingested =====
	// Events: 1 (file1 before deletion) + 5 (file2: 1 header + 4 unique) = 6 total
	filebeat.Start()
	filebeat.WaitLogsContains(
		fmt.Sprintf("End of file reached: %s; Backoff now.", file2),
		10*time.Second, "file was not read to EOF")

	filebeat.WaitPublishedEvents(10*time.Second, 6)

	// Verify events match the actual file contents, in order.
	// file1 was deleted, so only check file2; also verify file1 got
	// exactly 1 event (the header ingested before deletion).
	events := readOutputEvents(t, tempDir)
	f1Msgs := messagesForFile(events, file1)
	require.Len(t, f1Msgs, 1, "file1 should have 1 event (before deletion)")
	assertFileEvents(t, events, file2)
}

// TestFilestreamGrowingFingerprintTruncation tests that truncation with
// different content is treated as a new file (no prefix match = new entry).
func TestFilestreamGrowingFingerprintTruncation(t *testing.T) {
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	tempDir := filebeat.TempDir()
	printOutputOnFailure(t, tempDir)
	logDir := filepath.Join(tempDir, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("failed to create log directory: %s", err)
	}

	logFile := filepath.Join(logDir, "truncate.log")

	filebeat.WriteConfigFile(fmt.Sprintf(growingFingerprintCfg, logDir, "1s", tempDir))
	filebeat.Start()

	filebeat.WaitLogsContains("Input 'filestream' starting",
		10*time.Second, "filestream did not start")

	// ===== Phase 1: Create initial file =====
	originalContent := generateLines("original content", 10)
	writeTruncatingFile(t, logFile, originalContent)

	// Wait for file to be detected and fully read
	filebeat.WaitLogsContains(
		fmt.Sprintf("A new file %s has been found", logFile),
		10*time.Second,
		"file was not detected",
	)
	filebeat.WaitLogsContains(
		fmt.Sprintf("End of file reached: %s; Backoff now.", logFile),
		10*time.Second,
		"file was not fully read",
	)

	// 10 events from original content
	filebeat.WaitPublishedEvents(5*time.Second, 10)

	// ===== Phase 2: Truncate with different content =====
	// This should be treated as a new file since the fingerprint is completely
	// different (no prefix match because content starts differently)
	differentContent := generateLines("completely different", 8)
	writeTruncatingFile(t, logFile, differentContent) // overwrites the file

	// Wait for the truncated file to be read
	// The log message will appear again for the same file path
	filebeat.WaitLogsContains(
		fmt.Sprintf("End of file reached: %s; Backoff now.", logFile),
		10*time.Second,
		"truncated file was not read",
	)

	// 10 (original) + 8 (new content after truncate) = 18 events
	filebeat.WaitPublishedEvents(5*time.Second, 18)
}

// generateLines creates n lines with the given prefix, each line ~50 bytes
func generateLines(prefix string, n int) string {
	var sb strings.Builder
	for i := 1; i <= n; i++ {
		// Pad to make each line ~50 bytes
		line := fmt.Sprintf("%s %d", prefix, i)
		padding := 48 - len(line) // 48 + newline + null = ~50
		if padding > 0 {
			line += strings.Repeat(".", padding)
		}
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	return sb.String()
}

// writeTruncatingFile creates a new file with the given content
func writeTruncatingFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file %s: %s", path, err)
	}
}

// appendToFile appends content to an existing file
func appendToFile(t *testing.T, path, content string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		t.Fatalf("failed to open file %s for append: %s", path, err)
	}
	defer f.Close()
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("failed to append to file %s: %s", path, err)
	}
	if err := f.Sync(); err != nil {
		t.Fatalf("failed to sync file %s: %s", path, err)
	}
}

// outputEvent represents a parsed event from the output file
type outputEvent struct {
	Timestamp string `json:"@timestamp"`
	Message   string `json:"message"`
	Log       struct {
		Offset int64 `json:"offset"`
		File   struct {
			Path        string `json:"path"`
			DeviceID    string `json:"device_id"`
			Inode       string `json:"inode"`
			Fingerprint string `json:"fingerprint"`
		} `json:"file"`
	} `json:"log"`
	rawLine string
}

// TestPrintOutputFileSorted is a helper test to print output files sorted.
// Usage: go test -v -run TestPrintOutputFileSorted -temp-dir=/path/to/temp/dir
// Or: TEMP_DIR=/path/to/temp/dir go test -v -run TestPrintOutputFileSorted
var tempDirFlag = flag.String("dir", "", "path to the temp directory containing output files")

func TestPrintOutputFileSorted(t *testing.T) {
	tempDir := *tempDirFlag
	if tempDir == "" {
		t.Skip("no dir flag or TEMP_DIR environment variable provided")
	}
	printOutputFileSorted(t, tempDir)
}

// readOutputEvents reads all output files and returns parsed events sorted
// by file path, then by timestamp.
func readOutputEvents(t *testing.T, tempDir string) []outputEvent {
	t.Helper()

	pattern := filepath.Join(tempDir, "output-*.ndjson")
	files, err := filepath.Glob(pattern)
	require.NoError(t, err, "failed to glob output files")

	var events []outputEvent
	for _, outputFile := range files {
		f, err := os.Open(outputFile)
		if err != nil {
			t.Fatalf("failed to open output file %s: %s", outputFile, err)
		}

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			var event outputEvent
			if err := json.Unmarshal([]byte(line), &event); err != nil {
				t.Logf("failed to parse line: %s, error: %s", line, err)
				continue
			}
			event.rawLine = line
			events = append(events, event)
		}

		f.Close()

		require.NoError(t, scanner.Err(), "output file scanner returned an error")
	}

	sort.Slice(events, func(i, j int) bool {
		if events[i].Log.File.Path != events[j].Log.File.Path {
			return events[i].Log.File.Path < events[j].Log.File.Path
		}
		return events[i].Log.Offset < events[j].Log.Offset
	})

	return events
}

// messagesForFile returns the messages from events attributed to the given
// file path, preserving the order from the (already sorted) events slice.
func messagesForFile(events []outputEvent, path string) []string {
	var msgs []string
	for _, e := range events {
		if e.Log.File.Path == path {
			msgs = append(msgs, e.Message)
		}
	}
	return msgs
}

// readFileLines reads a text file from disk and returns its non-empty lines.
func readFileLines(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err, "failed to read %s", path)

	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// assertFileEvents verifies that the output events attributed to filePath
// match the actual file contents on disk, in order. This proves events were
// not mixed up between files and no data was re-read or lost.
func assertFileEvents(t *testing.T, events []outputEvent, filePath string) {
	t.Helper()
	expected := readFileLines(t, filePath)
	actual := messagesForFile(events, filePath)
	require.Equalf(t, expected, actual,
		"events for %s do not match file contents", filepath.Base(filePath))
}

// printOutputFileSorted reads the output file, parses each line as JSON,
// and prints the events sorted by file path, then by timestamp.
func printOutputFileSorted(t *testing.T, tempDir string) {
	t.Helper()

	events := readOutputEvents(t, tempDir)
	if len(events) == 0 {
		t.Log("No output events found")
		return
	}

	t.Log("=== Output events sorted by file path, then by timestamp ===")
	for _, event := range events {
		fmt.Printf("[%s] %s @ offset %6d: %s\n",
			filepath.Base(event.Log.File.Path),
			event.Timestamp,
			event.Log.Offset,
			event.Message)
	}
	t.Logf("=== Total: %d events ===", len(events))
}

// printOutputOnFailure registers a cleanup function that prints the sorted
// output events only if the test has failed. This aids debugging without
// cluttering passing test output.
func printOutputOnFailure(t *testing.T, tempDir string) {
	t.Helper()
	t.Cleanup(func() {
		if !t.Failed() {
			return
		}
		printOutputFileSorted(t, tempDir)
	})
}

// TestFilestreamGrowingFingerprint_rename_and_grow tests that when a file is
// renamed and subsequently grows, the growing_fingerprint identity preserves
// the read offset. Without this fix, the rename+grow would be misclassified
// as delete+create, causing the file to be re-read from offset 0.
//
// The test separates the rename and the growth into distinct scan cycles:
//   - First, rename the file and wait for filebeat to register the rename.
//   - Then, append new data and verify it is ingested from the correct offset.
//
// Note: the running harvester keeps publishing events under the ORIGINAL path
// because the file path is baked into the reader at startup. The rename updates
// the store metadata but does not affect the in-flight harvester's path.
// The key assertion is offset continuity: 5 total events (not 6 from re-read).
func TestFilestreamGrowingFingerprint_rename_and_grow(t *testing.T) {
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	tempDir := filebeat.TempDir()
	printOutputOnFailure(t, tempDir)
	logDir := filepath.Join(tempDir, "logs")
	require.NoError(t, os.MkdirAll(logDir, 0755), "failed to create log directory")

	appLog := filepath.Join(logDir, "app.log")
	appLogRenamed := filepath.Join(logDir, "app.log.1")

	filebeat.WriteConfigFile(fmt.Sprintf(growingFingerprintCfg, logDir, "5s", tempDir))
	filebeat.Start()

	filebeat.WaitLogsContains("Input 'filestream' starting",
		10*time.Second, "filestream did not start")

	// ===== Phase 1: Create file with unique content (short fingerprint) =====
	// With max_length=100 and ~50-byte lines, 1 line produces a ~50-byte
	// fingerprint (100 hex chars) — short, so it is a prefix-match candidate.
	appendToFile(t, appLog, generateLines("app original line", 1))

	filebeat.WaitLogsContains(
		fmt.Sprintf("End of file reached: %s; Backoff now.", appLog),
		15*time.Second,
		"app.log was not fully read",
	)
	filebeat.WaitPublishedEvents(10*time.Second, 1)

	// ===== Phase 2: Rename the file and wait for filebeat to register it =====
	// The rename alone does not change the fingerprint — the exact FileID
	// match detects it as OpRename.
	require.NoError(t, os.Rename(appLog, appLogRenamed),
		"failed to rename app.log -> app.log.1")

	filebeat.WaitLogsContains(
		fmt.Sprintf("File %s has been renamed to %s", appLog, appLogRenamed),
		15*time.Second,
		"filebeat did not detect the rename",
	)

	// ===== Phase 3: Append new data to the renamed file =====
	// The running harvester holds an open fd to the inode, so it reads the
	// new data regardless of the path change. The fingerprint grows on the
	// next scan, triggering prefix-match migration in the prospector.
	appendToFile(t, appLogRenamed, generateLines("app new line", 4))

	// Wait for the fingerprint migration — confirms the prospector detected
	// the fingerprint growth and migrated the registry key.
	filebeat.WaitLogsContains(
		"migrated growing fingerprint entry",
		15*time.Second,
		"fingerprint migration did not occur after rename+grow",
	)

	// ===== Phase 4: Assert offset continuity =====
	// Correct: 1 original + 4 new = 5 events.
	// Broken (re-read from 0): 1 original + 5 re-read = 6 events.
	//
	// All events appear under the original path (app.log) because the running
	// harvester's reader bakes in the path at startup and does not update it
	// on rename. This is expected — the important thing is offset continuity.
	filebeat.WaitPublishedEvents(15*time.Second, 5)

	events := readOutputEvents(t, tempDir)
	require.Len(t, events, 5,
		"expected exactly 5 events (1 original + 4 new); 6 would mean re-read from offset 0")

	// All events attributed to the original path (harvester does not update
	// its baked-in path on rename).
	msgs := messagesForFile(events, appLog)
	require.Len(t, msgs, 5, "all events should be attributed to the original path")

	// Verify no duplicate messages — each line is unique.
	seen := make(map[string]struct{}, len(msgs))
	for _, msg := range msgs {
		_, duplicate := seen[msg]
		require.False(t, duplicate, "duplicate event detected: %s", msg)
		seen[msg] = struct{}{}
	}

	// Verify offsets are monotonically increasing — proves the harvester
	// continued from where it left off, not re-read from 0.
	for i := 1; i < len(events); i++ {
		require.Greater(t, events[i].Log.Offset, events[i-1].Log.Offset,
			"offsets must be monotonically increasing (event %d vs %d)", i-1, i)
	}

	// ===== Phase 5: Stop Filebeat =====
	filebeat.Stop()
}
