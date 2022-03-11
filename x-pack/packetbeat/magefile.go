// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build mage
// +build mage

package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"

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

// NpcapVersion specifies the version of the OEM Npcap installer to bundle with
// the packetbeat executable. It is used to specify which npcap builder crossbuild
// image to use.
const NpcapVersion = "1.60"

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
	if devtools.Platform.GOOS == "windows" && (devtools.Platform.GOARCH == "amd64" || devtools.Platform.GOARCH == "386") {
		const installer = "npcap-" + NpcapVersion + "-oem.exe"
		err := sh.Copy("./npcap/installer/"+installer, "/installer/"+installer)
		if err != nil {
			return fmt.Errorf("failed to copy Npcap installer into source tree: %w", err)
		}
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

		devtools.ImageSelector(func(platform string) (string, error) {
			image, err := devtools.CrossBuildImage(platform)
			if err != nil {
				return "", err
			}
			if os.Getenv("CI") != "true" && os.Getenv("NPCAP_LOCAL") != "true" {
				return image, nil
			}
			if platform == "windows/amd64" || platform == "windows/386" {
				image = strings.ReplaceAll(image, "beats-dev", "observability-ci") // Temporarily work around naming of npcap image.
				image = strings.ReplaceAll(image, "main", "npcap-"+NpcapVersion+"-debian9")
			}
			return image, nil
		}),
	)
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

// TestPackages tests the generated packages (i.e. file modes, owners, groups).
func TestPackages() error {
	return devtools.TestPackages()
}
