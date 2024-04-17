// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build mage

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
	osquerybeat "github.com/elastic/beats/v7/x-pack/osquerybeat/scripts/mage"

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
	devtools.BeatDescription = "Osquerybeat is a beat implementation for osquery."
	devtools.BeatLicense = "Elastic License"
}

func Fmt() {
	mg.Deps(devtools.Format)
}

func AddLicenseHeaders() {
	mg.Deps(devtools.AddLicenseHeaders)
}

func Check() error {
	return devtools.Check()
}

func Build() error {
	// Building osquerybeat
	err := devtools.Build(devtools.DefaultBuildArgs())
	if err != nil {
		return err
	}
	return BuildExt()
}

// BuildExt builds the osquery-extension.
func BuildExt() error {
	params := devtools.DefaultBuildArgs()
	params.InputFiles = []string{"./ext/osquery-extension/."}
	params.Name = "osquery-extension"
	params.CGO = false
	err := devtools.Build(params)
	if err != nil {
		return err
	}

	// Rename osquery-extension to osquery-extension.ext on non windows platforms
	if runtime.GOOS != "windows" {
		err = os.Rename("osquery-extension", "osquery-extension.ext")
		if err != nil {
			return err
		}
	}
	return nil
}

// Clean cleans all generated files and build artifacts.
func Clean() error {
	paths := devtools.DefaultCleanPaths
	paths = append(paths, []string{
		"osquery-extension",
		"osquery-extension.exe",
		filepath.Join("ext", "osquery-extension", "build"),
	}...)
	return devtools.Clean(paths)
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
	// Building osquerybeat
	err := devtools.CrossBuild()
	if err != nil {
		return err
	}
	return CrossBuildExt()
}

// CrossBuildExt cross-builds the osquery-extension.
func CrossBuildExt() error {
	return devtools.CrossBuild(devtools.InDir("x-pack", "osquerybeat", "ext", "osquery-extension"))
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

	devtools.MustUsePackaging("osquerybeat", "x-pack/osquerybeat/dev-tools/packaging/packages.yml")

	// Add osquery distro binaries
	osquerybeat.CustomizePackaging()

	mg.Deps(Update, osquerybeat.FetchOsqueryDistros)
	mg.Deps(CrossBuild, CrossBuildGoDaemon)
	mg.SerialDeps(devtools.Package, TestPackages)
}

// Package packages the Beat for IronBank distribution.
//
// Use SNAPSHOT=true to build snapshots.
func Ironbank() error {
	fmt.Println(">> Ironbank: this module is not subscribed to the IronBank releases.")
	return nil
}

// TestPackages tests the generated packages (i.e. file modes, owners, groups).
func TestPackages() error {
	return devtools.TestPackages()
}

// Update is an alias for update:all. This is a workaround for
// https://github.com/magefile/mage/issues/217.
func Update() { mg.Deps(osquerybeat.Update.All) }

// Fields is an alias for update:fields. This is a workaround for
// https://github.com/magefile/mage/issues/217.
func Fields() { mg.Deps(osquerybeat.Update.Fields) }

// Config is an alias for update:config. This is a workaround for
// https://github.com/magefile/mage/issues/217.
func Config() { mg.Deps(osquerybeat.Update.Config) }
