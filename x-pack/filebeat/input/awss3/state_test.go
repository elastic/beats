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
		states [2]State
		isSame bool
	}

	lastModifed := time.Now()
	tests := map[string]stateTestCase{
		"two states pointing to the same key with same size and same last modified not stored": {
			[2]State{
				State{
					Bucket:       "bucket a",
					Key:          "/key/to/this/file/1",
					Size:         100,
					LastModified: lastModifed,
				},
				State{
					Bucket:       "bucket a",
					Key:          "/key/to/this/file/1",
					Size:         100,
					LastModified: lastModifed,
				},
			},
			true,
		},
		"two states pointing to the same key with same size and same last modified stored": {
			[2]State{
				State{
					Bucket:       "bucket a",
					Key:          "/key/to/this/file/1",
					Size:         100,
					LastModified: lastModifed,
					Stored:       true,
				},
				State{
					Bucket:       "bucket a",
					Key:          "/key/to/this/file/1",
					Size:         100,
					LastModified: lastModifed,
				},
			},
			false,
		},
		"two states pointing to the same key with different size and same last modified": {
			[2]State{
				State{
					Bucket:       "bucket a",
					Key:          "/key/to/this/file/1",
					Size:         100,
					LastModified: lastModifed,
				},
				State{
					Bucket:       "bucket a",
					Key:          "/key/to/this/file/1",
					Size:         101,
					LastModified: lastModifed,
				},
			},
			false,
		},
		"two states pointing to the same key with same size and different last modified": {
			[2]State{
				State{
					Bucket:       "bucket a",
					Key:          "/key/to/this/file/1",
					Size:         100,
					LastModified: time.Now(),
				},
				State{
					Bucket:       "bucket a",
					Key:          "/key/to/this/file/1",
					Size:         100,
					LastModified: time.Now().Add(10 * time.Second),
				},
			},
			false,
		},
		"two states pointing to different key": {
			[2]State{
				State{
					Bucket:       "bucket a",
					Key:          "/key/to/this/file/1",
					Size:         100,
					LastModified: lastModifed,
				},
				State{
					Bucket:       "bucket a",
					Key:          "/key/to/this/file/2",
					Size:         100,
					LastModified: lastModifed,
				},
			},
			false,
		},
		"two states pointing to different bucket": {
			[2]State{
				State{
					Bucket:       "bucket b",
					Key:          "/key/to/this/file/1",
					Size:         100,
					LastModified: lastModifed,
				},
				State{
					Bucket:       "bucket a",
					Key:          "/key/to/this/file/1",
					Size:         100,
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
			assert.Equal(t, isSame, test.isSame)
		})
	}
}
