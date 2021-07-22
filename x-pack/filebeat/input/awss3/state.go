// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"fmt"
	"strconv"
	"time"
)

// State is used to communicate the reading state of a file
type State struct {
	Id           string    `json:"id" struct:"id"`
	Bucket       string    `json:"bucket" struct:"bucket"`
	Key          string    `json:"key" struct:"key"`
	LastModified time.Time `json:"last_modified" struct:"last_modifed"`
	Size         int64     `json:"size" struct:"size"`
	Stored       bool      `json:"stored" struct:"stored"`
}

// NewState creates a new s3 object state
func NewState(bucket, key string, size int64, lastModified time.Time) State {
	s := State{
		Bucket:       bucket,
		Key:          key,
		LastModified: lastModified,
		Size:         size,
		Stored:       false,
	}

	s.Id = s.Bucket + s.Key + strconv.FormatInt(s.Size, 10) + s.LastModified.String()

	return s
}

// MarkAsStored set the stored flag to true
func (s *State) MarkAsStored() {
	s.Stored = true
}

// IsEqual checks if the two states point to the same file.
func (s *State) IsEqual(c *State) bool {
	return s.Bucket == c.Bucket && s.Key == c.Key && s.Size == c.Size && s.LastModified.Equal(c.LastModified)
}

// IsEmpty checks if the state is empty
func (s *State) IsEmpty() bool {
	c := State{}
	return s.Bucket == c.Bucket && s.Key == c.Key && s.Size == c.Size && s.LastModified.Equal(c.LastModified)
}

// String returns string representation of the struct
func (s *State) String() string {
	return fmt.Sprintf(
		"{Id: %v, Bucket: %v, Key: %v, Size: %v, LastModified: %v}",
		s.Id,
		s.Bucket,
		s.Key,
		s.Size,
		s.LastModified)
}
