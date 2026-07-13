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

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
)

// rawEntry is the expected in-memory form of raw material added via AddRaw.
func rawEntry(raw, source string) shortFingerprintEntry {
	return shortFingerprintEntry{
		Hash:   loginp.HashRawFingerprint(raw),
		HexLen: len(raw),
		Source: source,
	}
}

func TestShortFingerprintIndex_Add(t *testing.T) {
	set := newShortFingerprintSet()

	t.Run("adds entry with non-empty fingerprint", func(t *testing.T) {
		key := "key1"
		want := rawEntry("aabb", "/a.log")
		set.AddRaw(key, "aabb", "/a.log")
		assert.Equal(t, 1, set.Len(), "non-empty fingerprint should be added")
		assert.Equal(t, want, set.entries[key], "ShortFingerprintSet entry did not match")
	})

	t.Run("adds entry regardless of fingerprint length", func(t *testing.T) {
		key := "key2"
		// A full-length (64-char) fingerprint, so this subtest actually
		// exercises length-independent insertion rather than re-using the
		// 4-char value above.
		raw := "aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899"
		want := rawEntry(raw, "/a.log")
		set.AddRaw(key, raw, "/a.log")
		assert.Equal(t, 2, set.Len(), "long fingerprint should be added")
		assert.Equal(t, want, set.entries[key], "ShortFingerprintSet entry did not match")
	})

	t.Run("rejects empty fingerprint", func(t *testing.T) {
		key := "key3"
		set.AddRaw(key, "", "/a.log")
		assert.Equal(t, 2, set.Len(), "length should not change")
		assert.Empty(t, set.entries[key], "empty fingerprint should not be added")
	})

	t.Run("rejects empty hash or non-positive length", func(t *testing.T) {
		set.Add("key4", "", 4, "/a.log")
		set.Add("key5", loginp.HashRawFingerprint("aabb"), 0, "/a.log")
		set.Add("key6", loginp.HashRawFingerprint("aabb"), -2, "/a.log")
		assert.Equal(t, 2, set.Len(), "invalid entries should not be added")
	})

	t.Run("re-adding a key replaces its index slot", func(t *testing.T) {
		set := newShortFingerprintSet()
		set.AddRaw("key", "aabb", "/a.log")
		set.AddRaw("key", "aabbcc", "/a.log")
		assert.Equal(t, 1, set.Len())

		// Only the new material matches; the stale (shorter) slot is gone.
		_, _, found := set.FindPrefixMatch("aabbccdd", "")
		require.True(t, found, "new material should match")
		gotKey, _, found := set.FindPrefixMatch("aabbdd", "")
		assert.False(t, found, "stale material must not match, got key %q", gotKey)
	})
}

func TestShortFingerprintIndex_Remove(t *testing.T) {
	idx := newShortFingerprintSet()
	idx.AddRaw("key1", "aabb", "/a.log")
	idx.AddRaw("key2", "ccdd", "/b.log")
	require.Equal(t, 2, idx.Len())

	idx.Remove("key1")
	assert.Equal(t, 1, idx.Len())
	_, _, found := idx.FindPrefixMatch("aabbcc", "")
	assert.False(t, found, "removed entry must not match anymore")

	idx.Remove("nonexistent") // no-op
	assert.Equal(t, 1, idx.Len())
}

func TestShortFingerprintIndex_RemoveBySource(t *testing.T) {
	idx := newShortFingerprintSet()
	idx.AddRaw("key1", "aabb", "/a.log")
	idx.AddRaw("key2", "ccdd", "/b.log")

	idx.RemoveBySource("/a.log")
	assert.Equal(t, 1, idx.Len())

	_, entry, found := idx.FindPrefixMatch("ccddee", "")
	require.True(t, found, "key2 should still be present")
	assert.Equal(t, "/b.log", entry.Source)
	_, _, found = idx.FindPrefixMatch("aabbcc", "")
	assert.False(t, found, "removed source must not match anymore")
}

