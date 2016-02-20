package eventlogging

import (
	"time"
)

// Event represents a Windows Event within the Windows Event Log after it has
// been converted from bytes to a structure.
type Event struct {
	RecordID      uint32     `json:",omitempty"`
	TimeGenerated *time.Time `json:",omitempty"`
	TimeWritten   *time.Time `json:",omitempty"`
	EventID       uint32     `json:",omitempty"`
	Level         string     `json:",omitempty"`
	SourceName    string     `json:",omitempty"`
	Computer      string     `json:",omitempty"`

	UserSID    *SID `json:",omitempty"`
	UserSIDErr error

	// Strings that must be resolved by DLL lookups.
	Message  string `json:",omitempty"`
	Category string `json:",omitempty"`

	MessageInserts []string // Strings inserted into a message template to
	// create Message.
	MessageErr error // Possible error that occurred while formatting Message.
}
