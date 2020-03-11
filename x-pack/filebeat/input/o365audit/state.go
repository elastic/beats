// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package o365audit

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

var errNoUpdate = errors.New("new cursor doesn't preceed the existing cursor")

// Stream represents an event stream.
type stream struct {
	tenantID, contentType string
}

// A cursor represents a point in time within an event stream
// that can be persisted and used to resume processing from that point.
type cursor struct {
	// Identifier for the event stream.
	stream

	// createdTime for the last seen blob.
	timestamp time.Time
	// index of object count (1...n) within a blob.
	line int
	// startTime used in the last list content query.
	// This is necessary to ensure that the same blobs are observed.
	startTime time.Time
}

// Create a new cursor.
func newCursor(s stream, time time.Time) cursor {
	return cursor{
		stream:    s,
		timestamp: time,
	}
}

// TryAdvance advances the cursor to the given content blob
// if it's not in the past.
// Returns whether the given content needs to be processed.
func (c *cursor) TryAdvance(ct content) bool {
	if ct.Created.Before(c.timestamp) {
		return false
	}
	if ct.Created.Equal(c.timestamp) {
		// Only need to re-process the current content blob if we're
		// seeking to a line inside it.
		return c.line > 0
	}
	c.timestamp = ct.Created
	c.line = 0
	return true
}

// Before allows to compare cursors to see if the new cursor needs to be persisted.
func (c cursor) Before(b cursor) bool {
	if c.contentType != b.contentType || c.tenantID != b.tenantID {
		panic(fmt.Sprintf("assertion failed: %+v vs %+v", c, b))
	}

	if c.timestamp.Before(b.timestamp) {
		return true
	}
	if c.timestamp.Equal(b.timestamp) {
		return c.line < b.line
	}
	return false
}

// WithStartTime allows to create a cursor with an updated startTime.
func (c cursor) WithStartTime(s time.Time) cursor {
	c.startTime = s
	return c
}

// ForNextLine returns a new cursor for the next line within a blob.
func (c cursor) ForNextLine() cursor {
	c.line++
	return c
}

// String returns the printable representation of a cursor.
func (c cursor) String() string {
	return fmt.Sprintf("cursor{tenantID:%s contentType:%s timestamp:%s line:%d start:%s}",
		c.tenantID, c.contentType, c.timestamp, c.line, c.startTime)
}

// ErrStateNotFound is the error returned by a statePersister when a cursor
// is not found for a stream.
var errStateNotFound = errors.New("no saved state found")

type statePersister interface {
	Load(key stream) (cursor, error)
	Save(cursor cursor) error
}

type stateStorage struct {
	sync.Mutex
	saved     map[stream]cursor
	persister statePersister
}

func (s *stateStorage) Load(key stream) (cursor, error) {
	s.Lock()
	defer s.Unlock()
	if st, found := s.saved[key]; found {
		return st, nil
	}
	cur, err := s.persister.Load(key)
	if err != nil {
		if err != errStateNotFound {
			return cur, err
		}
		cur = newCursor(key, time.Time{})
	}
	return cur, s.saveUnsafe(cur)
}

func (s *stateStorage) Save(c cursor) error {
	s.Lock()
	defer s.Unlock()
	return s.saveUnsafe(c)
}

func (s *stateStorage) saveUnsafe(c cursor) error {
	if prev, found := s.saved[c.stream]; found {
		if !prev.Before(c) {
			return errNoUpdate
		}
	}
	if s.saved == nil {
		s.saved = make(map[stream]cursor)
	}
	s.saved[c.stream] = c
	return s.persister.Save(c)
}

func newStateStorage(underlying statePersister) *stateStorage {
	return &stateStorage{
		persister: underlying,
	}
}

type noopPersister struct{}

func (p noopPersister) Load(key stream) (cursor, error) {
	return cursor{}, errStateNotFound
}

func (p noopPersister) Save(cursor cursor) error {
	return nil
}
