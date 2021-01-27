// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package capabilities

import (
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/transpiler"
)

type inputCapability struct {
	Type  string `json:"rule" yaml:"rule"`
	Input string `json:"input" yaml:"input"`
}

func (c *inputCapability) Apply(in interface{}) (bool, interface{}) {
	ast, err := inputObject(in)
	if err != nil {
		// TODO: log error
		return false, in
	}

	inputs, ok := transpiler.Lookup(ast, "inputs")
	if ok {
		renderedInputs, err := c.renderInputs(inputs)
		if err != nil {
			// TODO: log error
			return false, in
		}

		err = transpiler.Insert(ast, renderedInputs, "inputs")
		if err != nil {
			// TODO: log error
			return false, in
		}

		return false, ast
	}

	return false, in
}

func (c *inputCapability) Rule() string {
	return c.Type
}

// NewInputCapability creates capability filter for input.
func NewInputCapability(r ruler) (Capability, error) {
	cap, ok := r.(*inputCapability)
	if !ok {
		return nil, nil
	}

	return cap, nil
}

func (c *inputCapability) renderInputs(inputs transpiler.Node) (transpiler.Node, error) {
	return inputs, nil

	// l, ok := inputs.Value().(*transpiler.List)
	// if !ok {
	// 	return nil, fmt.Errorf("inputs must be an array")
	// }

	// nodes := []*transpiler.Dict{}

	// for _, inputNode := range l.Value().([]transpiler.Node) {
	// 	// if input does not match definition continue
	// 	// TODO: get from input
	// var inputType = "system/metrics"

	// 	// if condition empty
	// 	// check if condition key already exists and if it has true/false value continue
	// 	// mark condition rule==allow
	// 	// endif condition empty

	// }

	// nInputs := []transpiler.Node{}
	// for _, node := range nodes {
	// 	nInputs = append(nInputs, node)
	// }
	// return transpiler.NewList(nInputs), nil

}

func inputObject(a interface{}) (*transpiler.AST, error) {
	if ast, ok := a.(*transpiler.AST); ok {
		return ast, nil
	}

	if mm, ok := a.(map[string]interface{}); ok {
		return transpiler.NewAST(mm)
	}

	return nil, nil
}
