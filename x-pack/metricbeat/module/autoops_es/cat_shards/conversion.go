// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cat_shards

import (
	"encoding/json"
)

func convertObjectArrayToMapArray[T any](objects []T) ([]map[string]any, error) {
	// Marshal the struct to JSON
	if data, err := json.Marshal(objects); err != nil {
		return nil, err
	} else {
		// Unmarshal the JSON into a map
		var result []map[string]any

		if err := json.Unmarshal(data, &result); err != nil {
			return nil, err
		}

		return result, nil
	}
}
