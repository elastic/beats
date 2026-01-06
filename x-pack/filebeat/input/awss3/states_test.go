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

// ============================================================================
// Tests for normalStateRegistry
// ============================================================================

func TestNormalStateRegistry_AddStateAndIsProcessed(t *testing.T) {
	type stateTestCase struct {
		// An initialization callback to invoke on the (initially empty) states.
		statesEdit func(registry stateRegistry) error

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
		"with empty registry": {
			state:               testState1,
			expectedIsProcessed: false,
		},
		"not existing state": {
			statesEdit: func(registry stateRegistry) error {
				return registry.AddState(testState2)
			},
			state:               testState1,
			expectedIsProcessed: false,
		},
		"existing state": {
			statesEdit: func(registry stateRegistry) error {
				return registry.AddState(testState1)
			},
			state:               testState1,
			expectedIsProcessed: true,
		},
		"existing stored state is persisted": {
			statesEdit: func(registry stateRegistry) error {
				state := testState1
				state.Stored = true
				return registry.AddState(state)
			},
			state:               testState1,
			shouldReload:        true,
			expectedIsProcessed: true,
		},
		"existing failed state is persisted": {
			statesEdit: func(registry stateRegistry) error {
				state := testState1
				state.Failed = true
				return registry.AddState(state)
			},
			state:               testState1,
			shouldReload:        true,
			expectedIsProcessed: true,
		},
		"existing unprocessed state is not persisted": {
			statesEdit: func(registry stateRegistry) error {
				return registry.AddState(testState1)
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
			registry, err := newStateRegistry(nil, store, "", false, 0)
			require.NoError(t, err, "registry creation must succeed")
			if test.statesEdit != nil {
				err = test.statesEdit(registry)
				require.NoError(t, err, "registry edit must succeed")
			}
			if test.shouldReload {
				registry, err = newStateRegistry(nil, store, "", false, 0)
				require.NoError(t, err, "registry creation must succeed")
			}

			isProcessed := registry.IsProcessed(test.state.ID())
			assert.Equal(t, test.expectedIsProcessed, isProcessed)
		})
	}
}

func TestNormalStateRegistry_CleanUp(t *testing.T) {
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
			registry, err := newStateRegistry(nil, store, "", false, 0)
			require.NoError(t, err, "registry creation must succeed")

			for _, s := range test.initStates {
				err := registry.AddState(s)
				require.NoError(t, err, "state initialization must succeed")
			}

			// perform cleanup
			err = registry.CleanUp(test.knownIDs)
			require.NoError(t, err, "state cleanup must succeed")

			// validate
			normalRegistry, ok := registry.(*normalStateRegistry)
			require.True(t, ok, "expected normalStateRegistry type")
			for _, id := range test.expectIDs {
				// must be in local state
				_, ok := normalRegistry.states[id]
				require.True(t, ok, fmt.Errorf("expected id %s in state, but got missing", id))
				// must be in store
				ok, err = normalRegistry.store.Has(getStoreKey(id))
				require.NoError(t, err, "state has must succeed")
				require.True(t, ok, fmt.Errorf("expected id %s in store, but got missing", id))
			}
		})
	}
}

