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

package beater

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/filebeat/config"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/statestore/backend/memlog"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"
)

func testOpenStore(t *testing.T, dir string) *filebeatStore {
	t.Helper()
	beatPaths := paths.New()
	beatPaths.Data = dir

	store, err := openStateStore(t.Context(), beat.Info{Beat: "test"}, logp.NewNopLogger(), config.Registry{
		Path:          "",
		Permissions:   0600,
		CleanInterval: 5 * time.Second,
	}, beatPaths)
	require.NoError(t, err)
	return store
}

func TestOpenStateStore_SamePathSharesRegistry(t *testing.T) {
	dir := t.TempDir()

	s1 := testOpenStore(t, dir)
	s2 := testOpenStore(t, dir)

	assert.Same(t, s1.shared, s2.shared, "stores with the same path should share the same sharedRegistries")

	globalMu.Lock()
	assert.Equal(t, 2, s1.shared.refCount)
	globalMu.Unlock()

	s1.Close()
	s2.Close()
}

func TestOpenStateStore_DifferentPathsGetDifferentRegistries(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	s1 := testOpenStore(t, dir1)
	s2 := testOpenStore(t, dir2)

	assert.NotSame(t, s1.shared, s2.shared, "stores with different paths should not share registries")

	s1.Close()
	s2.Close()
}

func TestOpenStateStore_CloseDecrementsRefCount(t *testing.T) {
	dir := t.TempDir()

	s1 := testOpenStore(t, dir)
	s2 := testOpenStore(t, dir)

	s1.Close()

	globalMu.Lock()
	assert.Equal(t, 1, s2.shared.refCount)
	globalMu.Unlock()

	s2.Close()
}

func TestOpenStateStore_LastCloseRemovesFromGlobal(t *testing.T) {
	dir := t.TempDir()

	s1 := testOpenStore(t, dir)
	resolvedKey := s1.storeKey

	s2 := testOpenStore(t, dir)

	s1.Close()

	globalMu.Lock()
	_, exists := globalStores[resolvedKey]
	globalMu.Unlock()
	assert.True(t, exists, "entry should still exist when refCount > 0")

	s2.Close()

	globalMu.Lock()
	_, exists = globalStores[resolvedKey]
	globalMu.Unlock()
	assert.False(t, exists, "entry should be removed when last store is closed")
}

func TestOpenStateStore_ConcurrentOpenClose(t *testing.T) {
	dir := t.TempDir()

	const n = 20
	stores := make([]*filebeatStore, n)
	var wg sync.WaitGroup

	// Open stores concurrently
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			stores[i] = testOpenStore(t, dir)
		}(i)
	}
	wg.Wait()

	// All should share the same sharedRegistries
	for i := 1; i < n; i++ {
		assert.Same(t, stores[0].shared, stores[i].shared)
	}

	globalMu.Lock()
	assert.Equal(t, n, stores[0].shared.refCount)
	globalMu.Unlock()

	// Close all concurrently
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			stores[i].Close()
		}(i)
	}
	wg.Wait()

	resolvedKey := stores[0].storeKey
	globalMu.Lock()
	_, exists := globalStores[resolvedKey]
	globalMu.Unlock()
	assert.False(t, exists, "entry should be cleaned up after all stores are closed")
}

// TestOpenStateStore_CheckpointSize verifies that the memlog checkpoint is
// triggered at the correct WAL size threshold, both for the default (10MB)
// and for a custom configured value.
//
// Important: memlog deletes old checkpoint data files after each new
// checkpoint, so at most 1 data file exists on disk at any time. We detect
// checkpoint events by checking whether a data file exists and the WAL log
// file has been reset (small size).
func TestOpenStateStore_CheckpointSize(t *testing.T) {
	const valueSize = 1024
	value := strings.Repeat("x", valueSize)

	testCases := []struct {
		name           string
		checkpointSize uint64 // 0 means use the default (10MB)
	}{
		{name: "custom 256KB", checkpointSize: 256 * 1024},
		{name: "custom 64KB", checkpointSize: 64 * 1024},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()

			cfg := config.Registry{
				Path:          "",
				Permissions:   0600,
				CleanInterval: 5 * time.Second,
			}
			if tc.checkpointSize > 0 {
				cfg.Memlog = memlog.Config{CheckpointSize: tc.checkpointSize}
			}

			s := testOpenStoreWithConfig(t, dir, cfg)
			defer s.Close()

			store, err := s.shared.registry.Get("test")
			require.NoError(t, err)
			defer store.Close()

			registryDir := filepath.Join(dir, "test")

			checkpointSize := tc.checkpointSize
			if checkpointSize == 0 {
				checkpointSize = 10 * 1 << 20 // memlog default: 10MB
			}

			written := 0

			// Phase 1: write entries until the WAL reaches ~80% of the
			// checkpointSize. This should NOT trigger a checkpoint.
			target80 := checkpointSize * 80 / 100
			for walFileSize(t, registryDir) < target80 {
				require.NoError(t, store.Set(fmt.Sprintf("key-%05d", written), map[string]any{"value": value}))
				written++
			}
			assert.False(t, hasCheckpointFile(t, registryDir),
				"no checkpoint should be triggered below the checkpointSize")
			t.Logf("phase 1: wrote %d entries, WAL size %d, checkpointSize %d",
				written, walFileSize(t, registryDir), checkpointSize)

			// Phase 2: write past the checkpointSize — a checkpoint should occur.
			// After the checkpoint, the WAL is truncated and the data file
			// appears. We write 2x the checkpointSize to be sure.
			target2x := checkpointSize * 2
			for walFileSize(t, registryDir) < target2x {
				require.NoError(t, store.Set(fmt.Sprintf("key-%05d", written), map[string]any{"value": value}))
				written++

				// Once a checkpoint fires, the WAL resets to near-zero.
				// Detect this to avoid writing forever.
				if hasCheckpointFile(t, registryDir) {
					break
				}
			}
			assert.True(t, hasCheckpointFile(t, registryDir),
				"a checkpoint should have been triggered after crossing the checkpointSize")
			t.Logf("phase 2: wrote %d total entries, checkpoint triggered", written)

			// Phase 3: after checkpoint the WAL was reset. Write past the
			// checkpointSize again and verify a second checkpoint occurs (the data
			// file changes).
			firstDataFile := currentCheckpointFile(t, registryDir)
			require.NotEmpty(t, firstDataFile, "sanity: data file should exist")

			for walFileSize(t, registryDir) < target2x {
				require.NoError(t, store.Set(fmt.Sprintf("key-%05d", written), map[string]any{"value": value}))
				written++

				if cf := currentCheckpointFile(t, registryDir); cf != "" && cf != firstDataFile {
					break
				}
			}

			secondDataFile := currentCheckpointFile(t, registryDir)
			assert.NotEqual(t, firstDataFile, secondDataFile,
				"a second checkpoint should produce a new data file (old one is deleted)")

			// The old data file should have been removed.
			assert.NoFileExists(t, filepath.Join(registryDir, firstDataFile),
				"the previous checkpoint file should be deleted")
		})
	}
}

