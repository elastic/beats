// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package capabilities

type upgradeCapability struct{}

func (c *upgradeCapability) Apply(in interface{}) (bool, interface{}) {
	return false, in
}

// NewUpgradeCapability creates capability filter for upgrade.
func NewUpgradeCapability(r rule) Capability {
	return nil
}