func TestNormalStateRegistry_PrefixHandling(t *testing.T) {
	logger := logp.NewLogger("state-prefix-testing")

	t.Run("if prefix was set, accept only states with prefix", func(t *testing.T) {
		// given
		store := openTestStatestore()

		// when - registry with prefix
		registry, err := newStateRegistry(logger, store, "staging-", false, 0)
		require.NoError(t, err)

		// then - fail for non prefixed
		err = registry.AddState(newState("bucket", "production-logA", "etag", time.Now()))
		require.Error(t, err)

		// then - pass for correctly prefixed
		err = registry.AddState(newState("bucket", "staging-logA", "etag", time.Now()))
		require.NoError(t, err)
	})

	t.Run("registry only loads entries matching the given prefix", func(t *testing.T) {
		// given
		store := openTestStatestore()

		sA := newState("bucket", "A", "etag", time.Unix(1733221244, 0))
		sA.Stored = true
		sStagingA := newState("bucket", "staging-A", "etag", time.Unix(1733224844, 0))
		sStagingA.Stored = true
		sProdB := newState("bucket", "production/B", "etag", time.Unix(1733228444, 0))
		sProdB.Stored = true
		sSpace := newState("bucket", "  B", "etag", time.Unix(1733230444, 0))
		sSpace.Stored = true

		// add various states first with no prefix
		registry, err := newStateRegistry(logger, store, "", false, 0)
		require.NoError(t, err)

		_ = registry.AddState(sA)
		_ = registry.AddState(sStagingA)
		_ = registry.AddState(sProdB)
		_ = registry.AddState(sSpace)

		// Reload states and validate

		// when - no prefix reload
		stNoPrefix, err := newStateRegistry(logger, store, "", false, 0)
		require.NoError(t, err)

		require.True(t, stNoPrefix.IsProcessed(sA.ID()))
		require.True(t, stNoPrefix.IsProcessed(sStagingA.ID()))
		require.True(t, stNoPrefix.IsProcessed(sProdB.ID()))
		require.True(t, stNoPrefix.IsProcessed(sSpace.ID()))

		// when - with prefix `staging-`
		stStaging, err := newStateRegistry(logger, store, "staging-", false, 0)
		require.NoError(t, err)

		require.False(t, stStaging.IsProcessed(sA.ID()))
		require.True(t, stStaging.IsProcessed(sStagingA.ID()))
		require.False(t, stStaging.IsProcessed(sProdB.ID()))
		require.False(t, stStaging.IsProcessed(sSpace.ID()))

		// when - with prefix `production/`
		stProd, err := newStateRegistry(logger, store, "production/", false, 0)
		require.NoError(t, err)

		require.False(t, stProd.IsProcessed(sA.ID()))
		require.False(t, stProd.IsProcessed(sStagingA.ID()))
		require.True(t, stProd.IsProcessed(sProdB.ID()))
		require.False(t, stProd.IsProcessed(sSpace.ID()))
	})
}

func TestNormalStateRegistry_GetOldestState(t *testing.T) {
	store := openTestStatestore()
	registry, err := newStateRegistry(nil, store, "", false, 0)
	require.NoError(t, err)

	// Normal mode doesn't use startAfterKey
	startAfterKey := registry.GetStartAfterKey()
	require.Empty(t, startAfterKey, "normal registry should return empty string for GetStartAfterKey")
}

// ============================================================================
// Tests for lexicographicalStateRegistry
// ============================================================================

func TestLexicographicalStateRegistry_AddStateAndIsProcessed(t *testing.T) {
	logger := logp.NewLogger("lexicographical-registry-test")

	t.Run("AddState uses IDWithLexicographicalOrdering", func(t *testing.T) {
		store := openTestStatestore()
		registry, err := newStateRegistry(logger, store, "", true, 10)
		require.NoError(t, err)

		state1 := newState("bucket", "key1", "etag1", time.Unix(1000, 0))
		state1.Stored = true
		err = registry.AddState(state1)
		require.NoError(t, err)

		require.True(t, registry.IsProcessed(state1.IDWithLexicographicalOrdering()))
		require.False(t, registry.IsProcessed(state1.ID()))
	})

	t.Run("AddState evicts oldest state when at capacity", func(t *testing.T) {
		store := openTestStatestore()
		capacity := 3
		registry, err := newStateRegistry(logger, store, "", true, capacity)
		require.NoError(t, err)

		lexicoRegistry := registry.(*lexicographicalStateRegistry) //nolint:errcheck // type assertion is safe in test

		stateA := newState("bucket", "a", "etag", time.Unix(1000, 0))
		stateA.Stored = true
		stateB := newState("bucket", "b", "etag", time.Unix(2000, 0))
		stateB.Stored = true
		stateC := newState("bucket", "c", "etag", time.Unix(3000, 0))
		stateC.Stored = true
		stateD := newState("bucket", "d", "etag", time.Unix(4000, 0))
		stateD.Stored = true

		err = registry.AddState(stateA)
		require.NoError(t, err)
		err = registry.AddState(stateB)
		require.NoError(t, err)
		err = registry.AddState(stateC)
		require.NoError(t, err)

		require.True(t, registry.IsProcessed(stateA.IDWithLexicographicalOrdering()))
		require.True(t, registry.IsProcessed(stateB.IDWithLexicographicalOrdering()))
		require.True(t, registry.IsProcessed(stateC.IDWithLexicographicalOrdering()))

		// This should evict stateA (lexicographically oldest)
		err = registry.AddState(stateD)
		require.NoError(t, err)

		require.False(t, registry.IsProcessed(stateA.IDWithLexicographicalOrdering()))
		require.True(t, registry.IsProcessed(stateB.IDWithLexicographicalOrdering()))
		require.True(t, registry.IsProcessed(stateC.IDWithLexicographicalOrdering()))
		require.True(t, registry.IsProcessed(stateD.IDWithLexicographicalOrdering()))

		ok, err := lexicoRegistry.store.Has(getStoreKey(stateA.IDWithLexicographicalOrdering()))
		require.NoError(t, err)
		require.False(t, ok, "stateA should be removed from store")
	})

}

