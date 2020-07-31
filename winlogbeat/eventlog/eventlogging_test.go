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
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/andrewkroh/sys/windows/svc/eventlog"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/winlogbeat/checkpoint"
	"github.com/elastic/beats/v7/winlogbeat/sys/eventlogging"
)

// Names that are registered by the test for logging events.
const (
	providerName = "WinlogbeatTestGo"
	sourceName   = "Integration Test"
)

// Event message files used when logging events.
const (
	// EventCreate.exe has valid event IDs in the range of 1-1000 where each
	// event message requires a single parameter.
	eventCreateMsgFile = "%SystemRoot%\\System32\\EventCreate.exe"
	// services.exe is used by the Service Control Manager as its event message
	// file; these tests use it to log messages with more than one parameter.
	servicesMsgFile = "%SystemRoot%\\System32\\services.exe"
	// netevent.dll has messages that require no message parameters.
	netEventMsgFile = "%SystemRoot%\\System32\\netevent.dll"
)

// Test messages.
var messages = map[uint32]struct {
	eventType uint16
	message   string
}{
	1: {
		eventType: eventlog.Info,
		message:   "Hmmmm.",
	},
	2: {
		eventType: eventlog.Success,
		message:   "I am so blue I'm greener than purple.",
	},
	3: {
		eventType: eventlog.Warning,
		message:   "I stepped on a Corn Flake, now I'm a Cereal Killer.",
	},
	4: {
		eventType: eventlog.Error,
		message:   "The quick brown fox jumps over the lazy dog.",
	},
	5: {
		eventType: eventlog.AuditSuccess,
		message:   "Where do random thoughts come from?",
	},
	6: {
		eventType: eventlog.AuditFailure,
		message:   "Login failure for user xyz!",
	},
}

var oneTimeLogpInit sync.Once

// Initializes logp if the verbose flag was set.
func configureLogp() {
	oneTimeLogpInit.Do(func() {
		if testing.Verbose() {
			logp.DevelopmentSetup(logp.WithSelectors("eventlog"))
			logp.Info("DEBUG enabled for eventlog.")
		} else {
			logp.DevelopmentSetup(logp.WithLevel(logp.WarnLevel))
		}
	})
}

// Verify that all messages are read from the event log.
func TestRead(t *testing.T) {
	configureLogp()
	writer, teardown := createLog(t)
	defer teardown()

	// Publish test messages:
	for k, m := range messages {
		if err := writer.Report(m.eventType, k, []string{m.message}); err != nil {
			t.Fatal(err)
		}
	}

	// Read messages:
	log := openEventLogging(t, 0, map[string]interface{}{"name": providerName})
	defer log.Close()

	records, err := log.Read()
	if err != nil {
		t.Fatal(err)
	}

	// Validate messages:
	assert.Len(t, records, len(messages))
	for _, record := range records {
		t.Log(record)
		m, exists := messages[record.EventIdentifier.ID]
		if !exists {
			t.Errorf("Unknown EventId %d Read() from event log. %v", record.EventIdentifier.ID, record)
			continue
		}
		assert.Equal(t, eventlogging.EventType(m.eventType).String(), record.Level)
		assert.Equal(t, m.message, strings.TrimRight(record.Message, "\r\n"))
	}

	// Validate getNumberOfEventLogRecords returns the correct number of messages.
	numMessages, err := eventlogging.GetNumberOfEventLogRecords(eventlogging.Handle(writer.Handle))
	assert.NoError(t, err)
	assert.Equal(t, len(messages), int(numMessages))
}

