// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package collections

import "encoding/json"

// Set is a collection data structure that retains single values of a given type.
type Set[T comparable] struct {
	m map[T]struct{}
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (s *Set[T]) UnmarshalJSON(b []byte) error {
	var rawValues []T

	if err := json.Unmarshal(b, &rawValues); err != nil {
		return err
	}

	*s = *NewSet[T](rawValues...)

	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (s *Set[T]) MarshalJSON() ([]byte, error) {
	if s == nil {
		return json.Marshal(nil)
	}
	v := s.Values()

	return json.Marshal(&v)
}

// Add adds one or more values to the Set.
func (s *Set[T]) Add(values ...T) {
	for _, v := range values {
		s.m[v] = struct{}{}
	}
}

// Has returns true if value is in the Set.
func (s *Set[T]) Has(value T) bool {
	if s == nil {
		return false
	}
	_, exists := s.m[value]

	return exists
}

// Values returns a slice of the elements contained within the Set. Order is
// not defined. For iterating over the elements, consider using ForEach.
func (s *Set[T]) Values() []T {
	if s == nil || len(s.m) == 0 {
		return nil
	}

	values := make([]T, 0, len(s.m))
	for k := range s.m {
		values = append(values, k)
	}

	return values
}

// Remove will remove element value from the Set.
func (s *Set[T]) Remove(value T) {
	if s == nil {
		return
	}
	delete(s.m, value)
}

// Len returns the length of the Set.
func (s *Set[T]) Len() int {
	if s == nil {
		return 0
	}
	return len(s.m)
}

// ForEach iterates over the Set, calling the function fn for each element.
func (s *Set[T]) ForEach(fn func(elem T)) {
	if s == nil {
		return
	}
	for k := range s.m {
		fn(k)
	}
}

// NewSet creates a new Set of type T. Values to add immediately may be provided.
func NewSet[T comparable](values ...T) *Set[T] {
	s := Set[T]{m: map[T]struct{}{}}
	s.Add(values...)

	return &s
}
