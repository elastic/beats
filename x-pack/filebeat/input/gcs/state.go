// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcs

import (
	"strings"
	"sync"
	"time"
)

const (
	maxFailedJobRetries int = 3
)

// state contains the the current state of the operation
type state struct {
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
	LatestEntryTime time.Time
	// list of failed jobs due to unexpected errors/download errors
	FailedJobs map[string]int
	// a mapping from object name to whether the object is having an array type as it's root.
	IsRootArray map[string]bool
	//  a mapping from object name to an array index that contains the last processed offset for that object.
	// if isRootArray == true for object, then LastProcessedOffset will treat offset as an array index
	LastProcessedOffset map[string]int64
}

func newState() *state {
	return &state{
		cp: &Checkpoint{
			FailedJobs:          make(map[string]int),
			LastProcessedOffset: make(map[string]int64),
			IsRootArray:         make(map[string]bool),
		},
	}
}

// saveForTx updates and returns the current state checkpoint, locks the state
// and returns an unlock function, done. The caller must call done when
// s and cp are no longer needed in a locked state. done may not be called
// more than once.
func (s *state) saveForTx(name string, lastModifiedOn time.Time) (cp *Checkpoint, done func()) {
	s.mu.Lock()
	delete(s.cp.LastProcessedOffset, name)
	delete(s.cp.IsRootArray, name)
	if _, ok := s.cp.FailedJobs[name]; !ok {
		if len(s.cp.ObjectName) == 0 {
			s.cp.ObjectName = name
		} else if strings.ToLower(name) > strings.ToLower(s.cp.ObjectName) {
			s.cp.ObjectName = name
		}

		if s.cp.LatestEntryTime.IsZero() {
			s.cp.LatestEntryTime = lastModifiedOn
		} else if lastModifiedOn.After(s.cp.LatestEntryTime) {
			s.cp.LatestEntryTime = lastModifiedOn
		}
	} else {
		// clear entry if this is a failed job
		delete(s.cp.FailedJobs, name)
	}
	return s.cp, func() { s.mu.Unlock() }
}

// savePartialForTx partially updates and returns the current state checkpoint, locks the state
// and returns an unlock function, done. The caller must call done when
// s and cp are no longer needed in a locked state. done may not be called
// more than once.
func (s *state) savePartialForTx(name string, offset int64) (cp *Checkpoint, done func()) {
	s.mu.Lock()
	s.cp.LastProcessedOffset[name] = offset
	return s.cp, func() { s.mu.Unlock() }
}

// setRootArray, sets boolean true for objects that have their roots defined as an array type
func (s *state) setRootArray(name string) {
	s.mu.Lock()
	s.cp.IsRootArray[name] = true
	s.mu.Unlock()
}

// updateFailedJobs, adds a job name to a failedJobs map, which helps
// in keeping track of failed jobs during edge cases when the state might
// move ahead in timestamp & objectName due to successful operations from other workers.
// A failed job will be re-tried a maximum of 3 times after which the
// entry is removed from the map
func (s *state) updateFailedJobs(jobName string) {
	s.mu.Lock()
	// we do not store partially processed jobs as failed jobs
	if _, ok := s.cp.LastProcessedOffset[jobName]; ok {
		return
	}
	s.cp.FailedJobs[jobName]++
	if s.cp.FailedJobs[jobName] > maxFailedJobRetries {
		delete(s.cp.FailedJobs, jobName)
	}
	s.mu.Unlock()
}

// setCheckpoint, sets checkpoint from source to current state instance
// If for some reason the current state is empty, assigns new states as
// a fail safe mechanism
func (s *state) setCheckpoint(chkpt *Checkpoint) {
	if chkpt.FailedJobs == nil {
		chkpt.FailedJobs = make(map[string]int)
	}
	if chkpt.IsRootArray == nil {
		chkpt.IsRootArray = make(map[string]bool)
	}
	if chkpt.LastProcessedOffset == nil {
		chkpt.LastProcessedOffset = make(map[string]int64)
	}
	s.cp = chkpt
}

// checkpoint, returns the current state checkpoint
func (s *state) checkpoint() *Checkpoint {
	return s.cp
}
