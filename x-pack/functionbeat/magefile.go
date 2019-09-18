// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build mage

package main

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/magefile/mage/mg"

	// mage:import
	_ "github.com/elastic/beats/dev-tools/mage/target/common"
	"github.com/elastic/beats/dev-tools/mage/target/unittest"
	// mage:import
	_ "github.com/elastic/beats/dev-tools/mage/target/pkg"
	// mage:import
	_ "github.com/elastic/beats/dev-tools/mage/target/integtest"
	// mage:import
	_ "github.com/elastic/beats/dev-tools/mage/target/unittest"

	devtools "github.com/elastic/beats/dev-tools/mage"
	functionbeat "github.com/elastic/beats/x-pack/functionbeat/scripts/mage"
)

func init() {
	devtools.BeatDescription = "Functionbeat is a beat implementation for a serverless architecture."
	devtools.BeatLicense = "Elastic License"
}

// Build builds the Beat binary and functions by provider.
func Build() error {
	params := devtools.DefaultBuildArgs()

	// Building functionbeat manager
	err := devtools.Build(params)
	if err != nil {
		return err
	}

	// Building functions to deploy
	for _, provider := range functionbeat.SelectedProviders {
		inputFiles := filepath.Join("provider", provider, "main.go")
		params.InputFiles = []string{inputFiles}
		params.Name = devtools.BeatName + "-" + provider
		params.OutputDir = filepath.Join("provider", provider)
		err := devtools.Build(params)
		if err != nil {
			return err
		}
	}
	return nil
}

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
	// Building functionbeat manager
	err := devtools.CrossBuild()
	if err != nil {
		return err
	}

	// Building functions to deploy
	for _, provider := range functionbeat.SelectedProviders {
		err := devtools.CrossBuild(devtools.AddPlatforms("linux/amd64"), devtools.InDir("x-pack", "functionbeat", "provider", provider))
		if err != nil {
			return err
		}
	}
	return nil
}

// CrossBuildGoDaemon cross-builds the go-daemon binary using Docker.
func CrossBuildGoDaemon() error {
	return devtools.CrossBuildGoDaemon()
}

// Update is an alias for update:all. This is a workaround for
// https://github.com/magefile/mage/issues/217.
func Update() { mg.Deps(functionbeat.Update.All) }

// Fields is an alias for update:fields. This is a workaround for
// https://github.com/magefile/mage/issues/217.
func Fields() { mg.Deps(functionbeat.Update.Fields) }

// Config is an alias for update:config. This is a workaround for
// https://github.com/magefile/mage/issues/217.
func Config() { mg.Deps(functionbeat.Update.Config) }

// Package packages the Beat for distribution.
// Use SNAPSHOT=true to build snapshots.
// Use PLATFORMS to control the target platforms.
func Package() {
	start := time.Now()
	defer func() { fmt.Println("package ran for", time.Since(start)) }()

	devtools.MustUsePackaging("functionbeat", "x-pack/functionbeat/dev-tools/packaging/packages.yml")

	mg.Deps(Update)
	mg.Deps(CrossBuild, CrossBuildGoDaemon)
	mg.SerialDeps(devtools.Package, TestPackages)
}

// TestPackages tests the generated packages (i.e. file modes, owners, groups).
func TestPackages() error {
	return devtools.TestPackages()
}

// GoTestUnit is an alias for goUnitTest.
func GoTestUnit() {
	mg.Deps(unittest.GoUnitTest)
}

// BuildSystemTestBinary build a binary for testing that is instrumented for
// testing and measuring code coverage. The binary is only instrumented for
// coverage when TEST_COVERAGE=true (default is false).
func BuildSystemTestBinary() error {
	err := devtools.BuildSystemTestBinary()
	if err != nil {
		return err
	}

	params := devtools.DefaultTestBinaryArgs()
	for _, provider := range functionbeat.SelectedProviders {
		params.Name = filepath.Join("provider", provider, devtools.BeatName+"-"+provider)
		inputFiles := make([]string, 0)
		for _, inputFileName := range []string{"main.go", "main_test.go"} {
			inputFiles = append(inputFiles, filepath.Join("provider", provider, inputFileName))
		}
		params.InputFiles = inputFiles
		err := devtools.BuildSystemTestGoBinary(params)
		if err != nil {
			return err
		}
	}
	return nil
}
