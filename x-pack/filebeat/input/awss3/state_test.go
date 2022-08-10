// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStateIsEqual(t *testing.T) {
	type stateTestCase struct {
		states [2]state
		isSame bool
	}

	lastModifed := time.Now()
	tests := map[string]stateTestCase{
		"two states pointing to the same key with same etag and same last modified not stored": {
			[2]state{
				{
					Bucket:       "bucket a",
					Key:          "/key/to/this/file/1",
					Etag:         "etag",
					LastModified: lastModifed,
				},
				{
					Bucket:       "bucket a",
					Key:          "/key/to/this/file/1",
					Etag:         "etag",
					LastModified: lastModifed,
				},
			},
			true,
		},
		"two states pointing to the same key with same etag and same last modified stored": {
			[2]state{
				{
					Bucket:       "bucket a",
					Key:          "/key/to/this/file/1",
					Etag:         "etag",
					LastModified: lastModifed,
					Stored:       true,
				},
				{
					Bucket:       "bucket a",
					Key:          "/key/to/this/file/1",
					Etag:         "etag",
					LastModified: lastModifed,
				},
			},
			true,
		},
		"two states pointing to the same key with same etag and same last modified error": {
			[2]state{
				{
					Bucket:       "bucket a",
					Key:          "/key/to/this/file/1",
					Etag:         "etag",
					LastModified: lastModifed,
					Error:        true,
				},
				{
					Bucket:       "bucket a",
					Key:          "/key/to/this/file/1",
					Etag:         "etag",
					LastModified: lastModifed,
				},
			},
			true,
		},
		"two states pointing to the same key with different etag and same last modified": {
			[2]state{
				{
					Bucket:       "bucket a",
					Key:          "/key/to/this/file/1",
					Etag:         "etag1",
					LastModified: lastModifed,
				},
				{
					Bucket:       "bucket a",
					Key:          "/key/to/this/file/1",
					Etag:         "etag2",
					LastModified: lastModifed,
				},
			},
			false,
		},
		"two states pointing to the same key with same etag and different last modified": {
			[2]state{
				{
					Bucket:       "bucket a",
					Key:          "/key/to/this/file/1",
					Etag:         "etag",
					LastModified: time.Now(),
				},
				{
					Bucket:       "bucket a",
					Key:          "/key/to/this/file/1",
					Etag:         "etag",
					LastModified: time.Now().Add(10 * time.Second),
				},
			},
			false,
		},
		"two states pointing to different key": {
			[2]state{
				{
					Bucket:       "bucket a",
					Key:          "/key/to/this/file/1",
					Etag:         "etag",
					LastModified: lastModifed,
				},
				{
					Bucket:       "bucket a",
					Key:          "/key/to/this/file/2",
					Etag:         "etag",
					LastModified: lastModifed,
				},
			},
			false,
		},
		"two states pointing to different bucket": {
			[2]state{
				{
					Bucket:       "bucket b",
					Key:          "/key/to/this/file/1",
					Etag:         "etag",
					LastModified: lastModifed,
				},
				{
					Bucket:       "bucket a",
					Key:          "/key/to/this/file/1",
					Etag:         "etag",
					LastModified: lastModifed,
				},
			},
			false,
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			isSame := test.states[0].IsEqual(&test.states[1])
			assert.Equal(t, test.isSame, isSame)
		})
	}
}
