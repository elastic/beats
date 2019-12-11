// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build mage

package main

import (
	"github.com/magefile/mage/mg"

	devtools "github.com/elastic/beats/dev-tools/mage"

	// mage:import
	_ "github.com/elastic/beats/dev-tools/mage/target/common"
	// mage:import
	_ "github.com/elastic/beats/dev-tools/mage/target/build"
	// mage:import
	_ "github.com/elastic/beats/dev-tools/mage/target/pkg"
	// mage:import
	_ "github.com/elastic/beats/dev-tools/mage/target/dashboards"
	// mage:import
	_ "github.com/elastic/beats/dev-tools/mage/target/test"
	// mage:import
	"github.com/elastic/beats/dev-tools/mage/target/unittest"
	// mage:import
	winlogbeat "github.com/elastic/beats/winlogbeat/scripts/mage"
)

func init() {
	winlogbeat.SelectLogic = devtools.XPackProject
	devtools.BeatLicense = "Elastic License"
}

// Update is an alias for update:all. This is a workaround for
// https://github.com/magefile/mage/issues/217.
func Update() { mg.Deps(winlogbeat.Update.All) }

// Fields is an alias for update:fields.
//
// TODO: dev-tools/jenkins_ci.ps1 uses this. This should be removed when all
// projects have update to use goUnitTest.
func Fields() { mg.Deps(winlogbeat.Update.Fields) }

// GoTestUnit is an alias for goUnitTest.
//
// TODO: dev-tools/jenkins_ci.ps1 uses this. This should be removed when all
// projects have update to use goUnitTest.
func GoTestUnit() { mg.Deps(unittest.GoUnitTest) }
