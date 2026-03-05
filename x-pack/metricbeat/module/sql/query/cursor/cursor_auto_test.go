// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cursor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/paths"
)

func TestManagerAutoType_InferFromDefault(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping manager test in short mode")
	}

	tmpDir := t.TempDir()
	beatPaths := &paths.Path{
		Home:   tmpDir,
		Config: tmpDir,
		Data:   tmpDir,
		Logs:   tmpDir,
	}
	logger := logp.NewNopLogger()

	cfg := Config{
		Enabled: true,
		Column:  "id",
		Default: "0",
	}

	store, _ := newTestStore(t, beatPaths, logger)
	mgr, err := NewManager(cfg, store, "localhost", "SELECT id FROM t WHERE id > :cursor ORDER BY id", logger)
	require.NoError(t, err)
	defer mgr.Close()

	assert.Equal(t, "0", mgr.CursorValueString())
	assert.EqualValues(t, int64(0), mgr.CursorValueForQuery())

	err = mgr.UpdateFromResults([]mapstr.M{
		{"id": int64(100)},
		{"id": int64(200)},
	})
	require.NoError(t, err)
	assert.Equal(t, "200", mgr.CursorValueString())
	assert.EqualValues(t, int64(200), mgr.CursorValueForQuery())
}

func TestManagerAutoType_RefineFromRowsAndReloadFromState(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping manager test in short mode")
	}

	tmpDir := t.TempDir()
	beatPaths := &paths.Path{
		Home:   tmpDir,
		Config: tmpDir,
		Data:   tmpDir,
		Logs:   tmpDir,
	}
	logger := logp.NewNopLogger()

	cfg := Config{
		Enabled: true,
		Column:  "price",
		Default: "0",
	}
	host := "localhost"
	query := "SELECT price FROM t WHERE price > :cursor ORDER BY price"

	store1, _ := newTestStore(t, beatPaths, logger)
	mgr1, err := NewManager(cfg, store1, host, query, logger)
	require.NoError(t, err)

	err = mgr1.UpdateFromResults([]mapstr.M{
		{"price": []byte("10.25")},
		{"price": []byte("20.50")},
	})
	require.NoError(t, err)
	assert.Equal(t, "20.5", mgr1.CursorValueString())
	arg1, ok := mgr1.CursorValueForQuery().(string)
	require.True(t, ok)
	assert.Equal(t, "20.5", arg1)
	require.NoError(t, mgr1.Close())

	store2, _ := newTestStore(t, beatPaths, logger)
	mgr2, err := NewManager(cfg, store2, host, query, logger)
	require.NoError(t, err)
	defer mgr2.Close()

	assert.Equal(t, "20.5", mgr2.CursorValueString())
	arg2, ok := mgr2.CursorValueForQuery().(string)
	require.True(t, ok)
	assert.Equal(t, "20.5", arg2)
}

func TestManagerStateID_PreservesStateAcrossDSNChanges(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping manager test in short mode")
	}

	tmpDir := t.TempDir()
	beatPaths := &paths.Path{
		Home:   tmpDir,
		Config: tmpDir,
		Data:   tmpDir,
		Logs:   tmpDir,
	}
	logger := logp.NewNopLogger()

	cfg := Config{
		Enabled: true,
		Column:  "id",
		Type:    CursorTypeInteger,
		Default: "0",
		StateID: "payments-prod",
	}
	query := "SELECT id FROM t WHERE id > :cursor ORDER BY id"

	store1, _ := newTestStore(t, beatPaths, logger)
	mgr1, err := NewManager(cfg, store1, "postgres://user:oldpass@localhost:5432/prod", query, logger)
	require.NoError(t, err)

	err = mgr1.UpdateFromResults([]mapstr.M{
		{"id": int64(150)},
	})
	require.NoError(t, err)
	require.NoError(t, mgr1.Close())

	store2, _ := newTestStore(t, beatPaths, logger)
	mgr2, err := NewManager(cfg, store2, "postgres://user:newpass@localhost:5432/prod", query, logger)
	require.NoError(t, err)
	defer mgr2.Close()

	assert.Equal(t, "150", mgr2.CursorValueString(), "state_id should keep cursor continuity across DSN changes")
}

func TestManagerWithoutStateID_ResetsOnDSNChanges(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping manager test in short mode")
	}

	tmpDir := t.TempDir()
	beatPaths := &paths.Path{
		Home:   tmpDir,
		Config: tmpDir,
		Data:   tmpDir,
		Logs:   tmpDir,
	}
	logger := logp.NewNopLogger()

	cfg := Config{
		Enabled: true,
		Column:  "id",
		Type:    CursorTypeInteger,
		Default: "0",
	}
	query := "SELECT id FROM t WHERE id > :cursor ORDER BY id"

	store1, _ := newTestStore(t, beatPaths, logger)
	mgr1, err := NewManager(cfg, store1, "postgres://user:oldpass@localhost:5432/prod", query, logger)
	require.NoError(t, err)

	err = mgr1.UpdateFromResults([]mapstr.M{
		{"id": int64(150)},
	})
	require.NoError(t, err)
	require.NoError(t, mgr1.Close())

	store2, _ := newTestStore(t, beatPaths, logger)
	mgr2, err := NewManager(cfg, store2, "postgres://user:newpass@localhost:5432/prod", query, logger)
	require.NoError(t, err)
	defer mgr2.Close()

	assert.Equal(t, "0", mgr2.CursorValueString(), "without state_id, DSN change should create a new state key")
}
