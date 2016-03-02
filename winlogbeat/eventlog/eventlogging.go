// +build windows

package eventlog

import (
	"fmt"
	"syscall"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	sys "github.com/elastic/beats/winlogbeat/sys/eventlogging"
)

const (
	// eventLoggingAPIName is the name used to identify the Event Logging API
	// as both an event type and an API.
	eventLoggingAPIName = "eventlogging"
)

// Validate that eventLogging implements the EventLog interface.
var _ EventLog = &eventLogging{}

// eventLogging implements the EventLog interface for reading from the Event
// Logging API.
type eventLogging struct {
	uncServerPath string               // UNC name of remote server.
	name          string               // Name of the log that is opened.
	handle        sys.Handle           // Handle to the event log.
	readBuf       []byte               // Buffer for reading in events.
	formatBuf     []byte               // Buffer for formatting messages.
	handles       *messageFilesCache   // Cached mapping of source name to event message file handles.
	logPrefix     string               // Prefix to add to all log entries.
	eventMetadata common.EventMetadata // Fields and tags to add to each event.

	recordNumber uint32 // First record number to read.
	seek         bool   // Read should use seek.
	ignoreFirst  bool   // Ignore first message returned from a read.
}

// Name returns the name of the event log (i.e. Application, Security, etc.).
func (l eventLogging) Name() string {
	return l.name
}

func (l *eventLogging) Open(recordNumber uint64) error {
	detailf("%s Open(recordNumber=%d) calling OpenEventLog(uncServerPath=%s, "+
		"providerName=%s)", l.logPrefix, recordNumber, l.uncServerPath, l.name)
	handle, err := sys.OpenEventLog(l.uncServerPath, l.name)
	if err != nil {
		return err
	}

	numRecords, err := sys.GetNumberOfEventLogRecords(handle)
	if err != nil {
		return err
	}

	var oldestRecord, newestRecord uint32
	if numRecords > 0 {
		l.recordNumber = uint32(recordNumber)
		l.seek = true
		l.ignoreFirst = true

		oldestRecord, err = sys.GetOldestEventLogRecord(handle)
		if err != nil {
			return err
		}
		newestRecord = oldestRecord + numRecords - 1

		if l.recordNumber < oldestRecord || l.recordNumber > newestRecord {
			l.recordNumber = oldestRecord
			l.ignoreFirst = false
		}
	} else {
		l.recordNumber = 0
		l.seek = false
		l.ignoreFirst = false
	}

	logp.Info("%s contains %d records. Record number range [%d, %d]. Starting "+
		"at %d (ignoringFirst=%t)", l.logPrefix, numRecords, oldestRecord,
		newestRecord, l.recordNumber, l.ignoreFirst)

	l.handle = handle
	return nil
}

func (l *eventLogging) Read() ([]Record, error) {
	flags := sys.EVENTLOG_SEQUENTIAL_READ | sys.EVENTLOG_FORWARDS_READ
	if l.seek {
		flags = sys.EVENTLOG_SEEK_READ | sys.EVENTLOG_FORWARDS_READ
		l.seek = false
	}

	var numBytesRead int
	err := retry(
		func() error {
			l.readBuf = l.readBuf[0:cap(l.readBuf)]
			// TODO: Use number of bytes to grow the buffer size as needed.
			var err error
			numBytesRead, err = sys.ReadEventLog(
				l.handle,
				flags,
				l.recordNumber,
				l.readBuf)
			return err
		},
		l.readRetryErrorHandler)
	if err != nil {
		debugf("%s ReadEventLog returned error %v", l.logPrefix, err)
		return readErrorHandler(err)
	}
	detailf("%s ReadEventLog read %d bytes", l.logPrefix, numBytesRead)

	l.readBuf = l.readBuf[0:numBytesRead]
	events, _, err := sys.RenderEvents(
		l.readBuf[:numBytesRead], 0, l.formatBuf, l.handles.get)
	if err != nil {
		return nil, err
	}
	detailf("%s RenderEvents returned %d events", l.logPrefix, len(events))

	records := make([]Record, 0, len(events))
	for _, e := range events {
		r := Record{
			API:            eventLoggingAPIName,
			EventLogName:   l.name,
			RecordNumber:   uint64(e.RecordID),
			EventID:        e.EventID, // TODO: Subtract out high order bytes (upper 2 bytes)
			Level:          e.Level,
			SourceName:     e.SourceName,
			ComputerName:   e.Computer,
			Category:       e.Category,
			Message:        e.Message,
			MessageInserts: e.MessageInserts,
			MessageErr:     e.MessageErr,
			EventMetadata:  l.eventMetadata,
		}

		if e.TimeGenerated != nil {
			r.TimeGenerated = *e.TimeGenerated
		} else if e.TimeWritten != nil {
			r.TimeGenerated = *e.TimeWritten
		}

		if e.UserSID != nil {
			r.User = &User{
				Identifier: e.UserSID.Identifier,
				Name:       e.UserSID.Name,
				Domain:     e.UserSID.Domain,
				Type:       e.UserSID.Type.String(),
			}
		}

		records = append(records, r)
	}

	if l.ignoreFirst && len(records) > 0 {
		debugf("%s Ignoring first event with record number %d", l.logPrefix,
			records[0].RecordNumber)
		records = records[1:]
		l.ignoreFirst = false
	}

	debugf("%s Read() is returning %d records", l.logPrefix, len(records))
	return records, nil
}

func (l *eventLogging) Close() error {
	debugf("%s Closing handle", l.logPrefix)
	return sys.CloseEventLog(l.handle)
}

// readRetryErrorHandler handles errors returned from the readEventLog function
// by attempting to correct the error through closing and reopening the event
// log.
func (l *eventLogging) readRetryErrorHandler(err error) error {
	if errno, ok := err.(syscall.Errno); ok {
		var reopen bool

		switch errno {
		case sys.ERROR_EVENTLOG_FILE_CHANGED:
			debugf("Re-opening event log because event log file was changed")
			reopen = true
		case sys.ERROR_EVENTLOG_FILE_CORRUPT:
			debugf("Re-opening event log because event log file is corrupt")
			reopen = true
		}

		if reopen {
			l.Close()
			return l.Open(uint64(l.recordNumber))
		}
	}
	return err
}

// readErrorHandler handles errors returned by the readEventLog function.
func readErrorHandler(err error) ([]Record, error) {
	switch err {
	case syscall.ERROR_HANDLE_EOF,
		sys.ERROR_EVENTLOG_FILE_CHANGED,
		sys.ERROR_EVENTLOG_FILE_CORRUPT:
		return []Record{}, nil
	}
	return nil, err
}

// newEventLogging creates and returns a new EventLog for reading event logs
// using the Event Logging API.
func newEventLogging(c Config) (EventLog, error) {
	return &eventLogging{
		uncServerPath: c.RemoteAddress,
		name:          c.Name,
		handles: newMessageFilesCache(c.Name, sys.QueryEventMessageFiles,
			sys.FreeLibrary),
		logPrefix:     fmt.Sprintf("EventLogging[%s]", c.Name),
		readBuf:       make([]byte, 0, sys.MaxEventBufferSize),
		formatBuf:     make([]byte, sys.MaxFormatMessageBufferSize),
		eventMetadata: c.EventMetadata,
	}, nil
}

func init() {
	// Register eventlogging API if it is available.
	available, _ := sys.IsAvailable()
	if available {
		Register(eventLoggingAPIName, 1, newEventLogging, nil)
	}
}
