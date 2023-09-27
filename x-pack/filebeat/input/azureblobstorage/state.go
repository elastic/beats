// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azureblobstorage

import (
	"strings"
	"sync"
	"time"
)

// state contains the the current state of the operation
type state struct {
	// Mutex lock to help in concurrent R/W
	mu sync.Mutex
	cp *Checkpoint
}

// Azure sdks do not return results based on timestamps, but only based on alphabetical order
// This forces us to maintain 2 different variables to calculate the
// exact checkpoint based on various scenarios
type Checkpoint struct {
	// name of the latest blob in alphabetical order
	BlobName string
	// timestamp to denote which is the latest blob
	LatestEntryTime time.Time
	// a mapping from object name to whether the object is having an array type as it's root.
	IsRootArray map[string]bool
	//  a mapping from object name to an array index that contains the last processed offset for that object.
	// if isRootArray == true for object, then PartiallyProcessed will treat offset as an array index
	PartiallyProcessed map[string]int64
}

func newState() *state {
	return &state{
		cp: &Checkpoint{
			PartiallyProcessed: make(map[string]int64),
			IsRootArray:        make(map[string]bool),
		},
	}
}

// saveForTx updates and returns the current state checkpoint, locks the state
// and returns an unlock function done(). The caller must call done when
// s and cp are no longer needed in a locked state. done may not be called
// more than once.
func (s *state) saveForTx(name string, lastModifiedOn time.Time) (cp *Checkpoint, done func()) {
	s.mu.Lock()
	delete(s.cp.PartiallyProcessed, name)
	delete(s.cp.IsRootArray, name)
	if len(s.cp.BlobName) == 0 {
		s.cp.BlobName = name
	} else if strings.ToLower(name) > strings.ToLower(s.cp.BlobName) {
		s.cp.BlobName = name
	}
	if s.cp.LatestEntryTime.IsZero() {
		s.cp.LatestEntryTime = lastModifiedOn
	} else if lastModifiedOn.After(s.cp.LatestEntryTime) {
		s.cp.LatestEntryTime = lastModifiedOn
	}
	return s.cp, func() { s.mu.Unlock() }
}

// savePartialForTx partially updates and returns the current state checkpoint, locks the state
// and returns an unlock function done(). The caller must call done when
// s and cp are no longer needed in a locked state. done may not be called
// more than once.
func (s *state) savePartialForTx(name string, offset int64) (cp *Checkpoint, done func()) {
	s.mu.Lock()
	s.cp.PartiallyProcessed[name] = offset
	return s.cp, func() { s.mu.Unlock() }
}

// setRootArray, sets boolean true for objects that have their roots defined as an array type, locks the state
// and returns an unlock function done(). The caller must call done when s is no longer needed in a locked state.
func (s *state) setRootArray(name string) (done func()) {
	s.mu.Lock()
	s.cp.IsRootArray[name] = true
	return func() { s.mu.Unlock() }
}

// isRootArray, returns true if the object has it's root defined as an array type and has been partially processed, it also locks the state
// and returns an unlock function done(). The caller must call done when 's' and 'result' are no longer needed in a locked state.
func (s *state) isRootArray(name string) (result bool, done func()) {
	s.mu.Lock()
	result = s.cp.IsRootArray[name]
	return result, func() { s.mu.Unlock() }
}

// setCheckpoint sets checkpoint from source to current state instance
func (s *state) setCheckpoint(chkpt *Checkpoint) {
	if chkpt.PartiallyProcessed == nil {
		chkpt.PartiallyProcessed = make(map[string]int64)
	}
	if chkpt.IsRootArray == nil {
		chkpt.IsRootArray = make(map[string]bool)
	}
	s.cp = chkpt
}

// checkpoint returns the current state checkpoint
func (s *state) checkpoint() *Checkpoint {
	return s.cp
}
