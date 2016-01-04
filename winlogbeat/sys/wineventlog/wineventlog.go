package wineventlog

import (
	"time"

	"github.com/elastic/beats/winlogbeat/sys/eventlogging"
)

// Event holds the data from the a log record.
type Event struct {
	// System context properties.
	ProviderName      string            `json:",omitempty"`
	ProviderGUID      string            `json:",omitempty"`
	EventID           uint16            `json:",omitempty"`
	Qualifiers        uint16            `json:",omitempty"`
	TimeCreated       *time.Time        `json:",omitempty"`
	RecordID          uint64            `json:",omitempty"`
	ActivityID        string            `json:",omitempty"`
	RelatedActivityID string            `json:",omitempty"`
	ProcessID         uint32            `json:",omitempty"`
	ThreadID          uint32            `json:",omitempty"`
	Channel           string            `json:",omitempty"`
	Computer          string            `json:",omitempty"`
	UserSID           *eventlogging.SID `json:",omitempty"`
	Version           uint8             `json:",omitempty"`

	// String properties that require a call to FormatMessage.

	Message    string `json:",omitempty"`
	MessageErr error

	Level    string `json:",omitempty"`
	LevelErr error

	Task    string `json:",omitempty"`
	TaskErr error

	Opcode    string `json:",omitempty"`
	OpcodeErr error

	Keywords      []string `json:",omitempty"`
	KeywordsError error
}
