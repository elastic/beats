// +build windows

package eventlog

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	elog "github.com/andrewkroh/sys/windows/svc/eventlog"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/winlogbeat/sys/eventlogging"
	"github.com/joeshaw/multierror"
	"github.com/stretchr/testify/assert"
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

const allLevels = elog.Success | elog.AuditFailure | elog.AuditSuccess | elog.Error | elog.Info | elog.Warning

// Test messages.
var messages = map[uint32]struct {
	eventType uint16
	message   string
}{
	1: {
		eventType: elog.Info,
		message:   "Hmmmm.",
	},
	2: {
		eventType: elog.Success,
		message:   "I am so blue I'm greener than purple.",
	},
	3: {
		eventType: elog.Warning,
		message:   "I stepped on a Corn Flake, now I'm a Cereal Killer.",
	},
	4: {
		eventType: elog.Error,
		message:   "The quick brown fox jumps over the lazy dog.",
	},
	5: {
		eventType: elog.AuditSuccess,
		message:   "Where do random thoughts come from?",
	},
	6: {
		eventType: elog.AuditFailure,
		message:   "Login failure for user xyz!",
	},
}

var oneTimeLogpInit sync.Once

// Initializes logp if the verbose flag was set.
func configureLogp() {
	oneTimeLogpInit.Do(func() {
		if testing.Verbose() {
			logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"eventlog", "eventlog_detail"})
			logp.Info("DEBUG enabled for eventlog.")
		} else {
			logp.LogInit(logp.LOG_WARNING, "", false, true, []string{})
		}

		// Clear the event log before starting.
		log, _ := elog.Open(sourceName)
		eventlogging.ClearEventLog(eventlogging.Handle(log.Handle), "")
		log.Close()
	})
}

// initLog initializes an event logger. It registers the source name with
// the registry if it does not already exist.
func initLog(provider, source, msgFile string) (*elog.Log, error) {
	// Install entry to registry:
	_, err := elog.Install(providerName, sourceName, msgFile, true, allLevels)
	if err != nil {
		return nil, err
	}

	// Open a new logger for writing events:
	log, err := elog.Open(sourceName)
	if err != nil {
		var errs multierror.Errors
		errs = append(errs, err)
		err := elog.RemoveSource(providerName, sourceName)
		if err != nil {
			errs = append(errs, err)
		}
		err = elog.RemoveProvider(providerName)
		if err != nil {
			errs = append(errs, err)
		}
		return nil, errs.Err()
	}

	return log, nil
}

// uninstallLog unregisters the event logger from the registry and closes the
// log's handle if it is open.
func uninstallLog(provider, source string, log *elog.Log) error {
	var errs multierror.Errors

	if log != nil {
		err := eventlogging.ClearEventLog(eventlogging.Handle(log.Handle), "")
		if err != nil {
			errs = append(errs, err)
		}

		err = log.Close()
		if err != nil {
			errs = append(errs, err)
		}
	}

	err := elog.RemoveSource(providerName, sourceName)
	if err != nil {
		errs = append(errs, err)
	}

	err = elog.RemoveProvider(providerName)
	if err != nil {
		errs = append(errs, err)
	}

	return errs.Err()
}

// Verify that all messages are read from the event log.
func TestRead(t *testing.T) {

	configureLogp()
	log, err := initLog(providerName, sourceName, eventCreateMsgFile)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := uninstallLog(providerName, sourceName, log)
		if err != nil {
			t.Fatal(err)
		}
	}()

	// Publish test messages:
	for k, m := range messages {
		err = log.Report(m.eventType, k, []string{m.message})
		if err != nil {
			t.Fatal(err)
		}
	}

	// Read messages:
	eventlog, err := newEventLogging(map[string]interface{}{"name": providerName})
	if err != nil {
		t.Fatal(err)
	}
	err = eventlog.Open(0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := eventlog.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()
	records, err := eventlog.Read()
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
	numMessages, err := eventlogging.GetNumberOfEventLogRecords(eventlogging.Handle(log.Handle))
	assert.NoError(t, err)
	assert.Equal(t, len(messages), int(numMessages))
}

