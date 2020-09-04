// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package eql

import (
	"fmt"
	"reflect"
)

// arrayContains check if value is a member of the array.
func arrayContains(args []interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("arrayContains: accepts minimum 2 arguments; recieved %d", len(args))
	}
	switch a := args[0].(type) {
	case *null:
		return false, nil
	case []interface{}:
		for _, check := range args[1:] {
			for _, i := range a {
				if reflect.DeepEqual(i, check) {
					return true, nil
				}
			}
		}
		return false, nil
	}
	return nil, fmt.Errorf("arrayContains: first argument must be an array; recieved %T", args[0])
}
