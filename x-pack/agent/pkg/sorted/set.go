// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package sorted

import (
	"sort"
)

// Set is a sorted set that allow to iterate on they keys in an ordered manner, when
// items are added or removed from the Set the keys are sorted.
type Set struct {
	mapped map[string]interface{}
	keys   []string
}

// NewSet returns an ordered set.
func NewSet() *Set {
	return &Set{
		mapped: make(map[string]interface{}),
	}
}

// Add adds an items to the set.
func (s *Set) Add(k string, v interface{}) {
	_, ok := s.mapped[k]
	if !ok {
		s.keys = append(s.keys, k)
		sort.Strings(s.keys)
	}

	s.mapped[k] = v
}

// Remove removes an items from the Set.
func (s *Set) Remove(k string) {
	_, ok := s.mapped[k]
	if !ok {
		return
	}

	delete(s.mapped, k)

	pos := sort.SearchStrings(s.keys, k)
	if pos < len(s.keys) && s.keys[pos] == k {
		s.keys = append(s.keys[:pos], s.keys[pos+1:]...)
	}
}

// Get retrieves a specific values from the map and will return false if the key is not found.
func (s *Set) Get(k string) (interface{}, bool) {
	v, ok := s.mapped[k]
	return v, ok
}

// Keys returns slice of keys where the keys are ordered alphabetically.
func (s *Set) Keys() []string {
	return append(s.keys[:0:0], s.keys...)
}
