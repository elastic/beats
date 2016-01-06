package eventlog

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// Debug logging functions for this package.
var (
	debugf  = logp.MakeDebug("eventlog")
	detailf = logp.MakeDebug("eventlog_detail")
)

// EventLog is an interface to a Windows Event Log.
type EventLog interface {
	// Open the event log. recordNumber is the last successfully read event log
	// record number. Read will resume from recordNumber + 1. To start reading
	// from the first event specify a recordNumber of 0.
	Open(recordNumber uint64) error

	// Read records from the event log.
	Read() ([]Record, error)

	// Close the event log. It should not be re-opened after closing.
	Close() error

	// Name returns the event log's name.
	Name() string
}

// Record represents a single event from the log.
type Record struct {
	API string // The event log API type used to read the record.

	EventLogName  string    // The name of the event log from which this record was read.
	SourceName    string    // The source of the event log record (the application or service that logged the record).
	ComputerName  string    // The name of the computer that generated the record.
	RecordNumber  uint64    // The record number of the event log record.
	EventID       uint32    // The event identifier. The value is specific to the source of the event.
	Level         string    // The level or severity of the event.
	Category      string    // The category for this event. The meaning of this value depends on the event source.
	TimeGenerated time.Time // The timestamp when the record was generated.
	User          *User     // The user that logged the record.

	Message        string   // The message from the event log.
	MessageInserts []string // The raw message data logged by an application.
	MessageErr     error    // The error that occurred while reading and formatting the message from the event log.
}

// String returns a string representation of Record.
func (r Record) String() string {
	return fmt.Sprintf("Record API[%s] EventLogName[%s] SourceName[%s] "+
		"ComputerName[%s] RecordNumber[%d] EventID[%d] Level[%s] "+
		"Category[%s] TimeGenerated[%s] User[%s] "+
		"Message[%s] MessageInserts[%s] MessageErr[%s]", r.API,
		r.EventLogName, r.SourceName, r.ComputerName, r.RecordNumber,
		r.EventID, r.Level, r.Category, r.TimeGenerated, r.User,
		r.Message, strings.Join(r.MessageInserts, ", "), r.MessageErr)
}

// ToMapStr returns a new MapStr containing the data from this Record.
func (r Record) ToMapStr() common.MapStr {
	m := common.MapStr{
		"@timestamp":   common.Time(r.TimeGenerated),
		"eventLogName": r.EventLogName,
		"sourceName":   r.SourceName,
		"computerName": r.ComputerName,
		// Use a string to represent this uint64 data because its value can
		// be outside the range represented by a Java long.
		"recordNumber": strconv.FormatUint(r.RecordNumber, 10),
		"eventID":      r.EventID,
		"level":        r.Level,
		"type":         r.API,
	}

	if r.Message != "" {
		m["message"] = r.Message
	} else {
		if len(r.MessageInserts) > 0 {
			m["messageInserts"] = r.MessageInserts
		}

		if r.MessageErr != nil {
			m["messageError"] = r.MessageErr.Error()
		}
	}

	if r.Category != "" {
		m["category"] = r.Category
	}

	if r.User != nil {
		user := common.MapStr{
			"identifier": r.User.Identifier,
		}
		m["user"] = user

		// Optional fields.
		if r.User.Name != "" {
			user["name"] = r.User.Name
		}
		if r.User.Domain != "" {
			user["domain"] = r.User.Domain
		}
		if r.User.Type != "" {
			user["type"] = r.User.Type
		}
	}

	return m
}

// User contains information about a Windows account.
type User struct {
	Identifier string // Unique identifier used by Windows to ID the account.
	Name       string // User name
	Domain     string // Domain that the user is a member of
	Type       string // Type of account (e.g. User, Computer, Service)
}

// String returns a string representation of Record.
func (u User) String() string {
	return fmt.Sprintf("User Name[%s] Domain[%s] Type[%s]",
		u.Name, u.Domain, u.Type)
}
