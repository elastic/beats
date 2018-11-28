// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build mage

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/magefile/mage/mg"

	"github.com/elastic/beats/dev-tools/mage"
)

func init() {
	mage.BeatDescription = "Metricbeat is a lightweight shipper for metrics."
	mage.BeatLicense = "Elastic"
}

// Build builds the Beat binary.
func Build() error {
	return mage.Build(mage.DefaultBuildArgs())
}

// GolangCrossBuild build the Beat binary inside of the golang-builder.
// Do not use directly, use crossBuild instead.
func GolangCrossBuild() error {
	return mage.GolangCrossBuild(mage.DefaultGolangCrossBuildArgs())
}

// CrossBuild cross-builds the beat for all target platforms.
func CrossBuild() error {
	return mage.CrossBuild()
}

// BuildGoDaemon builds the go-daemon binary (use crossBuildGoDaemon).
func BuildGoDaemon() error {
	return mage.BuildGoDaemon()
}

// CrossBuildGoDaemon cross-builds the go-daemon binary using Docker.
func CrossBuildGoDaemon() error {
	return mage.CrossBuildGoDaemon()
}

// Clean cleans all generated files and build artifacts.
func Clean() error {
	return mage.Clean()
}

// Package packages the Beat for distribution.
// Use SNAPSHOT=true to build snapshots.
// Use PLATFORMS to control the target platforms.
// Use BEAT_VERSION_QUALIFIER to control the version qualifier.
func Package() {
	start := time.Now()
	defer func() { fmt.Println("package ran for", time.Since(start)) }()

	mage.LoadLocalNamedSpec("xpack")

	mg.Deps(Update, prepareModulePackaging)
	mg.Deps(CrossBuild, CrossBuildGoDaemon)
	mg.SerialDeps(mage.Package, TestPackages)
}

// TestPackages tests the generated packages (i.e. file modes, owners, groups).
func TestPackages() error {
	return mage.TestPackages(mage.WithModulesD())
}

// Fields generates a fields.yml and fields.go for each module.
func Fields() {
	mg.Deps(fieldsYML, mage.GenerateModuleFieldsGo)
}

// fieldsYML generates a fields.yml based on filebeat + x-pack/filebeat/modules.
func fieldsYML() error {
	return mage.GenerateFieldsYAML(mage.OSSBeatDir("module"), "module")
}

// Dashboards collects all the dashboards and generates index patterns.
func Dashboards() error {
	return mage.KibanaDashboards(mage.OSSBeatDir("module"), "module")
}

// Config generates both the short and reference configs.
func Config() {
	mg.Deps(shortConfig, referenceConfig, createDirModulesD)
}

// Update is an alias for running fields, dashboards, config.
func Update() {
	mg.SerialDeps(Fields, Dashboards, Config, prepareModulePackaging,
		mage.GenerateModuleIncludeListGo)
}

// Fmt formats source code and adds file headers.
func Fmt() {
	mg.Deps(mage.Format)
}

// Check runs fmt and update then returns an error if any modifications are found.
func Check() {
	mg.SerialDeps(mage.Format, Update, mage.Check)
}

// IntegTest executes integration tests (it uses Docker to run the tests).
func IntegTest() {
	mage.AddIntegTestUsage()
	defer mage.StopIntegTestEnv()
	mg.SerialDeps(GoIntegTest, PythonIntegTest)
}

// UnitTest executes the unit tests.
func UnitTest() {
	mg.SerialDeps(GoUnitTest, PythonUnitTest)
}

// GoUnitTest executes the Go unit tests.
// Use TEST_COVERAGE=true to enable code coverage profiling.
// Use RACE_DETECTOR=true to enable the race detector.
func GoUnitTest(ctx context.Context) error {
	return mage.GoTest(ctx, mage.DefaultGoTestUnitArgs())
}

// GoIntegTest executes the Go integration tests.
// Use TEST_COVERAGE=true to enable code coverage profiling.
// Use RACE_DETECTOR=true to enable the race detector.
func GoIntegTest(ctx context.Context) error {
	return mage.RunIntegTest("goIntegTest", func() error {
		return mage.GoTest(ctx, mage.DefaultGoTestIntegrationArgs())
	})
}

