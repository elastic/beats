// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"fmt"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/storetest"
	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatesAddStateAndIsProcessed(t *testing.T) {
	type stateTestCase struct {
		// An initialization callback to invoke on the (initially empty) states.
		statesEdit func(states *states) error

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
			statesEdit: func(states *states) error {
				return states.AddState(testState2)
			},
			state:               testState1,
			expectedIsProcessed: false,
		},
		"existing state": {
			statesEdit: func(states *states) error {
				return states.AddState(testState1)
			},
			state:               testState1,
			expectedIsProcessed: true,
		},
		"existing stored state is persisted": {
			statesEdit: func(states *states) error {
				state := testState1
				state.Stored = true
				return states.AddState(state)
			},
			state:               testState1,
			shouldReload:        true,
			expectedIsProcessed: true,
		},
		"existing failed state is persisted": {
			statesEdit: func(states *states) error {
				state := testState1
				state.Failed = true
				return states.AddState(state)
			},
			state:               testState1,
			shouldReload:        true,
			expectedIsProcessed: true,
		},
		"existing unprocessed state is not persisted": {
			statesEdit: func(states *states) error {
				return states.AddState(testState1)
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
			states, err := newStates(nil, store, "", false, 0)
			require.NoError(t, err, "states creation must succeed")
			if test.statesEdit != nil {
				err = test.statesEdit(states)
				require.NoError(t, err, "states edit must succeed")
			}
			if test.shouldReload {
				states, err = newStates(nil, store, "", false, 0)
				require.NoError(t, err, "states creation must succeed")
			}

			isProcessed := states.IsProcessed(test.state.ID())
			assert.Equal(t, test.expectedIsProcessed, isProcessed)
		})
	}
}

func TestStatesCleanUp(t *testing.T) {
	bucketName := "test-bucket"
	lModifiedTime := time.Unix(0, 0)
	stateA := newState(bucketName, "a", "a-etag", lModifiedTime)
	stateB := newState(bucketName, "b", "b-etag", lModifiedTime)
	stateC := newState(bucketName, "c", "c-etag", lModifiedTime)

	tests := []struct {
		name       string
		initStates []state
		knownIDs   []string
		expectIDs  []string
	}{
		{
			name:       "No cleanup if not missing from known list",
			initStates: []state{stateA, stateB, stateC},
			knownIDs:   []string{stateA.ID(), stateB.ID(), stateC.ID()},
			expectIDs:  []string{stateA.ID(), stateB.ID(), stateC.ID()},
		},
		{
			name:       "Clean up if missing from known list",
			initStates: []state{stateA, stateB, stateC},
			knownIDs:   []string{stateA.ID()},
			expectIDs:  []string{stateA.ID()},
		},
		{
			name:       "Clean up everything",
			initStates: []state{stateA, stateC}, // given A, C
			knownIDs:   []string{stateB.ID()},   // but known B
			expectIDs:  []string{},              // empty state & store
		},
		{
			name:       "Empty known IDs are valid",
			initStates: []state{stateA}, // given A
			knownIDs:   []string{},      // Known nothing
			expectIDs:  []string{},      // empty state & store
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := openTestStatestore()
			statesInstance, err := newStates(nil, store, "", false, 0)
			require.NoError(t, err, "states creation must succeed")

			for _, s := range test.initStates {
				err := statesInstance.AddState(s)
				require.NoError(t, err, "state initialization must succeed")
			}

			// perform cleanup
			err = statesInstance.CleanUp(test.knownIDs)
			require.NoError(t, err, "state cleanup must succeed")

			// validate
			for _, id := range test.expectIDs {
				// must be in local state
				_, ok := statesInstance.states[id]
				require.True(t, ok, fmt.Errorf("expected id %s in state, but got missing", id))

				// must be in store
				ok, err := statesInstance.store.Has(getStoreKey(id))
				require.NoError(t, err, "state has must succeed")
				require.True(t, ok, fmt.Errorf("expected id %s in store, but got missing", id))
			}
		})
	}

}

