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
	a, ok := args[0].([]interface{})
	if !ok {
		return nil, fmt.Errorf("arrayContains: first argument must be an array; recieved %T", args[0])
	}
	for _, check := range args[1:] {
		for _, i := range a {
			if reflect.DeepEqual(i, check) {
				return true, nil
			}
		}
	}
	return false, nil
}
