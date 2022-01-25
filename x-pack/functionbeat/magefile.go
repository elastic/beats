// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build mage
// +build mage

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/magefile/mage/mg"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
	"github.com/elastic/beats/v7/dev-tools/mage/target/build"
	functionbeat "github.com/elastic/beats/v7/x-pack/functionbeat/scripts/mage"

	// mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/common"
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

	// Getting selected cloud providers
	selectedProviders, err := functionbeat.SelectedProviders()
	if err != nil {
		return err
	}

	// Building functions to deploy
	for _, provider := range selectedProviders {
		if !provider.Buildable {
			continue
		}

		inputFiles := filepath.Join("provider", provider.Name, "main.go")
		params.InputFiles = []string{inputFiles}
		params.Name = devtools.BeatName + "-" + provider.Name
		params.OutputDir = filepath.Join("provider", provider.Name)
		params.CGO = false
		params.Env = make(map[string]string)
		if provider.GOOS != "" {
			params.Env["GOOS"] = provider.GOOS
		}
		if provider.GOARCH != "" {
			params.Env["GOARCH"] = provider.GOARCH
		}
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

	// Getting selected cloud providers
	selectedProviders, err := functionbeat.SelectedProviders()
	if err != nil {
		return err
	}

	// Building functions to deploy
	for _, provider := range selectedProviders {
		if !provider.Buildable {
			continue
		}

		if runtime.GOARCH != "amd64" {
			fmt.Println("Crossbuilding functions only works on amd64 architecture.")
			return nil
		}

		err := devtools.CrossBuild(devtools.AddPlatforms("linux/amd64"), devtools.InDir("x-pack", "functionbeat", "provider", provider.Name))
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

// AssembleDarwinUniversal merges the darwin/amd64 and darwin/arm64 into a single
// universal binary using `lipo`. It assumes the darwin/amd64 and darwin/arm64
// were built and only performs the merge.
func AssembleDarwinUniversal() error {
	return build.AssembleDarwinUniversal()
}

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

// BuildPkgForFunctions creates a folder named pkg and adds functions to it.
// This makes testing the manager more comfortable.
func BuildPkgForFunctions() error {
	mg.Deps(Update, Build)

	err := os.RemoveAll("pkg")

	filesToCopy := map[string]string{
		filepath.Join("provider", "aws", "functionbeat-aws"): filepath.Join("pkg", "functionbeat-aws"),
	}
	for src, dest := range filesToCopy {
		c := &devtools.CopyTask{
			Source: src,
			Dest:   dest,
		}
		err = c.Execute()
		if err != nil {
			return err
		}
	}
	return nil
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

	// Getting selected cloud providers
	selectedProviders, err := functionbeat.SelectedProviders()
	if err != nil {
		return err
	}

	for _, provider := range selectedProviders {
		if !provider.Buildable {
			continue
		}

		params.Name = filepath.Join("provider", provider.Name, devtools.BeatName+"-"+provider.Name)
		inputFiles := make([]string, 0)
		for _, inputFileName := range []string{"main.go", "main_test.go"} {
			inputFiles = append(inputFiles, filepath.Join("provider", provider.Name, inputFileName))
		}
		params.InputFiles = inputFiles
		err := devtools.BuildSystemTestGoBinary(params)
		if err != nil {
			return err
		}
	}
	return nil
}
