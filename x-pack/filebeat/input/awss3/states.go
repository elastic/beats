// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"sync"

	"github.com/elastic/beats/v7/libbeat/logp"
)

// States handles list of s3 object state. One must use NewStates to instantiate a
// file states registry. Using the zero-value is not safe.
type States struct {
	sync.RWMutex

	// states store
	states []State

	// idx maps state IDs to state indexes for fast lookup and modifications.
	idx map[string]int
}

// NewStates generates a new states registry.
func NewStates() *States {
	return &States{
		states: nil,
		idx:    map[string]int{},
	}
}

// Update updates a state. If previous state didn't exist, new one is created
func (s *States) Update(newState State) {
	s.Lock()
	defer s.Unlock()

	id := newState.Bucket + newState.Key
	index := s.findPrevious(id)

	if index >= 0 {
		s.states[index] = newState
	} else {
		// No existing state found, add new one
		s.idx[id] = len(s.states)
		s.states = append(s.states, newState)
		logp.Debug("input", "New state added for %s", newState.Key)
	}
}

// FindPrevious lookups a registered state, that matching the new state.
// Returns a zero-state if no match is found.
func (s *States) FindPrevious(newState State) State {
	s.RLock()
	defer s.RUnlock()
	id := newState.Bucket + newState.Key
	i := s.findPrevious(id)
	if i < 0 {
		return State{}
	}
	return s.states[i]
}

func (s *States) IsNew(state State) bool {
	s.RLock()
	defer s.RUnlock()
	id := state.Bucket + state.Key
	i := s.findPrevious(id)

	if i < 0 {
		return true
	}

	return !s.states[i].IsEqual(&state)
}

// findPrevious returns the previous state for the file.
// In case no previous state exists, index -1 is returned
func (s *States) findPrevious(id string) int {
	if i, exists := s.idx[id]; exists {
		return i
	}
	return -1
}
