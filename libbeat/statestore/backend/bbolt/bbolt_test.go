// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package bbolt

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	"github.com/elastic/beats/v7/libbeat/statestore/internal/storecompliance"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func TestCompliance(t *testing.T) {
	storecompliance.TestBackendCompliance(t, func(testPath string) (backend.Registry, error) {
		logger := logptest.NewTestingLogger(t, "")
		return New(logger.Named("test"), Settings{
			Root:   testPath,
			Config: DefaultConfig(),
		})
	})
}

func TestStoreSetGet(t *testing.T) {
	path := t.TempDir()
	logger := logptest.NewTestingLogger(t, "")
	reg, err := New(logger.Named("test"), Settings{
		Root:   path,
		Config: DefaultConfig(),
	})
	require.NoError(t, err)
	defer reg.Close()

	store, err := reg.Access("test")
	require.NoError(t, err)
	defer store.Close()

	type entry struct {
		Field string
		Count int
	}

	require.NoError(t, store.Set("key1", entry{Field: "hello", Count: 42}))

	var got entry
	require.NoError(t, store.Get("key1", &got))
	assert.Equal(t, "hello", got.Field)
	assert.Equal(t, 42, got.Count)
}

func TestStoreHas(t *testing.T) {
	path := t.TempDir()
	logger := logptest.NewTestingLogger(t, "")
	reg, err := New(logger.Named("test"), Settings{
		Root:   path,
		Config: DefaultConfig(),
	})
	require.NoError(t, err)
	defer reg.Close()

	store, err := reg.Access("test")
	require.NoError(t, err)
	defer store.Close()

	has, err := store.Has("missing")
	require.NoError(t, err)
	assert.False(t, has)

	require.NoError(t, store.Set("present", map[string]any{"a": 1}))

	has, err = store.Has("present")
	require.NoError(t, err)
	assert.True(t, has)
}

func TestStoreRemove(t *testing.T) {
	path := t.TempDir()
	logger := logptest.NewTestingLogger(t, "")
	reg, err := New(logger.Named("test"), Settings{
		Root:   path,
		Config: DefaultConfig(),
	})
	require.NoError(t, err)
	defer reg.Close()

	store, err := reg.Access("test")
	require.NoError(t, err)
	defer store.Close()

	require.NoError(t, store.Set("key", map[string]any{"a": 1}))
	has, err := store.Has("key")
	require.NoError(t, err)
	assert.True(t, has)

	require.NoError(t, store.Remove("key"))
	has, err = store.Has("key")
	require.NoError(t, err)
	assert.False(t, has)
}

func TestStoreRemoveUnknownKey(t *testing.T) {
	path := t.TempDir()
	logger := logptest.NewTestingLogger(t, "")
	reg, err := New(logger.Named("test"), Settings{
		Root:   path,
		Config: DefaultConfig(),
	})
	require.NoError(t, err)
	defer reg.Close()

	store, err := reg.Access("test")
	require.NoError(t, err)
	defer store.Close()

	require.NoError(t, store.Remove("does-not-exist"))
}

