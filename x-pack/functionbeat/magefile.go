// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build mage

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/magefile/mage/mg"

	devtools "github.com/elastic/beats/dev-tools/mage"
	functionbeat "github.com/elastic/beats/x-pack/functionbeat/scripts/mage"
)

var (
	availableProviders = []string{
		"aws",
	}
	selectedProviders []string
)

func init() {
	devtools.BeatDescription = "Functionbeat is a beat implementation for a serverless architecture."
	devtools.BeatLicense = "Elastic License"
	selectedProviders = getConfiguredProviders()
}

// Build builds the Beat binary.
func Build() error {
	workingDir, err := os.Getwd()
	if err != nil {
		return err
	}

	params := devtools.DefaultBuildArgs()
	for _, provider := range getConfiguredProviders() {
		params.Name = devtools.BeatName + "-" + provider
		err = os.Chdir(workingDir + "/" + provider)
		if err != nil {
			return err
		}

		err = devtools.Build(params)
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
	return devtools.CrossBuild(devtools.AddPlatforms("linux/amd64"))
}

// CrossBuildGoDaemon cross-builds the go-daemon binary using Docker.
func CrossBuildGoDaemon() error {
	return devtools.CrossBuildGoDaemon()
}

// Clean cleans all generated files and build artifacts.
func Clean() error {
	return devtools.Clean()
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

// Update updates the generated files (aka make update).
func Update() {
	mg.SerialDeps(Fields, Config, includeFields, docs)
}

// GoTestUnit executes the Go unit tests.
// Use TEST_COVERAGE=true to enable code coverage profiling.
// Use RACE_DETECTOR=true to enable the race detector.
func GoTestUnit(ctx context.Context) error {
	return devtools.GoTest(ctx, devtools.DefaultGoTestUnitArgs())
}

// GoTestIntegration executes the Go integration tests.
// Use TEST_COVERAGE=true to enable code coverage profiling.
// Use RACE_DETECTOR=true to enable the race detector.
func GoTestIntegration(ctx context.Context) error {
	return devtools.GoTest(ctx, devtools.DefaultGoTestIntegrationArgs())
}

// Config generates both the short and reference configs.
func Config() error {
	for _, provider := range getConfiguredProviders() {
		err := devtools.Config(devtools.ShortConfigType|devtools.ReferenceConfigType, functionbeat.XPackConfigFileParams(provider), provider)
		if err != nil {
			return err
		}
	}
	return nil
}

// Fields generates a fields.yml for the Beat.
func Fields() error {
	for _, provider := range getConfiguredProviders() {
		output := filepath.Join(devtools.CWD(), provider, "fields.yml")
		err := devtools.GenerateFieldsYAMLTo(output)
		if err != nil {
			return err
		}
	}
	return nil
}

func includeFields() error {
	fnBeatDir := devtools.CWD()
	for _, provider := range getConfiguredProviders() {
		err := os.Chdir(filepath.Join(fnBeatDir, provider))
		if err != nil {
			return err
		}
		output := filepath.Join(fnBeatDir, provider, "include", "fields.go")
		err = devtools.GenerateFieldsGoWithName(devtools.BeatName+"-"+provider, "fields.yml", output)
		if err != nil {
			return err
		}
		os.Chdir(fnBeatDir)
	}
	return nil
}

func docs() error {
	for _, provider := range getConfiguredProviders() {
		fieldsYml := filepath.Join(devtools.CWD(), provider, "fields.yml")
		err := devtools.Docs.FieldDocs(fieldsYml)
		if err != nil {
			return err
		}
	}
	return nil
}

func getConfiguredProviders() []string {
	providers := os.Getenv("PROVIDERS")
	if len(providers) == 0 {
		return availableProviders
	}

	return strings.Split(providers, ",")
}