// Verify that messages whose text is larger than the read buffer cause a
// message error to be returned. Normally Winlogbeat is run with the largest
// possible buffer so this error should not occur.
func TestFormatMessageWithLargeMessage(t *testing.T) {
	configureLogp()
	writer, teardown := createLog(t)
	defer teardown()

	const message = "Hello"
	if err := writer.Report(eventlog.Info, 1, []string{message}); err != nil {
		t.Fatal(err)
	}

	// Messages are received as UTF-16 so we must have enough space in the read
	// buffer for the message, a windows newline, and a null-terminator.
	const requiredBufferSize = len(message+"\r\n")*2 + 2

	// Read messages:
	log := openEventLogging(t, 0, map[string]interface{}{
		"name": providerName,
		// Use a buffer smaller than what is required.
		"format_buffer_size": requiredBufferSize / 2,
	})
	defer log.Close()

	records, err := log.Read()
	if err != nil {
		t.Fatal(err)
	}

	// Validate messages:
	assert.Len(t, records, 1)
	for _, record := range records {
		t.Log(record)
		assert.Equal(t, []string{"The data area passed to a system call is too small."}, record.RenderErr)
	}
}

// Test that when an unknown Event ID is found, that a message containing the
// insert strings (the message parameters) is returned.
func TestReadUnknownEventId(t *testing.T) {
	configureLogp()
	writer, teardown := createLog(t, servicesMsgFile)
	defer teardown()

	const eventID uint32 = 1000
	const msg = "Test Message"
	if err := writer.Success(eventID, msg); err != nil {
		t.Fatal(err)
	}

	// Read messages:
	log := openEventLogging(t, 0, map[string]interface{}{"name": providerName})
	defer log.Close()

	records, err := log.Read()
	if err != nil {
		t.Fatal(err)
	}

	// Verify the error message:
	assert.Len(t, records, 1)
	if len(records) != 1 {
		t.FailNow()
	}
	assert.Equal(t, eventID, records[0].EventIdentifier.ID)
	assert.Equal(t, msg, records[0].EventData.Pairs[0].Value)
	assert.NotNil(t, records[0].RenderErr)
	assert.Equal(t, "", records[0].Message)
}

// Test that multiple event message files are searched for an event ID. This
// test configures the "EventMessageFile" registry value as a semi-color
// separated list of files. If the message for an event ID is not found in one
// of the files then the next file should be checked.
func TestReadTriesMultipleEventMsgFiles(t *testing.T) {
	configureLogp()
	writer, teardown := createLog(t, servicesMsgFile, eventCreateMsgFile)
	defer teardown()

	const eventID uint32 = 1000
	const msg = "Test Message"
	if err := writer.Success(eventID, msg); err != nil {
		t.Fatal(err)
	}

	// Read messages:
	log := openEventLogging(t, 0, map[string]interface{}{"name": providerName})
	defer log.Close()

	records, err := log.Read()
	if err != nil {
		t.Fatal(err)
	}

	// Verify the error message:
	assert.Len(t, records, 1)
	if len(records) != 1 {
		t.FailNow()
	}
	assert.Equal(t, eventID, records[0].EventIdentifier.ID)
	assert.Equal(t, msg, strings.TrimRight(records[0].Message, "\r\n"))
}

// Test event messages that require more than one message parameter.
func TestReadMultiParameterMsg(t *testing.T) {
	configureLogp()
	writer, teardown := createLog(t, servicesMsgFile)
	defer teardown()

	// EventID observed by exporting system event log to XML and doing calculation.
	// <EventID Qualifiers="16384">7036</EventID>
	// 1073748860 = 16384 << 16 + 7036
	// https://msdn.microsoft.com/en-us/library/windows/desktop/aa385206(v=vs.85).aspx
	const eventID uint32 = 1073748860
	const template = "The %s service entered the %s state."
	msgs := []string{"Windows Update", "running"}
	if err := writer.Report(eventlog.Info, eventID, msgs); err != nil {
		t.Fatal(err)
	}

	// Read messages:
	log := openEventLogging(t, 0, map[string]interface{}{"name": providerName})
	defer log.Close()

	records, err := log.Read()
	if err != nil {
		t.Fatal(err)
	}

	// Verify the message contents:
	assert.Len(t, records, 1)
	if len(records) != 1 {
		t.FailNow()
	}
	assert.Equal(t, eventID&0xFFFF, records[0].EventIdentifier.ID)
	assert.Equal(t, fmt.Sprintf(template, msgs[0], msgs[1]),
		strings.TrimRight(records[0].Message, "\r\n"))
}

