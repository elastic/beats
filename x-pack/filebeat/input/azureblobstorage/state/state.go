// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package state

import (
	"sync"
	"time"
)

// State contains the the current state of the operation
type State struct {
	// Mutex lock to help in concurrent R/W
	mu sync.Mutex
	cp *Checkpoint
}

type Checkpoint struct {
	// marker contains the last known position in the blob pager which was fetched
	Marker *string
	// name of the blob
	Name string
	// timestamp to denote when the blob was last modified
	LastModifiedOn *time.Time
}

func NewState() *State {
	return &State{
		cp: &Checkpoint{},
	}
}

// save functions , saves/updates the current state
func (s *State) Save(name string, marker *string, lastModifiedOn *time.Time) {
	s.mu.Lock()
	s.cp.Name = name
	s.cp.Marker = marker
	s.cp.LastModifiedOn = lastModifiedOn
	s.mu.Unlock()
}

func (s *State) SetCheckpoint(chkpt *Checkpoint) {
	s.cp = chkpt
}

func (s *State) Checkpoint() *Checkpoint {
	return s.cp
}
