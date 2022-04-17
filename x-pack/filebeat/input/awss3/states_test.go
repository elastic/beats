// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	v2 "github.com/menderesk/beats/v7/filebeat/input/v2"
	"github.com/menderesk/beats/v7/libbeat/logp"
)

var inputCtx = v2.Context{
	Logger:      logp.NewLogger("test"),
	Cancelation: context.Background(),
}

func TestStatesDelete(t *testing.T) {
	type stateTestCase struct {
		states   func() *states
		deleteID string
		expected []state
	}

	lastModified := time.Date(2021, time.July, 22, 18, 38, 0o0, 0, time.UTC)
	tests := map[string]stateTestCase{
		"delete empty states": {
			states: func() *states {
				return newStates(inputCtx)
			},
			deleteID: "an id",
			expected: []state{},
		},
		"delete not existing state": {
			states: func() *states {
				states := newStates(inputCtx)
				states.Update(newState("bucket", "key", "etag", lastModified), "")
				return states
			},
			deleteID: "an id",
			expected: []state{
				{
					ID:           "bucketkeyetag" + lastModified.String(),
					Bucket:       "bucket",
					Key:          "key",
					Etag:         "etag",
					LastModified: lastModified,
				},
			},
		},
		"delete only one existing": {
			states: func() *states {
				states := newStates(inputCtx)
				states.Update(newState("bucket", "key", "etag", lastModified), "")
				return states
			},
			deleteID: "bucketkey",
			expected: []state{},
		},
		"delete first": {
			states: func() *states {
				states := newStates(inputCtx)
				states.Update(newState("bucket", "key1", "etag1", lastModified), "")
				states.Update(newState("bucket", "key2", "etag2", lastModified), "")
				states.Update(newState("bucket", "key3", "etag3", lastModified), "")
				return states
			},
			deleteID: "bucketkey1",
			expected: []state{
				{
					ID:           "bucketkey3etag3" + lastModified.String(),
					Bucket:       "bucket",
					Key:          "key3",
					Etag:         "etag3",
					LastModified: lastModified,
				},
				{
					ID:           "bucketkey2etag2" + lastModified.String(),
					Bucket:       "bucket",
					Key:          "key2",
					Etag:         "etag2",
					LastModified: lastModified,
				},
			},
		},
		"delete last": {
			states: func() *states {
				states := newStates(inputCtx)
				states.Update(newState("bucket", "key1", "etag1", lastModified), "")
				states.Update(newState("bucket", "key2", "etag2", lastModified), "")
				states.Update(newState("bucket", "key3", "etag3", lastModified), "")
				return states
			},
			deleteID: "bucketkey3",
			expected: []state{
				{
					ID:           "bucketkey1etag1" + lastModified.String(),
					Bucket:       "bucket",
					Key:          "key1",
					Etag:         "etag1",
					LastModified: lastModified,
				},
				{
					ID:           "bucketkey2etag2" + lastModified.String(),
					Bucket:       "bucket",
					Key:          "key2",
					Etag:         "etag2",
					LastModified: lastModified,
				},
			},
		},
		"delete any": {
			states: func() *states {
				states := newStates(inputCtx)
				states.Update(newState("bucket", "key1", "etag1", lastModified), "")
				states.Update(newState("bucket", "key2", "etag2", lastModified), "")
				states.Update(newState("bucket", "key3", "etag3", lastModified), "")
				return states
			},
			deleteID: "bucketkey2",
			expected: []state{
				{
					ID:           "bucketkey1etag1" + lastModified.String(),
					Bucket:       "bucket",
					Key:          "key1",
					Etag:         "etag1",
					LastModified: lastModified,
				},
				{
					ID:           "bucketkey3etag3" + lastModified.String(),
					Bucket:       "bucket",
					Key:          "key3",
					Etag:         "etag3",
					LastModified: lastModified,
				},
			},
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			states := test.states()
			states.Delete(test.deleteID)
			assert.Equal(t, test.expected, states.GetStates())
		})
	}
}
