// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package eql

//go:generate antlr4 -Dlanguage=Go -o parser Eql.g4 -visitor

// Eval takes an expression, parse and evaluate it, everytime this method is called a new
// parser is created, if you want to reuse the parsed tree see the `New` method.
func Eval(expression string, store VarStore) (bool, error) {
	e, err := New(expression)
	if err != nil {
		return false, err
	}
	return e.Eval(store)
}
