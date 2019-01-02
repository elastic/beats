// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build mage

package main

import (
	"github.com/magefile/mage/mg"

	"github.com/elastic/beats/dev-tools/mage"

	// mage:import
	_ "github.com/elastic/beats/dev-tools/mage/target/common"
	// mage:import
	_ "github.com/elastic/beats/dev-tools/mage/target/pkg"
	// mage:import
	_ "github.com/elastic/beats/dev-tools/mage/target/dashboards"
	// mage:import
	journalbeat "github.com/elastic/beats/journalbeat/scripts/mage"
)

func init() {
	journalbeat.SelectLogic = mage.XPackProject

	mage.BeatLicense = "Elastic License"
}

// Update is an alias for update:all. This is a workaround for
// https://github.com/magefile/mage/issues/217.
func Update() { mg.Deps(journalbeat.Update.All) }
