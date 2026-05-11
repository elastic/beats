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

package datastore

import (
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
)

// snapshotEntry returns refCount and existence for path in r under the
// registry mutex. Tests use this to assert the lifecycle invariants
// (entries created on first open, removed on last release, no leaks on
// migration failure, etc.) without racing against the registry.
func snapshotEntry(r *registry, path string) (refCount int, exists bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if e, ok := r.dbs[path]; ok {
		return e.refCount, true
	}
	return 0, false
}

// TestRegistryRefCount: opening multiple buckets on the same path shares
// one registry entry; the entry stays in the map while any reference is
// outstanding and is removed exactly when the last bucket is closed.
func TestRegistryRefCount(t *testing.T) {
	p := pathsAt(t)
	path := resolvedDBPath(p)

	b1, err := OpenBucket("a", p)
	require.NoError(t, err, "first OpenBucket")
	b2, err := OpenBucket("b", p)
	require.NoError(t, err, "second OpenBucket on same path")

	rc, ok := snapshotEntry(defaultRegistry, path)
	require.True(t, ok, "registry must hold an entry while buckets are open")
	assert.Equal(t, 2, rc, "two open buckets on one path must share refCount=2")

	require.NoError(t, b1.Close(), "Close on first bucket")
	rc, ok = snapshotEntry(defaultRegistry, path)
	require.True(t, ok, "entry must persist while a second bucket is still open")
	assert.Equal(t, 1, rc, "refCount must drop to 1 after one Close")

	require.NoError(t, b2.Close(), "Close on second bucket")
	_, ok = snapshotEntry(defaultRegistry, path)
	assert.False(t, ok, "entry must be removed once the last bucket is closed")
}

// TestRegistryReopenAfterFullClose: after the last bucket on a path is
// closed, the bolt file lock is released and the registry entry is gone,
// so a subsequent OpenBucket on the same path opens a fresh DB. This
// exercises the open-after-close cycle that the original singleton
// implementation could not support.
func TestRegistryReopenAfterFullClose(t *testing.T) {
	p := pathsAt(t)
	path := resolvedDBPath(p)

	b, err := OpenBucket("a", p)
	require.NoError(t, err, "first OpenBucket")
	require.NoError(t, b.Close(), "Close before reopen")

	_, ok := snapshotEntry(defaultRegistry, path)
	require.False(t, ok, "entry must be cleaned up before reopen")

	b, err = OpenBucket("a", p)
	require.NoError(t, err, "OpenBucket after full close must succeed")
	t.Cleanup(func() { assert.NoError(t, b.Close(), "Close on reopened bucket") })

	rc, ok := snapshotEntry(defaultRegistry, path)
	require.True(t, ok, "reopen must create a fresh entry")
	assert.Equal(t, 1, rc, "fresh entry must start with refCount=1")
}

// TestRegistryMigrationFailureReleasesRef: when the migration callback
// returns an error, the acquired reference must be released so the path
// is left as if no open had been attempted. Without this, every failed
// migration would permanently leak a refCount.
func TestRegistryMigrationFailureReleasesRef(t *testing.T) {
	p := pathsAt(t)
	path := resolvedDBPath(p)

	_, err := OpenBucketWithMigration("a", p, func(*bolt.Tx) error {
		return errors.New("boom")
	})
	require.Error(t, err, "failing migration must surface an error")

	_, ok := snapshotEntry(defaultRegistry, path)
	assert.False(t, ok, "failed migration must not leave a registry entry")

	b, err := OpenBucket("a", p)
	require.NoError(t, err, "post-failure OpenBucket must succeed on the same path")
	t.Cleanup(func() { assert.NoError(t, b.Close(), "Close on post-failure bucket") })

	rc, _ := snapshotEntry(defaultRegistry, path)
	assert.Equal(t, 1, rc, "post-failure entry must start at refCount=1, proving the prior failure released its ref")
}

