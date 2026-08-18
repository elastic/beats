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

// Tests for global processor functionality, ported from test_processors.py.
// Processors are configured at the top-level processors: block.

package integration

import (
	"context"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	libbeatintegration "github.com/elastic/beats/v9/libbeat/testing/integration"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// TestProcessorsDropFields verifies that the drop_fields processor removes the
// specified fields from events.
// Corresponds to test_processors.py::test_dropfields.
//
// To get a reliable negative assertion with the current framework (which mixes
// stdout events and stderr debug logs into one stream), we inject a custom
// field with a unique value via the input config, then use drop_fields to
// remove it.  If the unique value appears in the output after the beat runs,
// the processor failed.
func TestProcessorsDropFields(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	dir := t.TempDir()
	logFile := filepath.Join(dir, "test.log")
	GenerateLogFile(t, logFile, 1, NewPlainTextGenerator("test message"))

	// Add a custom field then immediately drop it; the unique value should
	// not appear in any JSON event output.
	const droppedValue = "drop-fields-sentinel-v1"
	config := `
filebeat.inputs:
  - type: filestream
    id: drop-fields-test
    paths:
      - ` + logFile + `
    prospector.scanner.check_interval: 100ms
    file_identity.native: ~
    prospector.scanner.fingerprint.enabled: false
    fields:
      sentinel: ` + droppedValue + `

output.console:
  enabled: true

processors:
  - drop_fields:
      fields: [fields.sentinel, agent]
`
	test := NewTest(t, TestOptions{Config: config})

	reportOptions := libbeatintegration.ReportOptions{
		PrintLinesOnFail:  50,
		PrintConfigOnFail: true,
	}

	// The event message should still be present after dropping other fields.
	test.ExpectOutput("test message")

	// The injected sentinel value must not appear in any event.
	var sentinelCount atomic.Int64
	test.CountOutput(&sentinelCount, droppedValue)

	test.
		ExpectEOF(logFile).
		WithReportOptions(reportOptions).
		ExpectStart().
		Start(ctx).
		Wait()

	require.EqualValues(t, 0, sentinelCount.Load(),
		"drop_fields should have removed the sentinel field; unique value must not appear in output")
}

// TestProcessorsIncludeFields verifies that the include_fields processor keeps
// only the listed fields in each event.
// Corresponds to test_processors.py::test_include_fields.
func TestProcessorsIncludeFields(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	dir := t.TempDir()
	logFile := filepath.Join(dir, "test.log")
	GenerateLogFile(t, logFile, 1, NewPlainTextGenerator("test message"))

	config := FilestreamInputConfig("include-fields-test", logFile, FilestreamOptions{
		GlobalProcessors: "  - include_fields:\n      fields: [\"@timestamp\", message, log, input]\n",
	})
	test := NewTest(t, TestOptions{Config: config})

	reportOptions := libbeatintegration.ReportOptions{
		PrintLinesOnFail:  50,
		PrintConfigOnFail: true,
	}

	// message and input.type must survive the include_fields filter.
	test.ExpectJSONFields(mapstr.M{
		"message":    "test message test.log:1",
		"input.type": "filestream",
	})

	test.
		ExpectEOF(logFile).
		WithReportOptions(reportOptions).
		ExpectStart().
		Start(ctx).
		Wait()
}

// TestProcessorsDropEvent verifies that the drop_event processor removes events
// matching a condition, passing through all others.
// Corresponds to test_processors.py::test_drop_event.
func TestProcessorsDropEvent(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	dir := t.TempDir()
	logFile1 := filepath.Join(dir, "test1.log")
	logFile2 := filepath.Join(dir, "test2.log")

	GenerateLogFile(t, logFile1, 1, NewPlainTextGenerator("test1 message"))
	GenerateLogFile(t, logFile2, 1, NewPlainTextGenerator("test2 message"))

	// Drop events from test1.log; only test2 events should appear in output.
	config := FilestreamInputConfig("drop-event-test", filepath.Join(dir, "test*.log"), FilestreamOptions{
		GlobalProcessors: "  - drop_event:\n      when:\n        contains:\n          log.file.path: test1\n",
	})
	test := NewTest(t, TestOptions{Config: config})

	reportOptions := libbeatintegration.ReportOptions{
		PrintLinesOnFail:  50,
		PrintConfigOnFail: true,
	}

	// test2 events must appear.
	test.ExpectOutput("test2 message")

	// test1 events must NOT appear.  "test1 message" is specific enough that
	// it should not appear in filebeat's own debug logs.
	var droppedCount atomic.Int64
	test.CountOutput(&droppedCount, "test1 message")

	test.
		ExpectEOF(logFile2).
		WithReportOptions(reportOptions).
		ExpectStart().
		Start(ctx).
		Wait()

	require.EqualValues(t, 0, droppedCount.Load(),
		"test1 events should have been dropped by the drop_event processor")
}
