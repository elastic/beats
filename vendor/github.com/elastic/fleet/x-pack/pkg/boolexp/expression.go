// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package boolexp

import (
	"errors"
	"fmt"

	"github.com/antlr/antlr4/runtime/Go/antlr"

	"github.com/elastic/fleet/x-pack/pkg/boolexp/parser"
)

// VarStore is the interface to implements when you want the expression engine to be able to fetch
// the value of a variables. Variables are defined using the field reference syntax likes
/// this: `%{[hello.var]}`.
type VarStore interface {
	// Lookup allows to lookup a value of a variable from the store, the lookup method will received
	// the name of variable like this.
	//
	// %{[hello.var]} => hello.var
	Lookup(string) (interface{}, bool)
}

// Errors
var (
	ErrEmptyExpression = errors.New("expression must not be an empty string")
)

// Expression parse a boolean expression into a tree and allow to evaluate the expression.
type Expression struct {
	expression string
	tree       antlr.ParseTree
	methodsReg *MethodsReg
	vars       VarStore
}

// Eval evaluates the expression using a visitor and the provided methods registry, will return true
// or any evaluation errors.
func (e *Expression) Eval(store VarStore) (result bool, err error) {
	// Antlr can panic on errors so we have to recover somehow.
	defer func() {
		r := recover()
		if r != nil {
			err = fmt.Errorf("error in while parsing the expression %s, error %+v", e.expression, r)
		}
	}()

	visitor := &expVisitor{methodsReg: e.methodsReg, vars: store}
	r := visitor.Visit(e.tree)

	if visitor.err != nil {
		return false, visitor.err
	}

	return r.(bool), nil
}

// New create a new boolean expression parser will return an error if the expression if invalid.
func New(expression string, methods *MethodsReg) (*Expression, error) {
	if len(expression) == 0 {
		return nil, ErrEmptyExpression
	}

	input := antlr.NewInputStream(expression)
	lexer := parser.NewBoolexpLexer(input)
	lexer.RemoveErrorListeners()
	tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewBoolexpParser(tokens)
	p.RemoveErrorListeners()
	tree := p.ExpList()

	return &Expression{expression: expression, tree: tree, methodsReg: methods}, nil
}
