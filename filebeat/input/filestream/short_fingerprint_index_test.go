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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShortFingerprintIndex_Add(t *testing.T) {
	idx := newShortFingerprintIndex(10) // max 10 chars

	t.Run("adds entry with short fingerprint", func(t *testing.T) {
		ok := idx.Add("key1", "aabb", "/a.log")
		assert.True(t, ok, "short fingerprint should be added")
		assert.Equal(t, 1, idx.Len())
	})

	t.Run("rejects fingerprint at max length", func(t *testing.T) {
		ok := idx.Add("key2", "0123456789", "/b.log")
		assert.False(t, ok, "fingerprint at max length should not be added")
		assert.Equal(t, 1, idx.Len())
	})

	t.Run("rejects fingerprint over max length", func(t *testing.T) {
		ok := idx.Add("key3", "01234567890", "/c.log")
		assert.False(t, ok, "fingerprint over max length should not be added")
	})

	t.Run("rejects empty fingerprint", func(t *testing.T) {
		ok := idx.Add("key4", "", "/d.log")
		assert.False(t, ok, "empty fingerprint should not be added")
	})
}

func TestShortFingerprintIndex_Remove(t *testing.T) {
	idx := newShortFingerprintIndex(20)
	idx.Add("key1", "aabb", "/a.log")
	idx.Add("key2", "ccdd", "/b.log")
	require.Equal(t, 2, idx.Len())

	idx.Remove("key1")
	assert.Equal(t, 1, idx.Len())

	idx.Remove("nonexistent") // no-op
	assert.Equal(t, 1, idx.Len())
}

func TestShortFingerprintIndex_RemoveBySource(t *testing.T) {
	idx := newShortFingerprintIndex(20)
	idx.Add("key1", "aabb", "/a.log")
	idx.Add("key2", "ccdd", "/b.log")

	idx.RemoveBySource("/a.log")
	assert.Equal(t, 1, idx.Len())

	_, entry, found := idx.FindPrefixMatch("ccddee", "")
	require.True(t, found, "key2 should still be present")
	assert.Equal(t, "/b.log", entry.Source)
}

func TestShortFingerprintIndex_UpdateSource(t *testing.T) {
	idx := newShortFingerprintIndex(20)
	idx.Add("key1", "aabb", "/a.log")

	idx.UpdateSource("key1", "/a.log.1")
	assert.Equal(t, "/a.log.1", idx.entries["key1"].Source)

	idx.UpdateSource("nonexistent", "/x.log") // no-op
	assert.Equal(t, 1, idx.Len())
}

func TestShortFingerprintIndex_FindPrefixMatch(t *testing.T) {
	idx := newShortFingerprintIndex(20)
	idx.Add("key1", "aabb", "/a.log")
	idx.Add("key2", "ccdd", "/b.log")

	t.Run("finds prefix match without source check", func(t *testing.T) {
		key, entry, found := idx.FindPrefixMatch("aabbccdd", "")
		require.True(t, found, "should find prefix match")
		assert.Equal(t, "key1", key)
		assert.Equal(t, "aabb", entry.Fingerprint)
		assert.Equal(t, "/a.log", entry.Source)
	})

	t.Run("finds prefix match with source check", func(t *testing.T) {
		key, _, found := idx.FindPrefixMatch("aabbccdd", "/a.log")
		require.True(t, found, "should find prefix match with matching source")
		assert.Equal(t, "key1", key)
	})

	t.Run("rejects match with wrong source", func(t *testing.T) {
		_, _, found := idx.FindPrefixMatch("aabbccdd", "/wrong.log")
		assert.False(t, found, "should not match with wrong source")
	})

	t.Run("no match for unrelated fingerprint", func(t *testing.T) {
		_, _, found := idx.FindPrefixMatch("eeff0011", "")
		assert.False(t, found, "unrelated fingerprint should not match")
	})

	t.Run("empty target returns no match", func(t *testing.T) {
		_, _, found := idx.FindPrefixMatch("", "")
		assert.False(t, found, "empty target should not match")
	})

	t.Run("stored fingerprint not shorter than target returns no match", func(t *testing.T) {
		_, _, found := idx.FindPrefixMatch("aa", "")
		assert.False(t, found, "stored 'aabb' is not shorter than target 'aa'")
	})

	t.Run("exact same length is not a prefix match", func(t *testing.T) {
		_, _, found := idx.FindPrefixMatch("aabb", "")
		assert.False(t, found, "stored 'aabb' is not shorter than target 'aabb'")
	})

	t.Run("without source filter returns longest prefix match", func(t *testing.T) {
		idx := newShortFingerprintIndex(20)
		idx.Add("short", "aa", "/x.log")
		idx.Add("long", "aabb", "/y.log")

		key, entry, found := idx.FindPrefixMatch("aabbccdd", "")
		require.True(t, found, "should find prefix match")
		assert.Equal(t, "long", key, "should pick the longest prefix match")
		assert.Equal(t, "aabb", entry.Fingerprint)
	})
}
