// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package capabilities

type upgradeCapability struct {
	Type string `json:"rule" yaml:"rule"`
	// UpgradeEql is eql expression defining upgrade
	UpgradeEql string `json:"upgrade" yaml:"upgrade"`
}

func (c *upgradeCapability) Rule() string {
	return c.Type
}

func (c *upgradeCapability) Apply(in interface{}) (bool, interface{}) {
	return false, in
}

// NewUpgradeCapability creates capability filter for upgrade.
// Available variables:
// - version
// - source_uri
func NewUpgradeCapability(u ruler) Capability {
	return nil
}
