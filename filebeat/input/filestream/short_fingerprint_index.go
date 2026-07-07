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
	"slices"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
)

// shortFingerprintSet manages a set of entries whose fingerprint is still in the growing phase.
type shortFingerprintSet struct {
	entries map[string]shortFingerprintEntry // key → entry
	// byLen indexes entry keys by (hex length, hash) so a lookup is one hash snapshot and one map
	// access per distinct stored length, independent of the number of entries.
	byLen map[int]map[string]map[string]struct{} // hex length → hash → keys

	// Scratch state reused across FindPrefixMatchFunc calls to keep lookups allocation-free.
	hasher  *loginp.RawFingerprintHasher
	target  []byte
	lengths []int
}

// shortFingerprintEntry represents a tracked short fingerprint.
type shortFingerprintEntry struct {
	Hash   string // hex(sha256(raw)) — FingerprintID.Key() of the raw material
	HexLen int    // len(raw): hex characters, i.e. 2× the content bytes
	Source string // file path
}

func newShortFingerprintSet() *shortFingerprintSet {
	return &shortFingerprintSet{
		entries: make(map[string]shortFingerprintEntry),
		byLen:   make(map[int]map[string]map[string]struct{}),
		hasher:  loginp.NewRawFingerprintHasher(),
	}
}

// Add adds an entry from its already-derived (hash, hex length) form, e.g.
// loaded from the registry where the key carries the hash and the value the
// length. Callers must only add entries that are currently in the growing
// phase; the set does not enforce this. Entries with an empty hash or a
// non-positive length are ignored. Safe to call on a nil receiver.
func (s *shortFingerprintSet) Add(key, hash string, hexLen int, source string) {
	if s == nil || hash == "" || hexLen <= 0 {
		return
	}

	if old, ok := s.entries[key]; ok {
		s.deleteIndexSlot(old.HexLen, old.Hash, key)
	}
	s.entries[key] = shortFingerprintEntry{Hash: hash, HexLen: hexLen, Source: source}

	hashes := s.byLen[hexLen]
	if hashes == nil {
		hashes = make(map[string]map[string]struct{})
		s.byLen[hexLen] = hashes
	}
	keys := hashes[hash]
	if keys == nil {
		keys = make(map[string]struct{})
		hashes[hash] = keys
	}
	keys[key] = struct{}{}
}

// AddRaw adds an entry from in-memory raw fingerprint material, deriving the (hash, hex length)
// form. The raw material itself is not retained. Entries with an empty fingerprint are ignored.
// Safe to call on a nil receiver.
func (s *shortFingerprintSet) AddRaw(key, raw, source string) {
	if s == nil || raw == "" {
		return
	}
	s.Add(key, loginp.HashRawFingerprint(raw), len(raw), source)
}

// deleteIndexSlot removes key from the (hexLen, hash) bucket, pruning emptied
// maps: a long-lived set would otherwise accumulate dead lengths (every
// growth migration retires one) and pay a hash snapshot per dead length on
// every lookup.
func (s *shortFingerprintSet) deleteIndexSlot(hexLen int, hash, key string) {
	hashes := s.byLen[hexLen]
	if hashes == nil {
		return
	}
	keys := hashes[hash]
	if keys == nil {
		return
	}
	delete(keys, key)
	if len(keys) == 0 {
		delete(hashes, hash)
	}
	if len(hashes) == 0 {
		delete(s.byLen, hexLen)
	}
}

// Remove removes an entry by key. Safe to call on a nil receiver.
func (s *shortFingerprintSet) Remove(key string) {
	if s == nil {
		return
	}
	if e, ok := s.entries[key]; ok {
		s.deleteIndexSlot(e.HexLen, e.Hash, key)
		delete(s.entries, key)
	}
}

