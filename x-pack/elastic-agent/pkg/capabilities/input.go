// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package capabilities

type inputCapability struct {
	Type string `json:"rule" yaml:"rule"`
}

func (c *inputCapability) Apply(in interface{}) (bool, interface{}) {
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
