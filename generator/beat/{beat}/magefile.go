// +build mage

package main

import (
	"fmt"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"

	devtools "github.com/elastic/beats/dev-tools/mage"
	"github.com/elastic/beats/generator/common/beatgen"

	// mage:import
	"github.com/elastic/beats/dev-tools/mage/target/pkg"
	// mage:import
	"github.com/elastic/beats/dev-tools/mage/target/build"
	// mage:import
	_ "github.com/elastic/beats/dev-tools/mage/target/common"
	// mage:import
	_ "github.com/elastic/beats/dev-tools/mage/target/test"
	// mage:import
	_ "github.com/elastic/beats/dev-tools/mage/target/unittest"
	// mage:import
	_ "github.com/elastic/beats/dev-tools/mage/target/integtest"
)

func init() {
	devtools.SetBuildVariableSources(devtools.DefaultBeatBuildVariableSources)

	devtools.BeatDescription = "One sentence description of the Beat."
	devtools.BeatVendor = "{full_name}"
}

// VendorUpdate updates elastic/beats in the vendor dir
func VendorUpdate() error {
	return beatgen.VendorUpdate()
}

// Package packages the Beat for distribution.
// Use SNAPSHOT=true to build snapshots.
// Use PLATFORMS to control the target platforms.
func Package() {
	start := time.Now()
	defer func() { fmt.Println("package ran for", time.Since(start)) }()

	devtools.UseCommunityBeatPackaging()

	mg.Deps(Update)
	mg.Deps(build.CrossBuild, build.CrossBuildGoDaemon)
	mg.SerialDeps(devtools.Package, pkg.PackageTest)
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
	return devtools.Config(devtools.AllConfigTypes, devtools.ConfigFileParams{}, ".")
}
