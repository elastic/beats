// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"testing"
	"time"

	"github.com/elastic/beats/v7/filebeat/beater"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/storetest"
	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
				_ = states.AddState(testState2)
			},
			state:               testState1,
			expectedIsProcessed: false,
		},
		"existing state": {
			statesEdit: func(states *states) {
				_ = states.AddState(testState1)
			},
			state:               testState1,
			expectedIsProcessed: true,
		},
		"existing stored state is persisted": {
			statesEdit: func(states *states) {
				state := testState1
				state.Stored = true
				_ = states.AddState(state)
			},
			state:               testState1,
			shouldReload:        true,
			expectedIsProcessed: true,
		},
		"existing failed state is persisted": {
			statesEdit: func(states *states) {
				state := testState1
				state.Failed = true
				_ = states.AddState(state)
			},
			state:               testState1,
			shouldReload:        true,
			expectedIsProcessed: true,
		},
		"existing unprocessed state is not persisted": {
			statesEdit: func(states *states) {
				_ = states.AddState(testState1)
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
			states, err := newStates(nil, store, "")
			require.NoError(t, err, "states creation must succeed")
			if test.statesEdit != nil {
				test.statesEdit(states)
			}
			if test.shouldReload {
				states, err = newStates(nil, store, "")
				require.NoError(t, err, "states creation must succeed")
			}

			isProcessed := states.IsProcessed(test.state)
			assert.Equal(t, test.expectedIsProcessed, isProcessed)
		})
	}
}

func TestStatesPrefixHandling(t *testing.T) {
	logger := logp.NewLogger("state-prefix-testing")

	t.Run("if prefix was set, accept only states with prefix", func(t *testing.T) {
		// given
		registry := openTestStatestore()

		// when - registry with prefix
		st, err := newStates(logger, registry, "staging-")
		require.NoError(t, err)

		// then - fail for non prefixed
		err = st.AddState(newState("bucket", "production-logA", "etag", time.Now()))
		require.Error(t, err)

		// then - pass for correctly prefixed
		err = st.AddState(newState("bucket", "staging-logA", "etag", time.Now()))
		require.NoError(t, err)
	})

	t.Run("Store only load entries matching the given prefix", func(t *testing.T) {
		// given
		registry := openTestStatestore()

		sA := newState("bucket", "A", "etag", time.Unix(1733221244, 0))
		sA.Stored = true
		sStagingA := newState("bucket", "staging-A", "etag", time.Unix(1733224844, 0))
		sStagingA.Stored = true
		sProdB := newState("bucket", "production/B", "etag", time.Unix(1733228444, 0))
		sProdB.Stored = true
		sSpace := newState("bucket", "  B", "etag", time.Unix(1733230444, 0))
		sSpace.Stored = true

		// add various states first with no prefix
		st, err := newStates(logger, registry, "")
		require.NoError(t, err)

		_ = st.AddState(sA)
		_ = st.AddState(sStagingA)
		_ = st.AddState(sProdB)
		_ = st.AddState(sSpace)

		// Reload states and validate

		// when - no prefix reload
		stNoPrefix, err := newStates(logger, registry, "")
		require.NoError(t, err)

		require.True(t, stNoPrefix.IsProcessed(sA))
		require.True(t, stNoPrefix.IsProcessed(sStagingA))
		require.True(t, stNoPrefix.IsProcessed(sProdB))
		require.True(t, stNoPrefix.IsProcessed(sSpace))

		// when - with prefix `staging-`
		st, err = newStates(logger, registry, "staging-")
		require.NoError(t, err)

		require.False(t, st.IsProcessed(sA))
		require.True(t, st.IsProcessed(sStagingA))
		require.False(t, st.IsProcessed(sProdB))
		require.False(t, st.IsProcessed(sSpace))

		// when - with prefix `production/`
		st, err = newStates(logger, registry, "production/")
		require.NoError(t, err)

		require.False(t, st.IsProcessed(sA))
		require.False(t, st.IsProcessed(sStagingA))
		require.True(t, st.IsProcessed(sProdB))
		require.False(t, st.IsProcessed(sSpace))
	})

}
