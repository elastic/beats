// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build mage
// +build mage

package main

import (
	"fmt"
	"time"

	"github.com/magefile/mage/mg"

	auditbeat "github.com/elastic/beats/v8/auditbeat/scripts/mage"
	devtools "github.com/elastic/beats/v8/dev-tools/mage"
	"github.com/elastic/beats/v8/dev-tools/mage/target/build"

	// mage:import
	"github.com/elastic/beats/v8/dev-tools/mage/target/common"
	// mage:import
	"github.com/elastic/beats/v8/dev-tools/mage/target/unittest"
	// mage:import
	_ "github.com/elastic/beats/v8/dev-tools/mage/target/integtest"
	// mage:import
	_ "github.com/elastic/beats/v8/dev-tools/mage/target/test"
)

func init() {
	common.RegisterCheckDeps(Update)
	unittest.RegisterPythonTestDeps(fieldsYML)

	devtools.BeatDescription = "Audit the activities of users and processes on your system."
	devtools.BeatLicense = "Elastic License"
	devtools.Platforms = devtools.Platforms.Filter("!linux/ppc64 !linux/mips64")
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

// AssembleDarwinUniversal merges the darwin/amd64 and darwin/arm64 into a single
// universal binary using `lipo`. It assumes the darwin/amd64 and darwin/arm64
// were built and only performs the merge.
func AssembleDarwinUniversal() error {
	return build.AssembleDarwinUniversal()
}

// Package packages the Beat for distribution.
// Use SNAPSHOT=true to build snapshots.
// Use PLATFORMS to control the target platforms.
// Use VERSION_QUALIFIER to control the version qualifier.
func Package() {
	start := time.Now()
	defer func() { fmt.Println("package ran for", time.Since(start)) }()

	devtools.UseElasticBeatXPackPackaging()
	devtools.PackageKibanaDashboardsFromBuildDir()
	auditbeat.CustomizePackaging(auditbeat.XPackPackaging)

	mg.SerialDeps(Fields, Dashboards, Config, devtools.GenerateModuleIncludeListGo)
	mg.Deps(CrossBuild, CrossBuildGoDaemon)
	mg.SerialDeps(devtools.Package, TestPackages)
}

// TestPackages tests the generated packages (i.e. file modes, owners, groups).
func TestPackages() error {
	return devtools.TestPackages()
}

// Update is an alias for running fields, dashboards, config.
func Update() {
	mg.SerialDeps(Fields, Dashboards, Config, devtools.GenerateModuleIncludeListGo)
}

// Config generates both the short and reference configs.
func Config() error {
	return devtools.Config(devtools.AllConfigTypes, auditbeat.XPackConfigFileParams(), ".")
}

// Fields generates a fields.yml and include/fields.go.
func Fields() {
	mg.SerialDeps(fieldsYML, moduleFieldsGo)
}

func moduleFieldsGo() error {
	return devtools.GenerateModuleFieldsGo("module")
}

// fieldsYML generates the fields.yml file containing all fields.
func fieldsYML() error {
	return devtools.GenerateFieldsYAML(devtools.OSSBeatDir("module"), "module")
}

// ExportDashboard exports a dashboard and writes it into the correct directory.
//
// Required environment variables:
// - MODULE: Name of the module
// - ID:     Dashboard id
func ExportDashboard() error {
	return devtools.ExportDashboard()
}

// Dashboards collects all the dashboards and generates index patterns.
func Dashboards() error {
	return devtools.KibanaDashboards(devtools.OSSBeatDir("module"), "module")
}
