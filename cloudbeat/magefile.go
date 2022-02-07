//go:build mage
// +build mage

package main

import (
	"fmt"
	"time"

	"github.com/magefile/mage/mg"

	cloudbeat "github.com/elastic/beats/v7/cloudbeat/scripts/mage"
	devtools "github.com/elastic/beats/v7/dev-tools/mage"
	// mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/pkg"
	// mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/unittest"
	// mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/integtest/notests"
	// mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/test"
)

func init() {
	devtools.BeatDescription = "kubeat cis k8s benchmark."
	devtools.BeatLicense = "Elastic License"
}

// Package packages the Beat for distribution.
// Use SNAPSHOT=true to build snapshots.
// Use PLATFORMS to control the target platforms.
// Use VERSION_QUALIFIER to control the version qualifier.

// Check formats code, updates generated content, check for common errors, and
// checks for any modified files.
// func Check() {
// 	return devtools.Check()
// }

// Build builds the Beat binary.
func Build() error {
	params := devtools.DefaultBuildArgs()

	// Building cloudbeat
	err := devtools.Build(params)
	if err != nil {
		return err
	}

	//	params.
	err = devtools.Build(params)
	if err != nil {
		return err
	}

	return nil

}

// Todo write mage build & package functions for cloudbeat

// Clean cleans all generated files and build artifacts.
func Clean() error {
	return devtools.Clean()
}

// Update updates the generated files (aka make update).

// GolangCrossBuild build the Beat binary inside of the golang-builder.
// Do not use directly, use crossBuild instead.
func GolangCrossBuild() error {
	return devtools.GolangCrossBuild(devtools.DefaultGolangCrossBuildArgs())
}

// BuildGoDaemon builds the go-daemon binary (use crossBuildGoDaemon).
func BuildGoDaemon() error {
	return devtools.BuildGoDaemon()
}

// CrossBuild cross-builds the beat for all target platforms.
func CrossBuild() error {
	//building cloudbeat
	err := devtools.CrossBuild()
	if err != nil {
		return err
	}
	return nil
}

// CrossBuildGoDaemon cross-builds the go-daemon binary using Docker.
func CrossBuildGoDaemon() error {
	return devtools.CrossBuildGoDaemon()
}

func Package() {
	start := time.Now()
	defer func() { fmt.Println("package ran for", time.Since(start)) }()

	devtools.MustUsePackaging("cloudbeat", "cloudbeat/dev-tools/packaging/packages.yml")

	// ToDo decide whenther cloudbeat should move to x-pack dir & adjust accordingly

	mg.Deps(Update)
	mg.Deps(CrossBuild, CrossBuildGoDaemon)
	mg.SerialDeps(devtools.Package, TestPackages)
}

// TestPackages tests the generated packages (i.e. file modes, owners, groups).
func TestPackages() error {
	return devtools.TestPackages()
}

func Update() { mg.Deps(cloudbeat.Update.All) }

// Fields generates a fields.yml for the Beat.
func Fields() { mg.Deps(cloudbeat.Update.Fields) }

// Config generates both the short/reference/docker configs.
func Config() { mg.Deps(cloudbeat.Update.Config) }
