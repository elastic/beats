package eql

import "fmt"

// length returns the length of the string, array, or dictionary
func length(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("length: accepts exactly 1 argument; recieved %d", len(args))
	}
	switch a := args[0].(type) {
	case string:
		return len(a), nil
	case []interface{}:
		return len(a), nil
	case map[string]interface{}:
		return len(a), nil
	}
	return nil, fmt.Errorf("length: accepts only a string, array, or dictionary; recieved %T", args[0])
}
