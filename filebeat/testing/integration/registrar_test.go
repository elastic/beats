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

// Tests for the filestream registry (state persistence)

package integration

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestRegistrarFileContent verifies that the registry is created with correct
// initial content for a single ingested file.
func TestRegistrarFileContent(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	dir := t.TempDir()
	logFile := filepath.Join(dir, "test.log")
	GenerateLogFile(t, logFile, 5, NewPlainTextGenerator("hello world"))

	stat, err := os.Stat(logFile)
	require.NoError(t, err)
	expectedOffset := int(stat.Size())

	config := FilestreamInputConfig("registrar-content-test", logFile, FilestreamOptions{})
	test := NewTest(t, TestOptions{Config: config})
	test.
		ExpectEOF(logFile).
		ExpectStart().
		Start(ctx).
		Wait()

	entries := ReadRegistry(t, test.GetTempDir())
	require.Len(t, entries, 1)
	require.Equal(t, logFile, entries[0].Filename)
	require.Equal(t, expectedOffset, entries[0].Offset)
}

// TestRegistrarFiles verifies that multiple files are all tracked in the
// registry.
func TestRegistrarFiles(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	dir := t.TempDir()
	logFile1 := filepath.Join(dir, "test1.log")
	logFile2 := filepath.Join(dir, "test2.log")
	GenerateLogFile(t, logFile1, 5, NewPlainTextGenerator("hello world"))
	GenerateLogFile(t, logFile2, 5, NewPlainTextGenerator("goodbye world"))

	config := FilestreamInputConfig("registrar-files-test", filepath.Join(dir, "*.log"), FilestreamOptions{})
	test := NewTest(t, TestOptions{Config: config})
	test.
		ExpectEOF(logFile1).
		ExpectEOF(logFile2).
		ExpectStart().
		Start(ctx).
		Wait()

	entries := ReadRegistry(t, test.GetTempDir())
	require.Len(t, entries, 2)
}

// TestRegistrarCustomPath verifies that a custom filebeat.registry.path is
// honoured.
func TestRegistrarCustomPath(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	dir := t.TempDir()
	logFile := filepath.Join(dir, "test.log")
	GenerateLogFile(t, logFile, 1, NewPlainTextGenerator("hello world"))

	registryDir := filepath.Join(dir, "a", "b", "c", "registry")
	config := FilestreamInputConfig("registrar-custom-path-test", logFile, FilestreamOptions{
		RegistryPath: registryDir,
	})
	test := NewTest(t, TestOptions{Config: config})
	test.
		ExpectEOF(logFile).
		ExpectStart().
		Start(ctx).
		Wait()

	registryLog := filepath.Join(registryDir, "filebeat", "log.json")
	_, err := os.Stat(registryLog)
	require.NoError(t, err, "registry log.json should exist at custom path %s", registryLog)
}

// TestRegistrarDataPath verifies that the registry is written under a custom
// --path.data directory.
func TestRegistrarDataPath(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	dir := t.TempDir()
	logFile := filepath.Join(dir, "test.log")
	GenerateLogFile(t, logFile, 1, NewPlainTextGenerator("test message"))

	dataDir := filepath.Join(dir, "datapath")
	config := FilestreamInputConfig("registrar-data-path-test", logFile, FilestreamOptions{})
	test := NewTest(t, TestOptions{
		Config: config,
		Args:   []string{"--path.data", dataDir},
	})
	test.
		ExpectEOF(logFile).
		ExpectStart().
		Start(ctx).
		Wait()

	registryLog := filepath.Join(dataDir, "registry", "filebeat", "log.json")
	_, err := os.Stat(registryLog)
	require.NoError(t, err, "registry log.json should exist under --path.data at %s", registryLog)
}

