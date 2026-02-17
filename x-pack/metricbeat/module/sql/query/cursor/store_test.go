// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cursor

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/backend/memlog"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"
)

func TestGenerateStateKey(t *testing.T) {
	tests := []struct {
		name         string
		inputType    string
		host         string
		query        string
		cursorColumn string
		direction    string
	}{
		{
			name:         "basic asc",
			inputType:    "sql",
			host:         "localhost:5432",
			query:        "SELECT * FROM logs WHERE id > :cursor",
			cursorColumn: "id",
			direction:    "asc",
		},
		{
			name:         "basic desc",
			inputType:    "sql",
			host:         "localhost:5432",
			query:        "SELECT * FROM logs WHERE id > :cursor",
			cursorColumn: "id",
			direction:    "desc",
		},
		{
			name:         "different host",
			inputType:    "sql",
			host:         "remotehost:5432",
			query:        "SELECT * FROM logs WHERE id > :cursor",
			cursorColumn: "id",
			direction:    "asc",
		},
		{
			name:         "different query",
			inputType:    "sql",
			host:         "localhost:5432",
			query:        "SELECT * FROM events WHERE id > :cursor",
			cursorColumn: "id",
			direction:    "asc",
		},
		{
			name:         "different column",
			inputType:    "sql",
			host:         "localhost:5432",
			query:        "SELECT * FROM logs WHERE id > :cursor",
			cursorColumn: "event_id",
			direction:    "asc",
		},
	}

	// Generate keys and ensure they're unique
	keys := make(map[string]string)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := GenerateStateKey(tt.inputType, tt.host, tt.query, tt.cursorColumn, tt.direction)

			// Key should have expected prefix
			assert.Contains(t, key, "sql-cursor::")

			// Key should be hex formatted
			assert.Regexp(t, `^sql-cursor::[0-9a-f]+$`, key)

			// Keys should be unique for different inputs
			identifier := tt.inputType + tt.host + tt.query + tt.cursorColumn + tt.direction
			if existingKey, exists := keys[identifier]; exists {
				assert.Equal(t, existingKey, key, "same inputs should produce same key")
			} else {
				keys[identifier] = key
			}
		})
	}

	// Verify different inputs produce different keys
	key1 := GenerateStateKey("sql", "host1", "query", "col", "asc")
	key2 := GenerateStateKey("sql", "host2", "query", "col", "asc")
	assert.NotEqual(t, key1, key2, "different hosts should produce different keys")

	// Verify direction changes the key
	key3 := GenerateStateKey("sql", "host", "query", "col", "asc")
	key4 := GenerateStateKey("sql", "host", "query", "col", "desc")
	assert.NotEqual(t, key3, key4, "changing direction should change the key")

	// Verify whitespace in query changes the key (no normalization)
	key5 := GenerateStateKey("sql", "host", "SELECT * FROM logs", "col", "asc")
	key6 := GenerateStateKey("sql", "host", "SELECT  *  FROM  logs", "col", "asc")
	assert.NotEqual(t, key5, key6, "whitespace differences should produce different keys")
}

func TestStoreOperations(t *testing.T) {
	// Skip if running in short mode
	if testing.Short() {
		t.Skip("skipping store test in short mode")
	}

	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Create paths configuration pointing to temp dir
	beatPaths := &paths.Path{
		Home:   tmpDir,
		Config: tmpDir,
		Data:   tmpDir,
		Logs:   tmpDir,
	}

	logger := logp.NewLogger("test-cursor-store")

	// Test store creation
	store, err := newStore(beatPaths, logger)
	require.NoError(t, err)
	require.NotNil(t, store)
	defer store.Close()

	// Test saving state
	testKey := "test-key"
	testState := &State{
		Version:     StateVersion,
		CursorType:  CursorTypeInteger,
		CursorValue: "12345",
		UpdatedAt:   time.Now().UTC(),
	}

	err = store.Save(testKey, testState)
	require.NoError(t, err)

	// Test loading state
	loadedState, err := store.Load(testKey)
	require.NoError(t, err)
	require.NotNil(t, loadedState)
	assert.Equal(t, testState.Version, loadedState.Version)
	assert.Equal(t, testState.CursorType, loadedState.CursorType)
	assert.Equal(t, testState.CursorValue, loadedState.CursorValue)

	// Test loading non-existent key
	missingState, err := store.Load("non-existent-key")
	require.NoError(t, err)
	assert.Nil(t, missingState)

	// Test updating state
	testState.CursorValue = "67890"
	err = store.Save(testKey, testState)
	require.NoError(t, err)

	loadedState, err = store.Load(testKey)
	require.NoError(t, err)
	require.NotNil(t, loadedState)
	assert.Equal(t, "67890", loadedState.CursorValue)
}

