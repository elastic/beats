// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cat_shards

import (
	"encoding/json"
	"fmt"
)

func convertObjectToMap[T any](object T) (map[string]any, error) {
	// Marshal the struct to JSON
	if data, err := json.Marshal(object); err != nil {
		return nil, err
	} else {
		// Unmarshal the JSON into a map
		var result map[string]any

		if err := json.Unmarshal(data, &result); err != nil {
			return nil, err
		}

		if result["assignShards"] != nil {
			fmt.Printf("assignShards: %v for %v\n", result["assignShards"], result["index"])
		} else {
			fmt.Printf("assignShards is nil for %v\n", result["index"])
		}

		return result, nil
	}
}

func convertObjectArrayToMapArray[T any](objects []T) ([]map[string]any, error) {
	mapArray := make([]map[string]any, 0, len(objects))

	for _, object := range objects {
		if objectMap, err := convertObjectToMap(object); err != nil {
			return nil, err
		} else {
			mapArray = append(mapArray, objectMap)
		}
	}

	return mapArray, nil
}