func TestLexicographicalStateRegistry_TailTracking(t *testing.T) {
	logger := logp.NewLogger("lexicographical-registry-test")

	t.Run("GetStartAfterKey returns persisted tail", func(t *testing.T) {
		store := openTestStatestore()
		registry, err := newStateRegistry(logger, store, "", true, 10)
		require.NoError(t, err)

		// Initially no tail
		require.Empty(t, registry.GetStartAfterKey())

		// Mark object in-flight - tail should be set
		err = registry.MarkObjectInFlight("b")
		require.NoError(t, err)
		require.Equal(t, "b", registry.GetStartAfterKey())

		// Mark smaller key in-flight - tail should update
		err = registry.MarkObjectInFlight("a")
		require.NoError(t, err)
		require.Equal(t, "a", registry.GetStartAfterKey())

		// Unmark "a" - tail should move to "b"
		err = registry.UnmarkObjectInFlight("a")
		require.NoError(t, err)
		require.Equal(t, "b", registry.GetStartAfterKey())

		// Unmark "b" - no more in-flight, tail should be empty
		err = registry.UnmarkObjectInFlight("b")
		require.NoError(t, err)
		require.Empty(t, registry.GetStartAfterKey())
	})

	t.Run("Tail considers both in-flight and completed states", func(t *testing.T) {
		store := openTestStatestore()
		registry, err := newStateRegistry(logger, store, "", true, 10)
		require.NoError(t, err)

		// Add completed state
		stateC := newState("bucket", "c", "etag", time.Unix(1000, 0))
		stateC.Stored = true
		err = registry.AddState(stateC)
		require.NoError(t, err)

		// Mark "a" in-flight (smaller than "c")
		err = registry.MarkObjectInFlight("a")
		require.NoError(t, err)
		require.Equal(t, "a", registry.GetStartAfterKey())

		// Unmark "a" - tail should now be "c" (smallest completed)
		err = registry.UnmarkObjectInFlight("a")
		require.NoError(t, err)
		require.Equal(t, "c", registry.GetStartAfterKey())
	})

	t.Run("Tail persists and survives registry reload", func(t *testing.T) {
		store := openTestStatestore()

		// Create first registry and set a tail
		registry1, err := newStateRegistry(logger, store, "", true, 10)
		require.NoError(t, err)
		err = registry1.MarkObjectInFlight("a")
		require.NoError(t, err)
		require.Equal(t, "a", registry1.GetStartAfterKey())

		// Create new registry from same store - should load persisted tail
		registry2, err := newStateRegistry(logger, store, "", true, 10)
		require.NoError(t, err)
		require.Equal(t, "a", registry2.GetStartAfterKey())
	})
}

