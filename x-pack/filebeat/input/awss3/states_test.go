// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/elastic-agent-libs/logp"
)

var inputCtx = v2.Context{
	Logger:      logp.NewLogger("test"),
	Cancelation: context.Background(),
}

func TestStatesIsNew(t *testing.T) {
	type stateTestCase struct {
		states   func() *states
		state    state
		expected bool
	}
	lastModified := time.Date(2022, time.June, 30, 14, 13, 00, 0, time.UTC)
	tests := map[string]stateTestCase{
		"with empty states": {
			states: func() *states {
				return newStates(inputCtx)
			},
			state:    newState("bucket", "key", "etag", "listPrefix", lastModified),
			expected: true,
		},
		"not existing state": {
			states: func() *states {
				states := newStates(inputCtx)
				states.Update(newState("bucket", "key", "etag", "listPrefix", lastModified), "")
				return states
			},
			state:    newState("bucket1", "key1", "etag1", "listPrefix1", lastModified),
			expected: true,
		},
		"existing state": {
			states: func() *states {
				states := newStates(inputCtx)
				states.Update(newState("bucket", "key", "etag", "listPrefix", lastModified), "")
				return states
			},
			state:    newState("bucket", "key", "etag", "listPrefix", lastModified),
			expected: false,
		},
		"with different etag": {
			states: func() *states {
				states := newStates(inputCtx)
				states.Update(newState("bucket", "key", "etag1", "listPrefix", lastModified), "")
				return states
			},
			state:    newState("bucket", "key", "etag2", "listPrefix", lastModified),
			expected: true,
		},
		"with different lastmodified": {
			states: func() *states {
				states := newStates(inputCtx)
				states.Update(newState("bucket", "key", "etag", "listPrefix", lastModified), "")
				return states
			},
			state:    newState("bucket", "key", "etag", "listPrefix", lastModified.Add(1*time.Second)),
			expected: true,
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			states := test.states()
			isNew := states.IsNew(test.state)
			assert.Equal(t, test.expected, isNew)
		})
	}
}

func TestStatesDelete(t *testing.T) {
	type stateTestCase struct {
		states   func() *states
		deleteID string
		expected []state
	}

	lastModified := time.Date(2021, time.July, 22, 18, 38, 00, 0, time.UTC)
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
				states.Update(newState("bucket", "key", "etag", "listPrefix", lastModified), "")
				return states
			},
			deleteID: "an id",
			expected: []state{
				{
					ID:           stateID("bucket", "key", "etag", lastModified),
					Bucket:       "bucket",
					Key:          "key",
					Etag:         "etag",
					ListPrefix:   "listPrefix",
					LastModified: lastModified,
				},
			},
		},
		"delete only one existing": {
			states: func() *states {
				states := newStates(inputCtx)
				states.Update(newState("bucket", "key", "etag", "listPrefix", lastModified), "")
				return states
			},
			deleteID: stateID("bucket", "key", "etag", lastModified),
			expected: []state{},
		},
		"delete first": {
			states: func() *states {
				states := newStates(inputCtx)
				states.Update(newState("bucket", "key1", "etag1", "listPrefix", lastModified), "")
				states.Update(newState("bucket", "key2", "etag2", "listPrefix", lastModified), "")
				states.Update(newState("bucket", "key3", "etag3", "listPrefix", lastModified), "")
				return states
			},
			deleteID: "bucketkey1etag1" + lastModified.String(),
			expected: []state{
				{
					ID:           stateID("bucket", "key3", "etag3", lastModified),
					Bucket:       "bucket",
					Key:          "key3",
					Etag:         "etag3",
					ListPrefix:   "listPrefix",
					LastModified: lastModified,
				},
				{
					ID:           stateID("bucket", "key2", "etag2", lastModified),
					Bucket:       "bucket",
					Key:          "key2",
					Etag:         "etag2",
					ListPrefix:   "listPrefix",
					LastModified: lastModified,
				},
			},
		},
		"delete last": {
			states: func() *states {
				states := newStates(inputCtx)
				states.Update(newState("bucket", "key1", "etag1", "listPrefix", lastModified), "")
				states.Update(newState("bucket", "key2", "etag2", "listPrefix", lastModified), "")
				states.Update(newState("bucket", "key3", "etag3", "listPrefix", lastModified), "")
				return states
			},
			deleteID: "bucketkey3etag3" + lastModified.String(),
			expected: []state{
				{
					ID:           stateID("bucket", "key1", "etag1", lastModified),
					Bucket:       "bucket",
					Key:          "key1",
					Etag:         "etag1",
					ListPrefix:   "listPrefix",
					LastModified: lastModified,
				},
				{
					ID:           stateID("bucket", "key2", "etag2", lastModified),
					Bucket:       "bucket",
					Key:          "key2",
					Etag:         "etag2",
					ListPrefix:   "listPrefix",
					LastModified: lastModified,
				},
			},
		},
		"delete any": {
			states: func() *states {
				states := newStates(inputCtx)
				states.Update(newState("bucket", "key1", "etag1", "listPrefix", lastModified), "")
				states.Update(newState("bucket", "key2", "etag2", "listPrefix", lastModified), "")
				states.Update(newState("bucket", "key3", "etag3", "listPrefix", lastModified), "")
				return states
			},
			deleteID: "bucketkey2etag2" + lastModified.String(),
			expected: []state{
				{
					ID:           stateID("bucket", "key1", "etag1", lastModified),
					Bucket:       "bucket",
					Key:          "key1",
					Etag:         "etag1",
					ListPrefix:   "listPrefix",
					LastModified: lastModified,
				},
				{
					ID:           stateID("bucket", "key3", "etag3", lastModified),
					Bucket:       "bucket",
					Key:          "key3",
					Etag:         "etag3",
					ListPrefix:   "listPrefix",
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
