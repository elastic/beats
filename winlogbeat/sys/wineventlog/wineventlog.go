package wineventlog

import (
	"time"

	"github.com/elastic/beats/winlogbeat/eventlog"
)

// EvtHandle is a handle to the event log.
type EvtHandle uintptr

// Event holds the data from the a log record.
type Event struct {
	// System context properties.
	ProviderName      string        `json:",omitempty"`
	ProviderGUID      string        `json:",omitempty"`
	EventID           uint16        `json:",omitempty"`
	Qualifiers        uint16        `json:",omitempty"`
	TimeCreated       *time.Time    `json:",omitempty"`
	RecordID          uint64        `json:",omitempty"`
	ActivityID        string        `json:",omitempty"`
	RelatedActivityID string        `json:",omitempty"`
	ProcessID         uint32        `json:",omitempty"`
	ThreadID          uint32        `json:",omitempty"`
	Channel           string        `json:",omitempty"`
	Computer          string        `json:",omitempty"`
	UserSID           *eventlog.SID `json:",omitempty"`
	Version           uint8         `json:",omitempty"`

	// String properties
	Message  string   `json:",omitempty"`
	Level    string   `json:",omitempty"`
	Task     string   `json:",omitempty"`
	Opcode   string   `json:",omitempty"`
	Keywords []string `json:",omitempty"`
}
