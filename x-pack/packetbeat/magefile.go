// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build mage

package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
	"github.com/elastic/beats/v7/dev-tools/mage/target/build"
	packetbeat "github.com/elastic/beats/v7/packetbeat/scripts/mage"

	//mage:import
	"github.com/elastic/beats/v7/dev-tools/mage/target/common"
	//mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/unittest"
	//mage:import
	"github.com/elastic/beats/v7/dev-tools/mage/target/test"
)

// NpcapVersion specifies the version of the OEM Npcap installer to bundle with
// the packetbeat executable. It is used to specify which npcap builder crossbuild
// image to use and the installer to obtain from the cloud store for testing.
const (
	NpcapVersion = "1.78"
	installer    = "npcap-" + NpcapVersion + "-oem.exe"
)

func init() {
	common.RegisterCheckDeps(Update)

	test.RegisterDeps(SystemTest)

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
			if platform == "windows/amd64" {
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
