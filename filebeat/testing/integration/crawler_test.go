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

// Tests for file-discovery and crawling behaviour

package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	libbeatintegration "github.com/elastic/beats/v7/libbeat/testing/integration"
)

var crawlerReportOptions = libbeatintegration.ReportOptions{
	PrintLinesOnFail:  100,
	PrintConfigOnFail: true,
}

// TestCrawlerMultipleAppends verifies that Filebeat keeps picking up new lines
// as they are appended to a file in several rounds.
// Corresponds to test_crawler.py::test_multiple_appends.
func TestCrawlerMultipleAppends(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	EnsureCompiled(ctx, t)

	dir := t.TempDir()
	logFile := filepath.Join(dir, "test.log")
	WriteFile(t, logFile, "hello world\n")

	const finalMarker = "crawl-multi-append-done"
	config := FilestreamInputConfig("multi-append-test", filepath.Join(dir, "*.log"), FilestreamOptions{})
	test := NewTest(t, TestOptions{Config: config})

	// Count every event that contains "hello world" (the appended content).
	var count atomic.Int64
	test.CountOutput(&count, "hello world")

	// The test ends when the beat sees the final marker line.
	test.ExpectOutput(finalMarker)
	test.WithReportOptions(crawlerReportOptions)
	test.ExpectStart()
	test.Start(ctx)

	// Wait for the initial line to be ingested.
	WaitUntil(t, func() bool { return count.Load() >= 1 }, 15*time.Second, 100*time.Millisecond)

	// Append three rounds of lines and wait for each round to be ingested.
	totalExpected := int64(1) // the initial "hello world" line
	for n := range 3 {
		linesInRound := 20 + n
		for i := range linesInRound {
			AppendToFile(t, logFile, fmt.Sprintf("hello world %d %d\n", i, n))
			totalExpected++
		}
		WaitUntil(t, func() bool { return count.Load() >= totalExpected },
			60*time.Second, 200*time.Millisecond)
		t.Logf("round %d complete, events so far: %d", n, count.Load())
	}

	// All expected events are present; verify the exact count.
	require.EqualValues(t, 64, count.Load(),
		"expected 1 initial + 20 + 21 + 22 = 64 events")

	// Write the final marker to let the beat terminate cleanly.
	AppendToFile(t, logFile, finalMarker+"\n")
	test.Wait()
}

// TestCrawlerNewLineOnOpenFile verifies that Filebeat follows writes to an
// already-open file.
// Corresponds to test_crawler.py::test_new_line_on_open_file.
func TestCrawlerNewLineOnOpenFile(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	dir := t.TempDir()
	logFile := filepath.Join(dir, "test.log")

	// Open the file and write the first line while keeping the handle open.
	f, err := os.Create(logFile)
	require.NoError(t, err)

	_, err = f.WriteString("hello world\n")
	require.NoError(t, err)
	require.NoError(t, f.Sync())

	// Count "hello world" occurrences so we can wait for the first line.
	config := FilestreamInputConfig("open-file-test", filepath.Join(dir, "*.log"), FilestreamOptions{})
	test := NewTest(t, TestOptions{Config: config})

	var count atomic.Int64
	test.CountOutput(&count, "hello world")
	test.ExpectOutput("hello world 1")
	test.ExpectOutput("hello world 2")
	test.WithReportOptions(crawlerReportOptions)
	test.ExpectStart()
	test.Start(ctx)

	// Wait for the initial line to be ingested before writing more.
	WaitUntil(t, func() bool { return count.Load() >= 1 }, 15*time.Second, 100*time.Millisecond)

	// Append two more lines while the file handle is still open.
	_, err = f.WriteString("hello world 1\n")
	require.NoError(t, err)
	_, err = f.WriteString("hello world 2\n")
	require.NoError(t, err)
	require.NoError(t, f.Sync())
	f.Close()

	// The beat is killed once "hello world 1" and "hello world 2" are seen.
	test.Wait()
}

// TestCrawlerUTF8 verifies that UTF-8 characters are read correctly.
func TestCrawlerUTF8(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	dir := t.TempDir()
	logFile := filepath.Join(dir, "test.log")
	WriteFile(t, logFile, "ニコラスRuflin\n")

	config := FilestreamInputConfig("utf8-test", filepath.Join(dir, "*.log"), FilestreamOptions{})
	test := NewTest(t, TestOptions{Config: config})
	test.ExpectOutput("ニコラスRuflin")
	test.WithReportOptions(crawlerReportOptions)
	test.
		ExpectEOF(logFile).
		ExpectStart().
		Start(ctx).
		Wait()
}

// TestCrawlerFileDisappearAppear verifies that when a file is removed and a new
// file appears at the same path, Filebeat reads the new file from the start.
// Corresponds to test_crawler.py::test_file_disappear_appear.
func TestCrawlerFileDisappearAppear(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	dir := t.TempDir()
	logFile := filepath.Join(dir, "test.log")

	// Write first file's content.
	WriteFile(t, logFile, "disappearing file\n")

	config := FilestreamInputConfig("disappear-appear-test", filepath.Join(dir, "*.log"), FilestreamOptions{
		ScanInterval: "100ms",
		CloseRemoved: true,
	})
	test := NewTest(t, TestOptions{Config: config})

	// Count first-file events so we know when they have all been ingested.
	var firstCount atomic.Int64
	test.CountOutput(&firstCount, "disappearing file")

	// The test terminates when the new file's content is seen.
	test.ExpectOutput("new file content")
	test.WithReportOptions(crawlerReportOptions)
	test.ExpectStart()
	test.Start(ctx)

	// Wait for the original file to be fully ingested.
	WaitUntil(t, func() bool { return firstCount.Load() >= 1 }, 15*time.Second, 100*time.Millisecond)

	// Delete the original file; Filebeat should notice the removal and close it.
	require.NoError(t, os.Remove(logFile))

	// Wait for filebeat to log that it closed the removed file before creating a new one
	// at the same path.  A 500ms wait is a pragmatic minimum; on slow CI hosts
	// this may occasionally be too short, but close_removed fires on the next
	// scan tick (100ms), so 500ms provides 5 scan opportunities.
	time.Sleep(500 * time.Millisecond)

	// Create a new file at the same path with different content.
	WriteFile(t, logFile, "new file content\n")

	test.Wait()
}