// Verify that messages whose text is larger than the read buffer cause a
// message error to be returned. Normally Winlogbeat is run with the largest
// possible buffer so this error should not occur.
func TestFormatMessageWithLargeMessage(t *testing.T) {
	configureLogp()
	log, err := initLog(providerName, sourceName, eventCreateMsgFile)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := uninstallLog(providerName, sourceName, log)
		if err != nil {
			t.Fatal(err)
		}
	}()

	message := "Hello"
	err = log.Report(elog.Info, 1, []string{message})
	if err != nil {
		t.Fatal(err)
	}

	// Messages are received as UTF-16 so we must have enough space in the read
	// buffer for the message, a windows newline, and a null-terminator.
	requiredBufferSize := len(message+"\r\n")*2 + 2

	// Read messages:
	eventlog, err := newEventLogging(map[string]interface{}{
		"name": providerName,
		// Use a buffer smaller than what is required.
		"format_buffer_size": requiredBufferSize / 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	err = eventlog.Open(0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := eventlog.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()
	records, err := eventlog.Read()
	if err != nil {
		t.Fatal(err)
	}

	// Validate messages:
	assert.Len(t, records, 1)
	for _, record := range records {
		t.Log(record)
		assert.Equal(t, "The data area passed to a system call is too small.", record.RenderErr)
	}
}

// Test that when an unknown Event ID is found, that a message containing the
// insert strings (the message parameters) is returned.
func TestReadUnknownEventId(t *testing.T) {

	configureLogp()
	log, err := initLog(providerName, sourceName, servicesMsgFile)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := uninstallLog(providerName, sourceName, log)
		if err != nil {
			t.Fatal(err)
		}
	}()

	var eventID uint32 = 1000
	msg := "Test Message"
	err = log.Success(eventID, msg)
	if err != nil {
		t.Fatal(err)
	}

	// Read messages:
	eventlog, err := newEventLogging(map[string]interface{}{"name": providerName})
	if err != nil {
		t.Fatal(err)
	}
	err = eventlog.Open(0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := eventlog.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()
	records, err := eventlog.Read()
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
	log, err := initLog(providerName, sourceName,
		servicesMsgFile+";"+eventCreateMsgFile)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := uninstallLog(providerName, sourceName, log)
		if err != nil {
			t.Fatal(err)
		}
	}()

	var eventID uint32 = 1000
	msg := "Test Message"
	err = log.Success(eventID, msg)
	if err != nil {
		t.Fatal(err)
	}

	// Read messages:
	eventlog, err := newEventLogging(map[string]interface{}{"name": providerName})
	if err != nil {
		t.Fatal(err)
	}
	err = eventlog.Open(0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := eventlog.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()
	records, err := eventlog.Read()
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
	log, err := initLog(providerName, sourceName, servicesMsgFile)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := uninstallLog(providerName, sourceName, log)
		if err != nil {
			t.Fatal(err)
		}
	}()

	// EventID observed by exporting system event log to XML and doing calculation.
	// <EventID Qualifiers="16384">7036</EventID>
	// 1073748860 = 16384 << 16 + 7036
	// https://msdn.microsoft.com/en-us/library/windows/desktop/aa385206(v=vs.85).aspx
	var eventID uint32 = 1073748860
	template := "The %s service entered the %s state."
	msgs := []string{"Windows Update", "running"}
	err = log.Report(elog.Info, eventID, msgs)
	if err != nil {
		t.Fatal(err)
	}

	// Read messages:
	eventlog, err := newEventLogging(map[string]interface{}{"name": providerName})
	if err != nil {
		t.Fatal(err)
	}
	err = eventlog.Open(0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := eventlog.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()
	records, err := eventlog.Read()
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

	el, err := newEventLogging(map[string]interface{}{"name": "nonExistentProvider"})
	if err != nil {
		t.Fatal(err)
	}
	assert.NoError(t, el.Open(0), "Calling Open() on an unknown provider "+
		"should automatically open Application.")
	_, err = el.Read()
	assert.NoError(t, err)
}

// Test event messages that require no parameters.
func TestReadNoParameterMsg(t *testing.T) {

	configureLogp()
	log, err := initLog(providerName, sourceName, netEventMsgFile)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := uninstallLog(providerName, sourceName, log)
		if err != nil {
			t.Fatal(err)
		}
	}()

	var eventID uint32 = 2147489654 // 1<<31 + 6006
	template := "The Event log service was stopped."
	msgs := []string{}
	err = log.Report(elog.Info, eventID, msgs)
	if err != nil {
		t.Fatal(err)
	}

	// Read messages:
	eventlog, err := newEventLogging(map[string]interface{}{"name": providerName})
	if err != nil {
		t.Fatal(err)
	}
	err = eventlog.Open(0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := eventlog.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()
	records, err := eventlog.Read()
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
	log, err := initLog(providerName, sourceName, eventCreateMsgFile)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := uninstallLog(providerName, sourceName, log)
		if err != nil {
			t.Fatal(err)
		}
	}()

	eventlog, err := newEventLogging(map[string]interface{}{"name": providerName})
	if err != nil {
		t.Fatal(err)
	}
	err = eventlog.Open(0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := eventlog.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	log.Info(1, "Message 1")
	log.Info(2, "Message 2")
	lr, err := eventlog.Read()
	assert.NoError(t, err, "Expected 2 messages but received error")
	assert.Len(t, lr, 2, "Expected 2 messages")

	assert.NoError(t, eventlogging.ClearEventLog(eventlogging.Handle(log.Handle), ""))
	lr, err = eventlog.Read()
	assert.NoError(t, err, "Expected 0 messages but received error")
	assert.Len(t, lr, 0, "Expected 0 message")

	log.Info(3, "Message 3")
	lr, err = eventlog.Read()
	assert.NoError(t, err, "Expected 1 message but received error")
	assert.Len(t, lr, 1, "Expected 1 message")
	if len(lr) > 0 {
		assert.Equal(t, uint32(3), lr[0].EventIdentifier.ID)
	}
}

// TODO: Add more test cases:
// - Record number rollover (there may be an issue with this if ++ is used anywhere)
// - Reading from a source name instead of provider name (can't be done according to docs).
// - Persistent read mode shall support specifying a record number (or not specifying a record number).
// -- Invalid record number based on range (should start at first record).
// -- Invalid record number based on range timestamp match check (should start at first record).
// -- Valid record number
// --- Do not replay first record (it was already reported)
// -- First read (no saved state) should return the first record (send first reported record).
// - NewOnly read mode shall seek to end and ignore first.
// - ReadThenExit read mode shall seek to end, read backwards, honor the EOF, then exit.
