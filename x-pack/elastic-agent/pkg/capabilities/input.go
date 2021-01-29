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
	cfgMap, ok := in.(map[string]interface{})
	if !ok || cfgMap == nil {
		return false, in
	}

	inputsIface, ok := cfgMap["inputs"]
	if ok {
		inputs, ok := inputsIface.([]map[string]interface{})
		if ok {
			renderedInputs, err := c.renderInputs(inputs)
			if err != nil {
				// TODO: log error
				return false, in
			}

			cfgMap["inputs"] = renderedInputs
			return false, cfgMap
		}

		return false, in
	}

	return false, in
}

func (c *inputCapability) Rule() string {
	return c.Type
}

func (c *inputCapability) renderInputs(inputs []map[string]interface{}) ([]map[string]interface{}, error) {
	newInputs := make([]map[string]interface{}, 0, len(inputs))

	for _, input := range inputs {
		inputTypeIface, found := input["type"]
		if !found {
			newInputs = append(newInputs, input)
			continue
		}

		inputType, ok := inputTypeIface.(string)
		if !ok {
			newInputs = append(newInputs, input)
			continue
		}

		// if input does not match definition continue
		if !matchesExpr(c.Input, inputType) {
			newInputs = append(newInputs, input)
			continue
		}

		if _, found := input[conditionKey]; found {
			// we already visited
			newInputs = append(newInputs, input)
			continue
		}

		input[conditionKey] = c.Type == allowKey
		newInputs = append(newInputs, input)
	}

	return newInputs, nil

}

type multiInputsCapability struct {
	caps []Capability
}

func (c *multiInputsCapability) Apply(in interface{}) (bool, interface{}) {
	inputsMap, transform, err := inputObject(in)
	if err != nil {
		// TODO: log error
		return false, in
	}
	if inputsMap == nil {
		return false, in
	}

	var mapIface interface{} = inputsMap

	for _, cap := range c.caps {
		// input capability is not blocking
		_, mapIface = cap.Apply(mapIface)
	}

	inputsMap, ok := mapIface.(map[string]interface{})
	if !ok {
		// TODO: log failure
		return false, in
	}

	inputsMap, err = c.cleanupInput(inputsMap)
	if err != nil {
		// TODO: log error
		return false, in
	}

	if transform == nil {
		return false, inputsMap
	}

	return false, transform(inputsMap)
}

func (c *multiInputsCapability) cleanupInput(cfgMap map[string]interface{}) (map[string]interface{}, error) {
	inputsIface, found := cfgMap["inputs"]
	if !found {
		return cfgMap, nil
	}

	inputsList, ok := inputsIface.([]map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("inputs must be an array")
	}

	newInputs := make([]map[string]interface{}, 0, len(inputsList))

	for _, inputMap := range inputsList {
		acceptValue := true
		conditionIface, found := inputMap[conditionKey]
		if found {
			conditionVal, ok := conditionIface.(bool)
			if ok {
				acceptValue = conditionVal
			}
		}

		if !acceptValue {
			continue
		}

		delete(inputMap, conditionKey)
		newInputs = append(newInputs, inputMap)
	}

	cfgMap["inputs"] = newInputs
	return cfgMap, nil
}

func inputObject(a interface{}) (map[string]interface{}, func(interface{}) interface{}, error) {
	// TODO: transform input back to what it was
	if ast, ok := a.(*transpiler.AST); ok {
		fn := func(i interface{}) interface{} {
			mm, ok := i.(map[string]interface{})
			if !ok {
				return i
			}

			ast, err := transpiler.NewAST(mm)
			if err != nil {
				return i
			}
			return ast
		}
		mm, err := ast.Map()
		if err != nil {
			return nil, nil, err
		}

		return mm, fn, nil
	}

	if mm, ok := a.(map[string]interface{}); ok {
		fn := func(i interface{}) interface{} {
			// return as is
			return i
		}
		return mm, fn, nil
	}

	return nil, nil, nil
}
