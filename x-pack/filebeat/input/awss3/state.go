// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"fmt"
	"time"
)

// state is used to communicate the publishing state of a s3 object
type state struct {
	Bucket       string    `json:"bucket" struct:"bucket"`
	Key          string    `json:"key" struct:"key"`
	Etag         string    `json:"etag" struct:"etag"`
	LastModified time.Time `json:"last_modified" struct:"last_modified"`

	// A state has Stored = true when all events are ACKed.
	Stored bool `json:"stored" struct:"stored"`

	// Failed is true when ProcessS3Object returned an error other than
	// s3DownloadError.
	// Before 8.14, this field was called "error". However, that field was
	// set for many ephemeral reasons including client-side rate limiting
	// (see https://github.com/elastic/beats/issues/39114). Now that we
	// don't treat download errors as permanent, the field name was changed
	// so that users upgrading from old versions aren't prevented from
	// retrying old download failures.
	Failed bool `json:"failed" struct:"failed"`
}

// ID is used to identify the state in the store, and it is composed by
// Bucket + Key + Etag + LastModified.String(): changing this value or how it is
// composed will break backward compatibilities with entries already in the store.
func stateID(bucket, key, etag string, lastModified time.Time) string {
	return bucket + key + etag + lastModified.String()
}

// newState creates a new s3 object state
func newState(bucket, key, etag string, lastModified time.Time) state {
	return state{
		Bucket:       bucket,
		Key:          key,
		LastModified: lastModified,
		Etag:         etag,
	}
}

func (s *state) ID() string {
	return stateID(s.Bucket, s.Key, s.Etag, s.LastModified)
}

// IsEqual checks if the two states point to the same s3 object.
func (s *state) IsEqual(c *state) bool {
	return s.Bucket == c.Bucket && s.Key == c.Key && s.Etag == c.Etag && s.LastModified.Equal(c.LastModified)
}

// String returns string representation of the struct
func (s *state) String() string {
	return fmt.Sprintf(
		"{ID: %v, Bucket: %v, Key: %v, Etag: %v, LastModified: %v}",
		s.ID,
		s.Bucket,
		s.Key,
		s.Etag,
		s.LastModified)
}
