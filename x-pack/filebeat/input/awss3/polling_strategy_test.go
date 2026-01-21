// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp"
)

func TestNewPollingStrategy(t *testing.T) {
	t.Run("returns normalPollingStrategy when lexicographical ordering is false", func(t *testing.T) {
		strategy := newPollingStrategy(false)
		_, ok := strategy.(*normalPollingStrategy)
		assert.True(t, ok, "expected normalPollingStrategy")
	})

	t.Run("returns lexicographicalPollingStrategy when lexicographical ordering is true", func(t *testing.T) {
		strategy := newPollingStrategy(true)
		_, ok := strategy.(*lexicographicalPollingStrategy)
		assert.True(t, ok, "expected lexicographicalPollingStrategy")
	})
}

func TestNormalPollingStrategy(t *testing.T) {
	strategy := newNormalPollingStrategy()
	log := logp.NewLogger("normal_polling_strategy_test")

	t.Run("PrePollSetup does nothing", func(t *testing.T) {
		store := openTestStatestore()
		registry, err := newStateRegistry(nil, store, "", false, 0)
		require.NoError(t, err)

		strategy.PrePollSetup(log, registry)
	})

	t.Run("GetStartAfterKey returns empty string", func(t *testing.T) {
		store := openTestStatestore()
		registry, err := newStateRegistry(nil, store, "", false, 0)
		require.NoError(t, err)

		st := state{Bucket: "bucket", Key: "key1", Etag: "etag1", LastModified: time.Now()}
		err = registry.AddState(st)
		require.NoError(t, err)

		startKey := strategy.GetStartAfterKey(registry)
		assert.Empty(t, startKey, "normal mode should always return empty startAfterKey")
	})

	t.Run("ShouldSkipObject respects validity filter", func(t *testing.T) {
		st := state{Bucket: "bucket", Key: "key1", Etag: "etag1", LastModified: time.Now()}

		acceptAll := func(log *logp.Logger, s state) bool { return true }
		assert.False(t, strategy.ShouldSkipObject(log, st, acceptAll), "should not skip valid objects")

		rejectAll := func(log *logp.Logger, s state) bool { return false }
		assert.True(t, strategy.ShouldSkipObject(log, st, rejectAll), "should skip invalid objects")
	})

	t.Run("GetStateID returns ID without lexicographical suffix", func(t *testing.T) {
		st := state{Bucket: "bucket", Key: "key1", Etag: "etag1", LastModified: time.Now()}

		id := strategy.GetStateID(st)
		assert.Equal(t, st.ID(), id)
		assert.NotContains(t, id, "::lexicographical")
	})

	t.Run("ShouldSkipProcessed returns true for processed objects", func(t *testing.T) {
		store := openTestStatestore()
		registry, err := newStateRegistry(nil, store, "", false, 0)
		require.NoError(t, err)

		st := state{Bucket: "bucket", Key: "key1", Etag: "etag1", LastModified: time.Now(), Stored: true}
		err = registry.AddState(st)
		require.NoError(t, err)

		assert.True(t, strategy.ShouldSkipProcessed(registry, st.ID()), "should skip processed objects")
	})

	t.Run("ShouldSkipProcessed returns false for unprocessed objects", func(t *testing.T) {
		store := openTestStatestore()
		registry, err := newStateRegistry(nil, store, "", false, 0)
		require.NoError(t, err)

		assert.False(t, strategy.ShouldSkipProcessed(registry, "unknown-id"), "should not skip unprocessed objects")
	})
}