func TestStoreEach(t *testing.T) {
	path := t.TempDir()
	logger := logptest.NewTestingLogger(t, "")
	reg, err := New(logger.Named("test"), Settings{
		Root:   path,
		Config: DefaultConfig(),
	})
	require.NoError(t, err)
	defer reg.Close()

	store, err := reg.Access("test")
	require.NoError(t, err)
	defer store.Close()

	expected := map[string]any{
		"a": map[string]any{"field": "hello"},
		"b": map[string]any{"field": "world"},
	}

	for k, v := range expected {
		require.NoError(t, store.Set(k, v))
	}

	got := map[string]any{}
	err = store.Each(func(key string, dec backend.ValueDecoder) (bool, error) {
		var tmp any
		if err := dec.Decode(&tmp); err != nil {
			return false, err
		}
		got[key] = tmp
		return true, nil
	})
	require.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestStoreEachStopOnFalse(t *testing.T) {
	path := t.TempDir()
	logger := logptest.NewTestingLogger(t, "")
	reg, err := New(logger.Named("test"), Settings{
		Root:   path,
		Config: DefaultConfig(),
	})
	require.NoError(t, err)
	defer reg.Close()

	store, err := reg.Access("test")
	require.NoError(t, err)
	defer store.Close()

	require.NoError(t, store.Set("a", map[string]any{"x": 1}))
	require.NoError(t, store.Set("b", map[string]any{"x": 2}))

	count := 0
	err = store.Each(func(_ string, _ backend.ValueDecoder) (bool, error) {
		count++
		return false, nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestStorePersistence(t *testing.T) {
	path := t.TempDir()
	logger := logptest.NewTestingLogger(t, "")
	cfg := DefaultConfig()

	// First: open, write, close
	reg, err := New(logger.Named("test"), Settings{Root: path, Config: cfg})
	require.NoError(t, err)

	store, err := reg.Access("test")
	require.NoError(t, err)

	type entry struct{ Value string }
	require.NoError(t, store.Set("persistent", entry{Value: "survives restart"}))
	require.NoError(t, store.Close())
	require.NoError(t, reg.Close())

	// Second: reopen, read
	reg2, err := New(logger.Named("test"), Settings{Root: path, Config: cfg})
	require.NoError(t, err)
	defer reg2.Close()

	store2, err := reg2.Access("test")
	require.NoError(t, err)
	defer store2.Close()

	var got entry
	require.NoError(t, store2.Get("persistent", &got))
	assert.Equal(t, "survives restart", got.Value)
}

func TestStoreCompactionOnStart(t *testing.T) {
	path := t.TempDir()
	logger := logptest.NewTestingLogger(t, "")

	cfg := DefaultConfig()

	// Write data then close
	reg, err := New(logger.Named("test"), Settings{Root: path, Config: cfg})
	require.NoError(t, err)

	store, err := reg.Access("test")
	require.NoError(t, err)

	for i := 0; i < 100; i++ {
		require.NoError(t, store.Set("key", map[string]any{"i": i}))
	}
	require.NoError(t, store.Close())
	require.NoError(t, reg.Close())

	// Reopen with compaction on start
	cfg.Compaction.OnStart = true
	reg2, err := New(logger.Named("test"), Settings{Root: path, Config: cfg})
	require.NoError(t, err)
	defer reg2.Close()

	store2, err := reg2.Access("test")
	require.NoError(t, err)
	defer store2.Close()

	// Verify data survived compaction
	has, err := store2.Has("key")
	require.NoError(t, err)
	assert.True(t, has)
}

func TestStoreTTLCleanup(t *testing.T) {
	path := t.TempDir()
	logger := logptest.NewTestingLogger(t, "")

	cfg := DefaultConfig()
	cfg.Retention.TTL = 200 * time.Millisecond
	cfg.Retention.Interval = 100 * time.Millisecond

	reg, err := New(logger.Named("test"), Settings{Root: path, Config: cfg})
	require.NoError(t, err)
	defer reg.Close()

	store, err := reg.Access("test")
	require.NoError(t, err)
	defer store.Close()

	require.NoError(t, store.Set("old-key", map[string]any{"a": 1}))

	// Wait for TTL + cleanup interval to elapse
	require.Eventually(t, func() bool {
		has, err := store.Has("old-key")
		if err != nil {
			return false
		}
		return !has
	}, 2*time.Second, 50*time.Millisecond, "expected old-key to be removed by TTL cleanup")
}

func TestStoreTTLCleanupKeepsRecent(t *testing.T) {
	path := t.TempDir()
	logger := logptest.NewTestingLogger(t, "")

	cfg := DefaultConfig()
	cfg.Retention.TTL = 5 * time.Second
	cfg.Retention.Interval = 100 * time.Millisecond

	reg, err := New(logger.Named("test"), Settings{Root: path, Config: cfg})
	require.NoError(t, err)
	defer reg.Close()

	store, err := reg.Access("test")
	require.NoError(t, err)
	defer store.Close()

	require.NoError(t, store.Set("recent-key", map[string]any{"a": 1}))

	// Wait for a cleanup cycle but not for TTL
	time.Sleep(300 * time.Millisecond)

	has, err := store.Has("recent-key")
	require.NoError(t, err)
	assert.True(t, has, "recent key should not be removed before TTL expires")
}

func TestRegistryClosePreventsAccess(t *testing.T) {
	path := t.TempDir()
	logger := logptest.NewTestingLogger(t, "")
	reg, err := New(logger.Named("test"), Settings{
		Root:   path,
		Config: DefaultConfig(),
	})
	require.NoError(t, err)

	require.NoError(t, reg.Close())

	_, err = reg.Access("test")
	assert.ErrorIs(t, err, errRegClosed)
}

func TestCompactFailure_StoreStillWorks(t *testing.T) {
	dir := t.TempDir()
	logger := logptest.NewTestingLogger(t, "")
	cfg := DefaultConfig()

	dbPath := filepath.Join(dir, "test.db")
	s, err := openStore(logger, dbPath, 0600, cfg)
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, s.Set("key1", map[string]any{"value": "hello"}))

	has, err := s.Has("key1")
	require.NoError(t, err)
	require.True(t, has)

	// Make directory read-only so compact() fails at CreateTemp.
	require.NoError(t, os.Chmod(dir, 0555))
	defer os.Chmod(dir, 0755) //nolint:errcheck // restore for cleanup

	err = s.compact()
	require.Error(t, err)

	// Store must still be fully functional after the failed compaction.
	has, err = s.Has("key1")
	require.NoError(t, err)
	assert.True(t, has)

	var got map[string]any
	require.NoError(t, s.Get("key1", &got))
	assert.Equal(t, "hello", got["value"])

	// Restore permissions so Set can fsync.
	require.NoError(t, os.Chmod(dir, 0755))

	require.NoError(t, s.Set("key2", map[string]any{"value": "world"}))
	has, err = s.Has("key2")
	require.NoError(t, err)
	assert.True(t, has)
}

func TestCompactReopenFailure_NoPanic(t *testing.T) {
	dir := t.TempDir()
	logger := logptest.NewTestingLogger(t, "")
	cfg := DefaultConfig()

	dbPath := filepath.Join(dir, "test.db")
	s, err := openStore(logger, dbPath, 0600, cfg)
	require.NoError(t, err)

	require.NoError(t, s.Set("key1", map[string]any{"value": "hello"}))

	// Simulate the state compact() leaves when bolt.Open fails to reopen:
	// s.db is closed but not nil. Before the fix, bolt.Open would assign
	// nil to s.db, causing nil-pointer panics in all subsequent operations.
	s.mu.Lock()
	s.db.Close()
	s.mu.Unlock()

	assert.NotPanics(t, func() {
		_, err := s.Has("key1")
		assert.Error(t, err)
	})
	assert.NotPanics(t, func() {
		var v map[string]any
		err := s.Get("key1", &v)
		assert.Error(t, err)
	})
	assert.NotPanics(t, func() {
		err := s.Set("key2", map[string]any{"x": 1})
		assert.Error(t, err)
	})
	assert.NotPanics(t, func() {
		err := s.Remove("key1")
		assert.Error(t, err)
	})
	assert.NotPanics(t, func() {
		err := s.Each(func(_ string, _ backend.ValueDecoder) (bool, error) {
			return true, nil
		})
		assert.Error(t, err)
	})
}

func TestHasSkipsExpiredEntry(t *testing.T) {
	dir := t.TempDir()
	logger := logptest.NewTestingLogger(t, "")
	cfg := DefaultConfig()
	cfg.Retention.TTL = 100 * time.Millisecond

	s, err := openStore(logger, filepath.Join(dir, "test.db"), 0600, cfg)
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, s.Set("key", map[string]any{"v": 1}))

	has, err := s.Has("key")
	require.NoError(t, err)
	assert.True(t, has, "key should be visible before TTL expires")

	time.Sleep(150 * time.Millisecond)

	has, err = s.Has("key")
	require.NoError(t, err)
	assert.False(t, has, "key should be invisible after TTL expires")
}

func TestGetSkipsExpiredEntry(t *testing.T) {
	dir := t.TempDir()
	logger := logptest.NewTestingLogger(t, "")
	cfg := DefaultConfig()
	cfg.Retention.TTL = 100 * time.Millisecond

	s, err := openStore(logger, filepath.Join(dir, "test.db"), 0600, cfg)
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, s.Set("key", map[string]any{"v": 1}))

	var got map[string]any
	require.NoError(t, s.Get("key", &got), "key should be readable before TTL expires")

	time.Sleep(150 * time.Millisecond)

	err = s.Get("key", &got)
	assert.ErrorIs(t, err, errKeyUnknown, "expired key should return errKeyUnknown")
}

func TestEachSkipsExpiredEntries(t *testing.T) {
	dir := t.TempDir()
	logger := logptest.NewTestingLogger(t, "")
	cfg := DefaultConfig()
	cfg.Retention.TTL = 100 * time.Millisecond

	s, err := openStore(logger, filepath.Join(dir, "test.db"), 0600, cfg)
	require.NoError(t, err)
	defer s.Close()

	require.NoError(t, s.Set("old", map[string]any{"v": 1}))

	time.Sleep(150 * time.Millisecond)

	require.NoError(t, s.Set("new", map[string]any{"v": 2}))

	var keys []string
	err = s.Each(func(key string, dec backend.ValueDecoder) (bool, error) {
		keys = append(keys, key)
		return true, nil
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"new"}, keys, "Each should skip the expired entry")
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr string
	}{
		{
			name:   "valid default config",
			modify: func(_ *Config) {},
		},
		{
			name:    "negative timeout",
			modify:  func(c *Config) { c.Timeout = -1 * time.Second },
			wantErr: "timeout must not be negative",
		},
		{
			name:    "negative max_transaction_size",
			modify:  func(c *Config) { c.Compaction.MaxTransactionSize = -1 },
			wantErr: "max_transaction_size must not be negative",
		},
		{
			name:    "negative TTL",
			modify:  func(c *Config) { c.Retention.TTL = -1 * time.Second },
			wantErr: "TTL must not be negative",
		},
		{
			name: "negative interval with positive TTL",
			modify: func(c *Config) {
				c.Retention.TTL = time.Hour
				c.Retention.Interval = -1 * time.Second
			},
			wantErr: "interval must not be negative",
		},
		{
			name: "zero interval with positive TTL is valid",
			modify: func(c *Config) {
				c.Retention.TTL = time.Hour
				c.Retention.Interval = 0
			},
		},
		{
			name:   "zero max_transaction_size is valid",
			modify: func(c *Config) { c.Compaction.MaxTransactionSize = 0 },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.modify(&cfg)
			err := cfg.Validate()
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestMultipleStores(t *testing.T) {
	path := t.TempDir()
	logger := logptest.NewTestingLogger(t, "")
	reg, err := New(logger.Named("test"), Settings{
		Root:   path,
		Config: DefaultConfig(),
	})
	require.NoError(t, err)
	defer reg.Close()

	store1, err := reg.Access("store1")
	require.NoError(t, err)
	defer store1.Close()

	store2, err := reg.Access("store2")
	require.NoError(t, err)
	defer store2.Close()

	// Data in one store should not be visible in the other
	require.NoError(t, store1.Set("key", map[string]any{"from": "store1"}))

	has, err := store2.Has("key")
	require.NoError(t, err)
	assert.False(t, has, "store2 should not see data from store1")
}

func TestCleanupOnStartRemovesTempFiles(t *testing.T) {
	dir := t.TempDir()
	logger := logptest.NewTestingLogger(t, "")

	cfg := DefaultConfig()

	// Create a store, write data, close it.
	dbPath := filepath.Join(dir, "test.db")
	s, err := openStore(logger, dbPath, 0600, cfg)
	require.NoError(t, err)
	require.NoError(t, s.Set("key", map[string]any{"v": 1}))
	require.NoError(t, s.Close())

	// Plant fake leftover temp files that simulate a crashed compaction.
	for i := 0; i < 3; i++ {
		f, err := os.CreateTemp(dir, tempDbPrefix)
		require.NoError(t, err)
		require.NoError(t, f.Close())
	}

	matches, err := filepath.Glob(filepath.Join(dir, tempDbPrefix+"*"))
	require.NoError(t, err)
	require.Len(t, matches, 3, "precondition: 3 temp files should exist")

	// Reopen with CleanupOnStart enabled.
	cfg.Compaction.CleanupOnStart = true
	s2, err := openStore(logger, dbPath, 0600, cfg)
	require.NoError(t, err)
	defer s2.Close()

	matches, err = filepath.Glob(filepath.Join(dir, tempDbPrefix+"*"))
	require.NoError(t, err)
	assert.Empty(t, matches, "all temp files should be removed on start")

	has, err := s2.Has("key")
	require.NoError(t, err)
	assert.True(t, has, "original data should still be intact")
}

func TestCleanupExpiredBatching(t *testing.T) {
	dir := t.TempDir()
	logger := logptest.NewTestingLogger(t, "")

	cfg := DefaultConfig()
	cfg.Retention.TTL = 100 * time.Millisecond
	cfg.Compaction.MaxTransactionSize = 3

	dbPath := filepath.Join(dir, "test.db")
	s, err := openStore(logger, dbPath, 0600, cfg)
	require.NoError(t, err)
	defer s.Close()

	// Insert 10 keys that will all expire.
	for i := 0; i < 10; i++ {
		require.NoError(t, s.Set(fmt.Sprintf("key-%02d", i), map[string]any{"i": i}))
	}

	time.Sleep(150 * time.Millisecond)

	// Insert 2 fresh keys that should survive.
	require.NoError(t, s.Set("fresh-a", map[string]any{"v": 1}))
	require.NoError(t, s.Set("fresh-b", map[string]any{"v": 2}))

	require.NoError(t, s.cleanupExpired())

	// Verify all expired keys are gone.
	for i := 0; i < 10; i++ {
		has, err := s.Has(fmt.Sprintf("key-%02d", i))
		require.NoError(t, err)
		assert.False(t, has, "expired key-%02d should be removed", i)
	}

	// Verify fresh keys survived.
	has, err := s.Has("fresh-a")
	require.NoError(t, err)
	assert.True(t, has, "fresh-a should survive cleanup")

	has, err = s.Has("fresh-b")
	require.NoError(t, err)
	assert.True(t, has, "fresh-b should survive cleanup")
}

func TestStoreDoubleClose(t *testing.T) {
	dir := t.TempDir()
	logger := logptest.NewTestingLogger(t, "")

	s, err := openStore(logger, filepath.Join(dir, "test.db"), 0600, DefaultConfig())
	require.NoError(t, err)

	require.NoError(t, s.Set("key", map[string]any{"v": 1}))

	require.NoError(t, s.Close())
	require.NoError(t, s.Close(), "second Close should be a no-op and not return an error")
}

func TestOpenStoreResolvesSymlinks(t *testing.T) {
	dir := t.TempDir()
	logger := logptest.NewTestingLogger(t, "")

	realDir := filepath.Join(dir, "real")
	require.NoError(t, os.MkdirAll(realDir, 0755))

	// Create the database via the real path first.
	realDBPath := filepath.Join(realDir, "test.db")
	s, err := openStore(logger, realDBPath, 0600, DefaultConfig())
	require.NoError(t, err)
	require.NoError(t, s.Set("key", map[string]any{"v": 1}))
	require.NoError(t, s.Close())

	// Create a symlink to the real directory.
	linkDir := filepath.Join(dir, "link")
	require.NoError(t, os.Symlink(realDir, linkDir))

	// Reopen via the symlinked path. EvalSymlinks should resolve it.
	dbViaLink := filepath.Join(linkDir, "test.db")
	s2, err := openStore(logger, dbViaLink, 0600, DefaultConfig())
	require.NoError(t, err)
	defer s2.Close()

	assert.Equal(t, realDBPath, s2.dbPath, "dbPath should be resolved through the symlink")

	has, err := s2.Has("key")
	require.NoError(t, err)
	assert.True(t, has, "data written via real path should be accessible via resolved symlink")
}
