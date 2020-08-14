// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package boolexp

import "fmt"

// CallFunc is a function called while the expression evaluation is done, the function is responsable
// of doing the type conversion and allow checking the arity of the function.
type CallFunc func(args []interface{}) (interface{}, error)

// Method encapsulate a method.
type Method struct {
	Name string
	Func CallFunc
}

// MethodsReg is the registry of the methods, when the evaluation is done and a function is found we
// will lookup the function in the registry. If the method is found the methods will be executed,
// otherwise the evaluation will fail.
//
// NOTE: Define methods must have a unique name and capitalization is important.
type MethodsReg struct {
	methods map[string]Method
}

// Register registers a new methods, the method will return an error if the method with the same
// name already exists in the registry.
func (m *MethodsReg) Register(name string, f CallFunc) error {
	_, ok := m.methods[name]
	if ok {
		return fmt.Errorf("method %s already exists", name)
	}
	m.methods[name] = Method{Name: name, Func: f}
	return nil
}

// MustRegister registers a new methods and will panic on any error.
func (m *MethodsReg) MustRegister(name string, f CallFunc) {
	err := m.Register(name, f)
	if err != nil {
		panic(err)
	}
}

// Lookup search a methods by name and return it, will return false if the method is not found.
//
// NOTE: When looking methods name capitalization is important.
func (m *MethodsReg) Lookup(name string) (Method, bool) {
	v, ok := m.methods[name]
	return v, ok
}

// NewMethodsReg returns a new methods registry.
func NewMethodsReg() *MethodsReg {
	return &MethodsReg{methods: make(map[string]Method)}
}
