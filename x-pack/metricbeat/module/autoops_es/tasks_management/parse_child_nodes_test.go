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
func TestParseChildNodesReturnsEmpty(t *testing.T) {
	empty := []interface{}{}

	require.Equal(t, 0, len(parseChildNodes(empty)))
}

// Expect an empty array to returns node IDs without recursion.
func TestParseChildNodesReturnsNodeIdsWithDuplicates(t *testing.T) {
	children := []interface{}{
		map[string]interface{}{
			"node": "node1",
		},
		map[string]interface{}{
			"node": "node2",
		},
		map[string]interface{}{
			"node": "node1",
		},
	}

	require.ElementsMatch(t, []string{"node1", "node2"}, maps.Keys(parseChildNodes(children)))
}

// Expect an empty array to returns node IDs without recursion.
func TestParseChildNodesReturnsFlatNodeIds(t *testing.T) {
	children := []interface{}{
		map[string]interface{}{
			"node": "node1",
		},
		map[string]interface{}{
			"node": "node2",
		},
	}

	require.ElementsMatch(t, []string{"node1", "node2"}, maps.Keys(parseChildNodes(children)))
}

// Expect an empty array to returns node IDs with recursion.
func TestParseChildNodesReturnsRecursiveNodeIds(t *testing.T) {
	children := []interface{}{
		map[string]interface{}{
			"node": "node1",
		},
		map[string]interface{}{
			"node": "node2",
			"children": []interface{}{
				map[string]interface{}{
					"node": "node3",
					"children": []interface{}{
						map[string]interface{}{
							"node": "node4",
						},
					},
				},
			},
		},
		map[string]interface{}{
			"node": "node5",
		},
	}

	require.ElementsMatch(t, []string{"node1", "node2", "node3", "node4", "node5"}, maps.Keys(parseChildNodes(children)))
}

// Expect an empty array to returns node IDs with recursion and ignores unspecified node IDs.
func TestParseChildNodesIgnoresEmptyNodeId(t *testing.T) {
	children := []interface{}{
		map[string]interface{}{
			"node": "node1",
		},
		map[string]interface{}{
			"node": "node2",
			"children": []interface{}{
				map[string]interface{}{
					"children": []interface{}{
						map[string]interface{}{
							"node": "node3",
						},
					},
				},
			},
		},
		map[string]interface{}{},
		map[string]interface{}{
			"node": "node4",
		},
	}

	require.ElementsMatch(t, []string{"node1", "node2", "node3", "node4"}, maps.Keys(parseChildNodes(children)))
}
