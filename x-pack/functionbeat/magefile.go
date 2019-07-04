// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build mage

package main

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/magefile/mage/mg"

	// mage:import
	_ "github.com/elastic/beats/dev-tools/mage/target/common"
	// mage:import
	_ "github.com/elastic/beats/dev-tools/mage/target/pkg"

	devtools "github.com/elastic/beats/dev-tools/mage"
	"github.com/elastic/beats/dev-tools/mage/target/integtest"
	functionbeat "github.com/elastic/beats/x-pack/functionbeat/scripts/mage"
)

var ()

func init() {
	devtools.BeatDescription = "Functionbeat is a beat implementation for a serverless architecture."
	devtools.BeatLicense = "Elastic License"
}

// Build builds the Beat binary.
func Build() error {
	params := devtools.DefaultBuildArgs()
	for _, provider := range functionbeat.SelectedProviders {
		inputFiles := filepath.Join(provider, "main.go")
		params.InputFiles = []string{inputFiles}
		params.Name = devtools.BeatName + "-" + provider
		params.OutputDir = provider
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
	params := devtools.DefaultGolangCrossBuildArgs()
	params.Name = "functionbeat-" + params.Name
	return devtools.GolangCrossBuild(params)
}

// BuildGoDaemon builds the go-daemon binary (use crossBuildGoDaemon).
func BuildGoDaemon() error {
	return devtools.BuildGoDaemon()
}

// CrossBuild cross-builds the beat for all target platforms.
func CrossBuild() error {
	for _, provider := range functionbeat.SelectedProviders {
		err := devtools.CrossBuild(devtools.AddPlatforms("linux/amd64"), devtools.InDir("x-pack", "functionbeat", provider))
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
func Update() {
	fmt.Printf("baba %+v\n", functionbeat.Update.All)
	mg.Deps(functionbeat.Update.All)
}

// Update is an alias for update:fields. This is a workaround for
// https://github.com/magefile/mage/issues/217.
func Fields() { mg.Deps(functionbeat.Update.Fields) }

// Update is an alias for update:config. This is a workaround for
// https://github.com/magefile/mage/issues/217.
func Config() { mg.Deps(functionbeat.Update.Config) }

// Package packages the Beat for distribution.
// Use SNAPSHOT=true to build snapshots.
// Use PLATFORMS to control the target platforms.
func Package() {
	start := time.Now()
	defer func() { fmt.Println("package ran for", time.Since(start)) }()

	mg.Deps(Update)
	mg.Deps(CrossBuild, CrossBuildGoDaemon)
	for _, provider := range functionbeat.SelectedProviders {
		devtools.MustUsePackaging("functionbeat", "x-pack/functionbeat/dev-tools/packaging/packages.yml")
		for _, args := range devtools.Packages {
			args.Spec.ExtraVar("Provider", provider)
		}

		mg.Deps(devtools.Package)
		mg.Deps(devtools.TestPackages)
	}
}

// IntegTest executes integration tests (it uses Docker to run the tests).
func IntegTest() {
	devtools.AddIntegTestUsage()
	defer devtools.StopIntegTestEnv()
	mg.SerialDeps(integtest.GoIntegTest, PythonIntegTest)
}

// PythonUnitTest executes the python system tests.
func PythonUnitTest() error {
	mg.Deps(BuildSystemTestBinary)
	args := devtools.DefaultPythonTestIntegrationArgs()
	for _, provider := range functionbeat.SelectedProviders {
		args.Env = map[string]string{
			"CURRENT_PROVIDER": provider,
		}
		err := devtools.PythonNoseTest(args)
		if err != nil {
			return err
		}
	}
	return nil
}

// PythonIntegTest executes the python system tests in the integration environment (Docker).
func PythonIntegTest(ctx context.Context) error {
	if !devtools.IsInIntegTestEnv() {
		mg.Deps(functionbeat.Update.Fields)
	}
	return devtools.RunIntegTest("pythonIntegTest", func() error {
		return PythonUnitTest()
	})
}

// BuildSystemTestBinary build a binary for testing that is instrumented for
// testing and measuring code coverage. The binary is only instrumented for
// coverage when TEST_COVERAGE=true (default is false).
func BuildSystemTestBinary() error {
	params := devtools.DefaultTestBinaryArgs()
	for _, provider := range functionbeat.SelectedProviders {
		params.Name = devtools.BeatName + "-" + provider
		inputFiles := make([]string, 0)
		for _, inputFileName := range []string{"main.go", "main_test.go"} {
			inputFiles = append(inputFiles, filepath.Join(provider, inputFileName))
		}
		params.InputFiles = inputFiles
		err := devtools.BuildSystemTestBinary(params)
		if err != nil {
			return err
		}
	}
	return nil
}
