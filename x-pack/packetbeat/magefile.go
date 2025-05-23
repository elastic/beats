// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build mage

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/magefile/mage/mg"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
	"github.com/elastic/beats/v7/dev-tools/mage/target/build"
	packetbeat "github.com/elastic/beats/v7/packetbeat/scripts/mage"
	xpacketbeat "github.com/elastic/beats/v7/x-pack/packetbeat/scripts/mage"

	//mage:import
	"github.com/elastic/beats/v7/dev-tools/mage/target/common"
	//mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/unittest"
	//mage:import
	"github.com/elastic/beats/v7/dev-tools/mage/target/test"
)

func init() {
	common.RegisterCheckDeps(Update)

	test.RegisterDeps(SystemTest)

	devtools.BeatDescription = "Packetbeat analyzes network traffic and sends the data to Elasticsearch."
	devtools.BeatLicense = "Elastic License"
	packetbeat.SelectLogic = devtools.XPackProject
}

// Update updates the generated files.
func Update() {
	mg.SerialDeps(packetbeat.FieldsYML, Dashboards, Config)
}

// Config generates the config files.
func Config() error {
	return devtools.Config(devtools.AllConfigTypes, packetbeat.ConfigFileParams(), ".")
}

// Dashboards packages kibana dashboards
func Dashboards() error {
	return devtools.KibanaDashboards(devtools.OSSBeatDir("protos"))
}

// Build builds the Beat binary.
func Build() error {
	return devtools.Build(devtools.DefaultBuildArgs())
}

// GolangCrossBuild build the Beat binary inside of the golang-builder.
// Do not use directly, use crossBuild instead.
func GolangCrossBuild() error {
	if err := xpacketbeat.CopyNPCAPInstaller("./npcap/installer/"); err != nil {
		return err
	}

	return packetbeat.GolangCrossBuild()
}

// CrossBuild cross-builds the beat for all target platforms.
//
// On Windows platforms, if CrossBuild is invoked with the environment variables
// CI or NPCAP_LOCAL set to "true", a private cross-build image is selected that
// provides the OEM Npcap installer for the build. This behaviour requires access
// to the private image.
func CrossBuild() error {
	return devtools.CrossBuild(
		// Run all builds serially to try to address failures that might be caused
		// by concurrent builds. See https://github.com/elastic/beats/issues/24304.
		devtools.Serially(),

		devtools.ImageSelector(xpacketbeat.ImageSelector),
	)
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

	if v, found := os.LookupEnv("AGENT_PACKAGING"); found && v != "" {
		devtools.UseElasticBeatXPackReducedPackaging()
	} else {
		devtools.UseElasticBeatXPackPackaging()
	}

	devtools.PackageKibanaDashboardsFromBuildDir()
	packetbeat.CustomizePackaging()

	mg.Deps(Update)
	mg.Deps(CrossBuild)
	mg.SerialDeps(devtools.Package, TestPackages)
}

// Package packages the Beat for IronBank distribution.
//
// Use SNAPSHOT=true to build snapshots.
func Ironbank() error {
	start := time.Now()
	defer func() { fmt.Println("ironbank ran for", time.Since(start)) }()
	return devtools.Ironbank()
}

// TestPackages tests the generated packages (i.e. file modes, owners, groups).
func TestPackages() error {
	return devtools.TestPackages()
}

func SystemTest(ctx context.Context) error {
	mg.SerialDeps(xpacketbeat.GetNpcapInstallerFn("./"), devtools.BuildSystemTestBinary)

	args := devtools.DefaultGoTestIntegrationArgs()
	args.Packages = []string{"./tests/system/..."}
	return devtools.GoTest(ctx, args)
}