func TestShortFingerprintIndex_UpdateSource(t *testing.T) {
	idx := newShortFingerprintSet()
	idx.AddRaw("key1", "aabb", "/a.log")

	idx.UpdateSource("key1", "/a.log.1")
	assert.Equal(t, "/a.log.1", idx.entries["key1"].Source)

	idx.UpdateSource("nonexistent", "/x.log") // no-op
	assert.Equal(t, 1, idx.Len())
}

func TestShortFingerprintIndex_FindPrefixMatch(t *testing.T) {
	idx := newShortFingerprintSet()
	idx.AddRaw("key1", "aabb", "/a.log")
	idx.AddRaw("key2", "ccdd", "/b.log")

	t.Run("finds prefix match without source check", func(t *testing.T) {
		key, entry, found := idx.FindPrefixMatch("aabbccdd", "")
		require.True(t, found, "should find prefix match")
		assert.Equal(t, "key1", key)
		assert.Equal(t, rawEntry("aabb", "/a.log"), entry)
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

	t.Run("same length different content is not a match", func(t *testing.T) {
		_, _, found := idx.FindPrefixMatch("aacc", "")
		assert.False(t, found, "stored 'aabb' does not hash-match target prefix 'aacc'")
	})

	t.Run("without source filter returns longest prefix match", func(t *testing.T) {
		idx := newShortFingerprintSet()
		idx.AddRaw("short", "aa", "/x.log")
		idx.AddRaw("long", "aabb", "/y.log")

		key, entry, found := idx.FindPrefixMatch("aabbccdd", "")
		require.True(t, found, "should find prefix match")
		assert.Equal(t, "long", key, "should pick the longest prefix match")
		assert.Equal(t, rawEntry("aabb", "/y.log"), entry)
	})

	t.Run("filtered-out longer match falls back to shorter match", func(t *testing.T) {
		idx := newShortFingerprintSet()
		idx.AddRaw("short", "aa", "/x.log")
		idx.AddRaw("long", "aabb", "/y.log")

		key, _, found := idx.FindPrefixMatch("aabbccdd", "/x.log")
		require.True(t, found, "shorter entry matching the source filter should be found")
		assert.Equal(t, "short", key)
	})

	t.Run("distinct keys sharing identical material both stay findable", func(t *testing.T) {
		// The watcher's set is path-keyed, so two identical-content files
		// produce two keys with the same (length, hash). Removing one must not
		// orphan the other.
		idx := newShortFingerprintSet()
		idx.AddRaw("/x.log", "aabb", "/x.log")
		idx.AddRaw("/y.log", "aabb", "/y.log")
		require.Equal(t, 2, idx.Len())

		idx.Remove("/x.log")
		key, entry, found := idx.FindPrefixMatch("aabbccdd", "")
		require.True(t, found, "the remaining twin must still match")
		assert.Equal(t, "/y.log", key)
		assert.Equal(t, "/y.log", entry.Source)
	})
}

// TestShortFingerprintIndex_HashFormulaMatchesRegistryKey pins the invariant
// the whole scheme rests on: hashing a target's prefix during FindPrefixMatch
// produces exactly the value FingerprintID.Key() produced when that prefix was
// the full raw material — which is also the registry key's identity tail that
// buildShortFingerprintSet feeds to Add. If the streaming implementation ever
// diverges from HashRawFingerprint, restart migration silently breaks; this
// test makes that loud.
func TestShortFingerprintIndex_HashFormulaMatchesRegistryKey(t *testing.T) {
	target := "00112233445566778899aabbccddeeff"
	for hexLen := 2; hexLen < len(target); hexLen += 2 {
		prefix := target[:hexLen]

		idx := newShortFingerprintSet()
		// Simulate the registry-load path: the hash comes from the key that
		// FingerprintID.Key() produced for the prefix, the length from the
		// persisted fileMeta.FingerprintLen.
		keyHash := loginp.FingerprintID{Raw: prefix}.Key()
		idx.Add("key", keyHash, hexLen, "/a.log")

		got, _, found := idx.FindPrefixMatch(target, "")
		require.True(t, found, "prefix of hex length %d must match", hexLen)
		assert.Equal(t, "key", got)
	}
}
