// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package state

import (
	"strings"
	"sync"
	"time"
)

// State contains the the current state of the operation
type State struct {
	// Mutex lock to help in concurrent R/W
	mu sync.Mutex
	cp *Checkpoint
}

// Azure sdks do not return results based on timestamps , but only based on alphabetical order
// This forces us to maintain 2 different vars in addition to the page marker to calculate the
// exact checkpoint based on various scenarios
type Checkpoint struct {
	// marker contains the last known position in the blob pager which was fetched
	Marker *string
	// name of the latest blob in alphabetical order
	BlobName string
	// timestamp to denote which is the latest blob
	LatestEntryTime *time.Time
}

func NewState() *State {
	return &State{
		cp: &Checkpoint{},
	}
}

// Save , saves/updates the current state for cursor checkpoint
func (s *State) Save(name string, marker *string, lastModifiedOn *time.Time) {
	s.mu.Lock()
	if len(s.cp.BlobName) == 0 {
		s.cp.BlobName = name
	} else if strings.ToLower(name) > strings.ToLower(s.cp.BlobName) {
		s.cp.BlobName = name
	}
	s.cp.Marker = marker
	if s.cp.LatestEntryTime == nil {
		s.cp.LatestEntryTime = lastModifiedOn
	} else if lastModifiedOn.After(*s.cp.LatestEntryTime) {
		s.cp.LatestEntryTime = lastModifiedOn
	}
	s.mu.Unlock()
}

// SetCheckpoint , sets checkpoint from source to current state instance
func (s *State) SetCheckpoint(chkpt *Checkpoint) {
	s.cp = chkpt
}

// Checkpoint , returns the current state checkpoint
func (s *State) Checkpoint() *Checkpoint {
	return s.cp
}
