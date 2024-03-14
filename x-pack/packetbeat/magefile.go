// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build mage
// +build mage

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/magefile/mage/mg"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
	packetbeat "github.com/elastic/beats/v7/packetbeat/scripts/mage"

	// mage:import
	"github.com/elastic/beats/v7/dev-tools/mage/target/common"
	// mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/compose"
	// mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/unittest"
	// mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/test"
)

func init() {
	common.RegisterCheckDeps(Update)

	devtools.BeatDescription = "Packetbeat analyzes network traffic and sends the data to Elasticsearch."
	devtools.BeatLicense = "Elastic License"
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
	return packetbeat.GolangCrossBuild()
}

// CrossBuild cross-builds the beat for all target platforms.
func CrossBuild() error {
	return packetbeat.CrossBuild()
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
	mg.Deps(CrossBuild, CrossBuildGoDaemon)
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
	mg.SerialDeps(getNpcapInstaller, devtools.BuildSystemTestBinary)

	args := devtools.DefaultGoTestIntegrationArgs()
	args.Packages = []string{"./tests/system/..."}
	return devtools.GoTest(ctx, args)
}

func getBucketName() string {
	if os.Getenv("BUILDKITE") == "true" {
		return "ingest-buildkite-ci"
	}
	return "obs-ci-cache"
}

// getNpcapInstaller gets the installer from the Google Cloud Storage service.
//
// On Windows platforms, if getNpcapInstaller is invoked with the environment variables
// CI or NPCAP_LOCAL set to "true" and the OEM Npcap installer is not available it is
// obtained from the cloud storage. This behaviour requires access to the private store.
// If NPCAP_LOCAL is set to "true" and the file is in the npcap/installer directory, no
// fetch will be made.
func getNpcapInstaller() error {
	// TODO: Consider whether to expose this as a target.
	if runtime.GOOS != "windows" {
		return nil
	}
	if os.Getenv("CI") != "true" && os.Getenv("NPCAP_LOCAL") != "true" {
		return errors.New("only available if running in the CI or with NPCAP_LOCAL=true")
	}
	dstPath := filepath.Join("./npcap/installer", installer)
	if os.Getenv("NPCAP_LOCAL") == "true" {
		fi, err := os.Stat(dstPath)
		if err == nil && !fi.IsDir() {
			fmt.Println("using local Npcap installer with NPCAP_LOCAL=true")
			return nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	ciBucketName := getBucketName()

	fmt.Printf("getting %s from private cache\n", installer)
	return sh.RunV("gsutil", "cp", "gs://"+ciBucketName+"/private/"+installer, dstPath)
}
