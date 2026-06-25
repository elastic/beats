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

// shortFingerprintSet manages a set of entries whose fingerprint is still
// in the growing phase (raw-hex, below the configured threshold). Used by both
// the filewatcher (for rename+grow detection) and the prospector (for key
// migration on growth and threshold transition).
type shortFingerprintSet struct {
	entries map[string]shortFingerprintEntry // key → entry
}

// shortFingerprintEntry represents a tracked short fingerprint.
type shortFingerprintEntry struct {
	Fingerprint string
	Source      string // file path
}

func newShortFingerprintSet() *shortFingerprintSet {
	return &shortFingerprintSet{
		entries: make(map[string]shortFingerprintEntry),
	}
}

// Add adds an entry. Callers must only add entries that are currently in the
// growing phase; the index does not enforce this. Entries with an empty
// fingerprint are ignored. Safe to call on a nil receiver.
func (idx *shortFingerprintSet) Add(key, fingerprint, source string) {
	if idx == nil || fingerprint == "" {
		return
	}

	idx.entries[key] = shortFingerprintEntry{Fingerprint: fingerprint, Source: source}
}

// Remove removes an entry by key. Safe to call on a nil receiver.
func (idx *shortFingerprintSet) Remove(key string) {
	if idx == nil {
		return
	}
	delete(idx.entries, key)
}

// RemoveBySource removes every entry whose source matches.
// Used on truncation where the key is unknown (fingerprint changed). The set is
// keyed by registry key, not source, and the one-entry-per-source property is
// not enforced, so all matching entries must be removed — otherwise a stale
// entry would survive and keep participating in prefix matching.
// Safe to call on a nil receiver.
func (idx *shortFingerprintSet) RemoveBySource(source string) {
	if idx == nil {
		return
	}
	for key, entry := range idx.entries {
		if entry.Source == source {
			delete(idx.entries, key)
		}
	}
}

// UpdateSource updates the source path for an entry.
// Safe to call on a nil receiver.
func (idx *shortFingerprintSet) UpdateSource(key, newSource string) {
	if idx == nil {
		return
	}
	if entry, ok := idx.entries[key]; ok {
		entry.Source = newSource
		idx.entries[key] = entry
	}
}

// FindPrefixMatch finds the entry whose fingerprint is the longest strict
// prefix of targetFingerprint. If matchSource is non-empty, only entries with
// entry.Source == matchSource are considered. Picking the longest prefix (in
// both branches) avoids returning a shorter, less-specific entry when more than
// one entry shares a source: the set is keyed by registry key, not source, and
// the one-entry-per-source property is not enforced.
// Returns the key and entry on match. Safe to call on a nil receiver.
func (idx *shortFingerprintSet) FindPrefixMatch(targetFingerprint, matchSource string) (key string, entry shortFingerprintEntry, found bool) {
	keep := func(e shortFingerprintEntry) bool {
		return matchSource == "" || e.Source == matchSource
	}
	return idx.FindPrefixMatchFunc(targetFingerprint, keep)
}

// FindPrefixMatchFunc finds the entry with the longest fingerprint that is a
// strict prefix of targetFingerprint and for which keep(entry) returns true
// (a nil keep accepts every entry). It lets callers apply an arbitrary filter:
// the path-agnostic rename fallback uses it to select the longest
// rename-eligible candidate instead of testing only the single longest prefix
// match (which could discard a genuine rename in favor of a still-present
// distinct file that merely shares a longer header).
// Returns the key and entry on match. Safe to call on a nil receiver.
func (idx *shortFingerprintSet) FindPrefixMatchFunc(targetFingerprint string, keep func(shortFingerprintEntry) bool) (key string, entry shortFingerprintEntry, found bool) {
	if idx == nil || targetFingerprint == "" {
		return "", shortFingerprintEntry{}, false
	}
	for k, e := range idx.entries {
		if !isStrictPrefix(targetFingerprint, e.Fingerprint) {
			continue
		}
		if keep != nil && !keep(e) {
			continue
		}
		if !found || len(e.Fingerprint) > len(entry.Fingerprint) {
			key, entry, found = k, e, true
		}
	}
	return key, entry, found
}

// Len returns the number of entries. Safe to call on a nil receiver.
func (idx *shortFingerprintSet) Len() int {
	if idx == nil {
		return 0
	}
	return len(idx.entries)
}
