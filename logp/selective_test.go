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

func TestTypedCoreSelectors(t *testing.T) {
	logSelector := "enabled-log-selector"
	expectedMsg := "this should be logged"

	defaultCfg := DefaultConfig(DefaultEnvironment)
	eventsCfg := DefaultEventConfig(DefaultEnvironment)

	defaultCfg.Level = DebugLevel
	defaultCfg.toObserver = true
	defaultCfg.ToStderr = false
	defaultCfg.ToFiles = false
	defaultCfg.Selectors = []string{logSelector}

	eventsCfg.Level = defaultCfg.Level
	eventsCfg.toObserver = defaultCfg.toObserver
	eventsCfg.ToStderr = defaultCfg.ToStderr
	eventsCfg.ToFiles = defaultCfg.ToFiles
	eventsCfg.Selectors = defaultCfg.Selectors

	if err := ConfigureWithTypedOutput(defaultCfg, eventsCfg, "log.type", "event"); err != nil {
		t.Fatalf("could not configure logger: %s", err)
	}

	enabledSelector := NewLogger(logSelector)
	disabledSelector := NewLogger("foo-selector")

	enabledSelector.Debugw(expectedMsg)
	enabledSelector.Debugw(expectedMsg, "log.type", "event")
	disabledSelector.Debug("this should not be logged")

	logEntries := ObserverLogs().TakeAll()
	if len(logEntries) != 2 {
		t.Errorf("expecting 2 log entries, got %d", len(logEntries))
		t.Log("Log entries:")
		for _, e := range logEntries {
			t.Log("Message:", e.Message, "Fields:", e.Context)
		}
		t.FailNow()
	}

	for i, logEntry := range logEntries {
		msg := logEntry.Message
		if msg != expectedMsg {
			t.Fatalf("[%d] expecting log message '%s', got '%s'", i, expectedMsg, msg)
		}

		// The second entry should also contain `log.type: event`
		if i == 1 {
			fields := logEntry.Context
			if len(fields) != 1 {
				t.Errorf("expecting one field, got %d", len(fields))
			}

			k := fields[0].Key
			v := fields[0].String
			if k != "log.type" {
				t.Errorf("expecting key 'log.type', got '%s'", k)
			}
			if v != "event" {
				t.Errorf("expecting value 'event', got '%s'", v)
			}
		}
	}
}