// TestOpenStateStore_DefaultCheckpointSize verifies that the default memlog
// checkpoint (10MB) triggers. This test writes enough data to cross the
// threshold and checks that a checkpoint file appears.
func TestOpenStateStore_DefaultCheckpointSize(t *testing.T) {
	dir := t.TempDir()

	s := testOpenStoreWithConfig(t, dir, config.Registry{
		Path:          "",
		Permissions:   0600,
		CleanInterval: 5 * time.Second,
	})
	defer s.Close()

	store, err := s.shared.registry.Get("test")
	require.NoError(t, err)
	defer store.Close()

	registryDir := filepath.Join(dir, "test")

	const defaultThreshold = 10 * 1 << 20 // 10MB
	value := strings.Repeat("x", 4096)

	for i := 0; !hasCheckpointFile(t, registryDir); i++ {
		require.NoError(t, store.Set(fmt.Sprintf("key-%06d", i), map[string]any{"value": value}))

		if walFileSize(t, registryDir) > defaultThreshold*2 {
			t.Fatal("WAL exceeded 2x the default threshold without a checkpoint")
		}
	}

	assert.True(t, hasCheckpointFile(t, registryDir),
		"checkpoint should have been triggered at the default 10MB threshold")
}

// TestOpenStateStore_CustomCheckpointSmaller verifies that a smaller custom
// checkpoint size triggers sooner than the default 10MB.
func TestOpenStateStore_CustomCheckpointSmaller(t *testing.T) {
	const customThreshold uint64 = 32 * 1024 // 32KB

	dir := t.TempDir()
	s := testOpenStoreWithConfig(t, dir, config.Registry{
		Path:          "",
		Permissions:   0600,
		CleanInterval: 5 * time.Second,
		Memlog:        memlog.Config{CheckpointSize: customThreshold},
	})
	defer s.Close()

	store, err := s.shared.registry.Get("test")
	require.NoError(t, err)
	defer store.Close()

	registryDir := filepath.Join(dir, "test")
	value := strings.Repeat("x", 256)

	var checkpointedAt int
	for i := 0; i < 10000; i++ {
		require.NoError(t, store.Set(fmt.Sprintf("key-%05d", i), map[string]any{"value": value}))
		if hasCheckpointFile(t, registryDir) {
			checkpointedAt = i + 1
			break
		}
	}

	require.Greater(t, checkpointedAt, 0,
		"checkpoint should have been triggered")

	// With ~300-byte entries and a 32KB threshold, a checkpoint should happen
	// well before 200 entries. This verifies the custom threshold actually took
	// effect (default 10MB would need ~30,000+ entries).
	assert.Less(t, checkpointedAt, 200,
		"custom 32KB threshold should trigger much sooner than the default 10MB")
}

func testOpenStoreWithConfig(t *testing.T, dir string, cfg config.Registry) *filebeatStore {
	t.Helper()
	beatPaths := paths.New()
	beatPaths.Data = dir

	store, err := openStateStore(t.Context(), beat.Info{Beat: "test"}, logp.NewNopLogger(), cfg, beatPaths)
	require.NoError(t, err)
	return store
}

// hasCheckpointFile returns true if a checkpoint data file (numbered .json
// like "42.json") exists in the registry directory. Memlog deletes old
// checkpoint files after each new one, so at most 1 exists at a time.
func hasCheckpointFile(t *testing.T, registryDir string) bool {
	t.Helper()
	return currentCheckpointFile(t, registryDir) != ""
}

// currentCheckpointFile returns the name of the checkpoint data file in the
// registry directory, or "" if none exists.
func currentCheckpointFile(t *testing.T, registryDir string) string {
	t.Helper()
	entries, err := os.ReadDir(registryDir)
	require.NoError(t, err)

	for _, e := range entries {
		name := e.Name()
		if name == "log.json" || name == "meta.json" || name == "active.dat" ||
			strings.HasSuffix(name, ".new") {
			continue
		}
		if filepath.Ext(name) == ".json" {
			return name
		}
	}
	return ""
}

// walFileSize returns the size of the WAL log file in the registry directory.
func walFileSize(t *testing.T, registryDir string) uint64 {
	t.Helper()
	info, err := os.Stat(filepath.Join(registryDir, "log.json"))
	if os.IsNotExist(err) {
		return 0
	}
	require.NoError(t, err)
	return uint64(info.Size())
}
