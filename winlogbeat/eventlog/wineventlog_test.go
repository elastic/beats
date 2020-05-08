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

// +build windows

package eventlog

import (
	"io"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/andrewkroh/sys/windows/svc/eventlog"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/winlogbeat/checkpoint"
	"github.com/elastic/beats/v7/winlogbeat/sys/wineventlog"
)

func TestWindowsEventLogAPI(t *testing.T) {
	testWindowsEventLog(t, winEventLogAPIName)
}

func TestWindowsEventLogAPIExperimental(t *testing.T) {
	testWindowsEventLog(t, winEventLogExpAPIName)
}

func testWindowsEventLog(t *testing.T, api string) {
	writer, teardown := createLog(t)
	defer teardown()

	setLogSize(t, providerName, gigabyte)

	// Publish large test messages.
	const totalEvents = 1000
	for i := 0; i < totalEvents; i++ {
		err := writer.Report(eventlog.Info, uint32(i%1000), []string{strconv.Itoa(i) + " " + randomSentence(31800)})
		if err != nil {
			t.Fatal(err)
		}
	}

	openLog := func(t testing.TB, config map[string]interface{}) EventLog {
		return openLog(t, api, nil, config)
	}

	t.Run("batch_read_size_config", func(t *testing.T) {
		const batchReadSize = 2

		log := openLog(t, map[string]interface{}{"name": providerName, "batch_read_size": batchReadSize})
		defer log.Close()

		records, err := log.Read()
		if err != nil {
			t.Fatal(err)
		}

		assert.Len(t, records, batchReadSize)
	})

	// Test reading from an event log using a large batch_read_size parameter.
	// When combined with large messages this causes EvtNext to fail with
	// RPC_S_INVALID_BOUND error. The reader should recover from the error.
	t.Run("large_batch_read", func(t *testing.T) {
		log := openLog(t, map[string]interface{}{"name": providerName, "batch_read_size": 1024})
		defer log.Close()

		var eventCount int

		for eventCount < totalEvents {
			records, err := log.Read()
			if err != nil {
				t.Fatal("read error", err)
			}
			if len(records) == 0 {
				t.Fatal("read returned 0 records")
			}

			t.Logf("Read() returned %d events.", len(records))
			eventCount += len(records)
		}

		assert.Equal(t, totalEvents, eventCount)
	})

	t.Run("evtx_file", func(t *testing.T) {
		path, err := filepath.Abs("../sys/wineventlog/testdata/sysmon-9.01.evtx")
		if err != nil {
			t.Fatal(err)
		}

		log := openLog(t, map[string]interface{}{
			"name":           path,
			"no_more_events": "stop",
		})
		defer log.Close()

		records, err := log.Read()

		// This implementation returns the EOF on the next call.
		if err == nil && api == winEventLogAPIName {
			_, err = log.Read()
		}

		if assert.Error(t, err, "no_more_events=stop requires io.EOF to be returned") {
			assert.Equal(t, io.EOF, err)
		}

		assert.Len(t, records, 32)
	})
}

// ---- Utility Functions -----

// createLog creates a new event log and returns a handle for writing events
// to the log.
func createLog(t testing.TB, messageFiles ...string) (log *eventlog.Log, tearDown func()) {
	const name = providerName
	const source = sourceName

	messageFile := eventCreateMsgFile
	if len(messageFiles) > 0 {
		messageFile = strings.Join(messageFiles, ";")
	}

	existed, err := eventlog.Install(name, source, messageFile, true, eventlog.Error|eventlog.Warning|eventlog.Info)
	if err != nil {
		t.Fatal(err)
	}

	if existed {
		wineventlog.EvtClearLog(wineventlog.NilHandle, name, "")
	}

	log, err = eventlog.Open(source)
	if err != nil {
		eventlog.RemoveSource(name, source)
		eventlog.RemoveProvider(name)
		t.Fatal(err)
	}

	tearDown = func() {
		log.Close()
		wineventlog.EvtClearLog(wineventlog.NilHandle, name, "")
		eventlog.RemoveSource(name, source)
		eventlog.RemoveProvider(name)
	}

	return log, tearDown
}

// setLogSize set the maximum number of bytes that an event log can hold.
func setLogSize(t testing.TB, provider string, sizeBytes int) {
	output, err := exec.Command("wevtutil.exe", "sl", "/ms:"+strconv.Itoa(sizeBytes), provider).CombinedOutput()
	if err != nil {
		t.Fatal("Failed to set log size", err, string(output))
	}
}

func openLog(t testing.TB, api string, state *checkpoint.EventLogState, config map[string]interface{}) EventLog {
	cfg, err := common.NewConfigFrom(config)
	if err != nil {
		t.Fatal(err)
	}

	var log EventLog
	switch api {
	case winEventLogAPIName:
		log, err = newWinEventLog(cfg)
	case winEventLogExpAPIName:
		log, err = newWinEventLogExp(cfg)
	case eventLoggingAPIName:
		log, err = newEventLogging(cfg)
	default:
		t.Fatalf("Unknown API name: '%s'", api)
	}
	if err != nil {
		t.Fatal(err)
	}

	var eventLogState checkpoint.EventLogState
	if state != nil {
		eventLogState = *state
	}

	if err = log.Open(eventLogState); err != nil {
		log.Close()
		t.Fatal(err)
	}

	return log
}
