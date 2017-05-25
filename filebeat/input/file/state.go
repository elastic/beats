package file

import (
	"os"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/logp"
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
	FileStateOS StateOS
}

// NewState creates a new file state
func NewState(fileInfo os.FileInfo, path string, t string) State {
	return State{
		Fileinfo:    fileInfo,
		Source:      path,
		Finished:    false,
		FileStateOS: GetOSState(fileInfo),
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

// States handles list of FileState
type States struct {
	states []State
	sync.RWMutex
}

func NewStates() *States {
	return &States{
		states: []State{},
	}
}

// Update updates a state. If previous state didn't exist, new one is created
func (s *States) Update(newState State) {
	s.Lock()
	defer s.Unlock()

	index, _ := s.findPrevious(newState)
	newState.Timestamp = time.Now()

	if index >= 0 {
		s.states[index] = newState
	} else {
		// No existing state found, add new one
		s.states = append(s.states, newState)
		logp.Debug("prospector", "New state added for %s", newState.Source)
	}
}

func (s *States) FindPrevious(newState State) State {
	s.RLock()
	defer s.RUnlock()
	_, state := s.findPrevious(newState)
	return state
}

// findPreviousState returns the previous state fo the file
// In case no previous state exists, index -1 is returned
func (s *States) findPrevious(newState State) (int, State) {

	// TODO: This could be made potentially more performance by using an index (harvester id) and only use iteration as fall back
	for index, oldState := range s.states {
		// This is using the FileStateOS for comparison as FileInfo identifiers can only be fetched for existing files
		if oldState.IsEqual(&newState) {
			return index, oldState
		}
	}

	return -1, State{}
}

// Cleanup cleans up the state array. All states which are older then `older` are removed
// The number of states that were cleaned up is returned
func (s *States) Cleanup() int {

	s.Lock()
	defer s.Unlock()

	statesBefore := len(s.states)

	currentTime := time.Now()
	states := s.states[:0]

	for _, state := range s.states {

		expired := (state.TTL > 0 && currentTime.Sub(state.Timestamp) > state.TTL)

		if state.TTL == 0 || expired {
			if state.Finished {
				logp.Debug("state", "State removed for %v because of older: %v", state.Source, state.TTL)
				continue // drop state
			} else {
				logp.Err("State for %s should have been dropped, but couldn't as state is not finished.", state.Source)
			}
		}

		states = append(states, state) // in-place copy old state
	}
	s.states = states

	return statesBefore - len(s.states)
}

// Count returns number of states
func (s *States) Count() int {
	s.RLock()
	defer s.RUnlock()

	return len(s.states)
}

// Returns a copy of the file states
func (s *States) GetStates() []State {
	s.RLock()
	defer s.RUnlock()

	newStates := make([]State, len(s.states))
	copy(newStates, s.states)

	return newStates
}

// SetStates overwrites all internal states with the given states array
func (s *States) SetStates(states []State) {
	s.Lock()
	defer s.Unlock()
	s.states = states
}

// Copy create a new copy of the states object
func (s *States) Copy() *States {
	states := NewStates()
	states.states = s.GetStates()
	return states
}
