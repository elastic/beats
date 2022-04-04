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

//go:build windows
// +build windows

package wineventlog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/andrewkroh/sys/windows/svc/eventlog"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/windows"
)

const (
	winlogbeatTestLogName = "WinEventLogTestGo"

	security4752File      = "../../../x-pack/winlogbeat/module/security/test/testdata/4752.evtx"
	sysmon9File           = "../../../x-pack/winlogbeat/module/sysmon/test/testdata/sysmon-9.01.evtx"
	winErrorReportingFile = "testdata/application-windows-error-reporting.evtx"
)

// createLog creates a new event log and returns a handle for writing events
// to the log.
func createLog(t testing.TB) (log *eventlog.Log, tearDown func()) {
	const name = winlogbeatTestLogName
	const source = "wineventlog_test"

	existed, err := eventlog.InstallAsEventCreate(name, source, eventlog.Error|eventlog.Warning|eventlog.Info)
	if err != nil {
		t.Fatal(err)
	}

	if existed {
		EvtClearLog(NilHandle, name, "")
	}

	log, err = eventlog.Open(source)
	if err != nil {
		eventlog.RemoveSource(name, source)
		eventlog.RemoveProvider(name)
		t.Fatal(err)
	}

	setLogSize(t, winlogbeatTestLogName, 1024*1024*1024) // 1 GiB

	tearDown = func() {
		log.Close()
		EvtClearLog(NilHandle, name, "")
		eventlog.RemoveSource(name, source)
		eventlog.RemoveProvider(name)
	}

	return log, tearDown
}

func safeWriteEvent(t testing.TB, log *eventlog.Log, etype uint16, eid uint32, msgs []string) {
	deadline := time.Now().Add(time.Second * 10)
	for {
		err := log.Report(etype, eid, msgs)
		if err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("Failed to write event to event log", err)
			return
		}
	}
}

// openLog opens an event log or .evtx file for reading.
func openLog(t testing.TB, log string, eventIDFilters ...string) EvtHandle {
	var (
		err   error
		path               = log
		flags EvtQueryFlag = EvtQueryReverseDirection
	)

	if info, err := os.Stat(log); err == nil && info.Mode().IsRegular() {
		flags |= EvtQueryFilePath
	} else {
		flags |= EvtQueryChannelPath
	}

	var query string
	if len(eventIDFilters) > 0 {
		// Convert to URI.
		abs, err := filepath.Abs(log)
		if err != nil {
			t.Fatal(err)
		}
		path = "file://" + filepath.ToSlash(abs)

		query, err = Query{Log: path, EventID: strings.Join(eventIDFilters, ",")}.Build()
		if err != nil {
			t.Fatal(err)
		}
		path = ""
	}

	h, err := EvtQuery(NilHandle, path, query, flags)
	if err != nil {
		t.Fatal("Failed to open log", log, err)
	}
	return h
}

// nextHandle reads one handle from the log. It returns done=true when there
// are no more items to read.
func nextHandle(t *testing.T, log EvtHandle) (handle EvtHandle, done bool) {
	var numReturned uint32
	var handles [1]EvtHandle

	err := _EvtNext(log, 1, &handles[0], 0, 0, &numReturned)
	if err != nil {
		if err == windows.ERROR_NO_MORE_ITEMS {
			return NilHandle, true
		}
		t.Fatal(err)
	}

	return handles[0], false
}

// mustNextHandle reads one handle from the log.
func mustNextHandle(t *testing.T, log EvtHandle) EvtHandle {
	h, done := nextHandle(t, log)
	if done {
		t.Fatal("No more items to read.")
	}
	return h
}

func logAsJSON(t testing.TB, object interface{}) {
	data, err := json.MarshalIndent(object, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(data))
}

func assertEqualIgnoreCase(t *testing.T, expected, actual string) {
	t.Helper()
	assert.Equal(t,
		strings.ToLower(expected),
		strings.ToLower(actual),
	)
}