func TestLexicographicalStateRegistry_CleanUp(t *testing.T) {
	logger := logp.NewLogger("lexicographical-registry-test")

	t.Run("CleanUp preserves newest state", func(t *testing.T) {
		store := openTestStatestore()
		registry, err := newStateRegistry(logger, store, "", true, 10)
		require.NoError(t, err)

		stateA := newState("bucket", "a", "etag", time.Unix(1000, 0))
		stateA.Stored = true
		stateB := newState("bucket", "b", "etag", time.Unix(2000, 0))
		stateB.Stored = true
		stateC := newState("bucket", "c", "etag", time.Unix(3000, 0))
		stateC.Stored = true

		err = registry.AddState(stateA)
		require.NoError(t, err)
		err = registry.AddState(stateB)
		require.NoError(t, err)
		err = registry.AddState(stateC)
		require.NoError(t, err)

		err = registry.CleanUp([]string{})
		require.NoError(t, err)

		// stateC (lexicographically greatest) should be preserved
		// Atleast one state should be preserved for startAfterKey
		require.False(t, registry.IsProcessed(stateA.IDWithLexicographicalOrdering()))
		require.False(t, registry.IsProcessed(stateB.IDWithLexicographicalOrdering()))
		require.True(t, registry.IsProcessed(stateC.IDWithLexicographicalOrdering()))
	})

	t.Run("CleanUp removes states not in known list", func(t *testing.T) {
		store := openTestStatestore()
		registry, err := newStateRegistry(logger, store, "", true, 10)
		require.NoError(t, err)

		lexicoRegistry := registry.(*lexicographicalStateRegistry) //nolint:errcheck // type assertion is safe in test

		stateA := newState("bucket", "a", "etag", time.Unix(1000, 0))
		stateA.Stored = true
		stateB := newState("bucket", "b", "etag", time.Unix(2000, 0))
		stateB.Stored = true
		stateC := newState("bucket", "c", "etag", time.Unix(3000, 0))
		stateC.Stored = true

		err = registry.AddState(stateA)
		require.NoError(t, err)
		err = registry.AddState(stateB)
		require.NoError(t, err)
		err = registry.AddState(stateC)
		require.NoError(t, err)

		err = registry.CleanUp([]string{stateA.IDWithLexicographicalOrdering()})
		require.NoError(t, err)

		require.True(t, registry.IsProcessed(stateA.IDWithLexicographicalOrdering()))
		require.False(t, registry.IsProcessed(stateB.IDWithLexicographicalOrdering()))
		require.False(t, registry.IsProcessed(stateC.IDWithLexicographicalOrdering()))
		require.Equal(t, 1, len(lexicoRegistry.states))
	})
}

func TestLexicographicalStateRegistry_TrimsOnLoad(t *testing.T) {
	logger := logp.NewLogger("lexicographical-registry-test")
	store := openTestStatestore()

	registry1, err := newStateRegistry(logger, store, "", false, 0)
	require.NoError(t, err)

	normalRegistry1 := registry1.(*normalStateRegistry) //nolint:errcheck // type assertion is safe in test

	stateA := newState("bucket", "a", "etag", time.Unix(1000, 0))
	stateA.Stored = true
	stateB := newState("bucket", "b", "etag", time.Unix(2000, 0))
	stateB.Stored = true
	stateC := newState("bucket", "c", "etag", time.Unix(3000, 0))
	stateC.Stored = true
	stateD := newState("bucket", "d", "etag", time.Unix(4000, 0))
	stateD.Stored = true

	// Store using lexicographical IDs to simulate previous lexicographical mode data
	err = normalRegistry1.store.Set(getStoreKey(stateA.IDWithLexicographicalOrdering()), stateA)
	require.NoError(t, err)
	err = normalRegistry1.store.Set(getStoreKey(stateB.IDWithLexicographicalOrdering()), stateB)
	require.NoError(t, err)
	err = normalRegistry1.store.Set(getStoreKey(stateC.IDWithLexicographicalOrdering()), stateC)
	require.NoError(t, err)
	err = normalRegistry1.store.Set(getStoreKey(stateD.IDWithLexicographicalOrdering()), stateD)
	require.NoError(t, err)

	// Now reload with lexicographical mode and capacity of 2
	capacity := 2
	registry2, err := newStateRegistry(logger, store, "", true, capacity)
	require.NoError(t, err)

	lexicoRegistry2 := registry2.(*lexicographicalStateRegistry) //nolint:errcheck // type assertion is safe in test

	// Should only have the 2 newest (lexicographically greatest) states: c and d
	require.Equal(t, capacity, len(lexicoRegistry2.states))
	require.False(t, registry2.IsProcessed(stateA.IDWithLexicographicalOrdering()))
	require.False(t, registry2.IsProcessed(stateB.IDWithLexicographicalOrdering()))
	require.True(t, registry2.IsProcessed(stateC.IDWithLexicographicalOrdering()))
	require.True(t, registry2.IsProcessed(stateD.IDWithLexicographicalOrdering()))

	// Verify trimmed states are also removed from store
	ok, err := lexicoRegistry2.store.Has(getStoreKey(stateA.IDWithLexicographicalOrdering()))
	require.NoError(t, err)
	require.False(t, ok, "stateA should be removed from store during trim")
	ok, err = lexicoRegistry2.store.Has(getStoreKey(stateB.IDWithLexicographicalOrdering()))
	require.NoError(t, err)
	require.False(t, ok, "stateB should be removed from store during trim")
}

