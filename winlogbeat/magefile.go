// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

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
	_ "github.com/elastic/beats/v8/dev-tools/mage/target/docs"
	// mage:import
	_ "github.com/elastic/beats/v8/dev-tools/mage/target/test"
	// mage:import
	"github.com/elastic/beats/v8/dev-tools/mage/target/unittest"
	// mage:import
	winlogbeat "github.com/elastic/beats/v8/winlogbeat/scripts/mage"
)

func init() {
	unittest.RegisterGoTestDeps(winlogbeat.Update.Fields)
	winlogbeat.SelectLogic = devtools.OSSProject
}

// Update is an alias for update:all. This is a workaround for
// https://github.com/magefile/mage/issues/217.
func Update() { mg.Deps(winlogbeat.Update.All) }

// Dashboards collects all the dashboards and generates index patterns.
func Dashboards() error {
	return devtools.KibanaDashboards()
}
