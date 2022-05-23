// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure_blob

import (
	"fmt"
	"time"
)

// state is used to communicate the publishing state of a s3 object
type state struct {
	// ID is used to identify the state in the store, and it is composed by
	// Container + Blob + Etag + LastModified.String(): changing this value or how it is
	// composed will break backward compatibilities with entries already in the store.
	ID           string    `json:"id" struct:"id"`
	Container    string    `json:"container" struct:"container"`
	Blob         string    `json:"blob" struct:"blob"`
	Etag         string    `json:"etag" struct:"etag"`
	LastModified time.Time `json:"last_modified" struct:"last_modified"`

	// A state has Stored = true when all events are ACKed.
	Stored bool `json:"stored" struct:"stored"`
	// A state has Error = true when ProcessS3Object returned an error
	Error bool `json:"error" struct:"error"`
}

// newState creates a new s3 object state
func newState(container, blob, etag string, lastModified time.Time) state {
	s := state{
		Container:    container,
		Blob:         blob,
		LastModified: lastModified,
		Etag:         etag,
		Stored:       false,
		Error:        false,
	}

	s.ID = s.Container + s.Blob + s.Etag + s.LastModified.String()

	return s
}

// MarkAsStored set the stored flag to true
func (s *state) MarkAsStored() {
	s.Stored = true
}

// MarkAsError set the error flag to true
func (s *state) MarkAsError() {
	s.Error = true
}

// IsEqual checks if the two states point to the same s3 object.
func (s *state) IsEqual(c *state) bool {
	return s.Container == c.Container && s.Blob == c.Blob && s.Etag == c.Etag && s.LastModified.Equal(c.LastModified)
}

// IsEmpty checks if the state is empty
func (s *state) IsEmpty() bool {
	c := state{}
	return s.Container == c.Container && s.Blob == c.Blob && s.Etag == c.Etag && s.LastModified.Equal(c.LastModified)
}

// String returns string representation of the struct
func (s *state) String() string {
	return fmt.Sprintf(
		"{ID: %v, Container: %v, Blob: %v, Etag: %v, LastModified: %v}",
		s.ID,
		s.Container,
		s.Blob,
		s.Etag,
		s.LastModified)
}
