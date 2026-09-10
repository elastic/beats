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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- globBase ---------------------------------------------------------------

func TestGlobBase(t *testing.T) {
	tests := []struct {
		pattern string
		want    string
	}{
		// glob patterns
		{"/var/log/containers/*.log", "/var/log/containers"},
		{"/var/log/**/*.log", "/var/log"},
		{"/var/log/*/containers.log", "/var/log"},
		{"*.log", "."},
		// literal file paths (no glob chars)
		{"/var/log/containers/pod.log", "/var/log/containers"},
		{"/var/log/containers/a/b.log", "/var/log/containers/a"},
	}
	for _, tc := range tests {
		t.Run(tc.pattern, func(t *testing.T) {
			assert.Equal(t, filepath.FromSlash(tc.want), globBase(filepath.FromSlash(tc.pattern)))
		})
	}
}

// ---- per-path TTL ref counting ----------------------------------------------

func TestPerPathTTLSingleRegistration(t *testing.T) {
	dr := newCachedDirReader(time.Second)
	defer dr.stop()

	deregFn := dr.registerPath("/var/log/containers", 200*time.Millisecond)
	defer deregFn()

	dr.mu.Lock()
	assert.Equal(t, 200*time.Millisecond, dr.ttlForDir("/var/log/containers"))
	dr.mu.Unlock()
}

func TestPerPathTTLMinOfMultipleRegistrations(t *testing.T) {
	dr := newCachedDirReader(time.Second)
	defer dr.stop()

	deregA := dr.registerPath("/var/log/containers", time.Second)
	deregB := dr.registerPath("/var/log/containers", 200*time.Millisecond)
	defer deregA()
	defer deregB()

	dr.mu.Lock()
	assert.Equal(t, 200*time.Millisecond, dr.ttlForDir("/var/log/containers"))
	dr.mu.Unlock()
}

func TestPerPathTTLRisesWhenFastWatcherDeregisters(t *testing.T) {
	dr := newCachedDirReader(time.Second)
	defer dr.stop()

	deregA := dr.registerPath("/var/log/containers", time.Second)
	deregB := dr.registerPath("/var/log/containers", 200*time.Millisecond)
	defer deregA()

	// B (fast) stops first: TTL rises back to A's interval
	deregB()

	dr.mu.Lock()
	assert.Equal(t, time.Second, dr.ttlForDir("/var/log/containers"))
	dr.mu.Unlock()
}

func TestPerPathTTLRevertsToDefaultWhenAllDeregister(t *testing.T) {
	const defaultTTL = time.Second
	dr := newCachedDirReader(defaultTTL)
	defer dr.stop()

	deregA := dr.registerPath("/var/log/containers", time.Second)
	deregB := dr.registerPath("/var/log/containers", 200*time.Millisecond)

	deregA()
	deregB()

	dr.mu.Lock()
	assert.Equal(t, defaultTTL, dr.ttlForDir("/var/log/containers"))
	dr.mu.Unlock()
}

func TestPerPathTTLDeregisterIdempotent(t *testing.T) {
	dr := newCachedDirReader(time.Second)
	defer dr.stop()

	deregFn := dr.registerPath("/var/log/containers", 200*time.Millisecond)
	deregFn()
	deregFn() // second call must be a no-op

	dr.mu.Lock()
	assert.Equal(t, time.Second, dr.ttlForDir("/var/log/containers"), "double deregister must not undercount")
	dr.mu.Unlock()
}

func TestPerPathTTLIndependentPaths(t *testing.T) {
	dr := newCachedDirReader(time.Second)
	defer dr.stop()

	deregA := dr.registerPath("/var/log/containers", 200*time.Millisecond)
	deregB := dr.registerPath("/var/log/pods", time.Second)
	defer deregA()
	defer deregB()

	dr.mu.Lock()
	assert.Equal(t, 200*time.Millisecond, dr.ttlForDir("/var/log/containers"))
	assert.Equal(t, time.Second, dr.ttlForDir("/var/log/pods"))
	dr.mu.Unlock()
}

// ---- cache hit / miss -------------------------------------------------------

func TestCachedDirReaderCacheHit(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.log"), nil, 0o600))

	// defaultTTL long enough that the per-path registration drives the TTL.
	dr := newCachedDirReader(time.Second)
	defer dr.stop()
	deregFn := dr.registerPath(dir, 50*time.Millisecond)
	defer deregFn()

	// Warm the cache.
	listing, err := dr.readDir(dir)
	require.NoError(t, err)
	assert.Equal(t, []string{"a.log"}, listing.names)

	// Add a file while the cache is warm.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.log"), nil, 0o600))

	// Second read within TTL: still sees only a.log.
	listing, err = dr.readDir(dir)
	require.NoError(t, err)
	assert.Equal(t, []string{"a.log"}, listing.names)
}

func TestCachedDirReaderCacheMissAfterTTL(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.log"), nil, 0o600))

	dr := newCachedDirReader(time.Second)
	defer dr.stop()
	deregFn := dr.registerPath(dir, 50*time.Millisecond)
	defer deregFn()

	// Warm the cache then add a file.
	_, err := dr.readDir(dir)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.log"), nil, 0o600))

	// Wait past the 50 ms per-path TTL.
	time.Sleep(100 * time.Millisecond)

	listing, err := dr.readDir(dir)
	require.NoError(t, err)
	assert.Equal(t, []string{"a.log", "b.log"}, listing.names)
}

func TestCachedDirReaderDeregisterInvalidatesCache(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.log"), nil, 0o600))

	// Use a very long TTL so the cache would not expire on its own.
	dr := newCachedDirReader(time.Hour)
	defer dr.stop()
	deregFn := dr.registerPath(dir, time.Hour)

	// Warm the cache then add a file.
	_, err := dr.readDir(dir)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.log"), nil, 0o600))

	// Deregister: must invalidate the cached entry.
	deregFn()

	listing, err := dr.readDir(dir)
	require.NoError(t, err)
	assert.Equal(t, []string{"a.log", "b.log"}, listing.names)
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

	dr1, release1 := acquireSharedDirReader()
	dr2, release2 := acquireSharedDirReader()

	rd1, ok1 := dr1.(*cachedDirReader)
	rd2, ok2 := dr2.(*cachedDirReader)
	require.True(t, ok1 && ok2, "acquireSharedDirReader must return *cachedDirReader")

	dirReaderMu.Lock()
	assert.Equal(t, 2, dirReaderRefs, "two acquires should give ref count 2")
	assert.Same(t, rd1, rd2, "both acquires should return the same instance")
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