func TestStatesPrefixHandling(t *testing.T) {
	logger := logp.NewLogger("state-prefix-testing")

	t.Run("if prefix was set, accept only states with prefix", func(t *testing.T) {
		// given
		registry := openTestStatestore()

		// when - registry with prefix
		st, err := newStates(logger, registry, "staging-", false, 0)
		require.NoError(t, err)

		// then - fail for non prefixed
		err = st.AddState(newState("bucket", "production-logA", "etag", time.Now()))
		require.Error(t, err)

		// then - pass for correctly prefixed
		err = st.AddState(newState("bucket", "staging-logA", "etag", time.Now()))
		require.NoError(t, err)
	})

	t.Run("states store only load entries matching the given prefix", func(t *testing.T) {
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
		st, err := newStates(logger, registry, "", false, 0)
		require.NoError(t, err)

		_ = st.AddState(sA)
		_ = st.AddState(sStagingA)
		_ = st.AddState(sProdB)
		_ = st.AddState(sSpace)

		// Reload states and validate

		// when - no prefix reload
		stNoPrefix, err := newStates(logger, registry, "", false, 0)
		require.NoError(t, err)

		require.True(t, stNoPrefix.IsProcessed(sA.ID()))
		require.True(t, stNoPrefix.IsProcessed(sStagingA.ID()))
		require.True(t, stNoPrefix.IsProcessed(sProdB.ID()))
		require.True(t, stNoPrefix.IsProcessed(sSpace.ID()))

		// when - with prefix `staging-`
		st, err = newStates(logger, registry, "staging-", false, 0)
		require.NoError(t, err)

		require.False(t, st.IsProcessed(sA.ID()))
		require.True(t, st.IsProcessed(sStagingA.ID()))
		require.False(t, st.IsProcessed(sProdB.ID()))
		require.False(t, st.IsProcessed(sSpace.ID()))

		// when - with prefix `production/`
		st, err = newStates(logger, registry, "production/", false, 0)
		require.NoError(t, err)

		require.False(t, st.IsProcessed(sA.ID()))
		require.False(t, st.IsProcessed(sStagingA.ID()))
		require.True(t, st.IsProcessed(sProdB.ID()))
		require.False(t, st.IsProcessed(sSpace.ID()))
	})

}