// PythonUnitTest executes the python system tests.
func PythonUnitTest() error {
	mg.Deps(mage.BuildSystemTestBinary)
	return mage.PythonNoseTest(mage.DefaultPythonTestUnitArgs())
}

// PythonIntegTest executes the python system tests in the integration environment (Docker).
func PythonIntegTest(ctx context.Context) error {
	if !mage.IsInIntegTestEnv() {
		mg.Deps(Fields)
	}
	return mage.RunIntegTest("pythonIntegTest", func() error {
		mg.Deps(mage.BuildSystemTestBinary)
		return mage.PythonNoseTest(mage.DefaultPythonTestIntegrationArgs())
	})
}

// -----------------------------------------------------------------------------
// Customizations specific to Metricbeat.
// - Include modules.d directory in packages.

const (
	dirModulesDGenerated = "build/package/modules.d"
)

// prepareModulePackaging generates modules and modules.d directories
// for an x-pack distribution, excluding _meta and test files so that they are
// not included in packages.
func prepareModulePackaging() error {
	mg.Deps(createDirModulesD)

	err := mage.Clean([]string{
		dirModulesDGenerated,
	})
	if err != nil {
		return err
	}

	for _, copyAction := range []struct {
		src, dst string
	}{
		{mage.OSSBeatDir("modules.d"), dirModulesDGenerated},
		{"modules.d", dirModulesDGenerated},
	} {
		err := (&mage.CopyTask{
			Source:  copyAction.src,
			Dest:    copyAction.dst,
			Mode:    0644,
			DirMode: 0755,
		}).Execute()
		if err != nil {
			return err
		}
	}
	return nil
}

func shortConfig() error {
	var configParts = []string{
		mage.OSSBeatDir("_meta/common.yml"),
		mage.OSSBeatDir("_meta/setup.yml"),
		"{{ elastic_beats_dir }}/libbeat/_meta/config.yml",
	}

	for i, f := range configParts {
		configParts[i] = mage.MustExpand(f)
	}

	configFile := mage.BeatName + ".yml"
	mage.MustFileConcat(configFile, 0640, configParts...)
	mage.MustFindReplace(configFile, regexp.MustCompile("beatname"), mage.BeatName)
	mage.MustFindReplace(configFile, regexp.MustCompile("beat-index-prefix"), mage.BeatIndexPrefix)
	return nil
}

func referenceConfig() error {
	const modulesConfigYml = "build/config.modules.yml"
	err := mage.GenerateModuleReferenceConfig(modulesConfigYml, mage.OSSBeatDir("module"), "module")
	if err != nil {
		return err
	}
	defer os.Remove(modulesConfigYml)

	var configParts = []string{
		mage.OSSBeatDir("_meta/common.reference.yml"),
		modulesConfigYml,
		"{{ elastic_beats_dir }}/libbeat/_meta/config.reference.yml",
	}

	for i, f := range configParts {
		configParts[i] = mage.MustExpand(f)
	}

	configFile := mage.BeatName + ".reference.yml"
	mage.MustFileConcat(configFile, 0640, configParts...)
	mage.MustFindReplace(configFile, regexp.MustCompile("beatname"), mage.BeatName)
	mage.MustFindReplace(configFile, regexp.MustCompile("beat-index-prefix"), mage.BeatIndexPrefix)
	return nil
}

func createDirModulesD() error {
	if err := os.RemoveAll("modules.d"); err != nil {
		return err
	}

	shortConfigs, err := filepath.Glob("module/*/_meta/config.yml")
	if err != nil {
		return err
	}

	for _, f := range shortConfigs {
		parts := strings.Split(filepath.ToSlash(f), "/")
		if len(parts) < 2 {
			continue
		}
		moduleName := parts[1]

		cp := mage.CopyTask{
			Source: f,
			Dest:   filepath.Join("modules.d", moduleName+".yml.disabled"),
			Mode:   0644,
		}
		if err = cp.Execute(); err != nil {
			return err
		}
	}
	return nil
}
