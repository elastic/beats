// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package eql

// callFunc is a function called while the expression evaluation is done, the function is responsible
// of doing the type conversion and allow checking the arity of the function.
type callFunc func(args []interface{}) (interface{}, error)

// methods are the methods enabled in EQL.
var methods = map[string]callFunc{
	// array
	"arrayContains": arrayContains,

	// dict
	"hasKey": hasKey,

	// length:
	"length": length,

	// math
	"add":      add,
	"subtract": subtract,
	"multiply": multiply,
	"divide":   divide,
	"modulo":   modulo,

	// str
	"concat":         concat,
	"endsWith":       endsWith,
	"indexOf":        indexOf,
	"match":          match,
	"number":         number,
	"startsWith":     startsWith,
	"string":         str,
	"stringContains": stringContains,
}