// RemoveBySource removes every entry whose source matches.
// Used on truncation where the key is unknown (fingerprint changed). The set is
// keyed by registry key, not source, and the one-entry-per-source property is
// not enforced, so all matching entries must be removed — otherwise a stale
// entry would survive and keep participating in prefix matching.
// Safe to call on a nil receiver.
func (s *shortFingerprintSet) RemoveBySource(source string) {
	if s == nil {
		return
	}
	for key, entry := range s.entries {
		if entry.Source == source {
			s.deleteIndexSlot(entry.HexLen, entry.Hash, key)
			delete(s.entries, key)
		}
	}
}

// UpdateSource updates the source path for an entry.
// Safe to call on a nil receiver.
func (s *shortFingerprintSet) UpdateSource(key, newSource string) {
	if s == nil {
		return
	}
	if entry, ok := s.entries[key]; ok {
		entry.Source = newSource
		s.entries[key] = entry
	}
}

// FindPrefixMatch finds the entry whose fingerprint is the longest strict
// prefix of targetFingerprint. If matchSource is non-empty, only entries with
// entry.Source == matchSource are considered. Picking the longest prefix (in
// both branches) avoids returning a shorter, less-specific entry when more than
// one entry shares a source: the set is keyed by registry key, not source, and
// the one-entry-per-source property is not enforced.
// Returns the key and entry on match. Safe to call on a nil receiver.
func (s *shortFingerprintSet) FindPrefixMatch(targetFingerprint, matchSource string) (key string, entry shortFingerprintEntry, found bool) {
	keep := func(e shortFingerprintEntry) bool {
		return matchSource == "" || e.Source == matchSource
	}
	return s.FindPrefixMatchFunc(targetFingerprint, keep)
}

// FindPrefixMatchFunc finds the entry with the longest fingerprint that is a
// strict prefix of targetFingerprint and for which keep(entry) returns true
// (a nil keep accepts every entry). It lets callers apply an arbitrary filter:
// the path-agnostic rename fallback uses it to select the longest
// rename-eligible candidate instead of testing only the single longest prefix
// match (which could discard a genuine rename in favor of a still-present
// distinct file that merely shares a longer header).
//
// targetFingerprint is streamed through a single hasher; at every stored
// length below len(targetFingerprint) the hash is snapshotted and looked up in
// that length's bucket, so the cost is one pass over the target plus a map
// access per distinct stored length — no raw fingerprint material is needed or
// kept. Returns the key and entry on match. Safe to call on a nil receiver.
func (s *shortFingerprintSet) FindPrefixMatchFunc(targetFingerprint string, keep func(shortFingerprintEntry) bool) (key string, entry shortFingerprintEntry, found bool) {
	if s == nil || targetFingerprint == "" || len(s.entries) == 0 {
		return "", shortFingerprintEntry{}, false
	}

	// Only lengths strictly below the target's can hold strict prefixes.
	s.lengths = s.lengths[:0]
	for l := range s.byLen {
		if l < len(targetFingerprint) {
			s.lengths = append(s.lengths, l)
		}
	}
	if len(s.lengths) == 0 {
		return "", shortFingerprintEntry{}, false
	}
	slices.Sort(s.lengths)

	// Ascending walk so the hash is fed incrementally; a hit at a longer length overwrites earlier
	// ones, implementing longest-match-wins.
	s.hasher.Reset()
	s.target = append(s.target[:0], targetFingerprint...)
	fed := 0
	for _, l := range s.lengths {
		s.hasher.Feed(s.target[fed:l])
		fed = l
		candidates := s.byLen[l][string(s.hasher.Key())]
		for candidate := range candidates {
			e := s.entries[candidate]
			if keep == nil || keep(e) {
				key, entry, found = candidate, e, true
				break
			}
		}
	}
	return key, entry, found
}

// Len returns the number of entries. Safe to call on a nil receiver.
func (s *shortFingerprintSet) Len() int {
	if s == nil {
		return 0
	}
	return len(s.entries)
}
