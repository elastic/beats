// +build mage

package main

import (
	"fmt"
	"time"

	"github.com/magefile/mage/mg"

	devtools "github.com/elastic/beats/dev-tools/mage"
	"github.com/elastic/beats/dev-tools/mage/target/build"
	"github.com/elastic/beats/dev-tools/mage/target/collectors"
	"github.com/elastic/beats/dev-tools/mage/target/common"
	"github.com/elastic/beats/dev-tools/mage/target/pkg"
	"github.com/elastic/beats/dev-tools/mage/target/unittest"
	"github.com/elastic/beats/dev-tools/mage/target/update"
	"github.com/elastic/beats/generator/common/beatgen"
	metricbeat "github.com/elastic/beats/metricbeat/scripts/mage"
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

// CollectAll generates the docs and the fields.
func CollectAll() {
	mg.Deps(collectors.CollectDocs, FieldsDocs)
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

// FieldsDocs generates docs/fields.asciidoc containing all fields
// (including x-pack).
func FieldsDocs() error {
	inputs := []string{
		devtools.OSSBeatDir("module"),
	}
	output := devtools.CreateDir("build/fields/fields.all.yml")
	if err := devtools.GenerateFieldsYAMLTo(output, inputs...); err != nil {
		return err
	}
	return devtools.Docs.FieldDocs(output)
}

// Fields generates a fields.yml for the Beat.
func Fields() error {
	return devtools.GenerateFieldsYAML("module")
}

// Config generates both the short/reference/docker configs.
func Config() {
	mg.Deps(configYML, metricbeat.GenerateDirModulesD)
}

func configYML() error {
	customDeps := devtools.ConfigFileParams{
		ShortParts:     []string{"_meta/short.yml", devtools.LibbeatDir("_meta/config.yml.tmpl")},
		ReferenceParts: []string{"_meta/reference.yml", devtools.LibbeatDir("_meta/config.reference.yml.tmpl")},
		DockerParts:    []string{"_meta/docker.yml", devtools.LibbeatDir("_meta/config.docker.yml")},
		ExtraVars:      map[string]interface{}{"BeatName": devtools.BeatName},
	}
	return devtools.Config(devtools.AllConfigTypes, customDeps, ".")
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

// Update updates the generated files (aka make update).
func Update() error {
	return update.Update()
}

// Imports generates an include/list.go file containing
// a import statement for each module and dataset.
func Imports() error {
	return devtools.GenerateModuleIncludeListGo()
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
