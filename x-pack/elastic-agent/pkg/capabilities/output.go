// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package capabilities

type outputCapability struct {
	Type string `json:"rule" yaml:"rule"`
}

func (c *outputCapability) Apply(in interface{}) (bool, interface{}) {
	return false, in
}

func (c *outputCapability) Rule() string {
	return c.Type
}

// NewOutputCapability creates capability filter for output.
func NewOutputCapability(r ruler) Capability {
	cap, ok := r.(*outputCapability)
	if !ok {
		return nil
	}

	return cap
}
