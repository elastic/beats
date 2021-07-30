// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package capabilities

import (
	"fmt"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/state"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/transpiler"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/status"
)

const (
	inputsKey = "inputs"
)

func newInputsCapability(log *logger.Logger, rd *ruleDefinitions, reporter status.Reporter) (Capability, error) {
	if rd == nil {
		return &multiInputsCapability{log: log, caps: []*inputCapability{}}, nil
	}

	caps := make([]*inputCapability, 0, len(rd.Capabilities))

	for _, r := range rd.Capabilities {
		c, err := newInputCapability(log, r, reporter)
		if err != nil {
			return nil, err
		}

		if c != nil {
			caps = append(caps, c)
		}
	}

	return &multiInputsCapability{log: log, caps: caps}, nil
}

func newInputCapability(log *logger.Logger, r ruler, reporter status.Reporter) (*inputCapability, error) {
	cap, ok := r.(*inputCapability)
	if !ok {
		return nil, nil
	}

	cap.log = log
	cap.reporter = reporter
	return cap, nil
}

type inputCapability struct {
	log      *logger.Logger
	reporter status.Reporter
	Name     string `json:"name,omitempty" yaml:"name,omitempty"`
	Type     string `json:"rule" yaml:"rule"`
	Input    string `json:"input" yaml:"input"`
}

func (c *inputCapability) Apply(cfgMap map[string]interface{}) (map[string]interface{}, error) {
	inputsIface, ok := cfgMap[inputsKey]
	if ok {
		if inputs := inputsMap(inputsIface, c.log); inputs != nil {
			renderedInputs, err := c.renderInputs(inputs)
			if err != nil {
				c.log.Errorf("marking inputs failed for capability '%s': %v", c.name(), err)
				return cfgMap, err
			}

			cfgMap[inputsKey] = renderedInputs
			return cfgMap, nil
		}

		return cfgMap, nil
	}

	return cfgMap, nil
}

func inputsMap(cfgInputs interface{}, l *logger.Logger) []map[string]interface{} {
	if inputs, ok := cfgInputs.([]map[string]interface{}); ok {
		return inputs
	}

	inputsSet, ok := cfgInputs.([]interface{})
	if !ok {
		l.Warn("inputs is not an array")
		return nil
	}

	inputsMap := make([]map[string]interface{}, 0, len(inputsSet))
	for _, s := range inputsSet {
		switch mm := s.(type) {
		case map[string]interface{}:
			inputsMap = append(inputsMap, mm)
		case map[interface{}]interface{}:
			newMap := make(map[string]interface{})
			for k, v := range mm {
				key, ok := k.(string)
				if !ok {
					continue
				}

				newMap[key] = v
			}
			inputsMap = append(inputsMap, newMap)
		default:
			continue
		}
	}

	return inputsMap
}

func (c *inputCapability) Rule() string {
	return c.Type
}

func (c *inputCapability) name() string {
	if c.Name != "" {
		return c.Name
	}

	t := "allow"
	if c.Type == denyKey {
		t = "deny"
	}

	// e.g IA(*) or ID(system/*)
	c.Name = fmt.Sprintf("I %s(%s)", t, c.Input)
	return c.Name
}

func (c *inputCapability) renderInputs(inputs []map[string]interface{}) ([]map[string]interface{}, error) {
	newInputs := make([]map[string]interface{}, 0, len(inputs))

	for i, input := range inputs {
		inputTypeIface, found := input[typeKey]
		if !found {
			return newInputs, errors.New(fmt.Sprintf("input '%d' is missing type key", i), errors.TypeConfig)
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

		isSupported := c.Type == allowKey

		input[conditionKey] = isSupported
		if !isSupported {
			msg := fmt.Sprintf("input '%s' is not run due to capability restriction '%s'", inputType, c.name())
			c.log.Infof(msg)
			c.reporter.Update(state.Degraded, msg, nil)
		}

		newInputs = append(newInputs, input)
	}

	return newInputs, nil
}

type multiInputsCapability struct {
	caps []*inputCapability
	log  *logger.Logger
}

func (c *multiInputsCapability) Apply(in interface{}) (interface{}, error) {
	inputsMap, transform, err := configObject(in)
	if err != nil {
		c.log.Errorf("constructing config object failed for 'multi-inputs' capability '%s': %v", err)
		return in, nil
	}
	if inputsMap == nil {
		return in, nil
	}

	for _, cap := range c.caps {
		// input capability is not blocking
		inputsMap, err = cap.Apply(inputsMap)
		if err != nil {
			return in, err
		}
	}

	inputsMap, err = c.cleanupInput(inputsMap)
	if err != nil {
		c.log.Errorf("cleaning up config object failed for capability 'multi-inputs': %v", err)
		return in, nil
	}

	if transform == nil {
		return inputsMap, nil
	}

	return transform(inputsMap), nil
}

func (c *multiInputsCapability) cleanupInput(cfgMap map[string]interface{}) (map[string]interface{}, error) {
	inputsIface, found := cfgMap[inputsKey]
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

	cfgMap[inputsKey] = newInputs
	return cfgMap, nil
}

func configObject(a interface{}) (map[string]interface{}, func(interface{}) interface{}, error) {
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