func TestNewStoreFromRegistry(t *testing.T) {
	// Skip if running in short mode
	if testing.Short() {
		t.Skip("skipping store test in short mode")
	}

	tmpDir := t.TempDir()

	// Create paths configuration pointing to temp dir
	beatPaths := &paths.Path{
		Home:   tmpDir,
		Config: tmpDir,
		Data:   tmpDir,
		Logs:   tmpDir,
	}

	logger := logp.NewLogger("test-cursor-store-shared")
	dataPath := beatPaths.Resolve(paths.Data, "sql-cursor")

	// Create a shared memlog registry (simulating what ModuleBuilder does)
	reg, err := memlog.New(logger.Named("memlog"), memlog.Settings{
		Root:     dataPath,
		FileMode: 0o600,
	})
	require.NoError(t, err)

	registry := statestore.NewRegistry(reg)
	defer registry.Close()

	// Create two stores from the same registry (simulating 2 MetricSets)
	store1, err := NewStoreFromRegistry(registry, logger.Named("store1"))
	require.NoError(t, err)
	require.NotNil(t, store1)

	store2, err := NewStoreFromRegistry(registry, logger.Named("store2"))
	require.NoError(t, err)
	require.NotNil(t, store2)

	// Store1 writes a key
	testState := &State{
		Version:     StateVersion,
		CursorType:  CursorTypeInteger,
		CursorValue: "100",
		UpdatedAt:   time.Now().UTC(),
	}
	err = store1.Save("key-from-store1", testState)
	require.NoError(t, err)

	// Store2 can read the same key (shared backend)
	loaded, err := store2.Load("key-from-store1")
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, "100", loaded.CursorValue)

	// Store2 writes a different key
	testState2 := &State{
		Version:     StateVersion,
		CursorType:  CursorTypeTimestamp,
		CursorValue: "2026-01-01T00:00:00Z",
		UpdatedAt:   time.Now().UTC(),
	}
	err = store2.Save("key-from-store2", testState2)
	require.NoError(t, err)

	// Store1 can read it
	loaded2, err := store1.Load("key-from-store2")
	require.NoError(t, err)
	require.NotNil(t, loaded2)
	assert.Equal(t, "2026-01-01T00:00:00Z", loaded2.CursorValue)

	// Close stores — should NOT close the shared registry
	require.NoError(t, store1.Close())
	require.NoError(t, store2.Close())

	// Registry is still usable (not closed by stores)
	store3, err := NewStoreFromRegistry(registry, logger.Named("store3"))
	require.NoError(t, err)
	require.NotNil(t, store3)

	// Can still read previously written data
	loaded3, err := store3.Load("key-from-store1")
	require.NoError(t, err)
	require.NotNil(t, loaded3)
	assert.Equal(t, "100", loaded3.CursorValue)

	require.NoError(t, store3.Close())
}

func TestStoreClose(t *testing.T) {
	// Skip if running in short mode
	if testing.Short() {
		t.Skip("skipping store test in short mode")
	}

	tmpDir := t.TempDir()
	beatPaths := &paths.Path{
		Home:   tmpDir,
		Config: tmpDir,
		Data:   tmpDir,
		Logs:   tmpDir,
	}

	logger := logp.NewLogger("test-cursor-store")

	store, err := newStore(beatPaths, logger)
	require.NoError(t, err)

	// First close should succeed
	err = store.Close()
	require.NoError(t, err)

	// Second close should also succeed (idempotent)
	err = store.Close()
	require.NoError(t, err)
}

