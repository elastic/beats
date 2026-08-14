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

// Tests for input-level functionality, ported from test_input.py.
// Each test uses a filestream input with output.console so that
// ExpectOutput / ExpectJSONFields assertions work against captured stdout.

package integration

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	libbeatintegration "github.com/elastic/beats/v7/libbeat/testing/integration"
)

var inputReportOptions = libbeatintegration.ReportOptions{
	PrintLinesOnFail:  100,
	PrintConfigOnFail: true,
}

// TestInputIgnoreOlderFiles verifies that files not modified within ignore_older
// are skipped.
func TestInputIgnoreOlderFiles(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	dir := t.TempDir()
	logFile := filepath.Join(dir, "test.log")

	// Write content, then back-date the file so it exceeds ignore_older.
	GenerateLogFile(t, logFile, 5, NewPlainTextGenerator("hello world"))
	pastTime := time.Now().Add(-5 * time.Second)
	require.NoError(t, os.Chtimes(logFile, pastTime, pastTime))

	config := FilestreamInputConfig("ignore-older-test", logFile, FilestreamOptions{
		IgnoreOlder: "2s",
	})
	test := NewTest(t, TestOptions{Config: config})
	test.
		WithReportOptions(inputReportOptions).
		ExpectOutput("Ignore file because ignore_older reached").
		ExpectStart().
		Start(ctx).
		Wait()
}

// TestInputNotIgnoreOldFiles verifies that files modified within ignore_older
// are read normally.  Corresponds to test_input.py::test_not_ignore_old_files.
func TestInputNotIgnoreOldFiles(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	dir := t.TempDir()
	logFile := filepath.Join(dir, "test.log")
	GenerateLogFile(t, logFile, 5, NewPlainTextGenerator("hello world"))

	config := FilestreamInputConfig("not-ignore-test", logFile, FilestreamOptions{
		IgnoreOlder: "15s",
	})
	test := NewTest(t, TestOptions{Config: config})

	var count atomic.Int64
	test.CountOutput(&count, "hello world")

	test.
		ExpectEOF(logFile).
		WithReportOptions(inputReportOptions).
		ExpectStart().
		Start(ctx).
		Wait()

	require.EqualValues(t, 5, count.Load(), "expected 5 events from the log file")
}

// TestInputExcludeFiles verifies that files matching exclude_files patterns
// are skipped.  Corresponds to test_input.py::test_exclude_files.
func TestInputExcludeFiles(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	dir := t.TempDir()
	gzFile := filepath.Join(dir, "test.gz")
	logFile := filepath.Join(dir, "test.log")

	GenerateLogFile(t, gzFile, 1, NewPlainTextGenerator("line in gz file"))
	GenerateLogFile(t, logFile, 1, NewPlainTextGenerator("line in log file"))

	config := FilestreamInputConfig("exclude-files-test", filepath.Join(dir, "*"), FilestreamOptions{
		ExcludeFiles: []string{`\.gz$`},
	})
	test := NewTest(t, TestOptions{Config: config})
	test.ExpectOutput("line in log file")
	test.
		ExpectEOF(logFile).
		WithReportOptions(inputReportOptions).
		ExpectStart().
		Start(ctx).
		Wait()
}

// TestInputFilesAddedLate verifies that files created after Filebeat starts
// are still discovered and read.  Corresponds to test_input.py::test_files_added_late.
func TestInputFilesAddedLate(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	dir := t.TempDir()
	logFile := filepath.Join(dir, "test.log")

	config := FilestreamInputConfig("files-added-late-test", filepath.Join(dir, "*.log"), FilestreamOptions{
		ScanInterval: "100ms",
	})
	test := NewTest(t, TestOptions{Config: config})

	// Count scanner ticks to detect when at least one scan has run.
	var scanCount atomic.Int64
	test.CountOutput(&scanCount, "Start next scan")

	// Register expectations before starting.
	test.ExpectOutput("Hello World Late")
	test.WithReportOptions(inputReportOptions)
	test.ExpectStart()
	test.Start(ctx)

	// Wait for at least one scan to complete, then create the file.
	require.Eventually(t,func() bool { return scanCount.Load() >= 1 }, 10*time.Second, 50*time.Millisecond)
	GenerateLogFile(t, logFile, 1, NewPlainTextGenerator("Hello World Late"))

	test.Wait()
}

