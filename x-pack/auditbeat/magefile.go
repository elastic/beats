// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build mage

package main

import (
	"fmt"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/pkg/errors"

	auditbeat "github.com/elastic/beats/v7/auditbeat/scripts/mage"
	devtools "github.com/elastic/beats/v7/dev-tools/mage"

	// mage:import
	"github.com/elastic/beats/v7/dev-tools/mage/target/common"
	// mage:import
	"github.com/elastic/beats/v7/dev-tools/mage/target/unittest"
	// mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/integtest"
	// mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/test"
)

func init() {
	common.RegisterCheckDeps(Update)
	unittest.RegisterPythonTestDeps(fieldsYML)

	devtools.BeatDescription = "Audit the activities of users and processes on your system."
	devtools.BeatLicense = "Elastic License"
	devtools.Platforms = devtools.Platforms.Filter("!linux/ppc64 !linux/mips64")
}

// Build builds the Beat binary.
func Build() error {
	return devtools.Build(devtools.DefaultBuildArgs())
}

// GolangCrossBuild build the Beat binary inside of the golang-builder.
// Do not use directly, use crossBuild instead.
func GolangCrossBuild() error {
	if d, ok := deps[devtools.Platform.Name]; ok {
		mg.Deps(d)
	}
	return devtools.GolangCrossBuild(devtools.DefaultGolangCrossBuildArgs())
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

// Package packages the Beat for distribution.
// Use SNAPSHOT=true to build snapshots.
// Use PLATFORMS to control the target platforms.
// Use VERSION_QUALIFIER to control the version qualifier.
func Package() {
	start := time.Now()
	defer func() { fmt.Println("package ran for", time.Since(start)) }()

	devtools.UseElasticBeatXPackPackaging()
	devtools.PackageKibanaDashboardsFromBuildDir()
	auditbeat.CustomizePackaging(auditbeat.XPackPackaging)

	mg.SerialDeps(Fields, Dashboards, Config, devtools.GenerateModuleIncludeListGo)
	mg.Deps(CrossBuild, CrossBuildGoDaemon)
	mg.SerialDeps(devtools.Package, TestPackages)
}

// TestPackages tests the generated packages (i.e. file modes, owners, groups).
func TestPackages() error {
	return devtools.TestPackages(devtools.WithRootUserContainer())
}

// Update is an alias for running fields, dashboards, config.
func Update() {
	mg.SerialDeps(Fields, Dashboards, Config, devtools.GenerateModuleIncludeListGo)
}

// Config generates both the short and reference configs.
func Config() error {
	return devtools.Config(devtools.AllConfigTypes, auditbeat.XPackConfigFileParams(), ".")
}

// Fields generates a fields.yml and include/fields.go.
func Fields() {
	mg.SerialDeps(fieldsYML, moduleFieldsGo)
}

func moduleFieldsGo() error {
	return devtools.GenerateModuleFieldsGo("module")
}

// fieldsYML generates the fields.yml file containing all fields.
func fieldsYML() error {
	return devtools.GenerateFieldsYAML(devtools.OSSBeatDir("module"), "module")
}

// ExportDashboard exports a dashboard and writes it into the correct directory.
//
// Required environment variables:
// - MODULE: Name of the module
// - ID:     Dashboard id
func ExportDashboard() error {
	return devtools.ExportDashboard()
}

// Dashboards collects all the dashboards and generates index patterns.
func Dashboards() error {
	return devtools.KibanaDashboards(devtools.OSSBeatDir("module"), "module")
}

// -----------------------------------------------------------------------------
// - Install the librpm-dev package
var (
	deps = map[string]func() error{
		"linux/386":      installLinux386,
		"linux/amd64":    installLinuxAMD64,
		"linux/arm64":    installLinuxARM64,
		"linux/armv5":    installLinuxARMLE,
		"linux/armv6":    installLinuxARMLE,
		"linux/armv7":    installLinuxARMHF,
		"linux/mips":     installLinuxMIPS,
		"linux/mipsle":   installLinuxMIPSLE,
		"linux/mips64le": installLinuxMIPS64LE,
		"linux/ppc64le":  installLinuxPPC64LE,
		"linux/s390x":    installLinuxS390X,

		//"linux/ppc64":  installLinuxPpc64,
		//"linux/mips64": installLinuxMips64,
	}
)

const (
	librpmDevPkgName = "librpm-dev"
)

func installLinuxAMD64() error {
	return installDependencies(librpmDevPkgName, "")
}

func installLinuxARM64() error {
	return installDependencies(librpmDevPkgName+":arm64", "arm64")
}

func installLinuxARMHF() error {
	return installDependencies(librpmDevPkgName+":armhf", "armhf")
}

func installLinuxARMLE() error {
	return installDependencies(librpmDevPkgName+":armel", "armel")
}

func installLinux386() error {
	return installDependencies(librpmDevPkgName+":i386", "i386")
}

func installLinuxMIPS() error {
	return installDependencies(librpmDevPkgName+":mips", "mips")
}

func installLinuxMIPS64LE() error {
	return installDependencies(librpmDevPkgName+":mips64el", "mips64el")
}

func installLinuxMIPSLE() error {
	return installDependencies(librpmDevPkgName+":mipsel", "mipsel")
}

func installLinuxPPC64LE() error {
	return installDependencies(librpmDevPkgName+":ppc64el", "ppc64el")
}

func installLinuxS390X() error {
	return installDependencies(librpmDevPkgName+":s390x", "s390x")
}

func installDependencies(pkg, arch string) error {
	if arch != "" {
		err := sh.Run("dpkg", "--add-architecture", arch)
		if err != nil {
			return errors.Wrap(err, "error while adding architecture")
		}
	}

	// TODO: This is only for debian 7 and should be removed when move to a newer OS. This flag is
	// going to be used unnecessary when building using non-debian7 images
	// (like when making the linux/arm binaries) and we should remove it soonish.
	// See https://github.com/elastic/beats/v7/issues/11750 for more details.
	if err := sh.Run("apt-get", "update", "-o", "Acquire::Check-Valid-Until=false"); err != nil {
		return err
	}

	return sh.Run("apt-get", "install", "-y", "--no-install-recommends", pkg)
}
