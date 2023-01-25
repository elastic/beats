// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package collections

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTree_JSON(t *testing.T) {
	tree := &Tree[int]{
		Edges: map[int]*Set[int]{
			1: NewSet[int](2),
			2: NewSet[int](3),
			3: NewSet[int](4),
			5: NewSet[int](6),
		},
	}

	data, err := tree.MarshalJSON()
	assert.NoError(t, err)

	parsedTree := NewTree[int]()
	err = parsedTree.UnmarshalJSON(data)
	assert.NoError(t, err)

	assert.Equal(t, tree, parsedTree)
}

func TestTree_AddVertex(t *testing.T) {
	tree := NewTree[int]()
	tree.addVertex(1)

	assert.Contains(t, tree.Edges, 1)
}

func TestTree_DeleteVertex(t *testing.T) {
	tree := &Tree[int]{
		Edges: map[int]*Set[int]{
			1: NewSet[int](3),
			2: NewSet[int](3),
		},
	}
	tree.DeleteVertex(3)

	want := &Tree[int]{
		Edges: map[int]*Set[int]{},
	}

	assert.Equal(t, want, tree)
}

func TestTree_HasVertex(t *testing.T) {
	tree := &Tree[int]{
		Edges: map[int]*Set[int]{
			1: NewSet[int](2),
		},
	}

	assert.True(t, tree.HasVertex(1))
	assert.True(t, tree.HasVertex(1))
	assert.False(t, tree.HasVertex(3))
}

func TestTree_AddEdge(t *testing.T) {
	tree := NewTree[int]()
	want := &Tree[int]{
		Edges: map[int]*Set[int]{
			1: NewSet[int](2),
		},
	}

	tree.AddEdge(1, 2)

	assert.Equal(t, want, tree)
}

func TestTree_DeleteEdge(t *testing.T) {
	tree := &Tree[int]{
		Edges: map[int]*Set[int]{
			1: NewSet[int](2),
		},
	}
	want := &Tree[int]{
		Edges: map[int]*Set[int]{},
	}

	tree.DeleteEdge(1, 2)

	assert.Equal(t, want, tree)
}

func TestTree_HasEdge(t *testing.T) {
	tree := &Tree[int]{
		Edges: map[int]*Set[int]{
			1: NewSet[int](2),
		},
	}

	assert.True(t, tree.HasEdge(1, 2))
	assert.False(t, tree.HasEdge(2, 1))
	assert.False(t, tree.HasEdge(3, 4))
}

func TestTree_Expand(t *testing.T) {
	tree := &Tree[int]{
		Edges: map[int]*Set[int]{
			1: NewSet[int](2),
			2: NewSet[int](3),
			3: NewSet[int](4),
			5: NewSet[int](6),
		},
	}

	want := NewSet[int](2, 3, 4)
	got := tree.Expand(2)

	assert.Equal(t, want, got)
}

func TestTree_ExpandFromSet(t *testing.T) {
	tree := &Tree[int]{
		Edges: map[int]*Set[int]{
			1: NewSet[int](2),
			2: NewSet[int](3),
			3: NewSet[int](4),
			5: NewSet[int](6),
		},
	}

	want := NewSet[int](2, 3, 4)
	got := tree.ExpandFromSet(NewSet[int](2))

	assert.Equal(t, want, got)
}
