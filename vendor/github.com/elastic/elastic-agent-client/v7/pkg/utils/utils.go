// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package utils

import "encoding/json"

// JSONMustMarshal marshals the input to JSON []byte and panics if it fails.
func JSONMustMarshal(input interface{}) []byte {
	res, err := json.Marshal(input)
	if err != nil {
		panic(err)
	}
	return res
}
