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
	cfg.TTL = 200 * time.Millisecond
	cfg.Compaction.CleanupInterval = 100 * time.Millisecond

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
	cfg.TTL = 5 * time.Second
	cfg.Compaction.CleanupInterval = 100 * time.Millisecond

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
