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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"

	"github.com/elastic/elastic-agent-libs/paths"
)

// pathsAt returns a *paths.Path whose Data dir is a fresh per-test temp dir.
func pathsAt(t *testing.T) *paths.Path {
	t.Helper()
	p := paths.New()
	p.Data = t.TempDir()
	return p
}

// resolvedDBPath returns the absolute beat.db path for p.
func resolvedDBPath(p *paths.Path) string {
	return p.Resolve(paths.Data, dbFileName)
}

// loadValue fetches key from b as a string. Fails the test on error.
func loadValue(t *testing.T, b Bucket, key string) string {
	t.Helper()
	var got []byte
	err := b.Load(key, func(blob []byte) error {
		got = append([]byte(nil), blob...)
		return nil
	})
	assert.NoError(t, err, "Load(%q) must succeed", key)
	return string(got)
}

// TestDistinctPathsAreIsolated is the regression test for the original bug:
// before the path-keyed registry, OpenBucket's *paths.Path argument was
// silently ignored on every call after the first. Two callers passing
// different paths must now read and write to different beat.db files.
func TestDistinctPathsAreIsolated(t *testing.T) {
	p1, p2 := pathsAt(t), pathsAt(t)
	require.NotEqual(t, resolvedDBPath(p1), resolvedDBPath(p2),
		"the two test paths must resolve to different beat.db files")

	b1, err := OpenBucket("k", p1)
	require.NoError(t, err, "OpenBucket on first path")
	t.Cleanup(func() { assert.NoError(t, b1.Close(), "Close on first bucket") })
	b2, err := OpenBucket("k", p2)
	require.NoError(t, err, "OpenBucket on second path")
	t.Cleanup(func() { assert.NoError(t, b2.Close(), "Close on second bucket") })

	assert.NoError(t, b1.Store("x", []byte("from-p1")), "Store on first bucket")
	assert.NoError(t, b2.Store("x", []byte("from-p2")), "Store on second bucket")

	assert.Equal(t, "from-p1", loadValue(t, b1, "x"),
		"first bucket must read its own write, not the second bucket's")
	assert.Equal(t, "from-p2", loadValue(t, b2, "x"),
		"second bucket must read its own write, not the first bucket's")
}

// TestDoubleCloseReturnsError documents the public-API contract of the
// defensive guard against double-close. Without it, refCount would
// underflow silently.
func TestDoubleCloseReturnsError(t *testing.T) {
	b, err := OpenBucket("k", pathsAt(t))
	require.NoError(t, err, "OpenBucket")
	assert.NoError(t, b.Close(), "first Close should succeed")

	err = b.Close()
	assert.Error(t, err, "second Close on the same bucket must return an error")
	assert.Contains(t, err.Error(), "no outstanding references",
		"error should identify the cause as a missing reference")
}

// TestOpenBucketWithMigrationVisible verifies the migration callback runs
// inside a transaction that commits before the named bucket is returned,
// so the migration's writes are observable through the returned Bucket.
func TestOpenBucketWithMigrationVisible(t *testing.T) {
	migrate := func(tx *bolt.Tx) error {
		bk, err := tx.CreateBucketIfNotExists([]byte("dst"))
		if err != nil {
			return err
		}
		return bk.Put([]byte("seed"), []byte("ok"))
	}

	b, err := OpenBucketWithMigration("dst", pathsAt(t), migrate)
	require.NoError(t, err, "OpenBucketWithMigration")
	t.Cleanup(func() { assert.NoError(t, b.Close(), "Close after migration") })

	assert.Equal(t, "ok", loadValue(t, b, "seed"),
		"migration's Put must be visible to the returned bucket's Load")
}

// TestOpenBucketWithMigrationErrorPropagated verifies a failing migration
// callback returns the error wrapped and yields no Bucket. The
// corresponding "no leaked reference" half of this contract is asserted in
// registry_test.go via the registry's internal state.
func TestOpenBucketWithMigrationErrorPropagated(t *testing.T) {
	wantErr := errors.New("boom")

	b, err := OpenBucketWithMigration("dst", pathsAt(t), func(*bolt.Tx) error {
		return wantErr
	})
	require.Error(t, err, "OpenBucketWithMigration must surface the migration error")
	assert.ErrorIs(t, err, wantErr, "the original error must remain unwrappable from the returned error")
	assert.Nil(t, b, "no Bucket must be returned when migration fails")
}
