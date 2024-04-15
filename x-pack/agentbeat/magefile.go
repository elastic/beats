// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build mage

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/magefile/mage/sh"
	"go.uber.org/multierr"

	"github.com/magefile/mage/mg"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
	"github.com/elastic/beats/v7/dev-tools/mage/target/build"
	packetbeat "github.com/elastic/beats/v7/packetbeat/scripts/mage"
	osquerybeat "github.com/elastic/beats/v7/x-pack/osquerybeat/scripts/mage"

	//mage:import
	"github.com/elastic/beats/v7/dev-tools/mage/target/common"
	//mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/integtest/docker"
	//mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/test"
)

// beats are the beats the agentbeat combines
var beats = []string{
	"auditbeat",
	"filebeat",
	"heartbeat",
	"metricbeat",
	"osquerybeat",
	"packetbeat",
}

func init() {
	common.RegisterCheckDeps(Update)

	devtools.BeatDescription = "Combined beat ran only by the Elastic Agent"
	devtools.BeatLicense = "Elastic License"

	// disabled from auditbeat (not supported by Elastic Agent either)
	devtools.Platforms = devtools.Platforms.Filter("!linux/ppc64 !linux/mips64")
}

// Build builds the Beat binary.
func Build() error {
	args := devtools.DefaultBuildArgs()
	if devtools.Platform.GOOS == "linux" {
		args.ExtraFlags = append(args.ExtraFlags, "-tags=agentbeat,withjournald")
	} else {
		args.ExtraFlags = append(args.ExtraFlags, "-tags=agentbeat")
	}
	return devtools.Build(args)
}

// BuildSystemTestBinary builds a binary instrumented for use with Python system tests.
func BuildSystemTestBinary() error {
	args := devtools.DefaultTestBinaryArgs()
	args.ExtraFlags = append(args.ExtraFlags, "-tags=agentbeat")
	return devtools.BuildSystemTestGoBinary(args)
}

// GolangCrossBuild build the Beat binary inside of the golang-builder.
// Do not use directly, use crossBuild instead.
func GolangCrossBuild() error {
	// need packetbeat build arguments as it address the requirements for libpcap
	args := packetbeat.GolangCrossBuildArgs()
	if devtools.Platform.GOOS == "linux" {
		args.ExtraFlags = append(args.ExtraFlags, "-tags=agentbeat,withjournald")
	} else {
		args.ExtraFlags = append(args.ExtraFlags, "-tags=agentbeat")
	}
	return multierr.Combine(
		devtools.GolangCrossBuild(args),
		devtools.TestLinuxForCentosGLIBC(),
	)
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

// CrossBuildDeps cross-builds the required dependencies.
func CrossBuildDeps() error {
	return callForBeat("crossBuildExt", "osquerybeat")
}

// Package packages the Beat for distribution.
// Use SNAPSHOT=true to build snapshots.
// Use PLATFORMS to control the target platforms.
// Use VERSION_QUALIFIER to control the version qualifier.
func Package() {
	start := time.Now()
	defer func() { fmt.Println("package ran for", time.Since(start)) }()

	// specific packaging just for agentbeat
	devtools.MustUsePackaging("agentbeat", "x-pack/agentbeat/dev-tools/packaging/packages.yml")

	// Add osquery distro binaries, required for the osquerybeat subcommand.
	osquerybeat.CustomizePackaging()

	mg.SerialDeps(Update, osquerybeat.FetchOsqueryDistros, CrossBuildDeps, CrossBuild, devtools.Package, TestPackages)
}

// TestPackages tests the generated packages (i.e. file modes, owners, groups).
func TestPackages() error {
	return devtools.TestPackages()
}

// Update is an alias for running fields, dashboards, config.
func Update() {
	callForEachBeat("update")
}

func callForEachBeat(target string) error {
	for _, beat := range beats {
		err := callForBeat(target, beat)
		if err != nil {
			return fmt.Errorf("failed to perform mage %s for beat %s: %w", target, beat, err)
		}
	}
	return nil
}

func callForBeat(target string, beat string) error {
	path, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to getwd: %w", err)
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get abs path: %w", err)
	}
	fmt.Printf(">> Changing into %s directory\n", beat)
	err = os.Chdir(filepath.Join("..", beat))
	if err != nil {
		return fmt.Errorf("failed to chdir to %s: %w")
	}
	defer os.Chdir(path)

	fmt.Printf(">> Executing mage %s for %s\n", target, beat)
	err = sh.RunV("mage", target)
	if err != nil {
		return fmt.Errorf("failed to exec: %w", err)
	}
	return nil
}

// IntegTest executes integration tests (it uses Docker to run the tests).
func IntegTest() {
	mg.SerialDeps(GoIntegTest, PythonIntegTest)
}

// GoIntegTest starts the docker containers and executes the Go integration tests.
func GoIntegTest(ctx context.Context) error {
	mg.Deps(BuildSystemTestBinary)
	args := devtools.DefaultGoTestIntegrationFromHostArgs()
	args.Tags = append(args.Tags, "agentbeat")
	args.Packages = append(args.Packages, "../auditbeat/...", "../filebeat/...", "../heartbeat/...", "../metricbeat/...", "../osquerybeat/...", "../packetbeat/...")
	return devtools.GoIntegTestFromHost(ctx, args)
}

// PythonIntegTest starts the docker containers and executes the Python integration tests.
func PythonIntegTest(ctx context.Context) error {
	mg.Deps(BuildSystemTestBinary)
	return devtools.PythonIntegTestFromHost(devtools.DefaultPythonTestIntegrationFromHostArgs())
}