func TestLexicographicalStateRegistry_HeapOrder(t *testing.T) {
	logger := logp.NewLogger("lexicographical-registry-test")
	store := openTestStatestore()
	registry, err := newStateRegistry(logger, store, "", true, 10)
	require.NoError(t, err)

	// Add states in non-sorted order
	stateC := newState("bucket", "c", "etag", time.Unix(3000, 0))
	stateA := newState("bucket", "a", "etag", time.Unix(1000, 0))
	stateB := newState("bucket", "b", "etag", time.Unix(2000, 0))

	err = registry.AddState(stateC)
	require.NoError(t, err)
	err = registry.AddState(stateA)
	require.NoError(t, err)
	err = registry.AddState(stateB)
	require.NoError(t, err)

	// After adding completed states, the tail should be computed from them
	// Mark an object in-flight to establish a tail
	err = registry.MarkObjectInFlight("z")
	require.NoError(t, err)
	// Unmark it - now tail should be the smallest completed key
	err = registry.UnmarkObjectInFlight("z")
	require.NoError(t, err)
	require.Equal(t, "a", registry.GetStartAfterKey(), "GetStartAfterKey should return lexicographically smallest key")
}

// ============================================================================
// Tests for newStateRegistry factory function
// ============================================================================

func TestNewStateRegistry(t *testing.T) {
	t.Run("returns normalStateRegistry when lexicographical ordering is false", func(t *testing.T) {
		store := openTestStatestore()
		registry, err := newStateRegistry(nil, store, "", false, 0)
		require.NoError(t, err)

		_, ok := registry.(*normalStateRegistry)
		assert.True(t, ok, "expected normalStateRegistry")
	})

	t.Run("returns lexicographicalStateRegistry when lexicographical ordering is true", func(t *testing.T) {
		store := openTestStatestore()
		registry, err := newStateRegistry(nil, store, "", true, 10)
		require.NoError(t, err)

		_, ok := registry.(*lexicographicalStateRegistry)
		assert.True(t, ok, "expected lexicographicalStateRegistry")
	})
}

// ============================================================================
// Tests documenting behavioral differences between implementations
// ============================================================================

