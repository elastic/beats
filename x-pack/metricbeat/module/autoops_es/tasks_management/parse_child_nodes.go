// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tasks_management

import (
	"golang.org/x/exp/maps"
)

// Extract the Node ID from each child task and add it to the map (treated as a Set).
func parseChildNodes(children []any, ok bool) map[string]bool {
	nodeMap := map[string]bool{}

	if ok {
		for _, child := range children {
			childMap, ok := child.(map[string]any)

			if !ok {
				continue
			}

			innerNode, ok := childMap["node"]

			if ok {
				innerNode, ok := innerNode.(string)

				if !ok {
					continue
				}

				nodeMap[innerNode] = true
			}

			innerChildren, ok := childMap["children"]

			if ok && innerChildren != nil {
				innerChildren, ok := innerChildren.([]any)

				for _, node := range maps.Keys(parseChildNodes(innerChildren, ok)) {
					nodeMap[node] = true
				}
			}
		}
	}

	return nodeMap
}
