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

package logp

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHasSelector(t *testing.T) {
	err := DevelopmentSetup(WithSelectors("*", "config"))
	require.NoError(t, err)
	assert.True(t, HasSelector("config"))
	assert.False(t, HasSelector("publish"))
}

func TestLoggerSelectors(t *testing.T) {
	if err := DevelopmentSetup(WithSelectors("good", " padded "), ToObserverOutput()); err != nil {
		t.Fatal(err)
	}

	assert.True(t, HasSelector("padded"))

	good := NewLogger("good")
	bad := NewLogger("bad")

	good.Debug("is logged")
	logs := ObserverLogs().TakeAll()
	assert.Len(t, logs, 1)

	// Selectors only apply to debug level logs.
	bad.Debug("not logged")
	logs = ObserverLogs().TakeAll()
	assert.Len(t, logs, 0)

	bad.Info("is also logged")
	logs = ObserverLogs().TakeAll()
	assert.Len(t, logs, 1)
}

func TestTypedAndCloserCoreSelectors(t *testing.T) {
	tempDir := t.TempDir()

	logSelector := "enabled-log-selector"
	expectedMsg := "this should be logged"

	defaultCfg := DefaultConfig(DefaultEnvironment)
	eventsCfg := DefaultEventConfig(DefaultEnvironment)

	defaultCfg.Level = DebugLevel
	defaultCfg.Beat = t.Name()
	defaultCfg.Selectors = []string{logSelector}
	defaultCfg.ToStderr = false
	defaultCfg.ToFiles = true
	defaultCfg.Files.Path = tempDir

	eventsCfg.Level = defaultCfg.Level
	eventsCfg.Beat = defaultCfg.Beat
	eventsCfg.Selectors = defaultCfg.Selectors
	eventsCfg.ToStderr = defaultCfg.ToStderr
	eventsCfg.ToFiles = defaultCfg.ToFiles
	eventsCfg.Files.Path = tempDir

	if err := ConfigureWithTypedOutput(defaultCfg, eventsCfg, "log.type", "event"); err != nil {
		t.Fatalf("could not configure logger: %s", err)
	}

	enabledSelector := NewLogger(logSelector)
	defer enabledSelector.Close()
	disabledSelector := NewLogger("foo-selector")
	defer disabledSelector.Close()

	enabledSelector.Debugw(expectedMsg)
	enabledSelector.Debugw(expectedMsg, "log.type", "event")
	disabledSelector.Debug("this should not be logged")

	logEntries := takeAllLogsFromPath(t, tempDir)
	if len(logEntries) != 2 {
		t.Errorf("expecting 2 log entries, got %d", len(logEntries))
		t.Log("Log entries:")
		for _, e := range logEntries {
			t.Log(e)
		}
		t.FailNow()
	}

	for i, logEntry := range logEntries {
		msg := logEntry["message"].(string) //nolint: errcheck // We know it's a string and it is a test
		if msg != expectedMsg {
			t.Fatalf("[%d] expecting log message '%s', got '%s'", i, expectedMsg, msg)
		}

		// The second entry should also contain `log.type: event`
		if i == 1 {
			logType := logEntry["log.type"].(string) //nolint: errcheck // We know it's a string and it is a test
			if logType != "event" {
				t.Errorf("expecting value 'event', got '%s'", logType)
			}
		}
	}
}

func takeAllLogsFromPath(t *testing.T, path string) []map[string]any {
	entries := []map[string]any{}

	glob := filepath.Join(path, "*.ndjson")
	files, err := filepath.Glob(glob)
	if err != nil {
		t.Fatalf("cannot get files for glob '%s': %s", glob, err)
	}

	for _, filePath := range files {
		f, err := os.Open(filePath)
		if err != nil {
			t.Fatalf("cannot open file '%s': %s", filePath, err)
		}
		defer f.Close()

		sc := bufio.NewScanner(f)
		for sc.Scan() {
			m := map[string]any{}
			bytes := sc.Bytes()
			if err := json.Unmarshal(bytes, &m); err != nil {
				t.Fatalf("cannot unmarshal log entry: '%s', err: %s", string(bytes), err)
			}

			entries = append(entries, m)
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		t1 := entries[i]["@timestamp"].(string) //nolint: errcheck // We know it's a string and it is a test
		t2 := entries[j]["@timestamp"].(string) //nolint: errcheck // We know it's a string and it is a test
		return t1 < t2
	})

	return entries
}
