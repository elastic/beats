// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package state

import (
	"strings"
	"sync"
	"time"
)

const (
	maxFailedJobRetries int = 3
)

// State contains the the current state of the operation
type State struct {
	// Mutex lock to help in concurrent R/W
	mu sync.Mutex
	cp *Checkpoint
}

// Gcs sdks do not return results based on timestamps , but only based on lexicographic order
// This forces us to maintain 2 different variables to calculate the exact checkpoint based on various scenarios
type Checkpoint struct {
	// name of the latest blob in alphabetical order
	ObjectName string
	// timestamp to denote which is the latest blob
	LatestEntryTime *time.Time
	// list of failed jobs due to unexpected errors/download errors
	FailedJobs map[string]interface{}
}

func NewState() *State {
	return &State{
		cp: &Checkpoint{
			FailedJobs: make(map[string]interface{}),
		},
	}
}

// Save, saves/updates the current state for cursor checkpoint
func (s *State) Save(name string, lastModifiedOn *time.Time) {
	s.mu.Lock()
	if s.cp.FailedJobs[name] == nil {
		if len(s.cp.ObjectName) == 0 {
			s.cp.ObjectName = name
		} else if strings.ToLower(name) > strings.ToLower(s.cp.ObjectName) {
			s.cp.ObjectName = name
		}

		if s.cp.LatestEntryTime == nil {
			s.cp.LatestEntryTime = lastModifiedOn
		} else if lastModifiedOn.After(*s.cp.LatestEntryTime) {
			s.cp.LatestEntryTime = lastModifiedOn
		}
	} else {
		// clear entry if this is a failed job
		delete(s.cp.FailedJobs, name)
	}
	s.mu.Unlock()
}

// UpdateFailedJobs, adds a job name to a failedJobs map, which helps
// in keeping track of failed jobs during edge cases when the state might
// move ahead in timestamp & objectName due to successful operations from other workers.
// A failed job will be re-tried a maximum of 3 times after which the
// entry is removed from the map
func (s *State) UpdateFailedJobs(jobName string) {
	s.mu.Lock()
	if s.cp.FailedJobs[jobName] == nil {
		s.cp.FailedJobs[jobName] = 0
	} else {
		count := s.cp.FailedJobs[jobName].(int)
		count++
		if count > maxFailedJobRetries {
			delete(s.cp.FailedJobs, jobName)
		} else {
			s.cp.FailedJobs[jobName] = count
		}
	}
	s.mu.Unlock()
}

// SetCheckpoint, sets checkpoint from source to current state instance
func (s *State) SetCheckpoint(chkpt *Checkpoint) {
	s.cp = chkpt
}

// Checkpoint, returns the current state checkpoint
func (s *State) Checkpoint() *Checkpoint {
	return s.cp
}
