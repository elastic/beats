// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package capabilities

import (
	"fmt"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/status"
)

const (
	outputKey = "outputs"
	typeKey   = "type"
)

func newOutputsCapability(log *logger.Logger, rd ruleDefinitions, reporter status.Reporter) (Capability, error) {
	caps := make([]Capability, 0, len(rd))

	for _, r := range rd {
		c, err := newOutputCapability(log, r, reporter)
		if err != nil {
			return nil, err
		}

		if c != nil {
			caps = append(caps, c)
		}
	}

	return &multiOutputsCapability{log: log, caps: caps}, nil
}

func newOutputCapability(log *logger.Logger, r ruler, reporter status.Reporter) (Capability, error) {
	cap, ok := r.(*outputCapability)
	if !ok {
		return nil, nil
	}

	cap.log = log
	cap.reporter = reporter
	return cap, nil
}

type outputCapability struct {
	log      *logger.Logger
	reporter status.Reporter
	Name     string `json:"name,omitempty" yaml:"name,omitempty"`
	Type     string `json:"rule" yaml:"rule"`
	Output   string `json:"output" yaml:"output"`
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
				c.log.Errorf("marking outputs failed for capability '%s': %v", c.name(), err)
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

func (c *outputCapability) name() string {
	if c.Name != "" {
		return c.Name
	}

	t := "A"
	if c.Type == denyKey {
		t = "D"
	}

	// e.g OA(*) or OD(logstash)
	c.Name = fmt.Sprintf("O%s(%s)", t, c.Output)
	return c.Name
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

		isSupported := c.Type == allowKey
		output[conditionKey] = isSupported
		outputs[outputName] = output

		if !isSupported {
			c.log.Errorf("output '%s' is left out due to capability restriction '%s'", outputName, c.name())
			c.reporter.Update(status.Degraded)
		}
	}

	return outputs, nil
}

type multiOutputsCapability struct {
	caps []Capability
	log  *logger.Logger
}

func (c *multiOutputsCapability) Apply(in interface{}) (bool, interface{}) {
	configMap, transform, err := configObject(in)
	if err != nil {
		c.log.Errorf("creating configuration object failed for capability 'multi-outputs': %v", err)
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
		c.log.Errorf("expecting map config object but got %T for capability 'multi-outputs': %v", mapIface, err)
		return false, in
	}

	configMap, err = c.cleanupOutput(configMap)
	if err != nil {
		c.log.Errorf("cleaning up config object failed for capability 'multi-outputs': %v", err)
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