func TestLexicographicalPollingStrategy(t *testing.T) {
	strategy := newLexicographicalPollingStrategy()
	log := logp.NewLogger("lexicographical_polling_strategy_test")

	t.Run("PrePollSetup sorts states", func(t *testing.T) {
		store := openTestStatestore()
		registry, err := newStateRegistry(log, store, "", true, 100)
		require.NoError(t, err)

		lexicoRegistry := registry.(*lexicographicalStateRegistry)

		// Add states in random order
		st1 := state{Bucket: "bucket", Key: "key-c", Etag: "etag1", LastModified: time.Now()}
		st2 := state{Bucket: "bucket", Key: "key-a", Etag: "etag2", LastModified: time.Now()}
		st3 := state{Bucket: "bucket", Key: "key-b", Etag: "etag3", LastModified: time.Now()}

		err = registry.AddState(st1)
		require.NoError(t, err)
		err = registry.AddState(st2)
		require.NoError(t, err)
		err = registry.AddState(st3)
		require.NoError(t, err)

		strategy.PrePollSetup(log, registry)

		// Verify sorting occurred ("key-a" should be head, "key-c" should be tail)
		require.NotNil(t, lexicoRegistry.head)
		require.NotNil(t, lexicoRegistry.tail)
		assert.Equal(t, "key-a", lexicoRegistry.head.Key, "head should be lexicographically smallest")
		assert.Equal(t, "key-c", lexicoRegistry.tail.Key, "tail should be lexicographically largest")

		// Verify traversal order
		keys := []string{}
		for current := lexicoRegistry.head; current != nil; current = current.next {
			keys = append(keys, current.Key)
		}
		assert.Equal(t, []string{"key-a", "key-b", "key-c"}, keys)
	})

	t.Run("GetStartAfterKey returns oldest state key", func(t *testing.T) {
		store := openTestStatestore()
		registry, err := newStateRegistry(log, store, "", true, 100)
		require.NoError(t, err)

		st1 := state{Bucket: "bucket", Key: "key-oldest", Etag: "etag1", LastModified: time.Now()}
		st2 := state{Bucket: "bucket", Key: "key-newest", Etag: "etag2", LastModified: time.Now()}

		err = registry.AddState(st1)
		require.NoError(t, err)
		err = registry.AddState(st2)
		require.NoError(t, err)

		// Should return head (key-oldest)
		startKey := strategy.GetStartAfterKey(registry)
		assert.Equal(t, "key-oldest", startKey)
	})

	t.Run("GetStartAfterKey returns empty string when no states", func(t *testing.T) {
		store := openTestStatestore()
		registry, err := newStateRegistry(log, store, "", true, 100)
		require.NoError(t, err)

		startKey := strategy.GetStartAfterKey(registry)
		assert.Empty(t, startKey)
	})

	t.Run("ShouldSkipObject always returns false", func(t *testing.T) {
		st := state{Bucket: "bucket", Key: "key1", Etag: "etag1", LastModified: time.Now()}

		rejectAll := func(log *logp.Logger, s state) bool { return false }
		assert.False(t, strategy.ShouldSkipObject(log, st, rejectAll), "lexicographical mode should never skip objects based on filter")
	})

	t.Run("GetStateID returns ID with lexicographical suffix", func(t *testing.T) {
		st := state{Bucket: "bucket", Key: "key1", Etag: "etag1", LastModified: time.Now()}

		id := strategy.GetStateID(st)
		assert.Equal(t, st.IDWithLexicographicalOrdering(), id)
		assert.Contains(t, id, "::lexicographical")
	})

	t.Run("ShouldSkipProcessed always returns false", func(t *testing.T) {
		store := openTestStatestore()
		registry, err := newStateRegistry(log, store, "", true, 100)
		require.NoError(t, err)

		st := state{Bucket: "bucket", Key: "key1", Etag: "etag1", LastModified: time.Now(), Stored: true}
		err = registry.AddState(st)
		require.NoError(t, err)

		// lexicographical mode should never skip
		assert.False(t, strategy.ShouldSkipProcessed(registry, st.IDWithLexicographicalOrdering()), "lexicographical mode should never skip based on processed status")
	})
}

func TestPollingStrategyBehaviorDifferences(t *testing.T) {
	normalStrategy := newNormalPollingStrategy()
	lexicoStrategy := newLexicographicalPollingStrategy()
	log := logp.NewLogger("polling_strategy_behavior_differences_test")

	t.Run("StartAfterKey behavior differs", func(t *testing.T) {
		normalStore := openTestStatestore()
		lexicoStore := openTestStatestore()

		normalRegistry, err := newStateRegistry(log, normalStore, "", false, 0)
		require.NoError(t, err)

		lexicoRegistry, err := newStateRegistry(log, lexicoStore, "", true, 100)
		require.NoError(t, err)

		st1 := state{Bucket: "bucket", Key: "key1", Etag: "etag1", LastModified: time.Now()}
		st2 := state{Bucket: "bucket", Key: "key2", Etag: "etag2", LastModified: time.Now()}

		err = normalRegistry.AddState(st1)
		require.NoError(t, err)
		err = lexicoRegistry.AddState(st1)
		require.NoError(t, err)
		err = normalRegistry.AddState(st2)
		require.NoError(t, err)
		err = lexicoRegistry.AddState(st2)
		require.NoError(t, err)

		assert.Empty(t, normalStrategy.GetStartAfterKey(normalRegistry))
		assert.Equal(t, "key1", lexicoStrategy.GetStartAfterKey(lexicoRegistry))
	})

	t.Run("Object filtering behavior differs", func(t *testing.T) {
		st := state{Bucket: "bucket", Key: "key1", Etag: "etag1", LastModified: time.Now()}
		rejectAll := func(log *logp.Logger, s state) bool { return false }

		assert.True(t, normalStrategy.ShouldSkipObject(log, st, rejectAll))
		// Lexicographical mode ignores filter
		assert.False(t, lexicoStrategy.ShouldSkipObject(log, st, rejectAll))
	})

	t.Run("State ID format differs", func(t *testing.T) {
		st := state{Bucket: "bucket", Key: "key1", Etag: "etag1", LastModified: time.Now()}

		normalID := normalStrategy.GetStateID(st)
		lexicoID := lexicoStrategy.GetStateID(st)

		assert.NotEqual(t, normalID, lexicoID, "IDs should be different between modes")
		assert.NotContains(t, normalID, "::lexicographical")
		assert.Contains(t, lexicoID, "::lexicographical")
	})

	t.Run("Processed object skip handling differs", func(t *testing.T) {
		normalStore := openTestStatestore()
		lexicoStore := openTestStatestore()

		normalRegistry, err := newStateRegistry(log, normalStore, "", false, 0)
		require.NoError(t, err)

		lexicoRegistry, err := newStateRegistry(log, lexicoStore, "", true, 100)
		require.NoError(t, err)

		st := state{Bucket: "bucket", Key: "key1", Etag: "etag1", LastModified: time.Now(), Stored: true}

		err = normalRegistry.AddState(st)
		require.NoError(t, err)
		err = lexicoRegistry.AddState(st)
		require.NoError(t, err)

		assert.True(t, normalStrategy.ShouldSkipProcessed(normalRegistry, st.ID()))
		// Lexicographical mode doesn't skip processed
		assert.False(t, lexicoStrategy.ShouldSkipProcessed(lexicoRegistry, st.IDWithLexicographicalOrdering()))
	})
}
