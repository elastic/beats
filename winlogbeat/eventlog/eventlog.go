package eventlog

import (
	"fmt"
	"time"
	"unicode/utf16"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// Debug logging functions for this package.
var (
	debugf  = logp.MakeDebug("eventlog")
	detailf = logp.MakeDebug("eventlog_detail")
)

// EventLoggingAPI provides an interface to the Event Logging API introduced in
// Windows 2000 (not the Windows Event Log API that was introduced in Windows
// Vista).
type EventLoggingAPI interface {
	// Open the event log. recordNumber is the last successfully read event log
	// record number. Read will resume from recordNumber + 1. To start reading
	// from the first event specify a recordNumber of 0.
	Open(recordNumber uint64) error

	// Read records from the event log.
	Read() ([]LogRecord, error)

	// Close the event log. It should not be re-opened after closing.
	Close() error

	// Name returns the event log's name. If the name is unknown to the host
	// system, then the Application event log is opened.
	Name() string
}

type eventLog struct {
	uncServerPath string       // UNC name of remote server.
	name          string       // Name of the log that is opened.
	handle        Handle       // Handle to the event log.
	readBuf       []byte       // Re-usable buffer for reading in events.
	formatBuf     []byte       // Re-usable buffer for formatting messages.
	handles       *handleCache // Cached mapping of source name to event message file handles.
	logPrefix     string       // Prefix to add to all log entries.

	recordNumber uint32 // First record number to read.
	seek         bool   // Read should use seek.
	ignoreFirst  bool   // Ignore first message returned from a read.
}

// Name returns the name of the event log (i.e. Application, Security, etc.).
func (el eventLog) Name() string {
	return el.name
}

func newEventLog(uncServerPath, eventLogName string) *eventLog {
	el := &eventLog{
		uncServerPath: uncServerPath,
		name:          eventLogName,
		handles: newHandleCache(eventLogName, queryEventMessageFiles,
			freeLibrary),
		logPrefix: fmt.Sprintf("EventLog[%s]", eventLogName),
	}
	return el
}

func NewEventLoggingAPI(eventLogName string) EventLoggingAPI {
	return newEventLog("", eventLogName)
}

func NewRemoteEventLoggingAPI(uncServerPath, eventLogName string) EventLoggingAPI {
	return newEventLog(uncServerPath, eventLogName)
}

// LogRecord represents a single record from an event log.
type LogRecord struct {
	EventLogName  string
	SourceName    string
	ComputerName  string
	RecordNumber  uint64
	EventID       uint32
	EventType     string
	EventCategory string
	TimeGenerated time.Time
	UserSID       *SID
	Message       string
}

// String returns string representation of LogRecord.
func (lr LogRecord) String() string {
	return fmt.Sprintf("LogRecord EventLogName[%s] SourceName[%s] "+
		"ComputerName[%s] RecordNumber[%d] EventID[%d] EventType[%s] "+
		"EventCategory[%s] TimeGenerated[%s] UserSID[%s] "+
		"Message[%s]", lr.EventLogName, lr.SourceName, lr.ComputerName,
		lr.RecordNumber, lr.EventID, lr.EventType, lr.EventCategory,
		lr.TimeGenerated, lr.UserSID, lr.Message)
}

func (lr LogRecord) ToMapStr() common.MapStr {
	m := common.MapStr{
		"eventLogName": lr.EventLogName,
		"sourceName":   lr.SourceName,
		"computerName": lr.ComputerName,
		"recordNumber": lr.RecordNumber,
		"eventID":      lr.EventID,
		"eventType":    lr.EventType,
		"message":      lr.Message,
		"@timestamp":   common.Time(lr.TimeGenerated),
		"type":         "eventlog",
	}

	if lr.EventCategory != "" {
		m["eventCategory"] = lr.EventCategory
	}

	if lr.UserSID != nil {
		m["userSID"] = common.MapStr{
			"name":   lr.UserSID.Name,
			"domain": lr.UserSID.Domain,
			"type":   lr.UserSID.SIDType.String(),
		}
	}

	return m
}

// SID represents the Windows Security Identifier for an account.
type SID struct {
	Name    string
	Domain  string
	SIDType SIDType
}

// String returns string representation of SID.
func (a SID) String() string {
	return fmt.Sprintf("SID Name[%s] Domain[%s] SIDType[%s]",
		a.Name, a.Domain, a.SIDType)
}

// EventType identifies the five types of events that can be logged by
// applications.
type EventType uint16

// EventType values.
const (
	// Do not reorder.
	EVENTLOG_SUCCESS    EventType = 0
	EVENTLOG_ERROR_TYPE           = 1 << (iota - 1)
	EVENTLOG_WARNING_TYPE
	EVENTLOG_INFORMATION_TYPE
	EVENTLOG_AUDIT_SUCCESS
	EVENTLOG_AUDIT_FAILURE
)

// Mapping of event types to their string representations.
var eventTypeToString = map[EventType]string{
	EVENTLOG_SUCCESS:          "Success",
	EVENTLOG_ERROR_TYPE:       "Error",
	EVENTLOG_AUDIT_FAILURE:    "Audit Failure",
	EVENTLOG_AUDIT_SUCCESS:    "Audit Success",
	EVENTLOG_INFORMATION_TYPE: "Information",
	EVENTLOG_WARNING_TYPE:     "Warning",
}

// String returns string representation of EventType.
func (et EventType) String() string {
	return eventTypeToString[et]
}

// SIDType identifies the type of a security identifier (SID).
type SIDType uint32

// SIDType values.
const (
	// Do not reorder.
	SidTypeUser SIDType = 1 + iota
	SidTypeGroup
	SidTypeDomain
	SidTypeAlias
	SidTypeWellKnownGroup
	SidTypeDeletedAccount
	SidTypeInvalid
	SidTypeUnknown
	SidTypeComputer
	SidTypeLabel
)

// Mapping of SID types to their string representations.
var sidTypeToString = map[SIDType]string{
	SidTypeUser:           "User",
	SidTypeGroup:          "Group",
	SidTypeDomain:         "Domain",
	SidTypeAlias:          "Alias",
	SidTypeWellKnownGroup: "Well Known Group",
	SidTypeDeletedAccount: "Deleted Account",
	SidTypeInvalid:        "Invalid",
	SidTypeUnknown:        "Unknown",
	SidTypeComputer:       "Unknown",
	SidTypeLabel:          "Label",
}

// String returns string representation of SIDType.
func (st SIDType) String() string {
	return sidTypeToString[st]
}

// UTF16BytesToString returns the Unicode code point sequence represented
// by the UTF-16 buffer b.
func UTF16BytesToString(b []byte) (string, int, error) {
	if len(b)%2 != 0 {
		return "", 0, fmt.Errorf("Must have even length byte slice")
	}

	offset := len(b)/2 + 2
	s := make([]uint16, len(b)/2)
	for i := range s {
		s[i] = uint16(b[i*2]) + uint16(b[(i*2)+1])<<8

		if s[i] == 0 {
			s = s[0:i]
			offset = i*2 + 2
			break
		}
	}

	return string(utf16.Decode(s)), offset, nil
}
