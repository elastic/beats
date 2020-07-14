// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package o365audit

import (
	"fmt"
	"time"
)

// A checkpoint represents a point in time within an event stream
// that can be persisted and used to resume processing from that point.
type checkpoint struct {
	// createdTime for the last seen blob.
	Timestamp time.Time `struct:"timestamp"`

	// index of object count (1...n) within a blob.
	Line int `struct:"line"`

	// startTime used in the last list content query.
	// This is necessary to ensure that the same blobs are observed.
	StartTime time.Time `struct:"start_time"`
}

func (c *checkpoint) Before(other checkpoint) bool {
	return c.Timestamp.Before(other.Timestamp) || (c.Timestamp.Equal(other.Timestamp) && c.Line < other.Line)
}

// TryAdvance advances the cursor to the given content blob
// if it's not in the past.
// Returns whether the given content needs to be processed.
func (c *checkpoint) TryAdvance(ct content) bool {
	if ct.Created.Before(c.Timestamp) {
		return false
	}
	if ct.Created.Equal(c.Timestamp) {
		// Only need to re-process the current content blob if we're
		// seeking to a line inside it.
		return c.Line > 0
	}
	c.Timestamp = ct.Created
	c.Line = 0
	return true
}

// WithStartTime allows to create a cursor with an updated startTime.
func (c checkpoint) WithStartTime(s time.Time) checkpoint {
	c.StartTime = s
	return c
}

// ForNextLine returns a new cursor for the next line within a blob.
func (c checkpoint) ForNextLine() checkpoint {
	c.Line++
	return c
}

// String returns the printable representation of a cursor.
func (c checkpoint) String() string {
	return fmt.Sprintf("cursor{timestamp:%s line:%d start:%s}",
		c.Timestamp, c.Line, c.StartTime)
}
