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

package logptest

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewFileLogger(t *testing.T) {
	logger := NewFileLogger(t, "")
	logger.Logger = logger.Named("test-logger")
	logger.Debug("foo")

	assertLogFormat(t, logger.logFile.Name())

	t.Run("FindInLogs", func(t *testing.T) {
		found, err := logger.FindInLogs("foo")
		if err != nil {
			t.Fatalf("did not expect an error from 'FindInLogs': %s", err)
		}

		if !found {
			t.Fatal("expecing 'FindInLogs' to return true")
		}
	})

	t.Run("LogContains", func(t *testing.T) {
		// Log again, because we track offset
		logger.Debug("foo")
		// Call it and expect the test not to fail
		logger.LogContains(t, "foo")
	})

	t.Run("WaitLogContains", func(t *testing.T) {
		// Log again, because we track offset
		logger.Debug("foo")
		// Call it and expect the test not to fail
		logger.WaitLogsContains(t, "foo", 200*time.Millisecond)
	})

	t.Run("ResetOffset", func(t *testing.T) {
		logger.ResetOffset()

		// We logged "foo" 3 times, so we should find it 3 times in a row.
		for range 3 {
			found, err := logger.FindInLogs("foo")
			if err != nil {
				t.Fatalf("did not expect an error from 'FindInLogs': %s", err)
			}

			if !found {
				t.Fatal("expecing 'FindInLogs' to return true")
			}
		}
	})
}

func assertLogFormat(t *testing.T, path string) {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("cannot open log file: %s", err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		entry := struct {
			Timestamp string `json:"@timestamp"`
			LogLevel  string `json:"log.level"`
			Logger    string `json:"log.logger"`
			Message   string `json:"message"`
		}{}

		if err := json.Unmarshal(sc.Bytes(), &entry); err != nil {
			t.Fatalf("%q is not a ndjson file: %s", path, err)
		}

		// ensure the basic fileds are populated
		if entry.Timestamp == "" {
			t.Error("'@timestamp' cannot be empty/zero value")
		}

		if entry.LogLevel == "" {
			t.Error("'log.level' cannot be empty")
		}

		if entry.Message == "" {
			t.Error("message cannot be empty")
		}

		if entry.Logger == "" {
			t.Error("'log.logger' cannot be empty")
		}
	}
}

func TestLoggerCustomDir(t *testing.T) {
	dir := t.TempDir()
	logger := NewFileLogger(t, dir)
	logger.Info("foo")

	files, err := filepath.Glob(filepath.Join(dir, "testing-logger-*.log"))
	if err != nil {
		t.Fatalf("cannot resolve glob: %s", err)
	}

	if len(files) != 1 {
		t.Fatalf("expecting a single log file, got: %d", len(files))
	}
}

func TestLoggerFileIsRemoved(t *testing.T) {
	toBeDeletedLogFile := ""
	// Use a subtest so it can finish and succeed before
	// we check if the log file has been removed
	t.Run("log file is removed", func(t *testing.T) {
		logger := NewFileLogger(t, "")
		logger.Info("foo")
		toBeDeletedLogFile = logger.logFile.Name()
	})

	_, err := os.Stat(toBeDeletedLogFile)
	if !errors.Is(err, os.ErrNotExist) {
		t.Error("the log file must be removed after a test passes")
	}
}

func TestLoggerFileIsKeptOnTestFailure(t *testing.T) {
	if os.Getenv("INNER_TEST") == "1" {
		// We're inside the subprocess, use Logger and make it fail the test,
		// so the actual test can ensure it is behaving correctly
		logger := NewFileLogger(t, "")
		logger.Info("foo")
		logger.Warn("foo")
		logger.Debug("foo")
		logger.LogContains(t, "bar")

		return
	}

	//nolint:gosec // This is intentionally running a subprocess
	cmd := exec.CommandContext(
		t.Context(),
		os.Args[0],
		fmt.Sprintf("-test.run=^%s$",
			t.Name()),
		"-test.v")
	cmd.Env = append(cmd.Env, "INNER_TEST=1")

	out, cmdErr := cmd.CombinedOutput()
	if cmdErr != nil {
		// The test ran by cmd will fail and retrun 1 as the exit code. So we only
		// print the error if the main test fails.
		defer func() {
			if t.Failed() {
				t.Errorf("the test process returned an error (this is expected in on a normal test execution): %s", cmdErr)
			}
		}()
	}

	var path string
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		txt := sc.Text()
		// To extract the log file path we split txt in a way that the path
		// is the 2nd element.
		// The string we're parsing:
		// logger.go:103: Full logs written to /tmp/testing-logger-1901210787.log
		if strings.Contains(txt, "Full logs written to") {
			split := strings.Split(txt, "Full logs written to ")
			if len(split) != 2 {
				t.Fatalf("could not parse log file form test output, invalid format %q", txt)
			}
			path = split[1]
			t.Cleanup(func() {
				if t.Failed() {
					t.Logf("Log file %q", path)
				}
			})
		}
	}

	// Finally ensure the log file was kept and has the correct number of lines
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Fatalf("log file %q not found after the test exited", path)
		}
		t.Fatalf("cannot open log file for reading: %s", err)
	}
	defer f.Close()

	lineCount := 0
	for sc := bufio.NewScanner(f); sc.Scan(); {
		lineCount++
	}

	if got, want := lineCount, 3; got != want {
		t.Fatalf("expecting log file to contain %d lines, got %d:", want, got)
	}
}
