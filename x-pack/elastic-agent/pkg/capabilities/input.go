// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package capabilities

import (
	"fmt"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/transpiler"
)

func newInputsCapability(rd ruleDefinitions) (Capability, error) {
	caps := make([]Capability, 0, len(rd))

	for _, r := range rd {
		c, err := newInputCapability(r)
		if err != nil {
			return nil, err
		}

		if c != nil {
			caps = append(caps, c)
		}
	}

	return &multiInputsCapability{caps: caps}, nil
}

func newInputCapability(r ruler) (Capability, error) {
	cap, ok := r.(*inputCapability)
	if !ok {
		return nil, nil
	}

	return cap, nil
}

type inputCapability struct {
	Type  string `json:"rule" yaml:"rule"`
	Input string `json:"input" yaml:"input"`
}

func (c *inputCapability) Apply(in interface{}) (bool, interface{}) {
	ast, ok := in.(*transpiler.AST)
	if !ok || ast == nil {
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

func (c *inputCapability) renderInputs(inputs transpiler.Node) (transpiler.Node, error) {
	l, ok := inputs.Value().(*transpiler.List)
	if !ok {
		return nil, fmt.Errorf("inputs must be an array")
	}

	nodes := []*transpiler.Dict{}

	for _, inputNode := range l.Value().([]transpiler.Node) {
		inputDict, ok := inputNode.Clone().(*transpiler.Dict)
		if !ok {
			continue
		}
		typeNode, found := inputDict.Find("type")
		if !found {
			nodes = append(nodes, inputDict)
			continue
		}

		inputTypeStr, ok := typeNode.Value().(*transpiler.StrVal)
		if !ok {
			continue
		}

		inputType := inputTypeStr.String()
		// if input does not match definition continue
		if !matchesExpr(c.Input, inputType) {
			nodes = append(nodes, inputDict)
			continue
		}

		if _, found := inputDict.Find(conditionKey); found {
			// we already visited
			nodes = append(nodes, inputDict)
			continue
		}

		conditionNode := transpiler.NewKey(conditionKey, transpiler.NewBoolVal(c.Type == allowKey))
		dctNodes := inputDict.Value().([]transpiler.Node)
		dctNodes = append(dctNodes, conditionNode)

		nodes = append(nodes, transpiler.NewDict(dctNodes))
	}

	nInputs := []transpiler.Node{}
	for _, node := range nodes {
		nInputs = append(nInputs, node)
	}
	return transpiler.NewList(nInputs), nil

}

type multiInputsCapability struct {
	caps []Capability
}

func (c *multiInputsCapability) Apply(in interface{}) (bool, interface{}) {
	ast, transform, err := inputObject(in)
	if err != nil {
		// TODO: log error
		return false, in
	}
	if ast == nil {
		return false, in
	}

	var astIface interface{} = ast
	for _, cap := range c.caps {
		// input capability is not blocking
		_, astIface = cap.Apply(astIface)
	}

	ast, ok := astIface.(*transpiler.AST)
	if !ok {
		// TODO: log failure
		return false, in
	}

	input, err := c.cleanupInput(ast)
	if err != nil {
		// TODO: log error
		return false, in
	}

	if transform == nil {
		return false, input
	}

	return false, transform(input)
}

func (c *multiInputsCapability) cleanupInput(ast *transpiler.AST) (*transpiler.AST, error) {
	inputs, ok := transpiler.Lookup(ast, "inputs")
	if !ok {
		return ast, nil
	}

	l, ok := inputs.Value().(*transpiler.List)
	if !ok {
		return nil, fmt.Errorf("inputs must be an array")
	}

	nodes := []transpiler.Node{}

	for _, inputNode := range l.Value().([]transpiler.Node) {
		inputDict, ok := inputNode.Clone().(*transpiler.Dict)
		if !ok {
			continue
		}

		acceptValue := true
		conditionNode, found := inputDict.Find(conditionKey)
		if found {
			conditionValBool, ok := conditionNode.Value().(*transpiler.BoolVal)
			if ok {
				conditionVal, ok := conditionValBool.Value().(bool)
				if ok {
					acceptValue = conditionVal
				}
			}
		}

		if !acceptValue {
			continue
		}

		// cope everything except condition
		dctDict := make([]transpiler.Node, 0)
		for _, kv := range inputDict.Value().([]transpiler.Node) {
			kvNode, ok := kv.(*transpiler.Key)
			if !ok {
				dctDict = append(dctDict, kv)
			}

			if kvNode.Name() != conditionKey {
				dctDict = append(dctDict, kv)
			}
		}
		nodes = append(nodes, transpiler.NewDict(dctDict))
	}

	newInputsList := transpiler.NewList(nodes)
	if err := transpiler.Insert(ast, newInputsList, "inputs"); err != nil {
		// TODO: log error
		return ast, err
	}

	return ast, nil
}

func inputObject(a interface{}) (*transpiler.AST, func(interface{}) interface{}, error) {
	// TODO: transform input back to what it was
	if ast, ok := a.(*transpiler.AST); ok {
		fn := func(i interface{}) interface{} {
			// return as is
			return i
		}
		return ast, fn, nil
	}

	if mm, ok := a.(map[string]interface{}); ok {
		ast, err := transpiler.NewAST(mm)
		fn := func(i interface{}) interface{} {
			ast, ok := i.(*transpiler.AST)
			if ok {
				if mm, err := ast.Map(); err == nil {
					// return map if possible
					return mm
				}
			}

			return i
		}

		return ast, fn, err
	}

	return nil, nil, nil
}
