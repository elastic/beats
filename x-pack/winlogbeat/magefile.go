// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build mage
// +build mage

package main

import (
	"github.com/magefile/mage/mg"

	devtools "github.com/elastic/beats/v8/dev-tools/mage"

	// mage:import
	_ "github.com/elastic/beats/v8/dev-tools/mage/target/common"
	// mage:import
	_ "github.com/elastic/beats/v8/dev-tools/mage/target/build"
	// mage:import
	_ "github.com/elastic/beats/v8/dev-tools/mage/target/pkg"
	// mage:import
	_ "github.com/elastic/beats/v8/dev-tools/mage/target/dashboards"
	// mage:import
	_ "github.com/elastic/beats/v8/dev-tools/mage/target/test"
	// mage:import
	"github.com/elastic/beats/v8/dev-tools/mage/target/unittest"
	// mage:import
	winlogbeat "github.com/elastic/beats/v8/winlogbeat/scripts/mage"
)

func init() {
	unittest.RegisterGoTestDeps(winlogbeat.Update.Fields)

	winlogbeat.SelectLogic = devtools.XPackProject
	devtools.BeatLicense = "Elastic License"
}

// Update is an alias for update:all. This is a workaround for
// https://github.com/magefile/mage/issues/217.
func Update() { mg.Deps(winlogbeat.Update.All) }
