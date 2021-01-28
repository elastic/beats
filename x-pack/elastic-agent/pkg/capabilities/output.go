// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package capabilities

func newOutputCapability(r ruler) (Capability, error) {
	cap, ok := r.(*outputCapability)
	if !ok {
		return nil, nil
	}

	return cap, nil
}

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

type outputCapability struct {
	Type string `json:"rule" yaml:"rule"`
}

func (c *outputCapability) Apply(in interface{}) (bool, interface{}) {
	// TODO: Not yet implemented
	return false, in
}

func (c *outputCapability) Rule() string {
	return c.Type
}

type multiOutputsCapability struct {
	caps []Capability
}

func (c *multiOutputsCapability) Apply(in interface{}) (bool, interface{}) {
	// TODO: Not yet implemented
	return false, in
}
