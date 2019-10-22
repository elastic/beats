// +build mage

package main

import (
	"fmt"
	"time"

	"github.com/magefile/mage/mg"

	devtools "github.com/elastic/beats/dev-tools/mage"
	"github.com/elastic/beats/dev-tools/mage/target/build"
	"github.com/elastic/beats/dev-tools/mage/target/common"
	"github.com/elastic/beats/dev-tools/mage/target/pkg"
	"github.com/elastic/beats/dev-tools/mage/target/unittest"
	"github.com/elastic/beats/dev-tools/mage/target/update"
)

func init() {
	devtools.SetBuildVariableSources(devtools.DefaultBeatBuildVariableSources)

	devtools.BeatDescription = "One sentence description of the Beat."
	devtools.BeatVendor = "{full_name}"
}

// Package packages the Beat for distribution.
// Use SNAPSHOT=true to build snapshots.
// Use PLATFORMS to control the target platforms.
func Package() {
	start := time.Now()
	defer func() { fmt.Println("package ran for", time.Since(start)) }()

	devtools.UseCommunityBeatPackaging()

	mg.Deps(update.Update)
	mg.Deps(build.CrossBuild, build.CrossBuildGoDaemon)
	mg.SerialDeps(devtools.Package, pkg.PackageTest)
}

// Config generates both the short/reference/docker configs.
func Config() error {
	return devtools.Config(devtools.AllConfigTypes, devtools.ConfigFileParams{}, ".")
}

//Fields generates a fields.yml for the Beat.
func Fields() error {
	return devtools.GenerateFieldsYAML()
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
	return build.Build()
}

// CrossBuild cross-builds the beat for all target platforms.
func CrossBuild() error {
	return build.CrossBuild()
}
