// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package capabilities

type outputCapability struct{}

func (c *outputCapability) Apply(in interface{}) (bool, interface{}) {
	return false, in
}

// NewOutputCapability creates capability filter for output.
func NewOutputCapability(r rule) Capability {
	return nil
}