func TestStoreOwnershipClosingBehavior(t *testing.T) {
	// Skip if running in short mode
	if testing.Short() {
		t.Skip("skipping store test in short mode")
	}

	t.Run("newStore closes registry when store closes", func(t *testing.T) {
		tmpDir := t.TempDir()
		beatPaths := &paths.Path{
			Home:   tmpDir,
			Config: tmpDir,
			Data:   tmpDir,
			Logs:   tmpDir,
		}

		logger := logp.NewLogger("test-ownership-owns")

		// Create store via newStore (owns registry)
		store, err := newStore(beatPaths, logger)
		require.NoError(t, err)
		require.NotNil(t, store)
		require.Equal(t, ownsRegistry, store.ownsRegistry)

		// Save some data
		testState := &State{
			Version:     StateVersion,
			CursorType:  CursorTypeInteger,
			CursorValue: "100",
			UpdatedAt:   time.Now().UTC(),
		}
		err = store.Save("test-key", testState)
		require.NoError(t, err)

		// Close the store (should close the registry)
		err = store.Close()
		require.NoError(t, err)

		// Verify the registry was closed by checking that store operations fail
		err = store.Save("another-key", testState)
		require.Error(t, err, "Store operations should fail after close")
		assert.Contains(t, err.Error(), "store is closed")
	})

	t.Run("NewStoreFromRegistry does NOT close registry when store closes", func(t *testing.T) {
		tmpDir := t.TempDir()
		beatPaths := &paths.Path{
			Home:   tmpDir,
			Config: tmpDir,
			Data:   tmpDir,
			Logs:   tmpDir,
		}

		logger := logp.NewLogger("test-ownership-shared")
		dataPath := beatPaths.Resolve(paths.Data, "sql-cursor")

		// Create a shared registry
		reg, err := memlog.New(logger.Named("memlog"), memlog.Settings{
			Root:     dataPath,
			FileMode: 0o600,
		})
		require.NoError(t, err)

		registry := statestore.NewRegistry(reg)
		defer registry.Close()

		// Create store via NewStoreFromRegistry (does NOT own registry)
		store, err := NewStoreFromRegistry(registry, logger)
		require.NoError(t, err)
		require.NotNil(t, store)
		require.Equal(t, doesNotOwnRegistry, store.ownsRegistry)
		require.Nil(t, store.registry, "Store should not hold registry reference when not owned")

		// Save some data
		testState := &State{
			Version:     StateVersion,
			CursorType:  CursorTypeInteger,
			CursorValue: "200",
			UpdatedAt:   time.Now().UTC(),
		}
		err = store.Save("test-key", testState)
		require.NoError(t, err)

		// Close the store (should NOT close the registry)
		err = store.Close()
		require.NoError(t, err)

		// Verify the registry is still open by creating another store
		store2, err := NewStoreFromRegistry(registry, logger)
		require.NoError(t, err, "Registry should still be open after closing store")
		require.NotNil(t, store2)

		// Verify we can read the data written by the first store
		loaded, err := store2.Load("test-key")
		require.NoError(t, err)
		require.NotNil(t, loaded)
		assert.Equal(t, "200", loaded.CursorValue)

		// Clean up
		require.NoError(t, store2.Close())
	})
}

func TestIsKeyNotFoundError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "key unknown error",
			err:  fmt.Errorf("failed in get operation on store 'cursor-state': key unknown"),
			want: true,
		},
		{
			name: "other error containing key word",
			err:  fmt.Errorf("primary key constraint violated"),
			want: false,
		},
		{
			name: "generic error",
			err:  fmt.Errorf("connection refused"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isKeyNotFoundError(tt.err))
		})
	}
}
