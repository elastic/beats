// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package eql

import "fmt"

// hasKey check if dict has anyone of the provided keys.
func hasKey(args []interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("hasKey: accepts minimum 2 arguments; recieved %d", len(args))
	}
	switch d := args[0].(type) {
	case *null:
		return false, nil
	case map[string]interface{}:
		for i, check := range args[1:] {
			switch c := check.(type) {
			case string:
				_, ok := d[c]
				if ok {
					return true, nil
				}
			default:
				return nil, fmt.Errorf("hasKey: %d argument must be a string; recieved %T", i+1, check)
			}
		}
		return false, nil
	}
	return nil, fmt.Errorf("hasKey: first argument must be a dictionary; recieved %T", args[0])
}
