// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package capabilities

import (
	"fmt"
)

const (
	outputKey = "outputs"
	typeKey   = "type"
)

func newOutputsCapability(rd ruleDefinitions) (Capability, error) {
	caps := make([]Capability, 0, len(rd))

	for _, r := range rd {
		c, err := newOutputCapability(r)
		if err != nil {
			return nil, err
		}

		if c != nil {
			caps = append(caps, c)
		}
	}

	return &multiOutputsCapability{caps: caps}, nil
}

func newOutputCapability(r ruler) (Capability, error) {
	cap, ok := r.(*outputCapability)
	if !ok {
		return nil, nil
	}

	return cap, nil
}

type outputCapability struct {
	Type   string `json:"rule" yaml:"rule"`
	Output string `json:"output" yaml:"output"`
}

func (c *outputCapability) Apply(in interface{}) (bool, interface{}) {
	cfgMap, ok := in.(map[string]interface{})
	if !ok || cfgMap == nil {
		return false, in
	}

	outputIface, ok := cfgMap[outputKey]
	if ok {
		outputs, ok := outputIface.(map[string]interface{})
		if ok {
			renderedOutputs, err := c.renderOutputs(outputs)
			if err != nil {
				// TODO: log error
				return false, in
			}

			cfgMap[outputKey] = renderedOutputs
			return false, cfgMap
		}

		return false, in
	}

	return false, in
}

func (c *outputCapability) Rule() string {
	return c.Type
}

func (c *outputCapability) renderOutputs(outputs map[string]interface{}) (map[string]interface{}, error) {
	for outputName, outputIface := range outputs {
		output, ok := outputIface.(map[string]interface{})
		if !ok {
			continue
		}

		outputTypeIface, ok := output[typeKey]
		if !ok {
			continue
		}

		outputType, ok := outputTypeIface.(string)
		if !ok {
			continue
		}

		// if input does not match definition continue
		if !matchesExpr(c.Output, outputType) {
			continue
		}

		if _, found := output[conditionKey]; found {
			// we already visited
			continue
		}

		output[conditionKey] = c.Type == allowKey
		outputs[outputName] = output
	}

	return outputs, nil
}

type multiOutputsCapability struct {
	caps []Capability
}

func (c *multiOutputsCapability) Apply(in interface{}) (bool, interface{}) {
	configMap, transform, err := configObject(in)
	if err != nil {
		// TODO: log error
		return false, in
	}
	if configMap == nil {
		return false, in
	}

	var mapIface interface{} = configMap

	for _, cap := range c.caps {
		// input capability is not blocking
		_, mapIface = cap.Apply(mapIface)
	}

	configMap, ok := mapIface.(map[string]interface{})
	if !ok {
		// TODO: log failure
		return false, in
	}

	configMap, err = c.cleanupOutput(configMap)
	if err != nil {
		// TODO: log error
		return false, in
	}

	if transform == nil {
		return false, configMap
	}

	return false, transform(configMap)
}

func (c *multiOutputsCapability) cleanupOutput(cfgMap map[string]interface{}) (map[string]interface{}, error) {
	outputsIface, found := cfgMap[outputKey]
	if !found {
		return cfgMap, nil
	}

	outputsMap, ok := outputsIface.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("outputs must be a map")
	}

	for outputName, outputIface := range outputsMap {
		acceptValue := true

		outputMap, ok := outputIface.(map[string]interface{})
		if ok {
			conditionIface, found := outputMap[conditionKey]
			if found {
				conditionVal, ok := conditionIface.(bool)
				if ok {
					acceptValue = conditionVal
				}
			}
		}

		if !acceptValue {
			delete(outputsMap, outputName)
			continue
		}

		delete(outputMap, conditionKey)
	}

	cfgMap[outputKey] = outputsMap
	return cfgMap, nil
}