// TestRegistryDeleteBucketReleasesRef: DeleteBucket also drops the
// boltBucket's reference on the underlying DB, so when no other buckets
// share the path the registry entry is removed (and the bolt DB closed).
// Without this the new ref-count introduces a leak path that the old code
// did not have.
func TestRegistryDeleteBucketReleasesRef(t *testing.T) {
	p := pathsAt(t)

	b, err := OpenBucket("a", p)
	require.NoError(t, err, "OpenBucket")
	require.NoError(t, b.DeleteBucket(), "DeleteBucket must succeed and release the bucket's reference")

	_, ok := snapshotEntry(defaultRegistry, resolvedDBPath(p))
	assert.False(t, ok, "DeleteBucket on the only outstanding bucket must drive refCount to 0 and remove the entry")
}

// TestDoubleCloseDoesNotStealSiblingRef: closing one bucket twice must
// not drive the shared refCount past 0 and invalidate a sibling bucket
// holding the same path's DB handle.
func TestDoubleCloseDoesNotStealSiblingRef(t *testing.T) {
	p := pathsAt(t)
	path := resolvedDBPath(p)

	b1, err := OpenBucket("a", p)
	require.NoError(t, err, "first OpenBucket")
	t.Cleanup(func() { _ = b1.Close() })
	b2, err := OpenBucket("b", p)
	require.NoError(t, err, "second OpenBucket on the same path")

	require.NoError(t, b2.Close(), "first Close on b2 must succeed")
	require.NoError(t, b2.Close(), "second Close on b2 must be a no-op and must not decrement the shared refcount")

	rc, ok := snapshotEntry(defaultRegistry, path)
	require.True(t, ok, "entry must persist while b1 is still open")
	assert.Equal(t, 1, rc, "double-Close on b2 must leave b1's reference intact (refCount=1, not 0)")

	assert.NoError(t, b1.Store("x", []byte("still alive")),
		"b1 must remain usable; its DB handle must not have been closed by b2's double-Close")
}

// TestDeleteBucketThenDeferredCloseDoesNotStealSiblingRef: DeleteBucket
// already calls Close internally, so a deferred Close after it must be a
// no-op and must not invalidate a sibling bucket on the same path.
func TestDeleteBucketThenDeferredCloseDoesNotStealSiblingRef(t *testing.T) {
	p := pathsAt(t)
	path := resolvedDBPath(p)

	sibling, err := OpenBucket("keep", p)
	require.NoError(t, err, "OpenBucket for sibling")
	t.Cleanup(func() { assert.NoError(t, sibling.Close(), "sibling Close") })

	victim, err := OpenBucket("drop", p)
	require.NoError(t, err, "OpenBucket for victim")

	assert.NoError(t, victim.DeleteBucket(), "DeleteBucket on victim")
	assert.NoError(t, victim.Close(),
		"deferred Close after DeleteBucket must be a no-op (not a refcount decrement)")

	rc, ok := snapshotEntry(defaultRegistry, path)
	require.True(t, ok, "registry entry must persist while sibling is still open")
	assert.Equal(t, 1, rc,
		"DeleteBucket+deferred Close on victim must leave sibling's refcount untouched")

	assert.NoError(t, sibling.Store("x", []byte("still alive")),
		"sibling must remain usable after victim's DeleteBucket+deferred Close")
}

// TestRegistryConcurrentOpenClose stresses the locking on the path-keyed
// registry. Run with -race; the post-condition is that no entries are
// leaked, i.e. the registry has no entry for the contended path once all
// goroutines finish.
func TestRegistryConcurrentOpenClose(t *testing.T) {
	p := pathsAt(t)
	const goroutines = 16
	const iters = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iters; j++ {
				b, err := OpenBucket("a", p)
				if err != nil {
					t.Errorf("concurrent OpenBucket: %v", err)
					return
				}
				if err := b.Close(); err != nil {
					t.Errorf("concurrent Close: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()

	_, ok := snapshotEntry(defaultRegistry, resolvedDBPath(p))
	assert.False(t, ok, "no registry entry must remain after concurrent open/close cycles")
}
