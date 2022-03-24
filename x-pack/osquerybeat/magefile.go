// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build mage
// +build mage

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/magefile/mage/mg"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/command"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/distro"
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

func Check() error {
	return devtools.Check()
}

func Build() error {
	params := devtools.DefaultBuildArgs()

	// Building osquerybeat
	err := devtools.Build(params)
	if err != nil {
		return err
	}

	params.InputFiles = []string{"./ext/osquery-extension/."}
	params.Name = "osquery-extension"
	params.CGO = false
	err = devtools.Build(params)
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

func extractFromMSI() error {
	if os.Getenv("GOOS") != "windows" {
		return nil
	}

	ctx := context.Background()

	execCommand := func(name string, args ...string) error {
		ps := strings.Join(append([]string{name}, args...), " ")
		fmt.Println(ps)
		output, err := command.Execute(ctx, name, args...)
		if err != nil {
			fmt.Println(ps, ", failed: ", err)
			return err
		}
		fmt.Print(output)
		return err
	}

	osArchs := osquerybeat.OSArchs(devtools.Platforms)

	for _, osarch := range osArchs {
		if osarch.OS != "windows" {
			continue
		}
		spec, err := distro.GetSpec(osarch)
		if err != nil {
			if errors.Is(err, distro.ErrUnsupportedOS) {
				continue
			} else {
				return err
			}
		}
		dip := distro.GetDataInstallDir(osarch)
		msiFile := spec.DistroFilepath(dip)

		// MSI extract
		err = execCommand("msiextract", "--directory", dip, msiFile)
		if err != nil {
			return err
		}

		fmt.Println("copy osqueryd.exe from MSI")
		dp := distro.OsquerydPathForOS(osarch.OS, dip)
		err = devtools.Copy(filepath.Join(dip, "osquery", "osqueryd", "osqueryd.exe"), dp)
		if err != nil {
			fmt.Println("copy osqueryd.exe from MSI failed: ", err)
			return err
		}
		// Chmod set to the same as other executables in the final package
		if err = os.Chmod(dp, 0755); err != nil {
			return err
		}
	}

	return nil
}

// GolangCrossBuild build the Beat binary inside of the golang-builder.
// Do not use directly, use crossBuild instead.
func GolangCrossBuild() error {
	// This is to fix a defect in the field where msiexec fails to extract the osqueryd.exe
	// from bundled osquery.msi, with error code 1603
	// https://docs.microsoft.com/en-us/troubleshoot/windows-server/application-management/msi-installation-error-1603
	// SDH: https://github.com/elastic/sdh-beats/issues/1575
	// Currently we can't reproduce this is issue, but here we can eliminate the need for calling msiexec
	// if extract the osqueryd.exe binary during the build.
	//
	// The cross build is currently called for two binaries osquerybeat and osqquery-extension
	// Only extract osqueryd.exe during osquerybeat build on windows
	args := devtools.DefaultGolangCrossBuildArgs()

	if !strings.HasPrefix(args.Name, "osquery-extension-") {
		// Extract osqueryd.exe from MSI
		if err := extractFromMSI(); err != nil {
			return err
		}
	}

	return devtools.GolangCrossBuild(args)
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

	err = devtools.CrossBuild(devtools.InDir("x-pack", "osquerybeat", "ext", "osquery-extension"))
	if err != nil {
		return err
	}
	return nil
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

	devtools.MustUsePackaging("osquerybeat", "x-pack/osquerybeat/dev-tools/packaging/packages.yml")

	// Add osquery distro binaries
	osquerybeat.CustomizePackaging()

	mg.Deps(Update, osquerybeat.FetchOsqueryDistros)
	mg.Deps(CrossBuild, CrossBuildGoDaemon)
	mg.SerialDeps(devtools.Package, TestPackages)
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
