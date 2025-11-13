// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build mage

package main

import (
	devtools "github.com/elastic/beats/v7/dev-tools/mage"
	//mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/common"
	//mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/unittest"
)

func init() {

	devtools.BeatDescription = "OTel components used by the Elastic Agent"
	devtools.BeatLicense = "Elastic License"
}
