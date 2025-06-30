// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cat_shards

import (
	"encoding/json"
)

// convertObjectArrayToMapArray converts an array of structs to an array of maps one by one.
func convertObjectArrayToMapArray[T any](objects []T) []map[string]any {
	mapArray := make([]map[string]any, 0, len(objects))

	for _, object := range objects {
		if data, err := json.Marshal(object); err == nil {
			// Unmarshal the JSON into a map
			var result map[string]any

			if err := json.Unmarshal(data, &result); err == nil {
				mapArray = append(mapArray, result)
			}
		}
	}

	return mapArray
}
