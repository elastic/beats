// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package capabilities

type inputCapability struct{}

func (c *inputCapability) Apply(in interface{}) (bool, interface{}) {
	return false, in
}

// NewInputCapability creates capability filter for input.
func NewInputCapability(r rule) Capability {
	return nil
}
