// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package testing

import "encoding/json"

// CloneMap clones the source and returns a deep copy of the source.
func CloneMap(source map[string]interface{}) (map[string]interface{}, error) {
	if source == nil {
		return nil, nil
	}
	bytes, err := json.Marshal(source)
	if err != nil {
		return nil, err
	}
	var dest map[string]interface{}
	err = json.Unmarshal(bytes, &dest)
	if err != nil {
		return nil, err
	}
	return dest, nil
}

// CloneMapArray clones the source and returns a deep copy of the source.
func CloneMapArray(source []map[string]interface{}) ([]map[string]interface{}, error) {
	if source == nil {
		return nil, nil
	}
	bytes, err := json.Marshal(source)
	if err != nil {
		return nil, err
	}
	var dest []map[string]interface{}
	err = json.Unmarshal(bytes, &dest)
	if err != nil {
		return nil, err
	}
	return dest, nil
}
