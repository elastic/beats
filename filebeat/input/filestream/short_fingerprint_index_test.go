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
	set := newShortFingerprintSet()

	t.Run("adds entry with non-empty fingerprint", func(t *testing.T) {
		key := "key1"
		want := shortFingerprintEntry{
			Fingerprint: "aabb",
			Source:      "/a.log",
		}
		set.Add(key, want.Fingerprint, want.Source)
		assert.Equal(t, 1, set.Len(), "non-empty fingerprint should be added")
		assert.Equal(t, want, set.entries[key], "ShortFingerprintSet entry did not match")
	})

	t.Run("adds entry regardless of fingerprint length", func(t *testing.T) {
		key := "key2"
		want := shortFingerprintEntry{
			Fingerprint: "aabb",
			Source:      "/a.log",
		}
		set.Add(key, want.Fingerprint, want.Source)
		assert.Equal(t, 2, set.Len(), "long fingerprint should be added")
		assert.Equal(t, want, set.entries[key], "ShortFingerprintSet entry did not match")
	})

	t.Run("rejects empty fingerprint", func(t *testing.T) {
		key := "key3"
		want := shortFingerprintEntry{
			Fingerprint: "",
			Source:      "/a.log",
		}
		set.Add("key3", want.Fingerprint, want.Source)
		assert.Equal(t, 2, set.Len(), "length should not change")
		assert.Empty(t, set.entries[key], "empty fingerprint should not be added")
	})
}

func TestShortFingerprintIndex_Remove(t *testing.T) {
	idx := newShortFingerprintSet()
	idx.Add("key1", "aabb", "/a.log")
	idx.Add("key2", "ccdd", "/b.log")
	require.Equal(t, 2, idx.Len())

	idx.Remove("key1")
	assert.Equal(t, 1, idx.Len())

	idx.Remove("nonexistent") // no-op
	assert.Equal(t, 1, idx.Len())
}

func TestShortFingerprintIndex_RemoveBySource(t *testing.T) {
	idx := newShortFingerprintSet()
	idx.Add("key1", "aabb", "/a.log")
	idx.Add("key2", "ccdd", "/b.log")

	idx.RemoveBySource("/a.log")
	assert.Equal(t, 1, idx.Len())

	_, entry, found := idx.FindPrefixMatch("ccddee", "")
	require.True(t, found, "key2 should still be present")
	assert.Equal(t, "/b.log", entry.Source)
}

func TestShortFingerprintIndex_UpdateSource(t *testing.T) {
	idx := newShortFingerprintSet()
	idx.Add("key1", "aabb", "/a.log")

	idx.UpdateSource("key1", "/a.log.1")
	assert.Equal(t, "/a.log.1", idx.entries["key1"].Source)

	idx.UpdateSource("nonexistent", "/x.log") // no-op
	assert.Equal(t, 1, idx.Len())
}

func TestShortFingerprintIndex_FindPrefixMatch(t *testing.T) {
	idx := newShortFingerprintSet()
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
		idx := newShortFingerprintSet()
		idx.Add("short", "aa", "/x.log")
		idx.Add("long", "aabb", "/y.log")

		key, entry, found := idx.FindPrefixMatch("aabbccdd", "")
		require.True(t, found, "should find prefix match")
		assert.Equal(t, "long", key, "should pick the longest prefix match")
		assert.Equal(t, "aabb", entry.Fingerprint)
	})
}
