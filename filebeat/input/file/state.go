package file

import (
	"os"
	"time"

	"github.com/elastic/beats/libbeat/common/file"
)

// State is used to communicate the reading state of a file
type State struct {
	Id          string        `json:"-"` // local unique id to make comparison more efficient
	Finished    bool          `json:"-"` // harvester state
	Fileinfo    os.FileInfo   `json:"-"` // the file info
	Source      string        `json:"source"`
	Offset      int64         `json:"offset"`
	Timestamp   time.Time     `json:"timestamp"`
	TTL         time.Duration `json:"ttl"`
	Type        string        `json:"type"`
	FileStateOS file.StateOS
}

// NewState creates a new file state
func NewState(fileInfo os.FileInfo, path string, t string) State {
	return State{
		Fileinfo:    fileInfo,
		Source:      path,
		Finished:    false,
		FileStateOS: file.GetOSState(fileInfo),
		Timestamp:   time.Now(),
		TTL:         -1, // By default, state does have an infinite ttl
		Type:        t,
	}
}

// ID returns a unique id for the state as a string
func (s *State) ID() string {
	// Generate id on first request. This is needed as id is not set when converting back from json
	if s.Id == "" {
		s.Id = s.FileStateOS.String()
	}
	return s.Id
}

// IsEqual compares the state to an other state supporing stringer based on the unique string
func (s *State) IsEqual(c *State) bool {
	return s.ID() == c.ID()
}

// IsEmpty returns true if the state is empty
func (s *State) IsEmpty() bool {
	return *s == State{}
}
