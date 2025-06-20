// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cat_shards

import (
	"encoding/json"

	"github.com/elastic/elastic-agent-libs/logp"
)

func convertObjectToMap[T any](object T, logger *logp.Logger) (map[string]any, error) {
	// Marshal the struct to JSON
	if data, err := json.Marshal(object); err != nil {
		return nil, err
	} else {
		// Unmarshal the JSON into a map
		var result map[string]any

		if err := json.Unmarshal(data, &result); err != nil {
			return nil, err
		}

		if logger != nil {
			if result["assignShards"] != nil && result["assignShards"].([]any) != nil {
				switch result["assignShards"].(type) {
				case []map[string]any:
					logger.Infof("[ARRAY] assignShards ([]map[string]any): %v for %v\n", result["assignShards"], result["index"])
				default:
					logger.Infof("[ARRAY] assignShards (not []map[string]any): %v for %v\n", result["assignShards"], result["index"])
				}

				switch result["assignShards"].([]any)[0].(type) {
				case map[string]any:
					logger.Infof("assignShards (map[string]any): %v for %v\n", result["assignShards"].([]any)[0], result["index"])
				default:
					logger.Infof("assignShards (not map[string]any): %v for %v\n", result["assignShards"].([]any)[0], result["index"])
				}
			}
		}

		return result, nil
	}
}

func convertObjectArrayToMapArray[T any](objects []T, logger *logp.Logger) ([]map[string]any, error) {
	mapArray := make([]map[string]any, 0, len(objects))

	for _, object := range objects {
		if objectMap, err := convertObjectToMap(object, logger); err != nil {
			return nil, err
		} else {
			mapArray = append(mapArray, objectMap)
		}
	}

	return mapArray, nil
}
