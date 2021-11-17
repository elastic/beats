//go:build mage
// +build mage

package main

import (
	"fmt"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
	"github.com/elastic/beats/v7/dev-tools/mage/target/build"
	"github.com/elastic/beats/v7/dev-tools/mage/target/common"
	"github.com/elastic/beats/v7/dev-tools/mage/target/pkg"
	"github.com/elastic/beats/v7/dev-tools/mage/target/unittest"
)

func init() {
	common.RegisterCheckDeps(Update)
	unittest.RegisterPythonTestDeps(Fields)
	integtest.RegisterPythonTestDeps(Fields)
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

	//	devtools.UseElasticBeatOSSPackaging()
	// community beat package
	// ToDo decide whenther kubebeat should move to x-pack dir & adjust accordingly

	devtools.PackageKibanaDashboardsFromBuildDir()

	mg.Deps(Update)
	mg.Deps(build.CrossBuild, build.CrossBuildGoDaemon)
	mg.SerialDeps(devtools.Package, TestPackages)
}

// Update updates the generated files (aka make update).
func Update() error {
	return sh.Run("make", "update")
}

// Fields generates a fields.yml for the Beat.
func Fields() error {
	return devtools.GenerateFieldsYAML()
}

// Config generates both the short/reference/docker configs.
func Config() error {
	p := devtools.DefaultConfigFileParams()
	p.Templates = append(p.Templates, "_meta/config/*.tmpl")
	return devtools.Config(devtools.AllConfigTypes, p, ".")
}

// Clean cleans all generated files and build artifacts.
func Clean() error {
	return devtools.Clean()
}

// Check formats code, updates generated content, check for common errors, and
// checks for any modified files.
func Check() {
	common.Check()
}

// Fmt formats source code (.go and .py) and adds license headers.
func Fmt() {
	common.Fmt()
}

// Test runs all available tests
func Test() {
	mg.Deps(unittest.GoUnitTest)
}

// Build builds the Beat binary.
func Build() error {
	params := devtools.DefaultBuildArgs()

	// Building kubebeat
	err := devtools.Build(params)
	if err != nil {
		return err
	}

	err = devtools.Build(params)
	if err != nil {
		return err
	}

	return nil

}

// CrossBuild cross-builds the beat for all target platforms.
func CrossBuild() error {
	return build.CrossBuild()
}

// BuildGoDaemon builds the go-daemon binary (use crossBuildGoDaemon).
func BuildGoDaemon() error {
	return build.BuildGoDaemon()
}

// GolangCrossBuild build the Beat binary inside of the golang-builder.
// Do not use directly, use crossBuild instead.
func GolangCrossBuild() error {
	return build.GolangCrossBuild()
}
