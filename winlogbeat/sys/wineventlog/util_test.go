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

package wineventlog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"golang.org/x/sys/windows/svc/eventlog"
)

const (
	winlogbeatTestLogName = "WinEventLogTestGo"

	security4752File      = "testdata/4752.evtx"
	security4738File      = "testdata/4738.evtx"
	winErrorReportingFile = "testdata/application-windows-error-reporting.evtx"
)

// createLog creates a new event log and returns a handle for writing events
// to the log.
//
//nolint:errcheck // Errors are not checked since they always precede termination.
func createLog(t testing.TB) (log *eventlog.Log, tearDown func()) {
	t.Helper()
	const name = winlogbeatTestLogName
	const source = "wineventlog_test"

	existed, err := installAsEventCreate(name, source, eventlog.Error|eventlog.Warning|eventlog.Info)
	if err != nil {
		t.Fatalf("eventlog.InstallAsEventCreate failed: %v", err)
	}

	if existed {
		EvtClearLog(NilHandle, name, "")
	}

	log, err = eventlog.Open(source)
	if err != nil {
		removeSource(name, source)
		removeProvider(name)
		t.Fatalf("eventlog.Open failed: %v", err)
	}

	setLogSize(t, winlogbeatTestLogName, 1024*1024*1024) // 1 GiB

	tearDown = func() {
		log.Close()
		EvtClearLog(NilHandle, name, "")
		removeSource(name, source)
		removeProvider(name)
	}

	return log, tearDown
}

func safeWriteEvent(t testing.TB, log *eventlog.Log, eid uint32, msg string) {
	t.Helper()
	deadline := time.Now().Add(time.Second * 10)
	for {
		err := log.Info(eid, msg)
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
	t.Helper()
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
	t.Helper()
	var numReturned uint32
	var handles [1]EvtHandle

	err := _EvtNext(log, 1, &handles[0], 0, 0, &numReturned)
	if err != nil {
		if err == windows.ERROR_NO_MORE_ITEMS { //nolint:errorlint // Bad linter! x/sys errors are not wrapped.
			return NilHandle, true
		}
		t.Fatal(err)
	}

	return handles[0], false
}

// mustNextHandle reads one handle from the log.
func mustNextHandle(t *testing.T, log EvtHandle) EvtHandle {
	t.Helper()
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

const Application = "Application"

const eventLogKeyName = `SYSTEM\CurrentControlSet\Services\EventLog`

// removeSource deletes all registry elements installed for an event logging source.
func removeSource(provider, src string) error {
	providerKeyName := fmt.Sprintf("%s\\%s", eventLogKeyName, provider)
	pk, err := registry.OpenKey(registry.LOCAL_MACHINE, providerKeyName, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer pk.Close()
	return registry.DeleteKey(pk, src)
}

// removeProvider deletes all registry elements installed for an event logging provider.
// Only use this method if you have installed a custom provider.
func removeProvider(provider string) error {
	// Protect against removing Application.
	if provider == Application {
		return fmt.Errorf("%s cannot be removed. Only custom providers can be removed.", provider)
	}

	eventLogKey, err := registry.OpenKey(registry.LOCAL_MACHINE, eventLogKeyName, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer eventLogKey.Close()
	return registry.DeleteKey(eventLogKey, provider)
}

func installAsEventCreate(provider, src string, eventsSupported uint32) (bool, error) {
	eventLogKey, err := registry.OpenKey(registry.LOCAL_MACHINE, eventLogKeyName, registry.CREATE_SUB_KEY)
	if err != nil {
		return false, err
	}
	defer eventLogKey.Close()

	pk, _, err := registry.CreateKey(eventLogKey, provider, registry.SET_VALUE)
	if err != nil {
		return false, err
	}
	defer pk.Close()

	sk, alreadyExist, err := registry.CreateKey(pk, src, registry.SET_VALUE)
	if err != nil {
		return false, err
	}
	defer sk.Close()
	if alreadyExist {
		return true, nil
	}

	err = sk.SetDWordValue("CustomSource", 1)
	if err != nil {
		return false, err
	}
	err = sk.SetExpandStringValue("EventMessageFile", "%SystemRoot%\\System32\\EventCreate.exe")
	if err != nil {
		return false, err
	}
	err = sk.SetDWordValue("TypesSupported", eventsSupported)
	if err != nil {
		return false, err
	}
	return false, nil
}
