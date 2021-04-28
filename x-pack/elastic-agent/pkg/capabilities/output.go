// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package capabilities

import (
	"fmt"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/state"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/status"
)

const (
	outputKey = "outputs"
	typeKey   = "type"
)

func newOutputsCapability(log *logger.Logger, rd *ruleDefinitions, reporter status.Reporter) (Capability, error) {
	if rd == nil {
		return &multiOutputsCapability{log: log, caps: []*outputCapability{}}, nil
	}

	caps := make([]*outputCapability, 0, len(rd.Capabilities))

	for _, r := range rd.Capabilities {
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

func newOutputCapability(log *logger.Logger, r ruler, reporter status.Reporter) (*outputCapability, error) {
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

func (c *outputCapability) Apply(cfgMap map[string]interface{}) (map[string]interface{}, error) {
	outputIface, ok := cfgMap[outputKey]
	if ok {
		outputs, ok := outputIface.(map[string]interface{})
		if ok {
			renderedOutputs, err := c.renderOutputs(outputs)
			if err != nil {
				c.log.Errorf("marking outputs as failed for the capability '%s': %v", c.name(), err)
				return cfgMap, err
			}

			cfgMap[outputKey] = renderedOutputs
			return cfgMap, nil
		}

		return cfgMap, nil
	}

	return cfgMap, nil
}

func (c *outputCapability) Rule() string {
	return c.Type
}

func (c *outputCapability) name() string {
	if c.Name != "" {
		return c.Name
	}

	t := "allow"
	if c.Type == denyKey {
		t = "deny"
	}

	// e.g OA(*) or OD(logstash)
	c.Name = fmt.Sprintf("Output %s(%s)", t, c.Output)
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
			return nil, errors.New(fmt.Sprintf("output '%s' is missing type key", outputName), errors.TypeConfig)
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
			msg := fmt.Sprintf("output '%s' is left out due to capability restriction '%s'", outputName, c.name())
			c.log.Errorf(msg)
			c.reporter.Update(state.Degraded, msg, nil)
		}
	}

	return outputs, nil
}

type multiOutputsCapability struct {
	caps []*outputCapability
	log  *logger.Logger
}

func (c *multiOutputsCapability) Apply(in interface{}) (interface{}, error) {
	configMap, transform, err := configObject(in)
	if err != nil {
		c.log.Errorf("creating configuration object failed for capability 'multi-outputs': %v", err)
		return in, nil
	}
	if configMap == nil {
		return in, nil
	}

	for _, cap := range c.caps {
		// input capability is not blocking
		configMap, err = cap.Apply(configMap)
		if err != nil {
			return in, err
		}
	}

	configMap, err = c.cleanupOutput(configMap)
	if err != nil {
		c.log.Errorf("cleaning up config object failed for capability 'multi-outputs': %v", err)
		return in, nil
	}

	if transform == nil {
		return configMap, nil
	}

	return transform(configMap), nil
}

func (c *multiOutputsCapability) cleanupOutput(cfgMap map[string]interface{}) (map[string]interface{}, error) {
	outputsIface, found := cfgMap[outputKey]
	if !found {
		return cfgMap, nil
	}

	switch outputsMap := outputsIface.(type) {
	case map[string]interface{}:
		handleOutputMapStr(outputsMap)
		cfgMap[outputKey] = outputsMap
	case map[interface{}]interface{}:
		handleOutputMapIface(outputsMap)
		cfgMap[outputKey] = outputsMap
	default:
		return nil, fmt.Errorf("outputs must be a map")
	}

	return cfgMap, nil
}

func handleOutputMapStr(outputsMap map[string]interface{}) {
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
}

func handleOutputMapIface(outputsMap map[interface{}]interface{}) {
	for outputName, outputIface := range outputsMap {
		acceptValue := true

		outputMap, ok := outputIface.(map[interface{}]interface{})
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
}
