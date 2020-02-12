// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build mage

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/magefile/mage/mg"

	devtools "github.com/elastic/beats/dev-tools/mage"
	metricbeat "github.com/elastic/beats/metricbeat/scripts/mage"

	// mage:import
	"github.com/elastic/beats/dev-tools/mage/target/common"
	// mage:import
	_ "github.com/elastic/beats/dev-tools/mage/target/compose"
)

func init() {
	common.RegisterCheckDeps(Update)

	devtools.BeatDescription = "Metricbeat is a lightweight shipper for metrics."
	devtools.BeatLicense = "Elastic License"
}

// Aliases provides compatibility with CI while we transition all Beats
// to having common testing targets.
var Aliases = map[string]interface{}{
	"goTestUnit": GoUnitTest, // dev-tools/jenkins_ci.ps1 uses this.
}

// Build builds the Beat binary.
func Build() error {
	return devtools.Build(devtools.DefaultBuildArgs())
}

// GolangCrossBuild build the Beat binary inside of the golang-builder.
// Do not use directly, use crossBuild instead.
func GolangCrossBuild() error {
	return devtools.GolangCrossBuild(devtools.DefaultGolangCrossBuildArgs())
}

// CrossBuild cross-builds the beat for all target platforms.
func CrossBuild() error {
	return devtools.CrossBuild()
}

// BuildGoDaemon builds the go-daemon binary (use crossBuildGoDaemon).
func BuildGoDaemon() error {
	return devtools.BuildGoDaemon()
}

// CrossBuildGoDaemon cross-builds the go-daemon binary using Docker.
func CrossBuildGoDaemon() error {
	return devtools.CrossBuildGoDaemon()
}

// Package packages the Beat for distribution.
// Use SNAPSHOT=true to build snapshots.
// Use PLATFORMS to control the target platforms.
// Use BEAT_VERSION_QUALIFIER to control the version qualifier.
func Package() {
	start := time.Now()
	defer func() { fmt.Println("package ran for", time.Since(start)) }()

	if v, found := os.LookupEnv("AGENT_PACKAGING"); found && v != "" {
		devtools.UseElasticBeatXPackReducedPackaging()
	} else {
		devtools.UseElasticBeatXPackPackaging()
	}

	metricbeat.CustomizePackaging()
	devtools.PackageKibanaDashboardsFromBuildDir()

	mg.Deps(Update, metricbeat.PrepareModulePackagingXPack)
	mg.Deps(CrossBuild, CrossBuildGoDaemon)
	mg.SerialDeps(devtools.Package, TestPackages)
}

// TestPackages tests the generated packages (i.e. file modes, owners, groups).
func TestPackages() error {
	return devtools.TestPackages(
		devtools.WithModulesD(),
		devtools.WithModules(),

		// To be increased or removed when more light modules are added
		devtools.MinModules(5),
	)
}

// Fields generates a fields.yml and fields.go for each module.
func Fields() {
	mg.Deps(fieldsYML, moduleFieldsGo)
}

func moduleFieldsGo() error {
	return devtools.GenerateModuleFieldsGo("module")
}

// fieldsYML generates a fields.yml based on filebeat + x-pack/filebeat/modules.
func fieldsYML() error {
	return devtools.GenerateFieldsYAML(devtools.OSSBeatDir("module"), "module")
}

// Dashboards collects all the dashboards and generates index patterns.
func Dashboards() error {
	return devtools.KibanaDashboards(devtools.OSSBeatDir("module"), "module")
}

// Config generates both the short and reference configs.
func Config() {
	mg.Deps(configYML, devtools.GenerateDirModulesD)
}

func configYML() error {
	return devtools.Config(devtools.AllConfigTypes, metricbeat.XPackConfigFileParams(), ".")
}

// Update is an alias for running fields, dashboards, config.
func Update() {
	mg.SerialDeps(Fields, Dashboards, Config,
		metricbeat.PrepareModulePackagingXPack,
		devtools.GenerateModuleIncludeListGo)
}

// UnitTest executes the unit tests.
func UnitTest() {
	mg.SerialDeps(GoUnitTest, PythonUnitTest)
}

// GoUnitTest executes the Go unit tests.
// Use TEST_COVERAGE=true to enable code coverage profiling.
// Use RACE_DETECTOR=true to enable the race detector.
func GoUnitTest(ctx context.Context) error {
	return devtools.GoTest(ctx, devtools.DefaultGoTestUnitArgs())
}

// PythonUnitTest executes the python system tests.
func PythonUnitTest() error {
	mg.Deps(devtools.BuildSystemTestBinary)
	return devtools.PythonNoseTest(devtools.DefaultPythonTestUnitArgs())
}

// IntegTest executes integration tests (it uses Docker to run the tests).
func IntegTest() {
	devtools.AddIntegTestUsage()
	defer devtools.StopIntegTestEnv()
	mg.SerialDeps(GoIntegTest, PythonIntegTest)
}

// GoIntegTest executes the Go integration tests.
// Use TEST_COVERAGE=true to enable code coverage profiling.
// Use RACE_DETECTOR=true to enable the race detector.
func GoIntegTest(ctx context.Context) error {
	return devtools.GoTestIntegrationForModule(ctx)
}

// PythonIntegTest executes the python system tests in the integration environment (Docker).
func PythonIntegTest(ctx context.Context) error {
	if !devtools.IsInIntegTestEnv() {
		mg.SerialDeps(Fields, Dashboards)
	}
	return devtools.RunIntegTest("pythonIntegTest", func() error {
		mg.Deps(devtools.BuildSystemTestBinary)
		return devtools.PythonNoseTest(devtools.DefaultPythonTestIntegrationArgs())
	})
}
