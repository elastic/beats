// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package collections

import "encoding/json"

// Tree represents a directed graph. Vertexes are identified by comparable type T.
type Tree[T comparable] struct {
	Edges map[T]*Set[T]
}

// MarshalJSON implements the json.Marshaler interface.
func (t *Tree[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(&t.Edges)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (t *Tree[T]) UnmarshalJSON(b []byte) error {
	newTree := NewTree[T]()

	if err := json.Unmarshal(b, &newTree.Edges); err != nil {
		return err
	}

	for k, v := range newTree.Edges {
		if v == nil {
			newTree.Edges[k] = NewSet[T]()
		}
	}

	*t = *newTree

	return nil
}

// AddVertex adds a vertex to the graph.
func (t *Tree[T]) AddVertex(value T) {
	if _, ok := t.Edges[value]; !ok {
		t.Edges[value] = NewSet[T]()
	}
}

// DeleteVertex removes a vertex, and any of its edges, from the graph.
func (t *Tree[T]) DeleteVertex(value T) {
	delete(t.Edges, value)
	for _, v := range t.Edges {
		v.Remove(value)
	}
}

// HasVertex returns true if vertex of value exists.
func (t *Tree[T]) HasVertex(value T) bool {
	_, ok := t.Edges[value]

	return ok
}

// AddEdge adds an edge (from, to).
func (t *Tree[T]) AddEdge(from T, to ...T) {
	t.AddVertex(from)
	for _, v := range to {
		t.AddVertex(v)
	}
	t.Edges[from].Add(to...)
}

// DeleteEdge removes edge (from, to).
func (t *Tree[T]) DeleteEdge(from T, to T) {
	if v, ok := t.Edges[from]; ok && v != nil {
		v.Remove(to)
	}
}

// HasEdge returns true if edge (from, to) exists.
func (t *Tree[T]) HasEdge(from, to T) bool {
	v, ok := t.Edges[from]
	if !ok {
		return false
	}

	return v.Has(to)
}

// Expand will return a set of value(s) with transitive members included.
func (t *Tree[T]) Expand(value ...T) *Set[T] {
	found := NewSet[T]()

	for _, v := range value {
		t.expand(v, found)
	}

	return found
}

// ExpandFromSet is like Expand, but takes a Set instead of a slice of values.
func (t *Tree[T]) ExpandFromSet(values *Set[T]) *Set[T] {
	found := NewSet[T]()

	values.ForEach(func(elem T) {
		t.expand(elem, found)

	})

	return found
}

func (t *Tree[T]) expand(value T, seen *Set[T]) {
	// Prevent cycles.
	if seen.Has(value) {
		return
	}

	members, ok := t.Edges[value]
	if !ok {
		return
	}

	seen.Add(value)
	members.ForEach(func(member T) {
		t.expand(member, seen)
	})
}

// NewTree creates a new Tree of type T.
func NewTree[T comparable]() *Tree[T] {
	return &Tree[T]{
		Edges: map[T]*Set[T]{},
	}
}
