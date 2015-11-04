// Package provides access to the Event Logging API used in Windows 2000,
// Windows XP, and Windows Server 2003. This is distinct from the Windows
// Event Log API that was introduced in Windows Vista and  Windows 2008.
//
// TODO: Provide methods to access the newer Windows Event Log API.
package eventlog

import (
	"fmt"
	"time"

	"github.com/elastic/libbeat/common"
)

// Interface to the Event Logging API introduced in Windows 2000 (not the
// Windows Event Log API that was introduced in Windows Vista).
type EventLoggingAPI interface {
	// Open the event log. recordNumber is the last successfully read event log
	// record number. Read will resume from recordNumber + 1. To start reading
	// from the first event specify a recordNumber of 0.
	Open(recordNumber uint32) error

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
	recordNumber  uint32       // Last successfully read record number.
	handle        Handle       // Handle to the event log.
	readBuf       []byte       // Re-usable buffer for reading in events.
	formatBuf     []byte       // Re-usable buffer for formatting messages.
	handles       *handleCache // Cached mapping of source name to event message file handles.
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
	RecordNumber  uint32
	EventId       uint32
	EventType     EventType
	EventCategory string
	TimeGenerated time.Time
	TimeWritten   time.Time
	UserSid       *SID
	Message       string
}

// String returns string representation of LogRecord.
func (lr LogRecord) String() string {
	return fmt.Sprintf("LogRecord EventLogName[%s] SourceName[%s] "+
		"ComputerName[%s] RecordNumber[%d] EventId[%d] EventType[%s] "+
		"EventCategory[%s] TimeGenerated[%s] TimeWritten[%s] UserSid[%s] "+
		"Message[%s]", lr.EventLogName, lr.SourceName, lr.ComputerName,
		lr.RecordNumber, lr.EventId, lr.EventType, lr.EventCategory,
		lr.TimeGenerated, lr.TimeWritten, lr.UserSid, lr.Message)
}

func (lr LogRecord) ToMapStr() common.MapStr {
	m := common.MapStr{
		"eventLogName":  lr.EventLogName,
		"sourceName":    lr.SourceName,
		"computerName":  lr.ComputerName,
		"recordNumber":  lr.RecordNumber,
		"eventId":       lr.EventId,
		"eventType":     lr.EventType.String(),
		"eventCategory": lr.EventCategory,
		"message":       lr.Message,
		"@timestamp":    common.Time(lr.TimeGenerated),
		"type":          "eventlog",
	}

	if lr.UserSid != nil {
		m["userSid"] = common.MapStr{
			"name":   lr.UserSid.Name,
			"domain": lr.UserSid.Domain,
			"type":   lr.UserSid.SIDType.String(),
		}
	}

	return m
}

// Security Identifier for an account.
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
type EventType uint8

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
