// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"encoding/json"
	"fmt"
	"strings"
)

// parse returns array of string values from json string
func parse(rawJSON, key string) ([]string, error) {
	var data interface{}
	err := json.Unmarshal([]byte(rawJSON), &data)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("error while parsing json: %w", err)
	}
	values, err := parseInterface(data, key)
	if err != nil {
		return nil, fmt.Errorf("error while parsing json: %w", err)
	}
	return values, nil
}

func parseInterface(data interface{}, key string) (values []string, err error) {
	splitKey := strings.Split(key, ".")
	for i, key := range splitKey {
		switch key {
		case "#":
			// '#' means it's an array and we want to get the
			// next key from each object in the array
			if tmp, ok := data.([]interface{}); ok {
				data, err = jsonArr(splitKey[i+1], tmp)
				if err != nil {
					return nil, fmt.Errorf("error while parsing json: %w", err)
				}
			}
			// interate over []interface{}
			if _, ok := data.([]interface{}); ok {
				for _, jsoninterface := range data.([]interface{}) {
					if _, ok := jsoninterface.(string); ok {
						values = append(values, jsoninterface.(string))
					} else {
						joinKey := strings.Join(splitKey[i+2:], ".")
						final, err := parseInterface(jsoninterface, joinKey)
						if err != nil {
							return nil, fmt.Errorf("error while parsing json: %w", err)
						}
						values = append(values, final...)
					}
				}
			} else {
				return nil, fmt.Errorf("error while parsing json: %w", err)
			}
		default:
			// default assumes it's a JSON object, so it expects a
			// map[string]interface{}
			data, err = jsonNorm(key, data)
			if err != nil {
				return nil, fmt.Errorf("json is: %w", err)
			}
			if len(splitKey) == i+1 {
				if _, ok := data.(string); ok {
					values = append(values, data.(string))
				}
			}
		}
		if key == "#" {
			break
		}
	}
	return values, nil
}

// jsonArr returns array of interface value from interface
// input:
// key=a,
// arrayInterface=
// [
// 	{
// 		"a": "a_value_1",
// 	},
// 	{
// 		"a": "a_value_2",
// 	},
// ]
// output:
// ["a_value_1", "a_value_2"]
func jsonArr(key string, arrayInterface []interface{}) ([]interface{}, error) {
	var arrayInterfaces []interface{}
	for _, interfaces := range arrayInterface {
		if _, ok := interfaces.(map[string]interface{}); ok {
			if _, ok := interfaces.(map[string]interface{})[key]; ok {
				arrayInterfaces = append(arrayInterfaces, interfaces.(map[string]interface{})[key])
			} else {
				return nil, fmt.Errorf("key field not found")
			}
		} else {
			return nil, fmt.Errorf("invalid key")
		}
	}
	return arrayInterfaces, nil
}

// jsonNorm returns interface value from interface
// input:
// key=a
// mapStr=map[string]interface{}{
// 	"a": "a_value",
// }
// output:
// a_value

// input:
// key=a
// mapStr=map[string]interface{}{
// 	"a": map[string]interface{}{
// 		"b": "b_value",
// 	},
// }
// map[string]interface{}{
// 	"b": "b_value",
// }
func jsonNorm(key string, mapStr interface{}) (interface{}, error) {
	if _, ok := mapStr.(map[string]interface{}); ok {
		if _, ok := mapStr.(map[string]interface{})[key]; ok {
			return mapStr.(map[string]interface{})[key], nil
		}
	}
	return nil, fmt.Errorf("key field not found")
}