// TestRegistrarRotatingFile verifies that the registry is updated correctly
// after a log file is rotated.
func TestRegistrarRotatingFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipped on Windows: https://github.com/elastic/beats/issues/26378")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	dir := t.TempDir()
	logFile := filepath.Join(dir, "test.log")
	rotatedFile := filepath.Join(dir, "test.1.log")

	config := FilestreamInputConfig("rotating-file-test", filepath.Join(dir, "*.log"), FilestreamOptions{
		ScanInterval:  "100ms",
		CloseInactive: "1s",
	})
	test := NewTest(t, TestOptions{Config: config})

	var line1Count atomic.Int64
	test.CountOutput(&line1Count, "offset 9")

	var offsetTenCount atomic.Int64
	test.CountOutput(&offsetTenCount, "offset 10")

	var inactiveCount atomic.Int64
	test.CountOutput(&inactiveCount, "File is inactive")

	var renamedCount atomic.Int64
	test.CountOutput(&renamedCount, "has been renamed to")

	// Keep the beat running until all conditions are met; on fast systems "offset 10"
	// can be ingested before the 1s inactive timer fires.
	test.ExpectStop(exitSignalKilled)
	test.ExpectStart()
	test.Start(ctx)

	GenerateLogFile(t, logFile, 1, NewPlainTextGenerator("offset 9"))

	require.Eventually(t, func() bool { return line1Count.Load() >= 1 }, 15*time.Second, 100*time.Millisecond)

	// Capture size before rotation so we can assert the rotated file's offset.
	stat, err := os.Stat(logFile)
	require.NoError(t, err)
	rotatedExpectedOffset := int(stat.Size())

	RotateFile(t, logFile, rotatedFile)
	GenerateLogFile(t, logFile, 1, NewPlainTextGenerator("offset 10"))

	stat, err = os.Stat(logFile)
	require.NoError(t, err)
	origExpectedOffset := int(stat.Size())
	require.Eventually(t, func() bool { return offsetTenCount.Load() >= 1 }, 30*time.Second, 100*time.Millisecond)
	require.Eventually(t, func() bool { return inactiveCount.Load() >= 1 }, 30*time.Second, 100*time.Millisecond)
	require.Eventually(t, func() bool { return renamedCount.Load() >= 1 }, 30*time.Second, 100*time.Millisecond)

	cancel()
	test.Wait()

	entries := ReadRegistry(t, test.GetTempDir())
	require.Len(t, entries, 2, "both original and rotated file should be in the registry")

	orig, ok := FindRegistryEntryByFilename(entries, logFile)
	require.True(t, ok, "entry for %s not found in registry", logFile)
	require.Equal(t, origExpectedOffset, orig.Offset)

	rotated, ok := FindRegistryEntryByFilename(entries, rotatedFile)
	require.True(t, ok, "entry for %s not found in registry", rotatedFile)
	require.Equal(t, rotatedExpectedOffset, rotated.Offset)
}

// TestRegistrarStateAfterRotation verifies that offsets are written correctly
// after rotating pre-existing files.
func TestRegistrarStateAfterRotation(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	dir := t.TempDir()
	// Use bare filenames so the glob pattern "input*" matches without a log extension.
	logFile1 := filepath.Join(dir, "input")
	logFile2 := filepath.Join(dir, "input.1")
	logFile3 := filepath.Join(dir, "input.2")

	GenerateLogFile(t, logFile1, 1, NewPlainTextGenerator("entry10"))
	GenerateLogFile(t, logFile2, 1, NewPlainTextGenerator("entry0"))

	config := FilestreamInputConfig("state-after-rotation-test", filepath.Join(dir, "input*"), FilestreamOptions{
		ScanInterval:  "1s",
		IgnoreOlder:   "2m",
		CloseInactive: "1s",
	})
	test := NewTest(t, TestOptions{Config: config})

	var eventCount atomic.Int64
	test.CountOutput(&eventCount, "entry")

	var renamedCount atomic.Int64
	test.CountOutput(&renamedCount, "has been renamed to")

	// entry200 in new logFile1 is the terminal output.
	test.ExpectOutput("entry200")
	test.ExpectStart()
	test.Start(ctx)

	// Wait for both initial files to be ingested.
	require.Eventually(t, func() bool { return eventCount.Load() >= 2 }, 15*time.Second, 100*time.Millisecond)

	// Capture logFile1's size before rotation: this becomes logFile2's expected offset.
	stat, err := os.Stat(logFile1)
	require.NoError(t, err)
	entry2ExpectedOffset := int(stat.Size())

	// Rotate: input.1 → input.2, input → input.1; then write new input and remove input.2.
	RotateFile(t, logFile2, logFile3)
	RotateFile(t, logFile1, logFile2)
	require.NoError(t, os.Remove(logFile3))

	// Wait for rename detection before writing new content (ensures the
	// registry rename update precedes the terminal output that stops the beat).
	require.Eventually(t, func() bool { return renamedCount.Load() >= 1 }, 15*time.Second, 100*time.Millisecond)

	GenerateLogFile(t, logFile1, 1, NewPlainTextGenerator("entry200"))

	stat, err = os.Stat(logFile1)
	require.NoError(t, err)
	entry1ExpectedOffset := int(stat.Size())

	test.Wait()

	entries := ReadRegistry(t, test.GetTempDir())

	entry1, ok := FindRegistryEntryByFilename(entries, logFile1)
	require.True(t, ok, "entry for %s not found in registry", logFile1)
	require.Equal(t, entry1ExpectedOffset, entry1.Offset)

	entry2, ok := FindRegistryEntryByFilename(entries, logFile2)
	require.True(t, ok, "entry for %s not found in registry", logFile2)
	require.Equal(t, entry2ExpectedOffset, entry2.Offset)
}

