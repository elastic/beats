// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package tasks_management

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
)

// Expect an empty array to return an empty map.
func TestParseChildNodesBadCastReturnsEmpty(t *testing.T) {
	empty := []any{}

	require.Equal(t, 0, len(parseChildNodes(empty, false)))
}

// Expect an empty array to return an empty map.
func TestParseChildNodesReturnsEmpty(t *testing.T) {
	empty := []any{}

	require.Equal(t, 0, len(parseChildNodes(empty, true)))
}

// Expect an empty array to returns node IDs without recursion.
func TestParseChildNodesReturnsNodeIdsWithDuplicates(t *testing.T) {
	children := []any{
		map[string]any{
			"node": "node1",
		},
		map[string]any{
			"node": "node2",
		},
		map[string]any{
			"node": "node1",
		},
	}

	require.ElementsMatch(t, []string{"node1", "node2"}, maps.Keys(parseChildNodes(children, true)))
}

// Expect an empty array to returns node IDs without recursion.
func TestParseChildNodesReturnsFlatNodeIds(t *testing.T) {
	children := []any{
		map[string]any{
			"node": "node1",
		},
		map[string]any{
			"node": "node2",
		},
	}

	require.ElementsMatch(t, []string{"node1", "node2"}, maps.Keys(parseChildNodes(children, true)))
}

// Expect an empty array to returns node IDs with recursion.
func TestParseChildNodesReturnsRecursiveNodeIds(t *testing.T) {
	children := []any{
		map[string]any{
			"node": "node1",
		},
		map[string]any{
			"node": "node2",
			"children": []any{
				map[string]any{
					"node": "node3",
					"children": []any{
						map[string]any{
							"node": "node4",
						},
					},
				},
			},
		},
		map[string]any{
			"node": "node5",
		},
	}

	require.ElementsMatch(t, []string{"node1", "node2", "node3", "node4", "node5"}, maps.Keys(parseChildNodes(children, true)))
}

// Expect an empty array to returns node IDs with recursion and ignores unspecified node IDs.
func TestParseChildNodesIgnoresEmptyNodeId(t *testing.T) {
	children := []any{
		map[string]any{
			"node": "node1",
		},
		map[string]any{
			"node": "node2",
			"children": []any{
				map[string]any{
					"children": []any{
						map[string]any{
							"node": "node3",
						},
					},
				},
			},
		},
		map[string]any{},
		map[string]any{
			"node": "node4",
		},
	}

	require.ElementsMatch(t, []string{"node1", "node2", "node3", "node4"}, maps.Keys(parseChildNodes(children, true)))
}
