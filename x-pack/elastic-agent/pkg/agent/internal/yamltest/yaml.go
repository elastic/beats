// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package yamltest

import (
	"gopkg.in/yaml.v2"
)

// FromYAML read a bytes slice and return a map[string]interface{}.
// NOTE:OK, The YAML (v2 and v3) parser doesn't work with map as you would expect, it doesn't detect
// map[string]interface{} when parsing the document, it instead uses a map[interface{}]interface{},
// In the following expression, the left side is actually a bool and not a string.
//
// false: "awesome"
func FromYAML(in []byte, out *map[string]interface{}) error {
	var readTo map[interface{}]interface{}
	if err := yaml.Unmarshal(in, &readTo); err != nil {
		return err
	}

	*out = cleanMap(readTo)

	return nil
}

func cleanSlice(in []interface{}) []interface{} {
	result := make([]interface{}, len(in))
	for i, v := range in {
		result[i] = cleanValue(v)
	}
	return result
}

func cleanMap(in map[interface{}]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range in {
		key := k.(string)
		result[key] = cleanValue(v)
	}
	return result
}

func cleanValue(v interface{}) interface{} {
	switch v := v.(type) {
	case []interface{}:
		return cleanSlice(v)
	case map[interface{}]interface{}:
		return cleanMap(v)
	default:
		return v
	}
}