// TestRegistrarRestartContinue verifies that Filebeat resumes from the
// registry offset on restart and does not re-ingest already-read lines.
func TestRegistrarRestartContinue(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipped on Windows")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	dir := t.TempDir()
	logFile := filepath.Join(dir, "input.log")
	GenerateLogFile(t, logFile, 1, NewPlainTextGenerator("entry1"))

	config := FilestreamInputConfig("restart-continue-test", logFile, FilestreamOptions{
		ScanInterval: "1s",
	})

	// First run: ingest entry1.
	test1 := NewTest(t, TestOptions{Config: config})
	test1.
		ExpectOutput("entry1").
		ExpectStart().
		Start(ctx).
		Wait()

	homeDir := test1.GetTempDir()
	entries := ReadRegistry(t, homeDir)
	require.Len(t, entries, 1)
	require.Equal(t, logFile, entries[0].Filename)

	// Append entry2.
	AppendLogFile(t, logFile, 1, NewPlainTextGenerator("entry2"))

	// Second run reuses the same data directory so the registry is preserved.
	// Pass --path.data so the registry path matches the first run.
	test2 := NewTest(t, TestOptions{
		Config: config,
		Args:   []string{"--path.data", filepath.Join(homeDir, "data")},
	})

	var eventCount atomic.Int64
	test2.CountOutput(&eventCount, `"message":"entry`)

	test2.
		ExpectOutput("entry2").
		ExpectStart().
		Start(ctx).
		Wait()

	require.EqualValues(t, 1, eventCount.Load(), "second run should ingest only entry2, not entry1")

	entries = ReadRegistry(t, homeDir)
	require.Len(t, entries, 1)
}

// TestRegistrarCleanInactive verifies that registry entries are removed after
// the clean_inactive timeout expires.
func TestRegistrarCleanInactive(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipped on Windows: https://github.com/elastic/beats/issues/8102")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	dir := t.TempDir()
	logFile1 := filepath.Join(dir, "input1.log")
	logFile2 := filepath.Join(dir, "input2.log")
	logFile3 := filepath.Join(dir, "input3.log")

	GenerateLogFile(t, logFile1, 1, NewPlainTextGenerator("first file"))
	GenerateLogFile(t, logFile2, 1, NewPlainTextGenerator("second file"))

	// clean_inactive must exceed ignore_older so that the file scanner stops
	// reporting the files before the TTL expires; otherwise re-open cycles from
	// the 100ms scan interval prevent resource.Finished() from returning true.
	// This mirrors the proven configuration in TestFilestreamCleanInactive.
	config := FilestreamInputConfig("clean-inactive-test", filepath.Join(dir, "input*.log"), FilestreamOptions{
		ScanInterval:            "100ms",
		CloseInactive:           "1s",
		IgnoreOlder:             "2s",
		CleanInactive:           "3s",
		RegistryCleanupInterval: "1s",
	})
	test := NewTest(t, TestOptions{Config: config})

	var eventCount atomic.Int64
	test.CountOutput(&eventCount, "file")

	// "entries removed" is logged by the clean_inactive GC loop.
	var cleanCount atomic.Int64
	test.CountOutput(&cleanCount, "entries removed")

	var file3Count atomic.Int64
	test.CountOutput(&file3Count, "third file")

	// The GC log and its registry write are asynchronous with respect to any
	// particular stdout line; keep the beat alive until all conditions are met.
	test.ExpectStop(exitSignalKilled)
	test.ExpectStart()
	test.Start(ctx)

	// Wait for both initial files to be ingested.
	require.Eventually(t, func() bool { return eventCount.Load() >= 2 }, 15*time.Second, 100*time.Millisecond)

	// Wait for clean_inactive GC to fire and remove the entries for files 1 and 2.
	require.Eventually(t, func() bool { return cleanCount.Load() >= 1 }, 30*time.Second, 100*time.Millisecond)

	// Write file3; its event triggers a registry checkpoint flush for file3.
	GenerateLogFile(t, logFile3, 1, NewPlainTextGenerator("third file"))

	// Wait for file3 to be ingested (confirms the registry checkpoint was written).
	require.Eventually(t, func() bool { return file3Count.Load() >= 1 }, 15*time.Second, 100*time.Millisecond)

	cancel()
	test.Wait()

	entries := ReadRegistry(t, test.GetTempDir())
	require.Len(t, entries, 1, "only file3 should remain in the registry after clean_inactive")

	stat, err := os.Stat(logFile3)
	require.NoError(t, err)
	require.Equal(t, int(stat.Size()), entries[0].Offset)
}

