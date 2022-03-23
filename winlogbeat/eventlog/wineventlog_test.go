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

package eventlog

import (
	"io"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/andrewkroh/sys/windows/svc/eventlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/winlogbeat/checkpoint"
	"github.com/elastic/beats/v7/winlogbeat/sys/wineventlog"
)

const (
	// Names that are registered by the test for logging events.
	providerName   = "WinlogbeatTestGo"
	sourceName     = "Integration Test"
	customXMLQuery = `<QueryList>
    <Query Id="0" Path="WinlogbeatTestGo">
        <Select Path="WinlogbeatTestGo">*</Select>
    </Query>
</QueryList>`

	// Event message files used when logging events.

	// EventCreate.exe has valid event IDs in the range of 1-1000 where each
	// event message requires a single parameter.
	eventCreateMsgFile = "%SystemRoot%\\System32\\EventCreate.exe"
)

func TestWinEventLogConfig_Validate(t *testing.T) {
	tests := []struct {
		In      winEventLogConfig
		WantErr bool
		Desc    string
	}{
		{
			In: winEventLogConfig{
				ConfigCommon: ConfigCommon{
					ID:       "test",
					XMLQuery: customXMLQuery,
				},
			},
			WantErr: false,
			Desc:    "xml query: all good",
		},
		{
			In: winEventLogConfig{
				ConfigCommon: ConfigCommon{
					ID:       "test",
					XMLQuery: customXMLQuery[:len(customXMLQuery)-4], // Malformed XML by truncation.
				},
			},
			WantErr: true,
			Desc:    "xml query: malformed XML",
		},
		{
			In: winEventLogConfig{
				ConfigCommon: ConfigCommon{
					XMLQuery: customXMLQuery,
				},
			},
			WantErr: true,
			Desc:    "xml query: missing ID",
		},
		{
			In: winEventLogConfig{
				ConfigCommon: ConfigCommon{
					ID:       "test",
					Name:     "test",
					XMLQuery: customXMLQuery,
				},
			},
			WantErr: true,
			Desc:    "xml query: conflicting keys (xml query and name)",
		},
		{
			In: winEventLogConfig{
				ConfigCommon: ConfigCommon{
					ID:       "test",
					XMLQuery: customXMLQuery,
				},
				SimpleQuery: query{IgnoreOlder: 1},
			},
			WantErr: true,
			Desc:    "xml query: conflicting keys (xml query and ignore_older)",
		},
		{
			In: winEventLogConfig{
				ConfigCommon: ConfigCommon{
					ID:       "test",
					XMLQuery: customXMLQuery,
				},
				SimpleQuery: query{Level: "error"},
			},
			WantErr: true,
			Desc:    "xml query: conflicting keys (xml query and level)",
		},
		{
			In: winEventLogConfig{
				ConfigCommon: ConfigCommon{
					ID:       "test",
					XMLQuery: customXMLQuery,
				},
				SimpleQuery: query{EventID: "1000"},
			},
			WantErr: true,
			Desc:    "xml query: conflicting keys (xml query and event_id)",
		},
		{
			In: winEventLogConfig{
				ConfigCommon: ConfigCommon{
					ID:       "test",
					XMLQuery: customXMLQuery,
				},
				SimpleQuery: query{Provider: []string{providerName}},
			},
			WantErr: true,
			Desc:    "xml query: conflicting keys (xml query and provider)",
		},
		{
			In: winEventLogConfig{
				ConfigCommon: ConfigCommon{},
			},
			WantErr: true,
			Desc:    "missing name",
		},
	}

	for _, tc := range tests {
		gotErr := tc.In.Validate()

		if tc.WantErr {
			assert.NotNil(t, gotErr, tc.Desc)
		} else {
			assert.Nil(t, gotErr, "%q got unexpected err: %v", tc.Desc, gotErr)
		}
	}
}

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
	const messageSize = 256 // Originally 31800, such a large value resulted in an empty eventlog under Win10.
	const totalEvents = 1000
	for i := 0; i < totalEvents; i++ {
		safeWriteEvent(t, writer, eventlog.Info, uint32(i%1000), []string{strconv.Itoa(i) + " " + randomSentence(messageSize)})
	}

	openLog := func(t testing.TB, config map[string]interface{}) EventLog {
		return openLog(t, api, nil, config)
	}

	t.Run("has_message", func(t *testing.T) {
		log := openLog(t, map[string]interface{}{"name": providerName, "batch_read_size": 1})
		defer log.Close()

		records, err := log.Read()
		require.NotEmpty(t, records)
		require.NoError(t, err)

		r := records[0]
		require.NotEmpty(t, r.Message, "message field is empty: errors:%v\nrecord:%#v", r.Event.RenderErr, r)
	})

	// Test reading from an event log using a custom XML query.
	t.Run("custom_xml_query", func(t *testing.T) {
		cfg := map[string]interface{}{
			"id":        "custom-xml-query",
			"xml_query": customXMLQuery,
		}

		log := openLog(t, cfg)
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
		wineventlog.EvtClearLog(wineventlog.NilHandle, name, "") //nolint:errcheck // This is just a resource release.
	}

	log, err = eventlog.Open(source)
	//nolint:errcheck // This is just a resource release.
	if err != nil {
		eventlog.RemoveSource(name, source)
		eventlog.RemoveProvider(name)
		t.Fatal(err)
	}

	//nolint:errcheck // This is just a resource release.
	tearDown = func() {
		log.Close()
		wineventlog.EvtClearLog(wineventlog.NilHandle, name, "")
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

// setLogSize set the maximum number of bytes that an event log can hold.
func setLogSize(t testing.TB, provider string, sizeBytes int) {
	output, err := exec.Command("wevtutil.exe", "sl", "/ms:"+strconv.Itoa(sizeBytes), provider).CombinedOutput() //nolint:gosec // No possibility of command injection.
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
