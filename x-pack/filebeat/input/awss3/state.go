// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"fmt"
	"time"
)

// state is used to communicate the reading state of a file
type state struct {
	Id           string    `json:"id" struct:"id"`
	Bucket       string    `json:"bucket" struct:"bucket"`
	Key          string    `json:"key" struct:"key"`
	Etag         string    `json:"etag" struct:"etag"`
	LastModified time.Time `json:"last_modified" struct:"last_modifed"`
	Stored       bool      `json:"stored" struct:"stored"`
}

// newState creates a new s3 object state
func newState(bucket, key, etag string, lastModified time.Time) state {
	s := state{
		Bucket:       bucket,
		Key:          key,
		LastModified: lastModified,
		Etag:         etag,
		Stored:       false,
	}

	s.Id = s.Bucket + s.Key + s.Etag + s.LastModified.String()

	return s
}

// MarkAsStored set the stored flag to true
func (s *state) MarkAsStored() {
	s.Stored = true
}

// IsEqual checks if the two states point to the same file.
func (s *state) IsEqual(c *state) bool {
	return s.Bucket == c.Bucket && s.Key == c.Key && s.Etag == c.Etag && s.LastModified.Equal(c.LastModified)
}

// IsEmpty checks if the state is empty
func (s *state) IsEmpty() bool {
	c := state{}
	return s.Bucket == c.Bucket && s.Key == c.Key && s.Etag == c.Etag && s.LastModified.Equal(c.LastModified)
}

// String returns string representation of the struct
func (s *state) String() string {
	return fmt.Sprintf(
		"{Id: %v, Bucket: %v, Key: %v, Etag: %v, LastModified: %v}",
		s.Id,
		s.Bucket,
		s.Key,
		s.Etag,
		s.LastModified)
}