func TestStateRegistryBehaviorDifferences(t *testing.T) {
	logger := logp.NewLogger("state-registry-diff-test")

	t.Run("ID format differs between implementations", func(t *testing.T) {
		normalStore := openTestStatestore()
		lexicoStore := openTestStatestore()

		normalRegistry, err := newStateRegistry(logger, normalStore, "", false, 0)
		require.NoError(t, err)

		lexicoRegistry, err := newStateRegistry(logger, lexicoStore, "", true, 10)
		require.NoError(t, err)

		state := newState("bucket", "key", "etag", time.Unix(1000, 0))
		state.Stored = true

		err = normalRegistry.AddState(state)
		require.NoError(t, err)
		err = lexicoRegistry.AddState(state)
		require.NoError(t, err)

		require.True(t, normalRegistry.IsProcessed(state.ID()))
		require.False(t, normalRegistry.IsProcessed(state.IDWithLexicographicalOrdering()))

		require.False(t, lexicoRegistry.IsProcessed(state.ID()))
		require.True(t, lexicoRegistry.IsProcessed(state.IDWithLexicographicalOrdering()))
	})

	t.Run("GetStartAfterKey behavior differs", func(t *testing.T) {
		normalStore := openTestStatestore()
		lexicoStore := openTestStatestore()

		normalRegistry, err := newStateRegistry(logger, normalStore, "", false, 0)
		require.NoError(t, err)

		lexicoRegistry, err := newStateRegistry(logger, lexicoStore, "", true, 10)
		require.NoError(t, err)

		state := newState("bucket", "key", "etag", time.Unix(1000, 0))

		err = normalRegistry.AddState(state)
		require.NoError(t, err)
		err = lexicoRegistry.AddState(state)
		require.NoError(t, err)

		require.Empty(t, normalRegistry.GetStartAfterKey())
		// Mark and unmark to establish tail from completed state
		err = lexicoRegistry.MarkObjectInFlight("z")
		require.NoError(t, err)
		err = lexicoRegistry.UnmarkObjectInFlight("z")
		require.NoError(t, err)
		require.NotEmpty(t, lexicoRegistry.GetStartAfterKey())
	})

	t.Run("Capacity limiting only applies to lexicographical", func(t *testing.T) {
		normalStore := openTestStatestore()
		lexicoStore := openTestStatestore()

		normalRegistry, err := newStateRegistry(logger, normalStore, "", false, 2)
		require.NoError(t, err)

		lexicoRegistry, err := newStateRegistry(logger, lexicoStore, "", true, 2)
		require.NoError(t, err)

		stateA := newState("bucket", "a", "etag", time.Unix(1000, 0))
		stateB := newState("bucket", "b", "etag", time.Unix(2000, 0))
		stateC := newState("bucket", "c", "etag", time.Unix(3000, 0))

		for _, s := range []state{stateA, stateB, stateC} {
			err = normalRegistry.AddState(s)
			require.NoError(t, err)
			err = lexicoRegistry.AddState(s)
			require.NoError(t, err)
		}

		// Normal keeps all 3
		require.True(t, normalRegistry.IsProcessed(stateA.ID()))
		require.True(t, normalRegistry.IsProcessed(stateB.ID()))
		require.True(t, normalRegistry.IsProcessed(stateC.ID()))

		// Lexicographical removes A, keeps only 2
		require.False(t, lexicoRegistry.IsProcessed(stateA.IDWithLexicographicalOrdering()))
		require.True(t, lexicoRegistry.IsProcessed(stateB.IDWithLexicographicalOrdering()))
		require.True(t, lexicoRegistry.IsProcessed(stateC.IDWithLexicographicalOrdering()))
	})
}

func TestStatesStoreForRouting(t *testing.T) {
	logger := logp.NewLogger("states-store-routing-test")

	t.Run("lexicographical ordering passes aws-s3 to StoreFor", func(t *testing.T) {
		store := &trackingInputStore{
			testInputStore: testInputStore{
				registry: statestore.NewRegistry(storetest.NewMemoryStoreBackend()),
			},
		}

		_, err := newStateRegistry(logger, store, "", true, 10)
		require.NoError(t, err)

		require.Equal(t, inputName, store.lastStoreForType, "StoreFor should be called with input name when lexicographical ordering is enabled")
	})

	t.Run("non-lexicographical ordering passes empty string to StoreFor", func(t *testing.T) {
		store := &trackingInputStore{
			testInputStore: testInputStore{
				registry: statestore.NewRegistry(storetest.NewMemoryStoreBackend()),
			},
		}

		_, err := newStateRegistry(logger, store, "", false, 0)
		require.NoError(t, err)

		require.Equal(t, "", store.lastStoreForType, "StoreFor should be called with empty string when lexicographical ordering is disabled")
	})
}

var _ statestore.States = (*testInputStore)(nil)

type testInputStore struct {
	registry *statestore.Registry
}

// trackingInputStore wraps testInputStore to track StoreFor calls
type trackingInputStore struct {
	testInputStore
	lastStoreForType string
}

func (s *trackingInputStore) StoreFor(typ string) (*statestore.Store, error) {
	s.lastStoreForType = typ
	return s.testInputStore.StoreFor(typ)
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
