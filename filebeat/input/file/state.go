package file

import (
	"os"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/common/file"
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

// States handles list of FileState
type States struct {
	states map[string]State
	sync.RWMutex
}

func NewStates() *States {
	return &States{
		states: map[string]State{},
	}
}

// Update updates a state. If previous state didn't exist, new one is created
func (s *States) Update(newState State) {
	s.UpdateWithTimestamp(newState, time.Now())
}

// UpdateWithTimestamp updates a state, using the passed timestamp. If the
// previous state didn't exist, a new entry is created.
func (s *States) UpdateWithTimestamp(newState State, ts time.Time) {
	s.Lock()
	defer s.Unlock()

	// ensure ID is set, so it won't be generated in find and on insert
	id := newState.ID()

	if logp.IsDebug("input") {
		_, exists := s.findPrevious(newState)
		if !exists {
			logp.Debug("input", "New state added for %s", newState.Source)
		}
	}

	newState.Timestamp = ts
	s.states[id] = newState
}

func (s *States) FindPrevious(newState State) State {
	s.RLock()
	defer s.RUnlock()
	state, _ := s.findPrevious(newState)
	return state
}

// findPreviousState returns the previous state fo the file
// In case no previous state exists, index -1 is returned
func (s *States) findPrevious(newState State) (State, bool) {
	state, exists := s.states[newState.ID()]
	return state, exists
}

// Cleanup cleans up the state array. All states which are older then `older` are removed
// The number of states that were cleaned up is returned
func (s *States) Cleanup() (int, int) {
	s.Lock()
	defer s.Unlock()

	statesBefore := len(s.states)
	numCanExpire := 0

	currentTime := time.Now()
	for id, state := range s.states {
		canExpire := state.TTL >= 0
		if canExpire {
			numCanExpire++
		}

		expired := (state.TTL > 0 && currentTime.Sub(state.Timestamp) > state.TTL)
		if state.TTL == 0 || expired {
			if !state.Finished {
				logp.Err("State for %s should have been dropped, but couldn't as state is not finished.", state.Source)
			} else {
				logp.Debug("state", "State removed for %v because of older: %v", state.Source, state.TTL)

				delete(s.states, id)
				numCanExpire-- // event removed -> reduce count of pending events again
			}
		}
	}

	return statesBefore - len(s.states), numCanExpire
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

	newStates, i := make([]State, len(s.states)), 0
	for _, state := range s.states {
		newStates[i], i = state, i+1
	}

	return newStates
}

// GetIndexedStates returns a copy of the states, indexed by ID.
func (s *States) GetIndexedStates() map[string]State {
	m := make(map[string]State, len(s.states))
	for k, v := range s.states {
		m[k] = v
	}
	return m
}

// SetStates overwrites all internal states with the given states array
func (s *States) SetStates(states []State) {
	s.Lock()
	defer s.Unlock()
	newStates := make(map[string]State, len(states))
	for _, state := range states {
		newStates[state.ID()] = state
	}
	s.states = newStates
}

// Copy create a new copy of the states object
func (s *States) Copy() *States {
	states := NewStates()
	states.states = s.GetIndexedStates()
	return states
}
