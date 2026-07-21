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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/filebeat/testing/gziptest"
	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

const (
	fingerprintCfgBase = `
filebeat.inputs:
  - type: filestream
    id: test-enhanced-fingerprint
    enabled: true
    compression: auto
    paths:
      - %s/*.log*
    prospector.scanner:
      check_interval: %s
    %s

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
	fingerprintEnhanced            = "file_identity.fingerprint: ~"
	fingerprintStatic              = "file_identity.fingerprint:\n      growing: false"
	fingerprintEnhancedKeepRemoved = fingerprintEnhanced + "\n    clean_removed: false"
)

func fingerprintCfg(logDir, checkInterval, fingerprintBlock, pathHome string) string {
	return fmt.Sprintf(fingerprintCfgBase, logDir, checkInterval, fingerprintBlock, pathHome)
}

func enhancedFingerprintCfg(logDir, checkInterval, pathHome string) string {
	return fingerprintCfg(logDir, checkInterval, fingerprintEnhanced, pathHome)
}

func staticFingerprintCfg(logDir, checkInterval, pathHome string) string {
	return fingerprintCfg(logDir, checkInterval, fingerprintStatic, pathHome)
}

// newFingerprintFilebeat builds a Filebeat process and the standard directory
// layout shared by every test in this file: a home (temp) dir, a failure-only
// output dump, and a logs/ subdir for the input's glob. It stops short of
// writing config or starting so callers can control write/start ordering
// (some tests create files before Start, some after).
func newFingerprintFilebeat(t *testing.T) (filebeat *integration.BeatProc, tempDir, logDir string) {
	t.Helper()
	filebeat = integration.NewFilebeat(t)
	tempDir = filebeat.TempDir()
	printOutputOnFailure(t, tempDir)
	logDir = filepath.Join(tempDir, "logs")
	require.NoError(t, os.MkdirAll(logDir, 0o755), "failed to create log directory")
	return filebeat, tempDir, logDir
}

// TestFilestreamFingerprintSmallFiles documents the static (non-growing)
// fingerprint behavior: files smaller than the fingerprint size (offset +
// length, 1024 bytes by default) are held back and not ingested until they
// grow past the threshold. TestFilestreamGrowingFingerprint is the growing
// counterpart, where below-threshold files ARE ingested immediately.
func TestFilestreamFingerprintSmallFiles(t *testing.T) {
	t.Parallel()
	filebeat, tempDir, logDir := newFingerprintFilebeat(t)

	file1 := filepath.Join(logDir, "file1.log")
	file2 := filepath.Join(logDir, "file2.log")
	file3 := filepath.Join(logDir, "file3.log")

	filebeat.WriteConfigFile(staticFingerprintCfg(logDir, "1s", tempDir))
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
		"ingestion from some files will be delayed, files need to be at "+
			"least 1024 in size for ingestion to start",
		5*time.Second,
		"expected the delayed-ingestion warning for the below-threshold files",
	)

	// output file isn't created yet as no event has been published. Thus, check
	// it manually
	path := filepath.Join(filebeat.TempDir(), "output-*.ndjson")
	files, err := filepath.Glob(path)
	require.NoError(t, err, "failed to glob output files")
	require.Empty(t, files, "expected no output file to be created yet as no event should have been published")

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
		"file size is too small for ingestion",
		5*time.Second, "expected the below-threshold files to be reported as too small",
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

// TestFilestreamGrowingFingerprint tests the growing_fingerprint file identity
// which allows files of any size to be ingested immediately. The fingerprint
// grows as the file grows, and the registry entry is migrated to the new key.
//
// This test includes both plain text and gzipped files to verify that growing
// fingerprint works correctly with compressed files.
//
// This is the counterpart to TestFilestreamFingerprintSmallFiles, which pins
// the static (non-growing) fingerprint behavior where below-threshold files are
// held back. The final phase grows the plain files past the fingerprint
// threshold and asserts they migrate from a growing (raw-hex) registry entry to
// a final SHA-256 entry, while the still-small gzipped files stay growing.
func TestFilestreamGrowingFingerprint(t *testing.T) {
	t.Parallel()
	filebeat, tempDir, logDir := newFingerprintFilebeat(t)

	file1 := filepath.Join(logDir, "file1.log")
	file2 := filepath.Join(logDir, "file2.log")
	file3 := filepath.Join(logDir, "file3.log")
	file4 := filepath.Join(logDir, "file4.log.gz")
	file5 := filepath.Join(logDir, "file5.log.gz")

	filebeat.WriteConfigFile(enhancedFingerprintCfg(logDir, "1s", tempDir))
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

	// Create a gzipped file with DIFFERENT header content. The fingerprint is
	// computed on the decompressed bytes, so file4 has its own fingerprint and
	// is NOT part of the file1-3 collision; it is tracked as a distinct file.
	headerGZ := gziptest.Compress(t,
		[]byte(generateLines("gzip header line", 1)), gziptest.CorruptNone)
	require.NoError(t, os.WriteFile(file4, headerGZ, 0644), "failed to write gzipped file")

	// The scanner reports the colliding duplicates by path only, never by the
	// raw fingerprint material.
	filebeat.WaitLogsContains(
		"points to an already known ingest target",
		10*time.Second,
		"expected the duplicate ingest targets to be reported",
	)

	// With growing_fingerprint, small files ARE ingested immediately (unlike regular fingerprint)
	// Due to collision (same content = same fingerprint), only ONE file's entry is created
	// but events ARE published. We wait for EOF on the first detected file.
	filebeat.WaitLogsContains(
		"End of file reached", // any of the 4 files might be the one ingested first
		10*time.Second,
		"file was not read to EOF",
	)

	// Two events: one from the file1-3 collision winner, plus one from the
	// distinct gzip file4.
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
	require.NoError(t, os.WriteFile(file5, file5ContentGZ, 0644),
		"failed to write gzipped file")

	// Wait for all files to be read to EOF, in any order (they are written at
	// nearly the same time). Gzipped files log "EOF has been reached. Closing."
	// instead of "End of file reached".
	filebeat.WaitLogsContainsAnyOrder(
		[]string{
			fmt.Sprintf("End of file reached: %s; Backoff now.", file1),
			fmt.Sprintf("End of file reached: %s; Backoff now.", file2),
			fmt.Sprintf("End of file reached: %s; Backoff now.", file3),
			fmt.Sprintf("EOF has been reached. Closing. Path='%s'", file5),
		},
		10*time.Second,
		"files were not fully read after growth/new gzipped file created",
	)

	// Total events: 4 files × 5 lines each = 20 events + 1 GZIP small file (1 line)
	filebeat.WaitPublishedEvents(10*time.Second, 21)

	// ===== Phase 3: Grow the plain files past the fingerprint threshold =====
	// Each plain file is ~250 bytes so far; +20 lines (~1000 bytes) pushes them
	// past the 1024-byte threshold, so their identity transitions from a
	// growing (raw-hex) key to a final SHA-256 key. Gzipped files cannot grow.
	appendToFile(t, file1, generateLines("file1 big line", 20))
	appendToFile(t, file2, generateLines("file2 big line", 20))
	appendToFile(t, file3, generateLines("file3 big line", 20))

	filebeat.WaitLogsContainsAnyOrder(
		[]string{
			fmt.Sprintf("End of file reached: %s; Backoff now.", file1),
			fmt.Sprintf("End of file reached: %s; Backoff now.", file2),
			fmt.Sprintf("End of file reached: %s; Backoff now.", file3),
		},
		10*time.Second,
		"plain files were not fully read after crossing the threshold",
	)

	// 21 (previous) + 3 files × 20 new lines = 81 events.
	filebeat.WaitPublishedEvents(15*time.Second, 81)

	// Harvester logs identify files by path in the source_file field.
	filebeat.WaitLogsContainsFromBeginning(
		`"source_file":"`+strings.ReplaceAll(file1, `\`, `\\`)+`"`,
		10*time.Second,
		"harvester logs must carry the file path in source_file",
	)

	// ===== Phase 4: Stop Filebeat =====
	filebeat.Stop()

	// Registry shape: the plain files crossed the threshold and are now keyed by
	// a final SHA-256 entry (no meta.fingerprint_len); the gzipped files stayed
	// below the threshold (fingerprint is computed on decompressed content) and
	// remain growing entries.
	assertSingleSHA256RegistryEntry(t, tempDir, file1)
	assertSingleSHA256RegistryEntry(t, tempDir, file2)
	assertSingleSHA256RegistryEntry(t, tempDir, file3)
	assertGrowingRegistryEntry(t, tempDir, file4)
	assertGrowingRegistryEntry(t, tempDir, file5)

	// The raw fingerprint material (hex of the file content, here the shared
	// header all growing raws start with) and the removed state-id field must
	// never appear in the logs.
	assertLogsDoNotContain(t, tempDir, hex.EncodeToString([]byte(headerContent)))
	assertLogsDoNotContain(t, tempDir, `"state-id"`)
}

// TestFilestreamGrowingFingerprint_update_while_stopped verifies growing
// fingerprint across a restart when files that start with identical content (a
// fingerprint collision) diverge with unique content while Filebeat is stopped.
// Each file must be tracked independently after the restart: no content is
// mixed up between files and nothing is re-read. The final assertion checks
// every file's events against its on-disk content, in order.
func TestFilestreamGrowingFingerprint_update_while_stopped(t *testing.T) {
	t.Parallel()
	filebeat, tempDir, logDir := newFingerprintFilebeat(t)

	file1 := filepath.Join(logDir, "file1.log")
	file2 := filepath.Join(logDir, "file2.log")
	file3 := filepath.Join(logDir, "file3.log")

	filebeat.WriteConfigFile(enhancedFingerprintCfg(logDir, "1s", tempDir))
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

	// ===== Phase 2: Grow only file1 =====
	// Only file1 is appended to (file2 and file3 stay at the shared header).
	// Once file1 diverges it leaves the collision; file2 and file3 still collide
	// with each other, so one of them may surface and publish its header here.
	// That makes the intermediate event count timing-dependent, so we only wait
	// for file1 to reach EOF (log-based) and defer all count/content assertions
	// to the deterministic final state after the restart.
	appendToFile(t, file1, generateLines("file1 unique line", 4))

	filebeat.WaitLogsContains(
		fmt.Sprintf("End of file reached: %s; Backoff now.", file1),
		10*time.Second, "file1 was not read to EOF")

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

	// Verify each file's events match its on-disk content, in order — proving
	// the shared collision header and the later per-file divergence were not
	// mixed up between files and nothing was re-read.
	events := readOutputEvents(t, tempDir)
	assertFileEvents(t, events, file1)
	assertFileEvents(t, events, file2)
	assertFileEvents(t, events, file3)
}

// TestFilestreamGrowingFingerprint_supersetFileNotConflated verifies that a
// file which appears already containing another file's full content PLUS more
// — so the existing file's fingerprint is a strict PREFIX of the new file's —
// is tracked as its own file and ingested from the beginning. The prefix
// relationship must NOT be mistaken for "the existing file grew".
//
// The prefix relationship is present from the first time the new file is seen
// (it is never identical to the existing one). It is covered both while Filebeat
// is running (b.log) and across a restart (c.log), which exercise different code
// paths: the watch loop's rename detection vs. the prospector's startup
// reconstruction of the short-fingerprint set.
func TestFilestreamGrowingFingerprint_supersetFileNotConflated(t *testing.T) {
	t.Parallel()
	filebeat, tempDir, logDir := newFingerprintFilebeat(t)

	aLog := filepath.Join(logDir, "a.log")
	bLog := filepath.Join(logDir, "b.log")
	cLog := filepath.Join(logDir, "c.log")

	// 4 lines, well below the 1024-byte threshold => tracked in the growing phase.
	shared := generateLines("shared line", 4)

	filebeat.WriteConfigFile(enhancedFingerprintCfg(logDir, "1s", tempDir))
	filebeat.Start()
	filebeat.WaitLogsContains("Input 'filestream' starting",
		10*time.Second, "filestream did not start")

	// ===== Phase 1: a.log is ingested (4 lines) =====
	writeTruncatingFile(t, aLog, shared)
	filebeat.WaitLogsContains(
		fmt.Sprintf("End of file reached: %s; Backoff now.", aLog),
		10*time.Second, "a.log was not read to EOF")
	filebeat.WaitPublishedEvents(5*time.Second, 4)

	// ===== Phase 2 (running): b.log = a.log's content + a unique line =====
	// b.log is never identical to a.log, but a.log's fingerprint is a strict
	// prefix of b.log's. b.log must be ingested in full (5 lines), not skipped
	// or conflated with a.log.
	writeTruncatingFile(t, bLog, shared+generateLines("b unique line", 1))
	filebeat.WaitLogsContains(
		fmt.Sprintf("End of file reached: %s; Backoff now.", bLog),
		10*time.Second, "b.log was not read to EOF")
	// 9 events: 4 (a.log) + 5 (b.log).
	filebeat.WaitPublishedEvents(10*time.Second, 9)

	// ===== Phase 3: stop, then create c.log (another superset) while stopped ==
	// c.log first appears to the prospector's startup reconstruction rather than
	// to the running watch loop.
	filebeat.Stop()
	writeTruncatingFile(t, cLog, shared+generateLines("c unique line", 1))

	// ===== Phase 4: restart; c.log is ingested in full, a/b not re-ingested ===
	filebeat.Start()
	filebeat.WaitLogsContains(
		fmt.Sprintf("End of file reached: %s; Backoff now.", cLog),
		10*time.Second, "c.log was not read to EOF")
	// 14 events: 9 (previous) + 5 (c.log).
	filebeat.WaitPublishedEvents(10*time.Second, 14)

	events := readOutputEvents(t, tempDir)
	assertFileEvents(t, events, aLog)
	assertFileEvents(t, events, bLog)
	assertFileEvents(t, events, cLog)
	require.Len(t, messagesForFile(events, aLog), 4, "a.log must not be re-ingested")
	require.Len(t, messagesForFile(events, bLog), 5, "b.log must be ingested in full")
	require.Len(t, messagesForFile(events, cLog), 5, "c.log must be ingested in full")
}

// TestFilestreamGrowingFingerprintTruncation tests that truncation with
// different content is treated as a new file (no prefix match = new entry).
func TestFilestreamGrowingFingerprintTruncation(t *testing.T) {
	t.Parallel()
	filebeat, tempDir, logDir := newFingerprintFilebeat(t)

	logFile := filepath.Join(logDir, "truncate.log")

	filebeat.WriteConfigFile(enhancedFingerprintCfg(logDir, "1s", tempDir))
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
}

// readOutputEvents reads all output files and returns parsed events sorted
// by file path, then by log offset.
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
				// Tolerate a torn final line: readOutputEvents may run while
				// Filebeat is still writing, so an incomplete last record is
				// expected and must not fail the test.
				t.Logf("failed to parse line: %s, error: %s", line, err)
				continue
			}
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
	for line := range strings.SplitSeq(string(data), "\n") {
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

// assertNoDuplicateMessages fails if any message appears more than once. Every
// generated line is unique, so a duplicate means content was re-ingested.
func assertNoDuplicateMessages(t *testing.T, msgs []string) {
	t.Helper()
	seen := make(map[string]struct{}, len(msgs))
	for _, msg := range msgs {
		_, duplicate := seen[msg]
		require.False(t, duplicate, "duplicate event detected: %s", msg)
		seen[msg] = struct{}{}
	}
}

// assertMonotonicOffsets fails unless every event's log offset is strictly
// greater than the previous one, proving the harvester continued from where it
// left off rather than re-reading from offset 0. Events must be pre-sorted (as
// readOutputEvents returns them).
func assertMonotonicOffsets(t *testing.T, events []outputEvent) {
	t.Helper()
	for i := 1; i < len(events); i++ {
		require.Greater(t, events[i].Log.Offset, events[i-1].Log.Offset,
			"offsets must be monotonically increasing (event %d vs %d)", i-1, i)
	}
}

// printOutputFileSorted reads the output file, parses each line as JSON,
// and prints the events sorted by file path, then by log offset.
func printOutputFileSorted(t *testing.T, tempDir string) {
	t.Helper()

	events := readOutputEvents(t, tempDir)
	if len(events) == 0 {
		t.Log("No output events found")
		return
	}

	t.Log("=== Output events sorted by file path, then by timestamp ===")
	for _, event := range events {
		t.Logf("[%s] %s @ offset %6d: %s",
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
	t.Parallel()
	filebeat, tempDir, logDir := newFingerprintFilebeat(t)

	appLog := filepath.Join(logDir, "app.log")
	appLogRenamed := filepath.Join(logDir, "app.log.1")

	filebeat.WriteConfigFile(enhancedFingerprintCfg(logDir, "1s", tempDir))
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

	assertNoDuplicateMessages(t, msgs)
	assertMonotonicOffsets(t, events)

	// ===== Phase 5: Stop Filebeat =====
	filebeat.Stop()
}

// TestFilestreamEnhancedFingerprint_ThresholdTransition verifies the SHA-256
// transition at the configured offset+length threshold. A file is created
// with content below the threshold,
// then grown past the threshold in the same Filebeat run. The prospector
// must migrate the registry key from the raw-hex form to a SHA-256 key
// without re-ingesting any content.
//
// Each generateLines line is ~50 bytes; default length is 1024.
//
//	Phase 1: 5 lines  ≈ 250 bytes  (below threshold)
//	Phase 2: + 25 lines ≈ +1250 bytes, total ≈ 1500 bytes (above threshold)
//
// Expected: 30 events total (5 + 25), one migration log, no duplicates.
func TestFilestreamEnhancedFingerprint_ThresholdTransition(t *testing.T) {
	t.Parallel()
	filebeat, tempDir, logDir := newFingerprintFilebeat(t)

	logFile := filepath.Join(logDir, "app.log")

	filebeat.WriteConfigFile(enhancedFingerprintCfg(logDir, "1s", tempDir))
	filebeat.Start()

	filebeat.WaitLogsContains("Input 'filestream' starting",
		10*time.Second, "filestream did not start")

	// Phase 1: small file (~250 bytes) — below threshold; raw-hex key.
	appendToFile(t, logFile, generateLines("phase1", 5))
	filebeat.WaitPublishedEvents(10*time.Second, 5)
	filebeat.WaitLogsContains(
		fmt.Sprintf("End of file reached: %s; Backoff now.", logFile),
		10*time.Second, "phase 1 did not reach EOF")

	// Phase 2: grow past threshold (~1500 bytes total) — must migrate to SHA-256.
	appendToFile(t, logFile, generateLines("phase2", 25))

	filebeat.WaitPublishedEvents(15*time.Second, 30)

	events := readOutputEvents(t, tempDir)
	require.Len(t, events, 30, "expected exactly 30 events (5 + 25); more means re-ingestion")

	msgs := messagesForFile(events, logFile)
	require.Len(t, msgs, 30, "all events should be attributed to the test file")

	assertNoDuplicateMessages(t, msgs)
	assertMonotonicOffsets(t, events)

	filebeat.Stop()

	// Registry state: the file should end up under a SHA-256 (64-char) key,
	// with the original raw-hex key having been removed by the migration.
	assertFingerprintMigratedToSHA256(t, tempDir, logFile)
}

// TestFilestreamEnhancedFingerprint_NoDuplicationOnUpgrade is the no-data-
// duplication guarantee end-to-end. A user running with the default
// (static) fingerprint identity already has files at or above threshold
// indexed in the registry. When they run with the enhanced fingerprint enabled,
// no re-ingestion of files already at threshold should happen.
//
// The test:
//  1. Starts Filebeat with the static fingerprint config.
//  2. Writes a large file (above threshold) and waits for ingestion.
//  3. Stops Filebeat.
//  4. Switches the config to `growing: true` (same input id, same path).
//  5. Starts Filebeat.
//  6. Verifies no additional events are published.
//
// The promise: opting in to growing is contained to small files.
func TestFilestreamEnhancedFingerprint_NoDuplicationOnUpgrade(t *testing.T) {
	t.Parallel()
	filebeat, tempDir, logDir := newFingerprintFilebeat(t)

	largeFile := filepath.Join(logDir, "large.log")
	smallFile := filepath.Join(logDir, "small.log")

	// Phase 1: static fingerprint config. Two files exist from the start:
	//   - largeFile: 30 lines (~1500 bytes), above threshold → ingested
	//     under static; ends up keyed by its SHA-256 in the registry.
	//   - smallFile: 5 lines  (~250 bytes), below threshold → dropped by
	//     static (errFileTooSmall); no registry entry created.
	filebeat.WriteConfigFile(staticFingerprintCfg(logDir, "1s", tempDir))
	appendToFile(t, largeFile, generateLines("large", 30))
	appendToFile(t, smallFile, generateLines("small", 5))

	filebeat.Start()
	filebeat.WaitLogsContains("Input 'filestream' starting",
		10*time.Second, "filestream did not start under static config")

	filebeat.WaitPublishedEvents(15*time.Second, 30)
	filebeat.WaitLogsContains(
		fmt.Sprintf("End of file reached: %s; Backoff now.", largeFile),
		10*time.Second, "static phase did not reach EOF for the large file")

	filebeat.Stop()

	staticEvents := readOutputEvents(t, tempDir)
	assert.Len(t, staticEvents, 30, "static phase should have ingested only the large file's 30 lines")
	assert.Empty(t, messagesForFile(staticEvents, smallFile),
		"static phase must not produce any events for the below-threshold small file")

	// Phase 2: switch to growing. The promise has two halves:
	//   1. largeFile's existing SHA-256 entry is reused → no re-ingestion.
	//   2. smallFile is now eligible (growing tracks below-threshold files)
	//      → its 5 lines are ingested for the first time.
	// Expected post-upgrade total: 30 (large, unchanged) + 5 (small, new) = 35.
	filebeat.WriteConfigFile(enhancedFingerprintCfg(logDir, "1s", tempDir))
	filebeat.Start()
	filebeat.WaitLogsContains("Input 'filestream' starting",
		10*time.Second, "filestream did not restart under growing config")

	filebeat.WaitPublishedEvents(15*time.Second, 35)
	filebeat.WaitLogsContains(
		fmt.Sprintf("End of file reached: %s; Backoff now.", smallFile),
		10*time.Second, "growing phase did not reach EOF for the small file")

	postUpgrade := readOutputEvents(t, tempDir)
	assert.Len(t, postUpgrade, 35,
		"expected 35 events after upgrade (30 large + 5 small); more means re-ingestion, fewer means small file not tracked")
	assert.Len(t, messagesForFile(postUpgrade, largeFile), 30,
		"opting in to growing must not re-ingest the large file")
	assert.Len(t, messagesForFile(postUpgrade, smallFile), 5,
		"it should have ingested the previously-dropped small file")

	filebeat.Stop()

	// Registry state: largeFile keyed by the same SHA-256 from the static
	// phase (no extra entries on upgrade); smallFile keyed by the bounded hash
	// with a non-zero meta.fingerprint_len (still growing, below threshold).
	assertSingleSHA256RegistryEntry(t, tempDir, largeFile)
	assertGrowingRegistryEntry(t, tempDir, smallFile)
}

// TestFilestreamEnhancedFingerprint_DisableGrowingAfterEnabling exercises the
// opt-out (revert) path: a deployment runs with Enhanced Fingerprint enabled
// (the 9.5 default), then falls back to the legacy static behavior with
// `file_identity.fingerprint.growing: false`.
//
// It documents the one transition that is not free of side effects, and why:
//
//   - A file already at or above the fingerprint size keys on its SHA-256,
//     identical in both modes, so opting out never re-ingests it.
//   - A file still below the fingerprint size was tracked in the growing phase
//     under a raw-hex key. Static mode never computes that key, so the entry is
//     orphaned and the file is held back until it reaches offset+length; when it
//     does, it is picked up under a fresh SHA-256 key from offset 0, re-ingesting
//     the bytes already read during the growing phase.
func TestFilestreamEnhancedFingerprint_DisableGrowingAfterEnabling(t *testing.T) {
	t.Parallel()
	filebeat, tempDir, logDir := newFingerprintFilebeat(t)

	largeFile := filepath.Join(logDir, "large.log")
	smallFile := filepath.Join(logDir, "small.log")

	// Phase 1 (growing enabled): large file (~1500 bytes) above threshold,
	// small file (~250 bytes) below it. Both are ingested (30 + 5 = 35 events).
	filebeat.WriteConfigFile(enhancedFingerprintCfg(logDir, "1s", tempDir))
	appendToFile(t, largeFile, generateLines("large", 30))
	appendToFile(t, smallFile, generateLines("small", 5))

	filebeat.Start()
	filebeat.WaitLogsContains("Input 'filestream' starting",
		10*time.Second, "filestream did not start under growing config")

	filebeat.WaitPublishedEvents(15*time.Second, 35)
	filebeat.WaitLogsContains(
		fmt.Sprintf("End of file reached: %s; Backoff now.", smallFile),
		10*time.Second, "growing phase did not reach EOF for the small file")

	filebeat.Stop()

	growingEvents := readOutputEvents(t, tempDir)
	assert.Len(t, growingEvents, 35,
		"growing phase should have ingested 30 (large) + 5 (small) = 35 lines")
	assert.Len(t, messagesForFile(growingEvents, largeFile), 30,
		"growing phase should have ingested all 30 lines of the large file")
	assert.Len(t, messagesForFile(growingEvents, smallFile), 5,
		"growing phase should have ingested the 5 lines of the below-threshold small file")

	// Large file keys on SHA-256; small file keys on a raw-hex growing entry.
	assertSingleSHA256RegistryEntry(t, tempDir, largeFile)
	assertGrowingRegistryEntry(t, tempDir, smallFile)

	// Phase 2: opt out (growing: false) and restart. The large file's SHA-256
	// entry is reused; the still-below-threshold small file is held back.
	filebeat.WriteConfigFile(staticFingerprintCfg(logDir, "1s", tempDir))
	filebeat.Start()
	filebeat.WaitLogsContains("Input 'filestream' starting",
		10*time.Second, "filestream did not restart under static config")

	// Held-back proof: static mode logs this every scan for the below-threshold file.
	filebeat.WaitLogsContains(
		"is too small for ingestion",
		10*time.Second, "static mode did not report the below-threshold small file as too small")
	filebeat.WaitLogsContains(
		fmt.Sprintf("End of file reached: %s; Backoff now.", largeFile),
		10*time.Second, "static phase did not reach EOF for the large file")

	afterOptOut := readOutputEvents(t, tempDir)
	assert.Len(t, afterOptOut, 35,
		"opting out must not re-ingest anything while the small file is still below threshold")
	assert.Len(t, messagesForFile(afterOptOut, smallFile), 5,
		"the below-threshold small file must not be re-read yet under static mode")

	// Phase 3: grow the small file past the threshold with distinct content.
	// Static mode computes its SHA-256 for the first time, finds no matching
	// entry, and harvests from offset 0 — re-ingesting the original 5 lines.
	appendToFile(t, smallFile, generateLines("small grow", 25))

	filebeat.WaitLogsContains(
		fmt.Sprintf("End of file reached: %s; Backoff now.", smallFile),
		15*time.Second, "static phase did not reach EOF for the grown small file")
	// 35 (phase 1) + 30 (small file re-read in full: 5 original + 25 new) = 65.
	filebeat.WaitPublishedEvents(15*time.Second, 65)

	filebeat.Stop()

	final := readOutputEvents(t, tempDir)
	assert.Len(t, final, 65,
		"expected 65 events: 35 from the growing phase + 30 from re-reading the "+
			"small file in full once it crossed the threshold under static mode")
	assert.Len(t, messagesForFile(final, largeFile), 30,
		"the completed large file must never be re-ingested when opting out")

	// Concrete proof of the duplication: the first original line appears twice
	// (once in the growing phase, once re-ingested under static mode).
	firstOriginal := strings.TrimSuffix(generateLines("small", 1), "\n")
	dupCount := 0
	for _, m := range messagesForFile(final, smallFile) {
		if m == firstOriginal {
			dupCount++
		}
	}
	assert.Equal(t, 2, dupCount, "the first below-threshold line should appear twice")

	// The small file now has an active final (SHA-256) entry. The orphaned
	// growing entry may linger until clean_inactive removes it, so we assert on
	// the presence of the final entry rather than exclusivity (which is why the
	// existing assertSingleSHA256RegistryEntry helper is not used here).
	smallFinal, smallGrowing := activeFingerprintEntries(readFingerprintRegistry(t, tempDir), smallFile)
	assert.Len(t, smallFinal, 1,
		"expected exactly one active final SHA-256 entry for the small file; got final=%d growing=%d",
		len(smallFinal), len(smallGrowing))
	assertSingleSHA256RegistryEntry(t, tempDir, largeFile)
}

// seedLegacyFingerprintRegistry writes a filestream registry in the exact
// on-disk format a pre-growing-fingerprint Filebeat produced: a single static
// fingerprint entry keyed by the SHA-256 of the file's first 1024 bytes (the
// default fingerprint length at offset 0), with the cursor at the file's size
// and a meta of only {source, identifier_name} — no growing-fingerprint fields.
//
// The cursor carries only `offset` (no `eof`): the cursor's `eof` field was
// introduced by the later GZIP feature, so a pre-growing-fingerprint registry
// never set it. Seeding `eof:true` would also wrongly trip the "GZIP file
// already read to EOF, not reading it again" guard under compression:auto and
// prevent the file from being tailed.
//
// The `updated` timestamp uses go-structform's time encoding (what the memlog
// writes); its exact value is immaterial here because clean_inactive is disabled.
func seedLegacyFingerprintRegistry(t *testing.T, tempDir, inputID, filePath string, fileSize int64, sha256hex string) {
	t.Helper()
	regDir := filepath.Join(tempDir, "data", "registry", "filebeat")
	require.NoError(t, os.MkdirAll(regDir, 0o755), "failed to create registry dir")

	key := fmt.Sprintf("filestream::%s::fingerprint::%s", inputID, sha256hex)
	// The literal memlog log.json format: an op header line followed by the
	// entry line.
	op := `{"op":"set","id":1}`
	entry := fmt.Sprintf(
		`{"k":%q,"v":{"ttl":-1,"updated":[515683809191,1781771002],"cursor":{"offset":%d},"meta":{"source":%q,"identifier_name":"fingerprint"}}}`,
		key, fileSize, filePath)
	require.NoError(t,
		os.WriteFile(filepath.Join(regDir, "log.json"), []byte(op+"\n"+entry+"\n"), 0o644),
		"failed to write registry log.json")
	require.NoError(t,
		os.WriteFile(filepath.Join(regDir, "meta.json"), []byte(`{"version":"1"}`), 0o644),
		"failed to write registry meta.json")
}

// TestFilestreamEnhancedFingerprint_ReadsLegacyStaticRegistry is the
// cross-version backwards-compatibility guarantee: a registry written by a
// Filebeat from BEFORE growing fingerprint existed is read correctly by the new
// growing-fingerprint-by-default Filebeat. It covers both kinds of pre-existing
// file, and that ingestion continues for each after the upgrade:
//
//  1. A file SMALLER than the fingerprint threshold. The old static Filebeat
//     dropped it (errFileTooSmall, no registry entry), so growing mode ingests
//     it for the first time, then keeps ingesting as it grows.
//  2. A file LARGER than the threshold, already fully ingested by the old
//     Filebeat (a legacy SHA-256 entry at EOF). It must NOT be re-ingested, and
//     newly appended bytes must be ingested, continuing from the legacy offset.
//
// Unlike the config-switch upgrade tests (which write the registry with the new
// binary in static mode), this seeds the registry from raw bytes in the exact
// legacy on-disk format, so it pins on-disk format compatibility itself,
// independent of what the new binary writes.
func TestFilestreamEnhancedFingerprint_ReadsLegacyStaticRegistry(t *testing.T) {
	t.Parallel()
	filebeat, tempDir, logDir := newFingerprintFilebeat(t)

	// Case 2: a file above the 1024-byte threshold, already fully ingested by
	// the old (static) Filebeat -> a legacy SHA-256 registry entry at EOF.
	largeFile := filepath.Join(logDir, "large.log")
	largeOld := generateLines("large-old", 40) // ~2000 bytes, > 1024
	writeTruncatingFile(t, largeFile, largeOld)
	largeInfo, err := os.Stat(largeFile)
	require.NoError(t, err, "failed to stat the large file")
	// The legacy key is the SHA-256 of the first 1024 bytes (offset 0, default
	// length) — identical to what static fingerprinting computed.
	sum := sha256.Sum256([]byte(largeOld)[:1024])
	largeSHA := hex.EncodeToString(sum[:])
	seedLegacyFingerprintRegistry(t, tempDir, "test-enhanced-fingerprint", largeFile, largeInfo.Size(), largeSHA)

	// Case 1: a file below the threshold. The old (static) Filebeat dropped it,
	// so it has NO registry entry; growing mode ingests it now.
	smallFile := filepath.Join(logDir, "small.log")
	smallOld := generateLines("small-old", 5) // ~250 bytes, < 1024
	writeTruncatingFile(t, smallFile, smallOld)

	// Start the new Filebeat with growing enabled by default, over the legacy registry.
	filebeat.WriteConfigFile(enhancedFingerprintCfg(logDir, "1s", tempDir))
	filebeat.Start()
	filebeat.WaitLogsContains("Input 'filestream' starting",
		10*time.Second, "filestream did not start")

	// Phase 1: only the small file's 5 pre-existing lines are ingested. The
	// large file is matched to its legacy SHA-256 registry entry, resumes from
	// its EOF offset and contributes nothing — proof the old-format entry was
	// read and reused. WaitPublishedEvents asserts an exact count, so
	// re-ingesting the large file's 40 lines would overshoot 5 and fail here.
	filebeat.WaitPublishedEvents(20*time.Second, 5)

	// Phase 2: append to both files. The new bytes must be ingested, continuing
	// from each file's current offset (the large file from its legacy offset,
	// without re-reading its pre-existing content).
	appendToFile(t, largeFile, generateLines("large-new", 10))
	appendToFile(t, smallFile, generateLines("small-new", 10))

	// Total = 5 (small old) + 10 (small new) + 10 (large new) = 25.
	filebeat.WaitPublishedEvents(30*time.Second, 25)

	filebeat.Stop()

	events := readOutputEvents(t, tempDir)
	require.Len(t, events, 25,
		"expected 25 events (5 small-old + 10 small-new + 10 large-new); more means the large file was re-ingested")

	// The large file contributes ONLY its 10 appended lines — its pre-existing
	// 40 lines (from the legacy registry) are never re-ingested.
	largeMsgs := messagesForFile(events, largeFile)
	assert.Len(t, largeMsgs, 10, "the large file must contribute only its 10 appended lines, not its pre-existing content")
	for _, m := range largeMsgs {
		assert.Contains(t, m, "large-new",
			"the large file must not re-ingest pre-existing content; got %q", m)
	}

	// The small file contributes all 15 lines (5 pre-existing + 10 appended):
	// it had no legacy entry, so growing mode ingests it from the start and
	// continues as it grows.
	smallMsgs := messagesForFile(events, smallFile)
	assert.Len(t, smallMsgs, 15, "the small file must contribute its 5 pre-existing + 10 appended lines")

	// The large file's entry remains a single active SHA-256 entry keyed by the
	// original fingerprint (the appends stayed above threshold; no new key).
	assertSingleSHA256RegistryEntry(t, tempDir, largeFile)
	state := readFingerprintRegistry(t, tempDir)
	wantKey := fmt.Sprintf("filestream::test-enhanced-fingerprint::fingerprint::%s", largeSHA)
	e, ok := state[wantKey]
	require.True(t, ok, "the original legacy key %q must still be present", wantKey)
	assert.False(t, e.removed, "the original legacy key must remain active (not removed/migrated)")
}

// TestFilestreamEnhancedFingerprint_NoDuplicationConfigReload is
// the same no-data-duplication guarantee as
// TestFilestreamEnhancedFingerprint_NoDuplicationOnUpgrade but exercises
// Filebeat's live config-reload path (filebeat.config.inputs with
// reload.enabled). The input is restarted, when the inputs.d/*.yml
// file is edited. On both configs, the inputs have the same id, so the registry
// state is preserved across the reload — the new growing-enabled prospector
// must reuse the existing SHA-256 entry and produce no new events for a file
// that was already at threshold.
func TestFilestreamEnhancedFingerprint_NoDuplicationConfigReload(t *testing.T) {
	t.Parallel()
	filebeat, tempDir, logDir := newFingerprintFilebeat(t)
	inputsDir := filepath.Join(tempDir, "inputs.d")
	require.NoError(t, os.MkdirAll(inputsDir, 0o755), "failed to create inputs.d directory")

	inputConfigFile := filepath.Join(inputsDir, "filestream.yml")
	largeFile := filepath.Join(logDir, "large.log")
	smallFile := filepath.Join(logDir, "small.log")

	// Filebeat config
	configTemplate := `
filebeat.config.inputs:
  path: %s/*.yml
  reload.enabled: true
  reload.period: 1s

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

	// Initial input config: static fingerprint.
	staticInputCfg := `
- type: filestream
  id: test-enhanced-fingerprint
  enabled: true
  paths:
    - %s/*.log
  prospector.scanner:
    check_interval: 1s
  file_identity.fingerprint:
    growing: false
`

	// Post-reload input config: growing enabled.
	growingInputCfg := `
- type: filestream
  id: test-enhanced-fingerprint
  enabled: true
  paths:
    - %s/*.log
  prospector.scanner:
    check_interval: 1s
  file_identity.fingerprint:
    growing: true
`

	filebeat.WriteConfigFile(fmt.Sprintf(configTemplate, inputsDir, tempDir))

	// Phase 1: write the initial (static) input config and the log files.
	// Two files exist from the start:
	//   - largeFile: 30 lines (~1500 bytes), above threshold → ingested
	//     under static; SHA-256 entry in the registry.
	//   - smallFile: 5 lines  (~250 bytes), below threshold → dropped by
	//     static (errFileTooSmall); no registry entry.
	require.NoError(t,
		os.WriteFile(inputConfigFile, fmt.Appendf(nil, staticInputCfg, logDir), 0o644),
		"failed to write initial static input config")
	appendToFile(t, largeFile, generateLines("large", 30))
	appendToFile(t, smallFile, generateLines("small", 5))

	filebeat.Start()
	filebeat.WaitLogsContains("Input 'filestream' starting",
		15*time.Second, "filestream did not start under the static input config")

	filebeat.WaitPublishedEvents(15*time.Second, 30)
	filebeat.WaitLogsContains(
		fmt.Sprintf("End of file reached: %s; Backoff now.", largeFile),
		10*time.Second, "static phase did not reach EOF for the large file")

	staticEvents := readOutputEvents(t, tempDir)
	require.Len(t, staticEvents, 30, "static phase should have ingested only the large file's 30 lines")
	require.Empty(t, messagesForFile(staticEvents, smallFile),
		"static phase must not produce any events for the below-threshold small file")

	// Phase 2: replace the input config to enable growing fingerprint.
	// Expected post-reload total: 30 (large, unchanged) + 5 (small, new) = 35.
	require.NoError(t,
		os.WriteFile(inputConfigFile, fmt.Appendf(nil, growingInputCfg, logDir), 0o644),
		"failed to replace input config with growing-enabled version")

	// Wait for the input to be stopped
	filebeat.WaitLogsContains("Runner: 'filestream' has stopped",
		30*time.Second, "input runner was not stopped by the reloader")
	filebeat.WaitLogsContains("Input 'filestream' starting",
		15*time.Second, "filestream did not restart under the growing input config")

	filebeat.WaitPublishedEvents(15*time.Second, 35)
	filebeat.WaitLogsContains(
		fmt.Sprintf("End of file reached: %s; Backoff now.", smallFile),
		10*time.Second, "growing phase did not reach EOF for the small file")

	postReload := readOutputEvents(t, tempDir)
	assert.Len(t, postReload, 35,
		"expected 35 events after reload (30 large + 5 small); more means re-ingestion, fewer means small file not tracked")
	assert.Len(t, messagesForFile(postReload, largeFile), 30,
		"enabling growing via config reload must not re-ingest the large file")
	assert.Len(t, messagesForFile(postReload, smallFile), 5,
		"config reload to growing must now ingest the previously-dropped small file")

	filebeat.Stop()

	// Registry state: largeFile keyed by the same SHA-256 produced under
	// static (no extra entries from the reload); smallFile keyed by raw-hex
	// (still below threshold under growing).
	assertSingleSHA256RegistryEntry(t, tempDir, largeFile)
	assertGrowingRegistryEntry(t, tempDir, smallFile)
}

// TestFilestreamEnhancedFingerprint_ThresholdTransitionAcrossRestart covers
// the case where a file crosses the threshold while Filebeat is stopped.
// On restart, the scanner sees a file at/above threshold whose registry
// entry is still a raw-hex (growing) key. On the watch loop's first scan the
// descriptor carries both the final SHA-256 (Fingerprint.Sum) and the raw
// header (Fingerprint.Raw); the prospector prefix-matches the raw header
// against the raw-hex registry entry and migrates the key to the final
// SHA-256, resuming from the stored offset without re-ingesting content
// already harvested before the stop.
func TestFilestreamEnhancedFingerprint_ThresholdTransitionAcrossRestart(t *testing.T) {
	t.Parallel()
	filebeat, tempDir, logDir := newFingerprintFilebeat(t)

	logFile := filepath.Join(logDir, "app.log")

	filebeat.WriteConfigFile(enhancedFingerprintCfg(logDir, "1s", tempDir))

	// Phase 1: small file (~250 bytes), tracked with raw-hex (growing).
	appendToFile(t, logFile, generateLines("before-restart", 5))

	filebeat.Start()
	filebeat.WaitLogsContains("Input 'filestream' starting",
		10*time.Second, "filestream did not start")
	filebeat.WaitPublishedEvents(10*time.Second, 5)
	filebeat.WaitLogsContains(
		fmt.Sprintf("End of file reached: %s; Backoff now.", logFile),
		10*time.Second, "phase 1 did not reach EOF")

	// Phase 2: stop Filebeat. Append content past threshold while stopped.
	filebeat.Stop()
	appendToFile(t, logFile, generateLines("after-restart", 25)) // total ~1500 bytes, above threshold

	// Phase 3: restart. The first-scan migration described above runs and the
	// harvester resumes from the stored offset.
	filebeat.Start()
	filebeat.WaitLogsContains("Input 'filestream' starting",
		10*time.Second, "filestream did not restart")

	filebeat.WaitPublishedEvents(15*time.Second, 30)

	events := readOutputEvents(t, tempDir)
	require.Len(t, events, 30,
		"expected exactly 30 events (5 before + 25 after); more means content was re-ingested across the restart")

	assertNoDuplicateMessages(t, messagesForFile(events, logFile))

	filebeat.Stop()

	// Registry state: the file should end up under a SHA-256 (64-char) key,
	// with the raw-hex key from the pre-restart growing phase removed.
	// This is the strongest evidence that the threshold-transition migration
	// across the restart actually happened.
	assertFingerprintMigratedToSHA256(t, tempDir, logFile)
}

// TestFilestreamEnhancedFingerprint_RenameAndThresholdCrossing verifies the
// rename + threshold-crossing case while filebeat is running: a
// file is renamed AND grown past the configured threshold within a single
// scan interval. The fileWatcher's prefix-match phase
// recognises the renamed-and-grown file as the
// same identity as the previous registry entry, emits OpRename, and the
// prospector migrates the raw-hex registry key to the SHA-256 form. No
// content is re-read.
//
//	Phase 1: 5 lines  ≈ 250 bytes  in app.log  (below threshold)
//	Phase 2: rename app.log → app.log.1 + append 25 lines (~1500 bytes total, above threshold)
//
// Expected: 30 events total, no duplicates, monotonic offsets. Registry
// ends up with one active SHA-256 entry for app.log.1 and the raw-hex
// entry from before the rename is removed.
func TestFilestreamEnhancedFingerprint_RenameAndThresholdCrossing(t *testing.T) {
	t.Parallel()
	filebeat, homeDir, logDir := newFingerprintFilebeat(t)

	appLog := filepath.Join(logDir, "app.log")
	appLogRenamed := filepath.Join(logDir, "app.log.1")

	// 1s scan interval ensures the scanner runs again after the rename +
	// append below, so the in-process prefix-match rename detection actually
	// observes the file under its new path AND past threshold within the
	// test window. (With a longer interval the harvester may finish reading
	// via the still-open fd before the scanner re-scans, leaving the
	// registry under the original raw-hex key.)
	filebeat.WriteConfigFile(enhancedFingerprintCfg(logDir, "1s", homeDir))
	filebeat.Start()

	filebeat.WaitLogsContains("Input 'filestream' starting",
		10*time.Second, "filestream did not start")

	// Phase 1: small file (~250 bytes) — below threshold, raw-hex key.
	appendToFile(t, appLog, generateLines("app original line", 5))
	filebeat.WaitPublishedEvents(15*time.Second, 5)
	filebeat.WaitLogsContains(
		fmt.Sprintf("End of file reached: %s; Backoff now.", appLog),
		15*time.Second, "phase 1 did not reach EOF")

	// Phase 2: in a single inter-scan window, rename AND grow past threshold.
	require.NoError(t, os.Rename(appLog, appLogRenamed),
		"failed to rename app.log -> app.log.1")
	appendToFile(t, appLogRenamed, generateLines("app new line", 25))

	// Wait for the rename + grow to be observed and the additional lines
	// ingested. 30 events total = 5 original + 25 new.
	filebeat.WaitPublishedEvents(30*time.Second, 30)

	// Wait for the scanner to observe the file under its new path (the
	// prefix-match rename pass detects the rename + threshold crossing and
	// emits OpRename) — without this, the test can end before the migration
	// has had a chance to run.
	filebeat.WaitLogsContains(
		fmt.Sprintf("File %s has been renamed to %s", appLog, appLogRenamed),
		15*time.Second,
		"scanner did not observe the rename + threshold crossing within the test window")

	events := readOutputEvents(t, homeDir)
	assert.Len(t, events, 30,
		"expected exactly 30 events (5 original + 25 new); more means re-read from offset 0")

	// All events should be attributed to the same harvester. (The harvester
	// caches the open path at startup, so the message-path of events from
	// after the rename remains the original path — same behaviour as
	// TestFilestreamGrowingFingerprint_rename_and_grow.)
	msgs := messagesForFile(events, appLog)
	assert.Len(t, msgs, 30, "all events should be attributed to the original path (harvester known behaviour)")

	assertNoDuplicateMessages(t, msgs)
	assertMonotonicOffsets(t, events)

	filebeat.Stop()

	// Registry state: the renamed file ends up under a SHA-256 (64-char)
	// key with Source pointing to the new path; the original raw-hex key
	// (from the pre-rename, pre-threshold scan) has been removed by the
	// migration. This is direct evidence the prefix-match rename pass ran
	// against GrowingFingerprint and the prospector migrated the key.
	assertFingerprintMigratedToSHA256(t, homeDir, appLogRenamed)
}

// TestFilestreamEnhancedFingerprint_RenameAndThresholdAcrossRestart is the
// hardest of the threshold scenarios: BOTH rename and threshold-crossing
// happen while Filebeat is stopped.
// Requires `clean_removed: false`.
func TestFilestreamEnhancedFingerprint_RenameAndThresholdAcrossRestart(t *testing.T) {
	t.Parallel()
	filebeat, tempDir, logDir := newFingerprintFilebeat(t)

	appLog := filepath.Join(logDir, "app.log")
	appLogRenamed := filepath.Join(logDir, "app.log.1")

	// clean_removed disabled so the stopped-rename entry survives startup and the prospector can
	// migrate it.
	filebeat.WriteConfigFile(
		fingerprintCfg(logDir, "1s", fingerprintEnhancedKeepRemoved, tempDir))

	// Phase 1: small file (~250 bytes) — below threshold, raw-hex registry key.
	appendToFile(t, appLog, generateLines("before-restart line", 5))
	filebeat.Start()
	filebeat.WaitLogsContains("Input 'filestream' starting",
		10*time.Second, "filestream did not start")
	filebeat.WaitPublishedEvents(10*time.Second, 5)
	filebeat.WaitLogsContains(
		fmt.Sprintf("End of file reached: %s; Backoff now.", appLog),
		10*time.Second, "phase 1 did not reach EOF")

	// Phase 2: stop Filebeat. Rename + append past threshold while stopped.
	filebeat.Stop()
	require.NoError(t, os.Rename(appLog, appLogRenamed),
		"failed to rename app.log -> app.log.1 while filebeat is stopped")
	appendToFile(t, appLogRenamed, generateLines("after-restart line", 25)) // total ~1500 bytes

	// Phase 3: restart. fileWatcher sees app.log.1 as new.
	filebeat.Start()
	filebeat.WaitLogsContains("Input 'filestream' starting",
		10*time.Second, "filestream did not restart")

	filebeat.WaitPublishedEvents(15*time.Second, 30)

	events := readOutputEvents(t, tempDir)
	require.Len(t, events, 30,
		"expected exactly 30 events (5 before-restart + 25 after-restart); more means content was re-ingested across the restart")

	// Pre-restart events were emitted under the original path (app.log) during the first filebeat
	// run; post-restart events come out under the new path (app.log.1).
	preMsgs := messagesForFile(events, appLog)
	postMsgs := messagesForFile(events, appLogRenamed)
	require.Len(t, preMsgs, 5, "5 before-restart events should be attributed to the original path")
	require.Len(t, postMsgs, 25, "25 after-restart events should be attributed to the post-rename path")

	assertNoDuplicateMessages(t, append(append([]string{}, preMsgs...), postMsgs...))

	filebeat.Stop()

	// Registry state: app.log.1 ends up under a SHA-256 (64-char) key, raw-hex key for the old
	// app.log path is removed by the migration.
	assertFingerprintMigratedToSHA256(t, tempDir, appLogRenamed)
}

// TestFilestreamEnhancedFingerprint_RenameBelowThresholdAcrossRestartKeepRemoved
// documents the deliberate behaviour for a file that is renamed AND appended
// while Filebeat is stopped, staying below the fingerprint threshold, with
// clean_removed:false (so the pre-rename entry is not cleaned by path at
// startup).
//
// The renamed file reappears under a NEW path as OpCreate on the first scan.
// Its raw-hex fingerprint carries the stored entry's fingerprint as a strict
// prefix, but a below-threshold fingerprint is not Complete(), and the
// prospector deliberately does NOT accept a non-Complete() cross-path prefix
// match as a continuation: a short prefix is too weak to tell "the same file
// grew" apart from "a distinct file that merely shares a header prefix".
// Resuming here would reintroduce the silent data loss tracked in
// https://github.com/elastic/beats/issues/51417 (a distinct new file adopting a
// vanished file's cursor). So the file is re-ingested from offset 0:
// at-least-once duplication, which is recoverable, is preferred over data loss,
// which is not.
//
// The full-size fingerprint path is unaffected: once above the threshold the
// identity is a hash over the whole fingerprint window and a moved+appended
// file resumes from its cursor as before (see
// TestFilestreamEnhancedFingerprint_RenameAndThresholdAcrossRestart).
func TestFilestreamEnhancedFingerprint_RenameBelowThresholdAcrossRestartKeepRemoved(t *testing.T) {
	t.Parallel()
	filebeat, tempDir, logDir := newFingerprintFilebeat(t)

	appLog := filepath.Join(logDir, "app.log")
	appLogRenamed := filepath.Join(logDir, "app.log.1")

	// clean_removed disabled so the pre-rename entry (source app.log) survives
	// startup and can be recovered by the prospector's prefix-match fallback.
	filebeat.WriteConfigFile(
		fingerprintCfg(logDir, "1s", fingerprintEnhancedKeepRemoved, tempDir))

	// Phase 1: small file (one line, ~50 bytes) — below threshold, raw-hex key.
	appendToFile(t, appLog, generateLines("app original line", 1))
	filebeat.Start()
	filebeat.WaitLogsContains("Input 'filestream' starting",
		10*time.Second, "filestream did not start")
	filebeat.WaitPublishedEvents(10*time.Second, 1)
	filebeat.WaitLogsContains(
		fmt.Sprintf("End of file reached: %s; Backoff now.", appLog),
		10*time.Second, "phase 1 did not reach EOF")

	// Phase 2: stop, then rename + append while stopped, staying below the
	// threshold (5 lines total ≈ 250 bytes < 1024).
	filebeat.Stop()
	require.NoError(t, os.Rename(appLog, appLogRenamed),
		"failed to rename app.log -> app.log.1 while filebeat is stopped")
	appendToFile(t, appLogRenamed, generateLines("app appended line", 4))

	// Phase 3: restart. The renamed file arrives as OpCreate under the new path.
	// Below threshold it is NOT recognised as the grown app.log (see docstring),
	// so it is re-ingested in full from offset 0.
	filebeat.Start()
	filebeat.WaitLogsContains("Input 'filestream' starting",
		10*time.Second, "filestream did not restart")
	filebeat.WaitLogsContains(
		fmt.Sprintf("End of file reached: %s; Backoff now.", appLogRenamed),
		10*time.Second, "renamed file was not read to EOF")

	// 1 (original, old path) + 5 (full re-ingest, new path) = 6. Resuming from
	// the stored cursor would instead yield 5.
	filebeat.WaitPublishedEvents(15*time.Second, 6)

	filebeat.Stop()

	events := readOutputEvents(t, tempDir)
	require.Len(t, events, 6,
		"expected 6 events (1 original + 5 re-ingested); 5 would mean the renamed file resumed from the cursor")

	// The original line is published twice: once under the old path (phase 1)
	// and once as part of the full re-ingest under the new path. This
	// at-least-once duplication is the accepted trade-off (see docstring).
	require.Len(t, messagesForFile(events, appLog), 1,
		"the original line should be attributed once to the pre-rename path")
	require.Len(t, messagesForFile(events, appLogRenamed), 5,
		"the renamed file is re-ingested in full (1 original + 4 appended) under the new path")
	assertFileEvents(t, events, appLogRenamed)

	// The re-ingested file is still below threshold, so it is tracked as a
	// growing entry under the new path.
	assertGrowingRegistryEntry(t, tempDir, appLogRenamed)
}

// TestFilestreamEnhancedFingerprint_Gzip covers Enhanced Fingerprint on
// GZIP-compressed files. The fingerprint is computed on DECOMPRESSED content,
// so the threshold check is against the decompressed size, not the on-disk
// gzip-file size.
//
// Two files exist from the start:
//   - smallGz: decompressed ~250 bytes → below threshold → growing tracking
//     (bounded-hash registry key with a non-zero meta.fingerprint_len).
//   - largeGz: decompressed ~1500 bytes → above threshold → SHA-256 tracking.
//
// Both must be ingested completely under growing mode.
func TestFilestreamEnhancedFingerprint_Gzip(t *testing.T) {
	t.Parallel()
	filebeat, tempDir, logDir := newFingerprintFilebeat(t)

	smallGzFile := filepath.Join(logDir, "small.log.gz")
	largeGzFile := filepath.Join(logDir, "large.log.gz")

	smallContent := generateLines("small gzip", 5)  // ~250 bytes decompressed
	largeContent := generateLines("large gzip", 30) // ~1500 bytes decompressed

	smallGzBytes := gziptest.Compress(t, []byte(smallContent), gziptest.CorruptNone)
	largeGzBytes := gziptest.Compress(t, []byte(largeContent), gziptest.CorruptNone)

	require.NoError(t, os.WriteFile(smallGzFile, smallGzBytes, 0o644),
		"failed to write small gzip file")
	require.NoError(t, os.WriteFile(largeGzFile, largeGzBytes, 0o644),
		"failed to write large gzip file")

	filebeat.WriteConfigFile(enhancedFingerprintCfg(logDir, "1s", tempDir))
	filebeat.Start()

	filebeat.WaitLogsContains("Input 'filestream' starting",
		10*time.Second, "filestream did not start")

	// Expect 35 events: 5 (small, raw-hex growing) + 30 (large, SHA-256).
	filebeat.WaitPublishedEvents(15*time.Second, 35)
	// GZIP files are finite, so the harvester closes on EOF rather than
	// backing off.
	filebeat.WaitLogsContainsAnyOrder(
		[]string{
			fmt.Sprintf("EOF has been reached. Closing. Path='%s'", smallGzFile),
			fmt.Sprintf("EOF has been reached. Closing. Path='%s'", largeGzFile)},
		10*time.Second, "small gzip file was not fully read")

	events := readOutputEvents(t, tempDir)
	assert.Len(t, events, 35, "expected exactly 35 events (5 small + 30 large)")
	assert.Len(t, messagesForFile(events, smallGzFile), 5,
		"small gzip file should have 5 events")
	assert.Len(t, messagesForFile(events, largeGzFile), 30,
		"large gzip file should have 30 events")

	filebeat.Stop()

	// Registry: small gzip below threshold → raw-hex (growing) entry;
	// large gzip above threshold → SHA-256 entry.
	assertGrowingRegistryEntry(t, tempDir, smallGzFile)
	assertSingleSHA256RegistryEntry(t, tempDir, largeGzFile)
}

// TestFilestreamEnhancedFingerprint_TruncationAboveToBelowThreshold covers
// the edge case where a file already at threshold (SHA-256 key) is truncated
// to below threshold with different content. The new content has a different
// fingerprint identity (raw-hex of the new bytes), so the file is treated as
// a brand-new file — the new content is ingested from offset 0.
//
// The existing TestFilestreamGrowingFingerprintTruncation covers the
// below→below case (raw-hex → different raw-hex). This test covers the
// above→below case (SHA-256 → raw-hex), specific to Enhanced Fingerprint.
func TestFilestreamEnhancedFingerprint_TruncationAboveToBelowThreshold(t *testing.T) {
	t.Parallel()
	filebeat, tempDir, logDir := newFingerprintFilebeat(t)

	logFile := filepath.Join(logDir, "app.log")

	filebeat.WriteConfigFile(enhancedFingerprintCfg(logDir, "1s", tempDir))
	filebeat.Start()

	filebeat.WaitLogsContains("Input 'filestream' starting",
		10*time.Second, "filestream did not start")

	// Phase 1: write large file (~1500 bytes), above threshold → SHA-256 key.
	writeTruncatingFile(t, logFile, generateLines("large content", 30))
	filebeat.WaitPublishedEvents(15*time.Second, 30)
	filebeat.WaitLogsContains(
		fmt.Sprintf("End of file reached: %s; Backoff now.", logFile),
		10*time.Second, "phase 1 did not reach EOF")

	// Phase 2: truncate with smaller, DIFFERENT content (~400 bytes), below
	// threshold. The new fingerprint identity is raw-hex(new bytes), distinct
	// from the SHA-256 key from phase 1. SameFile sees no prefix relationship
	// → the fileWatcher treats this as a removal + creation (not OpTruncate,
	// which only fires when SameFile matched) → the new content is ingested
	// from offset 0 under a fresh registry entry.
	writeTruncatingFile(t, logFile, generateLines("completely different", 8))

	// Expect 30 + 8 = 38 events.
	filebeat.WaitPublishedEvents(15*time.Second, 38)

	events := readOutputEvents(t, tempDir)
	assert.Len(t, events, 38,
		"expected 38 events (30 large + 8 truncated); fewer means truncated content not ingested, more means re-ingestion")

	// All events under the same path (single harvester rotated through
	// truncation).
	msgs := messagesForFile(events, logFile)
	assert.Len(t, msgs, 38, "all events should be attributed to logFile")

	// First 30 are the large content; next 8 are the post-truncation content.
	// Spot-check that the new content's first message is present (proves the
	// new content was ingested from offset 0).
	var sawTruncatedFirst bool
	for _, m := range msgs {
		if strings.HasPrefix(m, "completely different 1") {
			sawTruncatedFirst = true
			break
		}
	}
	assert.True(t, sawTruncatedFirst,
		"expected the first line of the post-truncation content to appear in events")

	filebeat.Stop()

	// Registry shape after above→below truncation:
	// - An active raw-hex entry exists for the file under its new (post-
	//   truncation) identity.
	// - The pre-truncation SHA-256 entry may still be present too
	activeSHA256, activeRawHex := activeFingerprintEntries(readFingerprintRegistry(t, tempDir), logFile)
	assert.NotEmpty(t, activeRawHex,
		"expected at least one active raw-hex registry entry for %q (the post-truncation identity); got SHA-256=%d raw-hex=%d",
		logFile, len(activeSHA256), len(activeRawHex))
	assert.LessOrEqual(t, len(activeSHA256), 1,
		"expected at most one stale SHA-256 entry (pre-truncation orphan), got %d", len(activeSHA256))
}

// fingerprintRegistryEntry holds the parts of a fingerprint registry entry
// the Enhanced Fingerprint tests care about.
type fingerprintRegistryEntry struct {
	key         string // full registry key: filestream::<inputID>::fingerprint::<keypart>
	fingerprint string // the key part after ::fingerprint:: (a SHA-256 for final, a bounded hash for growing)
	source      string // Meta.Source
	removed     bool   // true if the latest op for this key was "remove"
	// growing is true while the entry is in the growing phase. With the
	// bounded-key optimization the key part is always a 64-char hash (so its
	// length no longer distinguishes growing from final); the growing
	// fingerprint's byte length lives in the value (Meta.FingerprintLen) and
	// a non-zero value is the marker of a still-growing entry.
	growing bool
}

// readFingerprintRegistry returns the latest state of every fingerprint
// entry in the filestream memlog, keyed by the registry key. "Latest state"
// folds successive set/remove ops into the final observation: a key whose
// last op was "remove" appears with removed=true; a key whose last op was
// "set" appears with removed=false and its source path populated.
func readFingerprintRegistry(t *testing.T, tempDir string) map[string]fingerprintRegistryEntry {
	t.Helper()
	registryFile := filepath.Join(tempDir, "data", "registry", "filebeat", "log.json")
	entries, _ := readFilestreamRegistryLog(t, registryFile)

	state := map[string]fingerprintRegistryEntry{}
	for _, e := range entries {
		idx := strings.LastIndex(e.Key, "::fingerprint::")
		if idx < 0 {
			continue
		}
		fp := e.Key[idx+len("::fingerprint::"):]
		prev := state[e.Key]
		// A remove op may carry no Meta payload; preserve the last-known
		// source from earlier set ops so callers can match by path.
		source := e.Filename
		if source == "" {
			source = prev.source
		}
		state[e.Key] = fingerprintRegistryEntry{
			key:         e.Key,
			fingerprint: fp,
			source:      source,
			removed:     e.Op == "remove",
			growing:     e.FingerprintLen > 0,
		}
	}
	return state
}

// activeFingerprintEntries returns the non-removed registry entries whose
// source is filePath, split into final (SHA-256) and growing (raw-hex)
// entries. It is the shared filter behind the registry assertions below.
func activeFingerprintEntries(state map[string]fingerprintRegistryEntry, filePath string) (final, growing []fingerprintRegistryEntry) {
	for _, e := range state {
		if e.source != filePath || e.removed {
			continue
		}
		if e.growing {
			growing = append(growing, e)
		} else {
			final = append(final, e)
		}
	}
	return final, growing
}

// assertFingerprintMigratedToSHA256 asserts that the given file has exactly
// one active fingerprint entry whose key uses the 64-char SHA-256 form, and
// that the migration left at least one removed raw-hex (non-SHA-256) entry
// in the log. The removed entry's source path may or may not equal filePath
// — across a rename the old entry was stored under the OLD path, so we
// do not path-filter the removed-entries check.
func assertFingerprintMigratedToSHA256(t *testing.T, homeDir, filePath string) {
	t.Helper()
	state := readFingerprintRegistry(t, homeDir)

	var (
		activeFinal   []fingerprintRegistryEntry
		activeGrowing []string
		removedKeys   []string
	)
	for _, e := range state {
		// "Migration removed the old key" check is source-path-agnostic: the
		// migration removes the old entry under whatever Source it was stored
		// with, which for a rename is the OLD path, not filePath.
		if e.removed {
			removedKeys = append(removedKeys, e.key)
			continue
		}
		// "Active entry for this file" check is path-filtered.
		if e.source != filePath {
			continue
		}
		// With the bounded-key optimization the growing key is also 64 chars,
		// so we classify by the value-side marker (Meta.FingerprintLen), not
		// key length.
		if e.growing {
			activeGrowing = append(activeGrowing, e.key)
		} else {
			activeFinal = append(activeFinal, e)
		}
	}

	require.Len(t, activeFinal, 1,
		"expected exactly one active final (SHA-256) registry entry for %q; got final=%d growing=%v",
		filePath, len(activeFinal), activeGrowing)
	assert.Empty(t, activeGrowing,
		"expected no active growing entries for %q after migration", filePath)
	assert.Len(t, activeFinal[0].fingerprint, 64,
		"expected the active entry to be keyed by a 64-char SHA-256; got %q", activeFinal[0].fingerprint)
	// Proof of migration: the old growing key (distinct from the final SHA-256
	// key) was removed from the registry.
	var removedOldKeys []string
	for _, k := range removedKeys {
		if k != activeFinal[0].key {
			removedOldKeys = append(removedOldKeys, k)
		}
	}
	assert.NotEmpty(t, removedOldKeys,
		"expected at least one removed key distinct from the final SHA-256 key (proof of migration); removed=%v",
		removedKeys)
}

// assertSingleSHA256RegistryEntry asserts that the file has exactly one
// active fingerprint entry with a 64-char SHA-256 key and no extra entries
// have been created.
func assertSingleSHA256RegistryEntry(t *testing.T, tempDir, filePath string) {
	t.Helper()
	final, growing := activeFingerprintEntries(readFingerprintRegistry(t, tempDir), filePath)

	require.Len(t, final, 1,
		"expected exactly one active final (SHA-256) registry entry for %q; got final=%d growing=%d",
		filePath, len(final), len(growing))
	assert.Empty(t, growing, "expected no active growing entries for %q", filePath)
	assert.Len(t, final[0].fingerprint, 64,
		"expected the active entry to be keyed by a SHA-256 (64-char) fingerprint; got len=%d",
		len(final[0].fingerprint))
}

// assertGrowingRegistryEntry asserts that the file has exactly one active
// fingerprint entry still in the growing phase. Growing is identified by a
// non-zero Meta.FingerprintLen (see fingerprintRegistryEntry.growing), not by
// key length: with the bounded-key optimization a growing key is also 64
// chars. Used to verify that a small file below the configured threshold is
// tracked under growing mode (and would migrate to SHA-256 if it ever grew
// past offset+length).
func assertGrowingRegistryEntry(t *testing.T, tempDir, filePath string) {
	t.Helper()
	final, growing := activeFingerprintEntries(readFingerprintRegistry(t, tempDir), filePath)

	require.Len(t, growing, 1,
		"expected exactly one active growing registry entry for %q; got growing=%d final=%d",
		filePath, len(growing), len(final))
	assert.Empty(t, final,
		"expected no active final entries for %q; got a final entry — file is treated as final, not growing",
		filePath)
}

// assertLogsDoNotContain fails if s is found anywhere in the Filebeat logs.
func assertLogsDoNotContain(t *testing.T, tempDir, s string) {
	t.Helper()
	glob := filepath.Join(tempDir, fmt.Sprintf("filebeat-%d*.ndjson", time.Now().Year()))
	files, err := filepath.Glob(glob)
	require.NoError(t, err, "failed to glob log files")
	require.NotEmpty(t, files, "no filebeat log file found")
	for _, f := range files {
		data, err := os.ReadFile(f)
		require.NoErrorf(t, err, "failed to read log file %q", f)
		if strings.Contains(string(data), s) {
			t.Errorf("log file %q must not contain %q", f, s)
		}
	}
}
