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

// +build mage

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/magefile/mage/mg"

	auditbeat "github.com/elastic/beats/auditbeat/scripts/mage"
	"github.com/elastic/beats/dev-tools/mage"
)

func init() {
	mage.BeatDescription = "Audit the activities of users and processes on your system."
}

// Aliases provides compatibility with CI while we transition all Beats
// to having common testing targets.
var Aliases = map[string]interface{}{
	"goTestUnit": GoUnitTest, // dev-tools/jenkins_ci.ps1 uses this.
}

// Build builds the Beat binary.
func Build() error {
	return mage.Build(mage.DefaultBuildArgs())
}

// GolangCrossBuild build the Beat binary inside of the golang-builder.
// Do not use directly, use crossBuild instead.
func GolangCrossBuild() error {
	return mage.GolangCrossBuild(mage.DefaultGolangCrossBuildArgs())
}

// BuildGoDaemon builds the go-daemon binary (use crossBuildGoDaemon).
func BuildGoDaemon() error {
	return mage.BuildGoDaemon()
}

// CrossBuild cross-builds the beat for all target platforms.
func CrossBuild() error {
	return mage.CrossBuild()
}

// CrossBuildGoDaemon cross-builds the go-daemon binary using Docker.
func CrossBuildGoDaemon() error {
	return mage.CrossBuildGoDaemon()
}

// Clean cleans all generated files and build artifacts.
func Clean() error {
	return mage.Clean()
}

// Package packages the Beat for distribution.
// Use SNAPSHOT=true to build snapshots.
// Use PLATFORMS to control the target platforms.
// Use VERSION_QUALIFIER to control the version qualifier.
func Package() {
	start := time.Now()
	defer func() { fmt.Println("package ran for", time.Since(start)) }()

	mage.UseElasticBeatOSSPackaging()
	mage.PackageKibanaDashboardsFromBuildDir()
	auditbeat.CustomizePackaging(auditbeat.OSSPackaging)

	mg.SerialDeps(Fields, Dashboards, Config, mage.GenerateModuleIncludeListGo)
	mg.Deps(CrossBuild, CrossBuildGoDaemon)
	mg.SerialDeps(mage.Package, TestPackages)
}

// TestPackages tests the generated packages (i.e. file modes, owners, groups).
func TestPackages() error {
	return mage.TestPackages()
}

// Update is an alias for running fields, dashboards, config, includes.
func Update() {
	mg.SerialDeps(Fields, Dashboards, Config,
		mage.GenerateModuleIncludeListGo, Docs)
}

// Config generates both the short/reference configs and populates the modules.d
// directory.
func Config() error {
	return mage.Config(mage.AllConfigTypes, auditbeat.OSSConfigFileParams(), ".")
}

// Fields generates fields.yml and fields.go files for the Beat.
func Fields() {
	mg.Deps(libbeatAndAuditbeatCommonFieldsGo, moduleFieldsGo)
	mg.Deps(fieldsYML)
}

// libbeatAndAuditbeatCommonFieldsGo generates a fields.go containing both
// libbeat and auditbeat's common fields.
func libbeatAndAuditbeatCommonFieldsGo() error {
	if err := mage.GenerateFieldsYAML(); err != nil {
		return err
	}
	return mage.GenerateAllInOneFieldsGo()
}

// moduleFieldsGo generates a fields.go for each module.
func moduleFieldsGo() error {
	return mage.GenerateModuleFieldsGo("module")
}

// fieldsYML generates the fields.yml file containing all fields.
func fieldsYML() error {
	return mage.GenerateFieldsYAML("module")
}

// ExportDashboard exports a dashboard and writes it into the correct directory.
//
// Required environment variables:
// - MODULE: Name of the module
// - ID:     Dashboard id
func ExportDashboard() error {
	return mage.ExportDashboard()
}

// Dashboards collects all the dashboards and generates index patterns.
func Dashboards() error {
	return mage.KibanaDashboards("module")
}

// Docs collects the documentation.
func Docs() {
	mg.Deps(auditbeat.ModuleDocs, auditbeat.FieldDocs)
}

// Fmt formats source code and adds file headers.
func Fmt() {
	mg.Deps(mage.Format)
}

// Check runs fmt and update then returns an error if any modifications are found.
func Check() {
	mg.SerialDeps(mage.Format, Update, mage.Check)
}

// IntegTest executes integration tests (it uses Docker to run the tests).
func IntegTest() {
	mage.AddIntegTestUsage()
	defer mage.StopIntegTestEnv()
	mg.SerialDeps(GoIntegTest, PythonIntegTest)
}

// UnitTest executes the unit tests.
func UnitTest() {
	mg.SerialDeps(GoUnitTest, PythonUnitTest)
}

// GoUnitTest executes the Go unit tests.
// Use TEST_COVERAGE=true to enable code coverage profiling.
// Use RACE_DETECTOR=true to enable the race detector.
func GoUnitTest(ctx context.Context) error {
	mg.Deps(Fields)
	return mage.GoTest(ctx, mage.DefaultGoTestUnitArgs())
}

// GoIntegTest executes the Go integration tests.
// Use TEST_COVERAGE=true to enable code coverage profiling.
// Use RACE_DETECTOR=true to enable the race detector.
func GoIntegTest(ctx context.Context) error {
	mg.Deps(Fields)
	return mage.RunIntegTest("goIntegTest", func() error {
		return mage.GoTest(ctx, mage.DefaultGoTestIntegrationArgs())
	})
}

// PythonUnitTest executes the python system tests.
func PythonUnitTest() error {
	mg.Deps(mage.BuildSystemTestBinary)
	return mage.PythonNoseTest(mage.DefaultPythonTestUnitArgs())
}

// PythonIntegTest executes the python system tests in the integration environment (Docker).
func PythonIntegTest(ctx context.Context) error {
	if !mage.IsInIntegTestEnv() {
		mg.SerialDeps(Fields, Dashboards)
	}
	return mage.RunIntegTest("pythonIntegTest", func() error {
		mg.Deps(mage.BuildSystemTestBinary)
		return mage.PythonNoseTest(mage.DefaultPythonTestIntegrationArgs())
	})
}