func TestStatesLexicographicalMode(t *testing.T) {
	logger := logp.NewLogger("states-lexicographical-test")

	t.Run("AddState with lexicographical ordering uses IDWithLexicographicalOrdering", func(t *testing.T) {
		store := openTestStatestore()
		states, err := newStates(logger, store, "", true, 10)
		require.NoError(t, err)

		state1 := newState("bucket", "key1", "etag1", time.Unix(1000, 0))
		state1.Stored = true
		err = states.AddState(state1)
		require.NoError(t, err)

		require.True(t, states.IsProcessed(state1.IDWithLexicographicalOrdering()))
		require.False(t, states.IsProcessed(state1.ID()))
	})

	t.Run("AddState evicts oldest state when at capacity", func(t *testing.T) {
		store := openTestStatestore()
		capacity := 3
		states, err := newStates(logger, store, "", true, capacity)
		require.NoError(t, err)

		stateA := newState("bucket", "a", "etag", time.Unix(1000, 0))
		stateA.Stored = true
		stateB := newState("bucket", "b", "etag", time.Unix(2000, 0))
		stateB.Stored = true
		stateC := newState("bucket", "c", "etag", time.Unix(3000, 0))
		stateC.Stored = true
		stateD := newState("bucket", "d", "etag", time.Unix(4000, 0))
		stateD.Stored = true

		err = states.AddState(stateA)
		require.NoError(t, err)
		err = states.AddState(stateB)
		require.NoError(t, err)
		err = states.AddState(stateC)
		require.NoError(t, err)

		require.True(t, states.IsProcessed(stateA.IDWithLexicographicalOrdering()))
		require.True(t, states.IsProcessed(stateB.IDWithLexicographicalOrdering()))
		require.True(t, states.IsProcessed(stateC.IDWithLexicographicalOrdering()))

		// This should evict stateA (lexicographically oldest)
		err = states.AddState(stateD)
		require.NoError(t, err)

		require.False(t, states.IsProcessed(stateA.IDWithLexicographicalOrdering()))
		require.True(t, states.IsProcessed(stateB.IDWithLexicographicalOrdering()))
		require.True(t, states.IsProcessed(stateC.IDWithLexicographicalOrdering()))
		require.True(t, states.IsProcessed(stateD.IDWithLexicographicalOrdering()))

		ok, err := states.store.Has(getStoreKey(stateA.IDWithLexicographicalOrdering()))
		require.NoError(t, err)
		require.False(t, ok, "stateA should be removed from store")
	})

	t.Run("AddState updates existing state without eviction", func(t *testing.T) {
		store := openTestStatestore()
		capacity := 2
		states, err := newStates(logger, store, "", true, capacity)
		require.NoError(t, err)

		state1 := newState("bucket", "key1", "etag", time.Unix(1000, 0))
		state2 := newState("bucket", "key2", "etag", time.Unix(2000, 0))

		err = states.AddState(state1)
		require.NoError(t, err)
		err = states.AddState(state2)
		require.NoError(t, err)

		// Update state1 (should not trigger eviction)
		state1Updated := newState("bucket", "key1", "etag", time.Unix(1000, 0))
		state1Updated.Stored = true
		err = states.AddState(state1Updated)
		require.NoError(t, err)

		require.True(t, states.IsProcessed(state1.IDWithLexicographicalOrdering()))
		require.True(t, states.IsProcessed(state2.IDWithLexicographicalOrdering()))
		require.Equal(t, 2, len(states.states))
	})

	t.Run("GetOldestState returns head of linked list", func(t *testing.T) {
		store := openTestStatestore()
		states, err := newStates(logger, store, "", true, 10)
		require.NoError(t, err)

		stateA := newState("bucket", "a", "etag", time.Unix(1000, 0))
		stateB := newState("bucket", "b", "etag", time.Unix(2000, 0))
		stateC := newState("bucket", "c", "etag", time.Unix(3000, 0))

		err = states.AddState(stateC)
		require.NoError(t, err)
		err = states.AddState(stateA)
		require.NoError(t, err)
		err = states.AddState(stateB)
		require.NoError(t, err)

		oldest := states.GetOldestState()
		require.NotNil(t, oldest)
		require.Equal(t, "c", oldest.Key)
	})

	t.Run("CleanUp preserves newest state in lexicographical mode", func(t *testing.T) {
		store := openTestStatestore()
		states, err := newStates(logger, store, "", true, 10)
		require.NoError(t, err)

		stateA := newState("bucket", "a", "etag", time.Unix(1000, 0))
		stateA.Stored = true
		stateB := newState("bucket", "b", "etag", time.Unix(2000, 0))
		stateB.Stored = true
		stateC := newState("bucket", "c", "etag", time.Unix(3000, 0))
		stateC.Stored = true

		err = states.AddState(stateA)
		require.NoError(t, err)
		err = states.AddState(stateB)
		require.NoError(t, err)
		err = states.AddState(stateC)
		require.NoError(t, err)

		err = states.CleanUp([]string{})
		require.NoError(t, err)

		// stateC (lexicographically greatest) should be preserved
		// Atleast one state should be preserved for startAfterKey
		require.False(t, states.IsProcessed(stateA.IDWithLexicographicalOrdering()))
		require.False(t, states.IsProcessed(stateB.IDWithLexicographicalOrdering()))
		require.True(t, states.IsProcessed(stateC.IDWithLexicographicalOrdering()))
	})

	t.Run("CleanUp removes states not in known list", func(t *testing.T) {
		store := openTestStatestore()
		states, err := newStates(logger, store, "", true, 10)
		require.NoError(t, err)

		stateA := newState("bucket", "a", "etag", time.Unix(1000, 0))
		stateA.Stored = true
		stateB := newState("bucket", "b", "etag", time.Unix(2000, 0))
		stateB.Stored = true
		stateC := newState("bucket", "c", "etag", time.Unix(3000, 0))
		stateC.Stored = true

		err = states.AddState(stateA)
		require.NoError(t, err)
		err = states.AddState(stateB)
		require.NoError(t, err)
		err = states.AddState(stateC)
		require.NoError(t, err)

		err = states.CleanUp([]string{stateA.IDWithLexicographicalOrdering()})
		require.NoError(t, err)

		require.True(t, states.IsProcessed(stateA.IDWithLexicographicalOrdering()))
		require.False(t, states.IsProcessed(stateB.IDWithLexicographicalOrdering()))
		require.False(t, states.IsProcessed(stateC.IDWithLexicographicalOrdering()))
		require.Equal(t, 1, len(states.states))
	})

	t.Run("newStates trims loaded states to capacity", func(t *testing.T) {
		store := openTestStatestore()

		states1, err := newStates(logger, store, "", false, 0)
		require.NoError(t, err)

		stateA := newState("bucket", "a", "etag", time.Unix(1000, 0))
		stateA.Stored = true
		stateB := newState("bucket", "b", "etag", time.Unix(2000, 0))
		stateB.Stored = true
		stateC := newState("bucket", "c", "etag", time.Unix(3000, 0))
		stateC.Stored = true
		stateD := newState("bucket", "d", "etag", time.Unix(4000, 0))
		stateD.Stored = true

		// Store using lexicographical IDs to simulate previous lexicographical mode data
		err = states1.store.Set(getStoreKey(stateA.IDWithLexicographicalOrdering()), stateA)
		require.NoError(t, err)
		err = states1.store.Set(getStoreKey(stateB.IDWithLexicographicalOrdering()), stateB)
		require.NoError(t, err)
		err = states1.store.Set(getStoreKey(stateC.IDWithLexicographicalOrdering()), stateC)
		require.NoError(t, err)
		err = states1.store.Set(getStoreKey(stateD.IDWithLexicographicalOrdering()), stateD)
		require.NoError(t, err)

		// Now reload with lexicographical mode and capacity of 2
		capacity := 2
		states2, err := newStates(logger, store, "", true, capacity)
		require.NoError(t, err)

		// Should only have the 2 newest (lexicographically greatest) states: c and d
		require.Equal(t, capacity, len(states2.states))
		require.False(t, states2.IsProcessed(stateA.IDWithLexicographicalOrdering()))
		require.False(t, states2.IsProcessed(stateB.IDWithLexicographicalOrdering()))
		require.True(t, states2.IsProcessed(stateC.IDWithLexicographicalOrdering()))
		require.True(t, states2.IsProcessed(stateD.IDWithLexicographicalOrdering()))

		// Verify trimmed states are also removed from store
		ok, err := states2.store.Has(getStoreKey(stateA.IDWithLexicographicalOrdering()))
		require.NoError(t, err)
		require.False(t, ok, "stateA should be removed from store during trim")
		ok, err = states2.store.Has(getStoreKey(stateB.IDWithLexicographicalOrdering()))
		require.NoError(t, err)
		require.False(t, ok, "stateB should be removed from store during trim")
	})

	t.Run("SortStatesByLexicographicalOrdering rebuilds linked list", func(t *testing.T) {
		store := openTestStatestore()
		states, err := newStates(logger, store, "", true, 10)
		require.NoError(t, err)

		stateC := newState("bucket", "c", "etag", time.Unix(3000, 0))
		stateA := newState("bucket", "a", "etag", time.Unix(1000, 0))
		stateB := newState("bucket", "b", "etag", time.Unix(2000, 0))

		err = states.AddState(stateC)
		require.NoError(t, err)
		err = states.AddState(stateA)
		require.NoError(t, err)
		err = states.AddState(stateB)
		require.NoError(t, err)

		states.SortStatesByLexicographicalOrdering(logger)

		// After sorting, verify the linked list structure: nil <- (head) a <-> b <-> c (tail) -> nil
		require.NotNil(t, states.head)
		require.Equal(t, "a", states.head.Key)

		require.NotNil(t, states.tail)
		require.Equal(t, "c", states.tail.Key)

		require.NotNil(t, states.head.next)
		require.Equal(t, "b", states.head.next.Key)
		require.NotNil(t, states.head.next.next)
		require.Equal(t, "c", states.head.next.next.Key)
		require.Nil(t, states.head.next.next.next)

		require.Nil(t, states.head.prev)
		require.Equal(t, states.head, states.head.next.prev)
		require.Equal(t, states.head.next, states.tail.prev)
		require.Nil(t, states.tail.next)
	})
}

var _ statestore.States = (*testInputStore)(nil)

type testInputStore struct {
	registry *statestore.Registry
}

func openTestStatestore() statestore.States {
	return &testInputStore{
		registry: statestore.NewRegistry(storetest.NewMemoryStoreBackend()),
	}
}

func (s *testInputStore) Close() {
	_ = s.registry.Close()
}

func (s *testInputStore) StoreFor(string) (*statestore.Store, error) {
	return s.registry.Get("filebeat")
}

func (s *testInputStore) CleanupInterval() time.Duration {
	return 24 * time.Hour
}