// TestInputCloseInactive verifies that close.on_state_change.inactive closes
// idle files and they are re-opened when new data is appended.
// Corresponds to test_input.py::test_close_inactive.
func TestInputCloseInactive(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	dir := t.TempDir()
	logFile := filepath.Join(dir, "test.log")

	config := FilestreamInputConfig("close-inactive-test", filepath.Join(dir, "*.log"), FilestreamOptions{
		ScanInterval:  "100ms",
		IgnoreOlder:   "1h",
		CloseInactive: "1s",
	})
	test := NewTest(t, TestOptions{Config: config})

	// Count Line 1 appearances so we can wait until it is ingested.
	var line1Count atomic.Int64
	test.CountOutput(&line1Count, "Line 1")
	// Count close_inactive log lines so we know when the file has been closed.
	var inactiveCount atomic.Int64
	test.CountOutput(&inactiveCount, "File is inactive")
	// Final expectation: Line 2 must also appear.
	test.ExpectOutput("Line 2")
	test.WithReportOptions(inputReportOptions)
	test.ExpectStart()
	test.Start(ctx)

	// Write the first line and wait until the beat ingests it.
	GenerateLogFile(t, logFile, 1, NewPlainTextGenerator("Line 1"))
	require.Eventually(t,func() bool { return line1Count.Load() >= 1 }, 15*time.Second, 100*time.Millisecond)

	// Wait for close_inactive to close the file.
	require.Eventually(t,func() bool { return inactiveCount.Load() >= 1 }, 10*time.Second, 100*time.Millisecond)

	// Write the second line; filestream should reopen the file.
	AppendLogFile(t, logFile, 1, NewPlainTextGenerator("Line 2"))

	test.Wait()
}

// TestInputSkipSymlinks verifies that symlinks are not followed unless
// symlinks option is enabled.  Corresponds to test_input.py::test_skip_symlinks.
func TestInputSkipSymlinks(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	dir := t.TempDir()
	realFile := filepath.Join(dir, "real-2016.log")
	symlinkFile := filepath.Join(dir, "symlink.log")

	GenerateLogFile(t, realFile, 1, NewPlainTextGenerator("Hello world"))
	CreateSymlink(t, realFile, symlinkFile)

	// Without symlinks enabled, only the real file should be read.
	// The filestream scanner logs "is a symlink and they're disabled" (fscanner.go:809)
	// for each symlink it skips.
	config := FilestreamInputConfig("skip-symlinks-test", filepath.Join(dir, "*.log"), FilestreamOptions{
		Symlinks: false,
	})
	test := NewTest(t, TestOptions{Config: config})
	test.ExpectOutput("is a symlink and they're disabled")
	test.
		ExpectEOF(realFile).
		WithReportOptions(inputReportOptions).
		ExpectStart().
		Start(ctx).
		Wait()
}

// TestInputRotatingCloseInactiveLowWriteRate verifies that log rotation with
// close_inactive works at a low write rate.
// Corresponds to test_input.py::test_rotating_close_inactive_low_write_rate.
func TestInputRotatingCloseInactiveLowWriteRate(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	dir := t.TempDir()
	logFile := filepath.Join(dir, "test.log")
	rotatedFile := filepath.Join(dir, "test.log.1")

	config := FilestreamInputConfig("rotate-inactive-test", filepath.Join(dir, "test.log"), FilestreamOptions{
		ScanInterval:  "100ms",
		IgnoreOlder:   "10s",
		CloseInactive: "1s",
	})
	test := NewTest(t, TestOptions{Config: config})

	var line1Count atomic.Int64
	test.CountOutput(&line1Count, "Line 1")
	// Count close_inactive log lines so we know when the rotated file has been closed.
	var inactiveCount atomic.Int64
	test.CountOutput(&inactiveCount, "File is inactive")
	test.ExpectOutput("Line 2")
	test.WithReportOptions(inputReportOptions)
	test.ExpectStart()
	test.Start(ctx)

	GenerateLogFile(t, logFile, 1, NewPlainTextGenerator("Line 1"))

	// Wait for Line 1 to be ingested, then rotate.
	require.Eventually(t,func() bool { return line1Count.Load() >= 1 }, 15*time.Second, 100*time.Millisecond)
	RotateFile(t, logFile, rotatedFile)

	// Wait for close_inactive to close the rotated file.
	require.Eventually(t,func() bool { return inactiveCount.Load() >= 1 }, 10*time.Second, 100*time.Millisecond)

	// Write the second line into the new file at the original path.
	GenerateLogFile(t, logFile, 1, NewPlainTextGenerator("Line 2"))

	test.Wait()
}
