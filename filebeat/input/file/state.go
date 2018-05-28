package file

import (
	"os"
	"time"
)

// State is used to communicate the reading state of a file
type State struct {
	Id          string      `json:"-"` // local unique id for comparison and indexing states.
	Source      string      `json:"source"`
	Offset      int64       `json:"offset"`
	Finished    bool        `json:"-"` // harvester state
	Fileinfo    os.FileInfo `json:"-"` // the file info
	FileStateOS StateOS
	Timestamp   time.Time     `json:"timestamp"`
	TTL         time.Duration `json:"ttl"`
}

// NewState creates a new file state
func NewState(fileInfo os.FileInfo, path string) State {
	osState := GetOSState(fileInfo)
	return State{
		Id:          osState.String(),
		Fileinfo:    fileInfo,
		Source:      path,
		Finished:    false,
		FileStateOS: osState,
		Timestamp:   time.Now(),
		TTL:         -1, // By default, state does have an infinite ttl
	}
}

// ID returns a unique id for the state as a string
func (s *State) ID() string {
	return s.Id
}

// IsEmpty returns true if the state is empty
func (s *State) IsEmpty() bool {
	return *s == State{}
}
