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
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/filebeat/config"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"
)

func testOpenStore(t *testing.T, dir string) *filebeatStore {
	t.Helper()
	beatPaths := paths.New()
	beatPaths.Data = dir

	store, err := openStateStore(context.Background(), beat.Info{Beat: "test"}, logp.NewLogger("test"), config.Registry{
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
	resolvedPath := s1.path

	s2 := testOpenStore(t, dir)

	s1.Close()

	globalMu.Lock()
	_, exists := globalStores[resolvedPath]
	globalMu.Unlock()
	assert.True(t, exists, "entry should still exist when refCount > 0")

	s2.Close()

	globalMu.Lock()
	_, exists = globalStores[resolvedPath]
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

	resolvedPath := stores[0].path
	globalMu.Lock()
	_, exists := globalStores[resolvedPath]
	globalMu.Unlock()
	assert.False(t, exists, "entry should be cleaned up after all stores are closed")
}
