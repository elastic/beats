// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package eql

import "fmt"

// add performs x + y
func add(args []interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("add: accepts exactly 2 arguments; recieved %d", len(args))
	}
	return mathAdd(args[0], args[1])
}

// subtract performs x - y
func subtract(args []interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("subtract: accepts exactly 2 arguments; recieved %d", len(args))
	}
	return mathSub(args[0], args[1])
}

// multiply performs x * y
func multiply(args []interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("multiply: accepts exactly 2 arguments; recieved %d", len(args))
	}
	return mathMul(args[0], args[1])
}

// divide performs x / y
func divide(args []interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("divide: accepts exactly 2 arguments; recieved %d", len(args))
	}
	return mathDiv(args[0], args[1])
}

// modulo performs x % y
func modulo(args []interface{}) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("modulo: accepts exactly 2 arguments; recieved %d", len(args))
	}
	return mathMod(args[0], args[1])
}
