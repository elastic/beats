// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package collections

import (
	"encoding/json"

	"github.com/google/uuid"
)

type UUIDTree struct {
	edges map[uuid.UUID]*UUIDSet
}

func (t *UUIDTree) UnmarshalJSON(b []byte) error {
	newTree := UUIDTree{edges: map[uuid.UUID]*UUIDSet{}}

	if err := json.Unmarshal(b, &newTree.edges); err != nil {
		return err
	}
	*t = newTree

	return nil
}

func (t *UUIDTree) MarshalJSON() ([]byte, error) {
	if len(t.edges) == 0 {
		return json.Marshal(nil)
	}

	return json.Marshal(&t.edges)
}

// RemoveVertex removes a vertex, and any of its edges, from the graph.
func (t *UUIDTree) RemoveVertex(value uuid.UUID) {
	delete(t.edges, value)
	for k, v := range t.edges {
		v.Remove(value)
		if v.Len() == 0 {
			delete(t.edges, k)
		}
	}
}

// ContainsVertex returns true if vertex of value exists.
func (t *UUIDTree) ContainsVertex(value uuid.UUID) bool {
	_, ok := t.edges[value]

	return ok
}

// AddEdge adds an edge (from, to).
func (t *UUIDTree) AddEdge(from uuid.UUID, to ...uuid.UUID) {
	if t.edges == nil {
		t.edges = map[uuid.UUID]*UUIDSet{}
	}

	t.addVertex(from)
	vertex := t.edges[from]
	vertex.Add(to...)
}

// RemoveEdge removes edge (from, to).
func (t *UUIDTree) RemoveEdge(from uuid.UUID, to uuid.UUID) {
	if vertex, ok := t.edges[from]; ok {
		vertex.Remove(to)
		if vertex.Len() == 0 {
			delete(t.edges, from)
		}
	}
}

// ContainsEdge returns true if edge (from, to) exists.
func (t *UUIDTree) ContainsEdge(from, to uuid.UUID) bool {
	v, ok := t.edges[from]
	if !ok {
		return false
	}

	return v.Contains(to)
}

// Expand will return a set of value(s) with transitive members included.
func (t *UUIDTree) Expand(value ...uuid.UUID) *UUIDSet {
	var found UUIDSet

	for _, v := range value {
		t.expand(v, &found)
	}

	return &found
}

// ExpandFromSet is like Expand, but takes a Set instead of a slice of values.
func (t *UUIDTree) ExpandFromSet(values UUIDSet) *UUIDSet {
	var found UUIDSet

	values.ForEach(func(elem uuid.UUID) {
		t.expand(elem, &found)
	})

	return &found
}

func (t *UUIDTree) expand(value uuid.UUID, seen *UUIDSet) {
	// Prevent cycles.
	if seen.Contains(value) {
		return
	}
	members, ok := t.edges[value]
	if !ok {
		return
	}
	seen.Add(value)
	members.ForEach(func(member uuid.UUID) {
		t.expand(member, seen)
		seen.Add(member)
	})
}

// addVertex adds a vertex to the graph.
func (t *UUIDTree) addVertex(value uuid.UUID) {
	if _, ok := t.edges[value]; !ok {
		set := NewUUIDSet()
		t.edges[value] = &set
	}
}
