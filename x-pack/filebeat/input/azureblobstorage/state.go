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
	// map to contain offset data
	PartiallyProcessed map[string]int64
}

func newState() *state {
	return &state{
		cp: &Checkpoint{
			PartiallyProcessed: make(map[string]int64),
		},
	}
}

// Save, saves/updates the current state for cursor checkpoint
func (s *state) save(name string, lastModifiedOn time.Time) {
	s.mu.Lock()
	delete(s.cp.PartiallyProcessed, name)
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
	s.mu.Unlock()
}

// savePartial, partially saves/updates the current state for cursor checkpoint
func (s *state) savePartial(name string, offset int64, lastModifiedOn *time.Time) {
	s.mu.Lock()
	s.cp.PartiallyProcessed[name] = offset
	s.mu.Unlock()
}

// setCheckpoint, sets checkpoint from source to current state instance
func (s *state) setCheckpoint(chkpt *Checkpoint) {
	if chkpt.PartiallyProcessed == nil {
		chkpt.PartiallyProcessed = make(map[string]int64)
	}
	s.cp = chkpt
}

// checkpoint, returns the current state checkpoint
func (s *state) checkpoint() *Checkpoint {
	return s.cp
}