// TestRegistrarCleanRemoved verifies that a deleted file's registry entry is
// removed when clean_removed is enabled.
func TestRegistrarCleanRemoved(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("flaky on Windows: https://github.com/elastic/beats/issues/7690")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	dir := t.TempDir()
	logFile1 := filepath.Join(dir, "input1.log")
	logFile2 := filepath.Join(dir, "input2.log")

	GenerateLogFile(t, logFile1, 1, NewPlainTextGenerator("file to be removed"))
	GenerateLogFile(t, logFile2, 1, NewPlainTextGenerator("second file"))

	config := FilestreamInputConfig("clean-removed-test", filepath.Join(dir, "input*.log"), FilestreamOptions{
		ScanInterval: "100ms",
		CloseRemoved: true,
		CleanRemoved: true,
	})
	test := NewTest(t, TestOptions{Config: config})

	var file1Count atomic.Int64
	test.CountOutput(&file1Count, "file to be removed")

	var file2Count atomic.Int64
	test.CountOutput(&file2Count, "second file")

	// "Remove state for file as file removed" is logged by clean_removed.
	var removedCount atomic.Int64
	test.CountOutput(&removedCount, "Remove state for file as file removed")

	// The appended line triggers a registry flush and serves as terminal output.
	test.ExpectOutput("registry is written")
	test.ExpectStart()
	test.Start(ctx)

	// Wait for both files to be ingested.
	require.Eventually(t, func() bool {
		return file1Count.Load() >= 1 && file2Count.Load() >= 1
	}, 15*time.Second, 100*time.Millisecond)

	// Delete file1.
	require.NoError(t, os.Remove(logFile1))

	// Wait for Filebeat to detect the removal.
	require.Eventually(t, func() bool { return removedCount.Load() >= 1 }, 15*time.Second, 100*time.Millisecond)

	// Append to file2 to trigger a registry write and act as the terminal output.
	AppendLogFile(t, logFile2, 1, NewPlainTextGenerator("make sure registry is written"))

	test.Wait()

	entries := ReadRegistry(t, test.GetTempDir())
	require.Len(t, entries, 1, "only file2 should remain in the registry")

	stat, err := os.Stat(logFile2)
	require.NoError(t, err)
	require.Equal(t, int(stat.Size()), entries[0].Offset)
}

// TestRegistrarRestartStateReset verifies registry TTL behaviour across
// restarts: the first run records a positive TTL; after a second run whose
// config covers a different file, filestream leaves the unmanaged entry's TTL
// unchanged (the old log-input sentinel -2 does not apply to filestream).
func TestRegistrarRestartStateReset(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	dir := t.TempDir()
	logFile := filepath.Join(dir, "test.log")
	GenerateLogFile(t, logFile, 1, NewPlainTextGenerator("Hello World"))

	// First run: clean_inactive is set, so the registry entry should record a positive TTL.
	config1 := FilestreamInputConfig("restart-state-reset-1", logFile, FilestreamOptions{
		CleanInactive: "10s",
		IgnoreOlder:   "9s",
	})
	test1 := NewTest(t, TestOptions{Config: config1})
	test1.
		ExpectOutput("Hello World").
		ExpectStart().
		Start(ctx).
		Wait()

	homeDir := test1.GetTempDir()
	entries := ReadRegistry(t, homeDir)
	require.Len(t, entries, 1)
	require.Positive(t, int64(entries[0].TTL), "TTL should be positive when clean_inactive is set")

	// Second run: config points to a different path. Filestream does not reset
	// the TTL for entries that are no longer managed (unlike the old log input
	// which used a -2 sentinel). The entry persists with its original TTL.
	otherFile := filepath.Join(dir, "test2.log")
	config2 := FilestreamInputConfig("restart-state-reset-2", otherFile, FilestreamOptions{
		CleanInactive: "10s",
		IgnoreOlder:   "9s",
	})
	test2 := NewTest(t, TestOptions{
		Config: config2,
		Args:   []string{"--path.data", filepath.Join(homeDir, "data")},
	})
	test2.
		ExpectOutput("Starting input").
		ExpectStart().
		Start(ctx).
		Wait()

	entries = ReadRegistry(t, homeDir)
	require.Len(t, entries, 1)
	require.Equal(t, 10*time.Second, entries[0].TTL,
		"unmanaged entry should retain its TTL from the first run")
}
