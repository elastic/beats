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

// Tests for NDJSON/JSON parsing functionality, ported from test_json.py.
// These tests use the filestream parsers.ndjson block, which corresponds
// to the log input's json.* settings.

package integration

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	libbeatintegration "github.com/elastic/beats/v7/libbeat/testing/integration"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var jsonReportOptions = libbeatintegration.ReportOptions{
	PrintLinesOnFail:  100,
	PrintConfigOnFail: true,
}

// TestJSONDockerLogs verifies that Docker-format JSON log lines are parsed and
// that message_key extracts the correct field as the event message.
// Corresponds to test_json.py::test_docker_logs.
func TestJSONDockerLogs(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	dir := t.TempDir()
	logFile := filepath.Join(dir, "docker.log")

	// Docker log format: each line is a JSON object with "log", "stream", "time".
	WriteFile(
		t, logFile,
		`{"log":"fetching dependencies\n","stream":"stdout","time":"2016-03-02T22:58:51.338462311Z"}`+"\n"+
			`{"log":"execute build script\n","stream":"stdout","time":"2016-03-02T22:59:04.609292428Z"}`+"\n"+
			`{"log":"build complete\n","stream":"stderr","time":"2016-03-02T22:59:05.617434682Z"}`+"\n",
	)

	config := FilestreamInputConfig("json-docker-test", logFile, FilestreamOptions{
		NDJSON: &NDJSONOptions{
			MessageKey: "log",
		},
	})
	test := NewTest(t, TestOptions{Config: config})

	// With message_key="log", the "log" field becomes the message.
	// Remaining fields (stream, time) appear under json.*.
	test.ExpectJSONFields(mapstr.M{
		"message":     "fetching dependencies\n",
		"json.stream": "stdout",
	})
	test.
		ExpectEOF(logFile).
		WithReportOptions(jsonReportOptions).
		ExpectStart().
		Start(ctx).
		Wait()
}

// TestJSONDockerLogsFiltering verifies that line filtering works on NDJSON input.
// Corresponds to test_json.py::test_docker_logs_filtering.
func TestJSONDockerLogsFiltering(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	dir := t.TempDir()
	logFile := filepath.Join(dir, "docker.log")

	WriteFile(
		t, logFile,
		`{"log":"linux line\n","stream":"stdout","time":"2016-01-01T00:00:00Z"}`+"\n"+
			`{"log":"windows line\n","stream":"stdout","time":"2016-01-01T00:00:01Z"}`+"\n"+
			`{"log":"another linux\n","stream":"stdout","time":"2016-01-01T00:00:02Z"}`+"\n",
	)

	config := FilestreamInputConfig("json-filter-test", logFile, FilestreamOptions{
		NDJSON:       &NDJSONOptions{MessageKey: "log", KeysUnderRoot: true},
		ExcludeLines: []string{"windows"},
	})
	test := NewTest(t, TestOptions{Config: config})

	test.ExpectOutput("linux line")
	test.ExpectOutput("another linux")
	test.
		ExpectEOF(logFile).
		WithReportOptions(jsonReportOptions).
		ExpectStart().
		Start(ctx).
		Wait()
}

// TestJSONKeysUnderRoot verifies that keys_under_root merges JSON fields into
// the top-level event document.
func TestJSONKeysUnderRoot(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	dir := t.TempDir()
	logFile := filepath.Join(dir, "json.log")

	// Use field names that are not reserved by ECS to avoid shadowing.
	WriteFile(
		t, logFile,
		`{"app_name":"myapp","app_env":"staging","message":"hello json world"}`+"\n",
	)

	config := FilestreamInputConfig("json-under-root-test", logFile, FilestreamOptions{
		NDJSON: &NDJSONOptions{
			MessageKey:    "message",
			KeysUnderRoot: true,
		},
	})
	test := NewTest(t, TestOptions{Config: config})
	// With keys_under_root, app_name and app_env appear at the top level.
	test.ExpectJSONFields(mapstr.M{
		"message":  "hello json world",
		"app_name": "myapp",
		"app_env":  "staging",
	})
	test.
		ExpectEOF(logFile).
		WithReportOptions(jsonReportOptions).
		ExpectStart().
		Start(ctx).
		Wait()
}

// TestJSONOverwriteKeys verifies that overwrite_keys allows JSON fields to
// overwrite Filebeat-generated fields.
// Corresponds to test_json.py::test_simple_json_overwrite.
func TestJSONOverwriteKeys(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	dir := t.TempDir()
	logFile := filepath.Join(dir, "override.log")

	WriteFile(
		t, logFile,
		`{"source":"hello","message":"test source"}`+"\n",
	)

	config := FilestreamInputConfig("json-overwrite-test", logFile, FilestreamOptions{
		NDJSON: &NDJSONOptions{
			MessageKey:    "message",
			KeysUnderRoot: true,
			OverwriteKeys: true,
		},
	})
	test := NewTest(t, TestOptions{Config: config})
	test.ExpectJSONFields(mapstr.M{
		"source":  "hello",
		"message": "test source",
	})
	test.
		ExpectEOF(logFile).
		WithReportOptions(jsonReportOptions).
		ExpectStart().
		Start(ctx).
		Wait()
}

// TestJSONAddErrorKey verifies that ignore_decoding_error allows non-JSON lines
// to still produce events, and that add_error_key annotates them.
// Without a message_key, filestream sets the event message to the raw log line,
// so both the invalid text and the raw valid-JSON string appear in output.
// Corresponds to test_json.py::test_invalid_json_adds_tag.
func TestJSONAddErrorKey(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	EnsureCompiled(ctx, t)

	dir := t.TempDir()
	logFile := filepath.Join(dir, "invalid.log")

	// Mix of valid and invalid JSON lines.
	WriteFile(
		t, logFile,
		"this is not json\n"+
			`{"json_field":"valid_json_value"}`+"\n",
	)

	config := FilestreamInputConfig("json-error-key-test", logFile, FilestreamOptions{
		NDJSON: &NDJSONOptions{
			AddErrorKey:         true,
			IgnoreDecodingError: true,
		},
	})
	test := NewTest(t, TestOptions{Config: config})

	// Without message_key, filestream's raw line becomes the event message.
	// Both lines produce events; their raw content is visible in the output.
	test.ExpectOutput("this is not json")
	test.ExpectOutput("valid_json_value")
	test.
		ExpectEOF(logFile).
		WithReportOptions(jsonReportOptions).
		ExpectStart().
		Start(ctx).
		Wait()
}
