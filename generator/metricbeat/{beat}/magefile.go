// +build mage

package main

import (
	"fmt"
	"time"

	"github.com/magefile/mage/mg"

	devtools "github.com/elastic/beats/dev-tools/mage"
	metricbeat "github.com/elastic/beats/metricbeat/scripts/mage"

	// mage:import
	_ "github.com/elastic/beats/dev-tools/mage/target/common"
	// mage:import
	"github.com/elastic/beats/dev-tools/mage/target/build"
	// mage:import
	"github.com/elastic/beats/dev-tools/mage/target/pkg"
	// mage:import
	_ "github.com/elastic/beats/dev-tools/mage/target/test"
	// mage:import
	_ "github.com/elastic/beats/dev-tools/mage/target/unittest"
	// mage:import
	_ "github.com/elastic/beats/dev-tools/mage/target/integtest"
	// mage:import
	"github.com/elastic/beats/dev-tools/mage/target/collectors"
	// mage:import
	"github.com/elastic/beats/dev-tools/mage/target/update"
)

func init() {
	devtools.SetBuildVariableSources(devtools.DefaultBeatBuildVariableSources)

	devtools.BeatDescription = "One sentence description of the Beat."
	devtools.BeatVendor = "{full_name}"
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
