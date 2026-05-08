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

import "strings"

// shortFingerprintIndex manages a set of fingerprint entries shorter than maxLen
// and provides O(K) prefix matching. Used by both the filewatcher
// (for rename+grow detection) and the prospector (for key migration).
type shortFingerprintIndex struct {
	maxLen  int                              // max encoded fingerprint length
	entries map[string]shortFingerprintEntry // key → entry
}

// shortFingerprintEntry represents a tracked short fingerprint.
type shortFingerprintEntry struct {
	Fingerprint string
	Source      string // file path
}

func newShortFingerprintIndex(maxLen int) *shortFingerprintIndex {
	return &shortFingerprintIndex{
		maxLen:  maxLen,
		entries: make(map[string]shortFingerprintEntry),
	}
}

// Add adds an entry if its fingerprint is shorter than maxLen.
// Returns true if the entry was added (fingerprint short enough).
// Safe to call on a nil receiver.
func (idx *shortFingerprintIndex) Add(key, fingerprint, source string) bool {
	if idx == nil || fingerprint == "" || len(fingerprint) >= idx.maxLen {
		return false
	}
	idx.entries[key] = shortFingerprintEntry{Fingerprint: fingerprint, Source: source}
	return true
}

// Remove removes an entry by key. Safe to call on a nil receiver.
func (idx *shortFingerprintIndex) Remove(key string) {
	if idx == nil {
		return
	}
	delete(idx.entries, key)
}

// RemoveBySource removes the first entry whose source matches.
// Used on truncation where the key is unknown (fingerprint changed).
// Safe to call on a nil receiver.
func (idx *shortFingerprintIndex) RemoveBySource(source string) {
	if idx == nil {
		return
	}
	for key, entry := range idx.entries {
		if entry.Source == source {
			delete(idx.entries, key)
			return
		}
	}
}

// UpdateSource updates the source path for an entry.
// Safe to call on a nil receiver.
func (idx *shortFingerprintIndex) UpdateSource(key, newSource string) {
	if idx == nil {
		return
	}
	if entry, ok := idx.entries[key]; ok {
		entry.Source = newSource
		idx.entries[key] = entry
	}
}

// FindPrefixMatch finds an entry whose fingerprint is a prefix of targetFingerprint.
// If matchSource is non-empty, also requires entry.Source == matchSource and returns
// immediately on the first match (there is at most one entry per source path).
// If matchSource is empty, returns the longest prefix match to avoid ambiguity when
// multiple entries have prefix-related fingerprints.
// Returns the key and entry on match. Safe to call on a nil receiver.
func (idx *shortFingerprintIndex) FindPrefixMatch(targetFingerprint, matchSource string) (key string, entry shortFingerprintEntry, found bool) {
	if idx == nil || targetFingerprint == "" {
		return "", shortFingerprintEntry{}, false
	}
	for k, e := range idx.entries {
		if len(e.Fingerprint) >= len(targetFingerprint) {
			continue // stored is not shorter
		}
		if !strings.HasPrefix(targetFingerprint, e.Fingerprint) {
			continue
		}
		if matchSource != "" {
			if e.Source == matchSource {
				return k, e, true // exact path — at most one match, return immediately
			}
			continue
		}
		// No source filter: pick the longest prefix to avoid ambiguity.
		if len(e.Fingerprint) > len(entry.Fingerprint) {
			key, entry, found = k, e, true
		}
	}
	return key, entry, found
}

// Len returns the number of entries. Safe to call on a nil receiver.
func (idx *shortFingerprintIndex) Len() int {
	if idx == nil {
		return 0
	}
	return len(idx.entries)
}
