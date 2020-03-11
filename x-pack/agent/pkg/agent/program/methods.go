// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package program

import (
	"fmt"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/agent/transpiler"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/boolexp"
)

type env struct {
	ast  *transpiler.AST
	vars boolexp.VarStore
}

type envFunc = func(*env, []interface{}) (interface{}, error)

func methodsEnv(ast *transpiler.AST) *boolexp.MethodsReg {
	env := &env{
		ast:  ast,
		vars: &varStoreAST{ast: ast},
	}

	var methods = boolexp.NewMethodsReg()
	methods.MustRegister("HasItems", withEnv(env, hasItems))
	methods.MustRegister("HasNamespace", withEnv(env, hasNamespace))
	return methods
}

// hasItems the methods take a selector which must be a list, and look for the presence item in the
// list which are "enabled". The logic to determine if an item is enabled is the following:
// - When the "enabled" key is present and set to "true", The item is enabled.
// - When the "enabled" key is missing, the item is enabled.
// - When the "enabled" key is present and set to "false", The item is NOT enabled.
func hasItems(_ *env, args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return false, fmt.Errorf("expecting 1 argument received %d", len(args))
	}

	if args[0] == boolexp.Null {
		return false, nil
	}

	v, ok := args[0].(transpiler.Node).Value().(*transpiler.List)
	if !ok {
		return false, fmt.Errorf("expecting List and received %T", args[0])
	}

	for _, item := range v.Value().([]transpiler.Node) {
		d, ok := item.(*transpiler.Dict)
		if !ok {
			return false, fmt.Errorf("expecting Dict and received %T", args[0])
		}

		if isEnabled(d) {
			return true, nil
		}
	}

	return false, nil
}

// hasItems the methods take a selector which must be map and look if the map is enabled.
// The logic to determine if a map is enabled is the following:
// - When the "enabled" key is present and set to "true", The item is enabled.
// - When the "enabled" key is missing, the item is enabled.
// - When the "enabled" key is present and set to "false", The item is NOT enabled.
func hasNamespace(env *env, args []interface{}) (interface{}, error) {
	if len(args) < 2 {
		return false, fmt.Errorf("expecting at least 2 arguments received %d", len(args))
	}

	namespace, ok := args[0].(string)
	if !ok {
		return false, fmt.Errorf("invalid namespace %+v", args[0])
	}

	possibleSubKey := make([]string, 0, len(args))

	for _, v := range args[1:] {
		sk, ok := v.(string)
		if !ok {
			return false, fmt.Errorf("invalid sub key %+v for namespace", v)
		}
		possibleSubKey = append(possibleSubKey, sk)
	}

	var enabledCount int
	for _, key := range possibleSubKey {
		f := namespace + "." + key
		s, ok := transpiler.Lookup(env.ast, transpiler.Selector(f))
		if !ok {
			continue
		}

		if isEnabled(s) {
			enabledCount++
		}

		if enabledCount > 1 {
			return false, fmt.Errorf("only one namespace must be enabled in %s", namespace)
		}
	}

	if enabledCount == 0 {
		return false, nil
	}

	return true, nil
}

func withEnv(env *env, method envFunc) boolexp.CallFunc {
	return func(args []interface{}) (interface{}, error) {
		return method(env, args)
	}
}

func isEnabled(n transpiler.Node) bool {
	enabled, ok := n.Find("enabled")
	if !ok {
		return true
	}

	// Get the actual value of the node.
	value, ok := enabled.Value().(transpiler.Node).Value().(bool)
	if !ok {
		return false
	}

	return value
}

type varStoreAST struct {
	ast *transpiler.AST
}

func (v *varStoreAST) Lookup(needle string) (interface{}, bool) {
	return transpiler.Lookup(v.ast, transpiler.Selector(needle))
}
