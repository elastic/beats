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

package filestream

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- cache hit / miss -------------------------------------------------------

func TestDirCacheCacheHit(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.log"), nil, 0o600))

	c := newDirCache()
	maxAge := 100 * time.Millisecond

	// Warm the cache.
	names, err := c.readDirNames(dir, maxAge, true)
	require.NoError(t, err)
	assert.Equal(t, []string{"a.log"}, names)

	// Add a file while the cache is warm.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.log"), nil, 0o600))

	// Second read within TTL: still sees only a.log.
	names, err = c.readDirNames(dir, maxAge, true)
	require.NoError(t, err)
	assert.Equal(t, []string{"a.log"}, names)
}

func TestDirCacheCacheMissAfterTTL(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.log"), nil, 0o600))

	c := newDirCache()
	maxAge := 50 * time.Millisecond

	// Warm the cache then add a file.
	_, err := c.readDirNames(dir, maxAge, true)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.log"), nil, 0o600))

	// Wait past the TTL.
	time.Sleep(100 * time.Millisecond)

	names, err := c.readDirNames(dir, maxAge, true)
	require.NoError(t, err)
	assert.Equal(t, []string{"a.log", "b.log"}, names)
}

func TestDirCacheSharedFalseBypassesCache(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.log"), nil, 0o600))

	c := newDirCache()
	maxAge := time.Hour

	// Warm the cache.
	_, err := c.readDirNames(dir, maxAge, true)
	require.NoError(t, err)

	// Add a file then read with shared=false — must see new file despite cache.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.log"), nil, 0o600))
	names, err := c.readDirNames(dir, maxAge, false)
	require.NoError(t, err)
	assert.Equal(t, []string{"a.log", "b.log"}, names)
}

func TestDirCacheErrorNotCached(t *testing.T) {
	c := newDirCache()
	// Read a non-existent directory.
	_, err := c.readDirNames("/nonexistent/path/that/does/not/exist", time.Hour, true)
	require.Error(t, err)

	// Entry should be cleared so the next read retries the OS.
	e := c.entry("/nonexistent/path/that/does/not/exist")
	assert.Equal(t, int64(0), e.fetched.Load(), "error should clear the entry's fetched timestamp")
}

// ---- concurrent readers for the same directory ------------------------------

func TestDirCacheConcurrentReaders(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.log"), nil, 0o600))

	c := newDirCache()
	maxAge := time.Second

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)
	results := make([][]string, goroutines)
	errs := make([]error, goroutines)
	for i := range goroutines {
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = c.readDirNames(dir, maxAge, true)
		}(i)
	}
	wg.Wait()

	for i := range goroutines {
		require.NoError(t, errs[i])
		assert.Equal(t, []string{"a.log"}, results[i])
	}
}

// ---- singleton lifecycle ----------------------------------------------------

func TestAcquireSharedDirReaderSingleton(t *testing.T) {
	// Require clean global state — skip if another test left a reference.
	dirReaderMu.Lock()
	if dirReaderInst != nil {
		dirReaderMu.Unlock()
		t.Skip("singleton already held by another test")
	}
	dirReaderMu.Unlock()

	dc1, release1 := acquireSharedDirReader()
	dc2, release2 := acquireSharedDirReader()

	dirReaderMu.Lock()
	assert.Equal(t, 2, dirReaderRefs, "two acquires should give ref count 2")
	assert.Same(t, dc1, dc2, "both acquires should return the same instance")
	dirReaderMu.Unlock()

	release1()
	dirReaderMu.Lock()
	assert.Equal(t, 1, dirReaderRefs)
	assert.NotNil(t, dirReaderInst)
	dirReaderMu.Unlock()

	release2()
	dirReaderMu.Lock()
	assert.Equal(t, 0, dirReaderRefs)
	assert.Nil(t, dirReaderInst, "singleton should be nil after last release")
	dirReaderMu.Unlock()
}

func TestAcquireSharedDirReaderReleaseIdempotent(t *testing.T) {
	dirReaderMu.Lock()
	if dirReaderInst != nil {
		dirReaderMu.Unlock()
		t.Skip("singleton already held by another test")
	}
	dirReaderMu.Unlock()

	_, release := acquireSharedDirReader()
	release()
	release() // second call must be a no-op

	dirReaderMu.Lock()
	assert.Equal(t, 0, dirReaderRefs)
	assert.Nil(t, dirReaderInst)
	dirReaderMu.Unlock()
}

func TestAcquireSharedDirReaderResetOnLastRelease(t *testing.T) {
	dirReaderMu.Lock()
	if dirReaderInst != nil {
		dirReaderMu.Unlock()
		t.Skip("singleton already held by another test")
	}
	dirReaderMu.Unlock()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.log"), nil, 0o600))

	dc, release := acquireSharedDirReader()

	// Warm the cache.
	_, err := dc.readDirNames(dir, time.Hour, true)
	require.NoError(t, err)
	require.NotEmpty(t, dc.entries)

	// On last release, reset() clears the entries map — no leak.
	release()

	dirReaderMu.Lock()
	assert.Nil(t, dirReaderInst, "singleton should be nil after last release")
	dirReaderMu.Unlock()

	// The old dc pointer's entries were cleared by reset().
	dc.mu.Lock()
	assert.Empty(t, dc.entries, "reset should clear all cached entries")
	dc.mu.Unlock()
}