// Verify that opening an invalid provider succeeds. Windows opens the
// Application event log provider when this happens (unfortunately).
func TestOpenInvalidProvider(t *testing.T) {
	configureLogp()

	log := openEventLogging(t, 0, map[string]interface{}{"name": "nonExistentProvider"})
	defer log.Close()

	_, err := log.Read()
	assert.NoError(t, err)
}

// Test event messages that require no parameters.
func TestReadNoParameterMsg(t *testing.T) {
	configureLogp()
	writer, teardown := createLog(t, netEventMsgFile)
	defer teardown()

	const eventID uint32 = 2147489654 // 1<<31 + 6006
	const template = "The Event log service was stopped."
	msgs := []string{}
	if err := writer.Report(eventlog.Info, eventID, msgs); err != nil {
		t.Fatal(err)
	}

	// Read messages:
	log := openEventLogging(t, 0, map[string]interface{}{"name": providerName})
	defer log.Close()

	records, err := log.Read()
	if err != nil {
		t.Fatal(err)
	}

	// Verify the message contents:
	assert.Len(t, records, 1)
	if len(records) != 1 {
		t.FailNow()
	}
	assert.Equal(t, eventID&0xFFFF, records[0].EventIdentifier.ID)
	assert.Equal(t, template,
		strings.TrimRight(records[0].Message, "\r\n"))
}

// TestReadWhileCleared tests that the Read method recovers from the event log
// being cleared or reset while reading.
func TestReadWhileCleared(t *testing.T) {
	configureLogp()
	writer, teardown := createLog(t)
	defer teardown()

	log := openEventLogging(t, 0, map[string]interface{}{"name": providerName})
	defer log.Close()

	writer.Info(1, "Message 1")
	writer.Info(2, "Message 2")
	lr, err := log.Read()
	assert.NoError(t, err, "Expected 2 messages but received error")
	assert.Len(t, lr, 2, "Expected 2 messages")

	assert.NoError(t, eventlogging.ClearEventLog(eventlogging.Handle(writer.Handle), ""))
	lr, err = log.Read()
	assert.NoError(t, err, "Expected 0 messages but received error")
	assert.Len(t, lr, 0, "Expected 0 message")

	writer.Info(3, "Message 3")
	lr, err = log.Read()
	assert.NoError(t, err, "Expected 1 message but received error")
	assert.Len(t, lr, 1, "Expected 1 message")
	if len(lr) > 0 {
		assert.Equal(t, uint32(3), lr[0].EventIdentifier.ID)
	}
}

// Test event messages that include less parameters than required for message
// formatting (caused a crash in previous versions)
func TestReadMissingParameters(t *testing.T) {
	configureLogp()
	writer, teardown := createLog(t, servicesMsgFile)
	defer teardown()

	const eventID uint32 = 1073748860
	// Missing parameters will be substituted by "(null)"
	const template = "The %s service entered the (null) state."
	msgs := []string{"Windows Update"}
	if err := writer.Report(eventlog.Info, eventID, msgs); err != nil {
		t.Fatal(err)
	}

	// Read messages:
	log := openEventLogging(t, 0, map[string]interface{}{"name": providerName})
	defer log.Close()

	records, err := log.Read()
	if err != nil {
		t.Fatal(err)
	}

	// Verify the message contents:
	assert.Len(t, records, 1)
	if len(records) != 1 {
		t.FailNow()
	}
	assert.Equal(t, eventID&0xFFFF, records[0].EventIdentifier.ID)
	assert.Equal(t, fmt.Sprintf(template, msgs[0]),
		strings.TrimRight(records[0].Message, "\r\n"))
}

func openEventLogging(t *testing.T, recordID uint64, options map[string]interface{}) EventLog {
	t.Helper()
	return openLog(t, eventLoggingAPIName, &checkpoint.EventLogState{RecordNumber: recordID}, options)
}
