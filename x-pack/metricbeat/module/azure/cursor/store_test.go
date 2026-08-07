// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package cursor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/backend/memlog"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"
)

func newTestStore(t *testing.T) (*Store, *statestore.Registry) {
	t.Helper()
	tmpDir := t.TempDir()
	beatPaths := &paths.Path{
		Home: tmpDir, Config: tmpDir, Data: tmpDir, Logs: tmpDir,
	}
	logger := logp.NewNopLogger()
	dataPath := beatPaths.Resolve(paths.Data, "azure-cursor")
	reg, err := memlog.New(logger.Named("memlog"), memlog.Settings{
		Root:     dataPath,
		FileMode: 0o600,
	})
	require.NoError(t, err, "create memlog registry")
	registry := statestore.NewRegistry(reg)
	t.Cleanup(func() { registry.Close() })
	store, err := NewStoreFromRegistry(registry, logger)
	require.NoError(t, err, "open cursor store")
	return store, registry
}

func TestStoreMissingKey(t *testing.T) {
	store, _ := newTestStore(t)
	defer store.Close()

	state, err := store.Load("does-not-exist")
	require.NoError(t, err)
	assert.Nil(t, state, "missing key should return nil state, not an error")
}

func TestStoreSaveAndLoad(t *testing.T) {
	store, _ := newTestStore(t)
	defer store.Close()

	want := &State{
		Version:           StateVersion,
		LastCollectionEnd: time.Date(2024, 7, 30, 18, 56, 0, 0, time.UTC),
		UpdatedAt:         time.Now().UTC(),
	}

	require.NoError(t, store.Save("my-key", want))

	got, err := store.Load("my-key")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, want.Version, got.Version)
	assert.True(t, want.LastCollectionEnd.Equal(got.LastCollectionEnd),
		"LastCollectionEnd mismatch: want %v got %v", want.LastCollectionEnd, got.LastCollectionEnd)
}

func TestStoreCloseIsIdempotent(t *testing.T) {
	store, _ := newTestStore(t)
	require.NoError(t, store.Close())
	require.NoError(t, store.Close()) // second close must not panic or error
}

func TestStoreOperationsAfterClose(t *testing.T) {
	store, _ := newTestStore(t)
	require.NoError(t, store.Close())

	state := &State{Version: StateVersion, LastCollectionEnd: time.Now()}
	require.Error(t, store.Save("k", state), "Save after close should error")

	_, err := store.Load("k")
	require.Error(t, err, "Load after close should error")
}

func TestStoreSharedRegistry(t *testing.T) {
	// Two Store handles from the same registry share the same underlying data.
	if testing.Short() {
		t.Skip("skipping shared-registry test in short mode")
	}
	tmpDir := t.TempDir()
	beatPaths := &paths.Path{
		Home: tmpDir, Config: tmpDir, Data: tmpDir, Logs: tmpDir,
	}
	logger := logp.NewNopLogger()
	dataPath := beatPaths.Resolve(paths.Data, "azure-cursor")
	reg, err := memlog.New(logger.Named("memlog"), memlog.Settings{
		Root: dataPath, FileMode: 0o600,
	})
	require.NoError(t, err)
	registry := statestore.NewRegistry(reg)
	defer registry.Close()

	store1, err := NewStoreFromRegistry(registry, logger)
	require.NoError(t, err)
	store2, err := NewStoreFromRegistry(registry, logger)
	require.NoError(t, err)

	state := &State{
		Version:           StateVersion,
		LastCollectionEnd: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:         time.Now().UTC(),
	}
	require.NoError(t, store1.Save("shared-key", state))

	loaded, err := store2.Load("shared-key")
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.True(t, state.LastCollectionEnd.Equal(loaded.LastCollectionEnd))

	require.NoError(t, store1.Close())
	require.NoError(t, store2.Close())
}
