// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStatesDelete(t *testing.T) {
	type stateTestCase struct {
		states   func() *States
		deleteID string
		expected []State
	}

	lastModified := time.Date(2021, time.July, 22, 18, 38, 00, 0, time.UTC)
	tests := map[string]stateTestCase{
		"delete empty states": {
			states: func() *States {
				return NewStates()
			},
			deleteID: "an id",
			expected: []State{},
		},
		"delete not existing state": {
			states: func() *States {
				states := NewStates()
				states.Update(NewState("bucket", "key", 100, lastModified))
				return states
			},
			deleteID: "an id",
			expected: []State{
				{
					Id:           "bucketkey100" + lastModified.String(),
					Bucket:       "bucket",
					Key:          "key",
					Size:         100,
					LastModified: lastModified,
				},
			},
		},
		"delete only one existing": {
			states: func() *States {
				states := NewStates()
				states.Update(NewState("bucket", "key", 100, lastModified))
				return states
			},
			deleteID: "bucketkey",
			expected: []State{},
		},
		"delete first": {
			states: func() *States {
				states := NewStates()
				states.Update(NewState("bucket", "key1", 100, lastModified))
				states.Update(NewState("bucket", "key2", 100, lastModified))
				states.Update(NewState("bucket", "key3", 100, lastModified))
				return states
			},
			deleteID: "bucketkey1",
			expected: []State{
				{
					Id:           "bucketkey3100" + lastModified.String(),
					Bucket:       "bucket",
					Key:          "key3",
					Size:         100,
					LastModified: lastModified,
				},
				{
					Id:           "bucketkey2100" + lastModified.String(),
					Bucket:       "bucket",
					Key:          "key2",
					Size:         100,
					LastModified: lastModified,
				},
			},
		},
		"delete last": {
			states: func() *States {
				states := NewStates()
				states.Update(NewState("bucket", "key1", 100, lastModified))
				states.Update(NewState("bucket", "key2", 100, lastModified))
				states.Update(NewState("bucket", "key3", 100, lastModified))
				return states
			},
			deleteID: "bucketkey3",
			expected: []State{
				{
					Id:           "bucketkey1100" + lastModified.String(),
					Bucket:       "bucket",
					Key:          "key1",
					Size:         100,
					LastModified: lastModified,
				},
				{
					Id:           "bucketkey2100" + lastModified.String(),
					Bucket:       "bucket",
					Key:          "key2",
					Size:         100,
					LastModified: lastModified,
				},
			},
		},
		"delete any": {
			states: func() *States {
				states := NewStates()
				states.Update(NewState("bucket", "key1", 100, lastModified))
				states.Update(NewState("bucket", "key2", 100, lastModified))
				states.Update(NewState("bucket", "key3", 100, lastModified))
				return states
			},
			deleteID: "bucketkey2",
			expected: []State{
				{
					Id:           "bucketkey1100" + lastModified.String(),
					Bucket:       "bucket",
					Key:          "key1",
					Size:         100,
					LastModified: lastModified,
				},
				{
					Id:           "bucketkey3100" + lastModified.String(),
					Bucket:       "bucket",
					Key:          "key3",
					Size:         100,
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
