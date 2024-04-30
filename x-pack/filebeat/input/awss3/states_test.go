// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"testing"
	"time"

	"github.com/elastic/beats/v7/filebeat/beater"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/storetest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/elastic-agent-libs/logp"
)

type testInputStore struct {
	registry *statestore.Registry
}

func openTestStatestore() beater.StateStore {
	return &testInputStore{
		registry: statestore.NewRegistry(storetest.NewMemoryStoreBackend()),
	}
}

func (s *testInputStore) Close() {
	_ = s.registry.Close()
}

func (s *testInputStore) Access() (*statestore.Store, error) {
	return s.registry.Get("filebeat")
}

func (s *testInputStore) CleanupInterval() time.Duration {
	return 24 * time.Hour
}

var inputCtx = v2.Context{
	Logger:      logp.NewLogger("test"),
	Cancelation: context.Background(),
}

func TestStatesAddStateAndIsProcessed(t *testing.T) {
	type stateTestCase struct {
		// An initialization callback to invoke on the (initially empty) states.
		statesEdit func(states *states)

		// The state to call IsProcessed on and the expected result
		state               state
		expectedIsProcessed bool

		// If true, the test will run statesEdit, then create a new states
		// object from the same persistent store before calling IsProcessed
		// (to test persistence between restarts).
		shouldReload bool
	}
	lastModified := time.Date(2022, time.June, 30, 14, 13, 00, 0, time.UTC)
	testState1 := newState("bucket", "key", "etag", lastModified)
	testState2 := newState("bucket1", "key1", "etag1", lastModified)
	tests := map[string]stateTestCase{
		"with empty states": {
			state:               testState1,
			expectedIsProcessed: false,
		},
		"not existing state": {
			statesEdit: func(states *states) {
				states.AddState(testState2)
			},
			state:               testState1,
			expectedIsProcessed: false,
		},
		"existing state": {
			statesEdit: func(states *states) {
				states.AddState(testState1)
			},
			state:               testState1,
			expectedIsProcessed: true,
		},
		"existing stored state is persisted": {
			statesEdit: func(states *states) {
				state := testState1
				state.Stored = true
				states.AddState(state)
			},
			state:               testState1,
			shouldReload:        true,
			expectedIsProcessed: true,
		},
		"existing failed state is persisted": {
			statesEdit: func(states *states) {
				state := testState1
				state.Failed = true
				states.AddState(state)
			},
			state:               testState1,
			shouldReload:        true,
			expectedIsProcessed: true,
		},
		"existing unprocessed state is not persisted": {
			statesEdit: func(states *states) {
				states.AddState(testState1)
			},
			state:               testState1,
			shouldReload:        true,
			expectedIsProcessed: false,
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			store := openTestStatestore()
			persistentStore, err := store.Access()
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			states, err := newStates(persistentStore)
			require.NoError(t, err, "states creation must succeed")
			if test.statesEdit != nil {
				test.statesEdit(states)
			}
			if test.shouldReload {
				states, err = newStates(persistentStore)
				require.NoError(t, err, "states creation must succeed")
			}

			isProcessed := states.IsProcessed(test.state)
			assert.Equal(t, test.expectedIsProcessed, isProcessed)
		})
	}
}
