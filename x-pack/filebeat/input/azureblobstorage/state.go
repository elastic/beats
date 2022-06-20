// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package azureblobstorage

import (
	"sync"
	"time"
)

// State contains the the current state of the operation
type state struct {
	// Mutex lock to help in concurrent R/W
	mu sync.Mutex
	// marker contains the last known position in the blob pager which was fetched
	marker *string
	// name of the blob
	name string
	// timestamp to denote when the blob was last modified
	lastModifiedOn *time.Time
}

func newState() *state {
	return &state{}
}

// save functions , saves/updates the current state
func (s *state) save(name string, marker *string, lastModifiedOn *time.Time) {
	s.mu.Lock()
	s.name = name
	s.marker = marker
	s.lastModifiedOn = lastModifiedOn
	s.mu.Unlock()
}
