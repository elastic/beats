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

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"
)

func TestGenerateStateKey(t *testing.T) {
	tests := []struct {
		name         string
		inputType    string
		moduleID     string
		host         string
		query        string
		cursorColumn string
	}{
		{
			name:         "basic",
			inputType:    "sql",
			moduleID:     "",
			host:         "localhost:5432",
			query:        "SELECT * FROM logs WHERE id > :cursor",
			cursorColumn: "id",
		},
		{
			name:         "with module ID",
			inputType:    "sql",
			moduleID:     "my-module-1",
			host:         "localhost:5432",
			query:        "SELECT * FROM logs WHERE id > :cursor",
			cursorColumn: "id",
		},
		{
			name:         "different host",
			inputType:    "sql",
			moduleID:     "",
			host:         "remotehost:5432",
			query:        "SELECT * FROM logs WHERE id > :cursor",
			cursorColumn: "id",
		},
		{
			name:         "different query",
			inputType:    "sql",
			moduleID:     "",
			host:         "localhost:5432",
			query:        "SELECT * FROM events WHERE id > :cursor",
			cursorColumn: "id",
		},
		{
			name:         "different column",
			inputType:    "sql",
			moduleID:     "",
			host:         "localhost:5432",
			query:        "SELECT * FROM logs WHERE id > :cursor",
			cursorColumn: "event_id",
		},
	}

	// Generate keys and ensure they're unique
	keys := make(map[string]string)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := GenerateStateKey(tt.inputType, tt.moduleID, tt.host, tt.query, tt.cursorColumn)

			// Key should have expected prefix
			assert.Contains(t, key, "sql-cursor::")

			// Key should be hex formatted
			assert.Regexp(t, `^sql-cursor::[0-9a-f]+$`, key)

			// Keys should be unique for different inputs
			identifier := tt.inputType + tt.moduleID + tt.host + tt.query + tt.cursorColumn
			if existingKey, exists := keys[identifier]; exists {
				assert.Equal(t, existingKey, key, "same inputs should produce same key")
			} else {
				keys[identifier] = key
			}
		})
	}

	// Verify different inputs produce different keys
	key1 := GenerateStateKey("sql", "", "host1", "query", "col")
	key2 := GenerateStateKey("sql", "", "host2", "query", "col")
	assert.NotEqual(t, key1, key2, "different hosts should produce different keys")

	// Verify module ID changes the key
	key3 := GenerateStateKey("sql", "", "host", "query", "col")
	key4 := GenerateStateKey("sql", "module-1", "host", "query", "col")
	assert.NotEqual(t, key3, key4, "adding module ID should change the key")

	// Verify whitespace in query changes the key (no normalization)
	key5 := GenerateStateKey("sql", "", "host", "SELECT * FROM logs", "col")
	key6 := GenerateStateKey("sql", "", "host", "SELECT  *  FROM  logs", "col")
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
	store, err := NewStore(beatPaths, logger)
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

	store, err := NewStore(beatPaths, logger)
	require.NoError(t, err)

	// First close should succeed
	err = store.Close()
	require.NoError(t, err)

	// Second close should also succeed (idempotent)
	err = store.Close()
	require.NoError(t, err)
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
