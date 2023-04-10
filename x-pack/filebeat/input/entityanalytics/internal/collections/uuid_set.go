// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package collections

import (
	"bytes"
	"encoding/json"
	"sort"

	"github.com/google/uuid"
)

type UUIDSet struct {
	m map[uuid.UUID]struct{}
}

func NewUUIDSet(values ...uuid.UUID) UUIDSet {
	s := UUIDSet{}
	s.Add(values...)

	return s
}

func (s *UUIDSet) UnmarshalJSON(b []byte) error {
	var rawValues []uuid.UUID
	if err := json.Unmarshal(b, &rawValues); err != nil {
		return err
	}
	newSet := NewUUIDSet(rawValues...)

	*s = newSet

	return nil
}

func (s *UUIDSet) MarshalJSON() ([]byte, error) {
	if len(s.m) == 0 {
		return json.Marshal(nil)
	}
	values := s.Values()

	return json.Marshal(&values)
}

func (s *UUIDSet) Len() int {
	return len(s.m)
}

func (s *UUIDSet) Values() []uuid.UUID {
	if len(s.m) == 0 {
		return nil
	}

	values := make([]uuid.UUID, 0, len(s.m))
	for k := range s.m {
		values = append(values, k)
	}
	sort.Slice(values, func(i, j int) bool {
		return bytes.Compare(values[i][:], values[j][:]) == -1
	})

	return values
}

func (s *UUIDSet) Add(values ...uuid.UUID) {
	if s.m == nil && len(values) != 0 {
		s.m = map[uuid.UUID]struct{}{}
	}
	for _, v := range values {
		s.m[v] = struct{}{}
	}
}

func (s *UUIDSet) Remove(values ...uuid.UUID) {
	if s.m == nil {
		return
	}

	for _, v := range values {
		delete(s.m, v)
	}
	if len(s.m) == 0 {
		s.m = nil
	}
}

func (s *UUIDSet) Contains(value uuid.UUID) bool {
	if s.m == nil {
		return false
	}
	_, ok := s.m[value]

	return ok
}

// ForEach iterates over the Set, calling the function fn for each element.
func (s *UUIDSet) ForEach(fn func(elem uuid.UUID)) {
	if len(s.m) == 0 {
		return
	}
	for k := range s.m {
		fn(k)
	}
}
