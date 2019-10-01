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
	"time"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/pkg/errors"

	devtools "github.com/elastic/beats/dev-tools/mage"
	metricbeat "github.com/elastic/beats/metricbeat/scripts/mage"

	// mage:import
	"github.com/elastic/beats/dev-tools/mage/target/common"
)

const (
	dirModulesGenerated = "build/package/module"
)

func init() {
	common.RegisterCheckDeps(Update)

	devtools.BeatDescription = "Metricbeat is a lightweight shipper for metrics."
	devtools.BeatLicense = "Elastic License"
}

// Aliases provides compatibility with CI while we transition all Beats
// to having common testing targets.
var Aliases = map[string]interface{}{
	"goTestUnit": GoUnitTest, // dev-tools/jenkins_ci.ps1 uses this.
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
	params := devtools.DefaultGolangCrossBuildArgs()
	params.Env = map[string]string{
		//"CGO_LDFLAGS": "-Wl,-rpath.*",
		"CGO_CFLAGS": "-I/opt/mqm/inc/",
		"CGO_LDFLAGS_ALLOW": "-Wl,-rpath.*",
	}
	return devtools.GolangCrossBuild(params)
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
// Use BEAT_VERSION_QUALIFIER to control the version qualifier.
func Package() {
	start := time.Now()
	defer func() { fmt.Println("package ran for", time.Since(start)) }()

	devtools.UseElasticBeatXPackPackaging()
	metricbeat.CustomizePackaging()
	devtools.PackageKibanaDashboardsFromBuildDir()
	packageLightModules()

	mg.Deps(Update, metricbeat.PrepareModulePackagingXPack)
	mg.Deps(CrossBuild, CrossBuildGoDaemon)
	mg.SerialDeps(devtools.Package, TestPackages)
}

// TestPackages tests the generated packages (i.e. file modes, owners, groups).
func TestPackages() error {
	return devtools.TestPackages(
		devtools.WithModulesD(),
		devtools.WithModules(),

		// To be increased or removed when more light modules are added
		devtools.MinModules(3))
}

// Fields generates a fields.yml and fields.go for each module.
func Fields() {
	mg.Deps(fieldsYML, moduleFieldsGo)
}

func moduleFieldsGo() error {
	return devtools.GenerateModuleFieldsGo("module")
}

// fieldsYML generates a fields.yml based on filebeat + x-pack/filebeat/modules.
func fieldsYML() error {
	return devtools.GenerateFieldsYAML(devtools.OSSBeatDir("module"), "module")
}

// Dashboards collects all the dashboards and generates index patterns.
func Dashboards() error {
	return devtools.KibanaDashboards(devtools.OSSBeatDir("module"), "module")
}

// Config generates both the short and reference configs.
func Config() {
	mg.Deps(configYML, devtools.GenerateDirModulesD)
}

func configYML() error {
	return devtools.Config(devtools.AllConfigTypes, metricbeat.XPackConfigFileParams(), ".")
}

// Update is an alias for running fields, dashboards, config.
func Update() {
	mg.SerialDeps(Fields, Dashboards, Config,
		metricbeat.PrepareModulePackagingXPack,
		devtools.GenerateModuleIncludeListGo)
}

// IntegTest executes integration tests (it uses Docker to run the tests).
func IntegTest() {
	devtools.AddIntegTestUsage()
	defer devtools.StopIntegTestEnv()
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
	return devtools.GoTest(ctx, devtools.DefaultGoTestUnitArgs())
}

// GoIntegTest executes the Go integration tests.
// Use TEST_COVERAGE=true to enable code coverage profiling.
// Use RACE_DETECTOR=true to enable the race detector.
func GoIntegTest(ctx context.Context) error {
	return devtools.RunIntegTest("goIntegTest", func() error {
		return devtools.GoTest(ctx, devtools.DefaultGoTestIntegrationArgs())
	})
}

// PythonUnitTest executes the python system tests.
func PythonUnitTest() error {
	mg.Deps(devtools.BuildSystemTestBinary)
	return devtools.PythonNoseTest(devtools.DefaultPythonTestUnitArgs())
}

// PythonIntegTest executes the python system tests in the integration environment (Docker).
func PythonIntegTest(ctx context.Context) error {
	if !devtools.IsInIntegTestEnv() {
		mg.Deps(Fields)
	}
	return devtools.RunIntegTest("pythonIntegTest", func() error {
		mg.Deps(devtools.BuildSystemTestBinary)
		return devtools.PythonNoseTest(devtools.DefaultPythonTestIntegrationArgs())
	})
}

// prepareLightModules generates light modules
func prepareLightModules(path string) error {
	err := devtools.Clean([]string{dirModulesGenerated})
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dirModulesGenerated, 0755); err != nil {
		return err
	}

	filePatterns := []string{
		"*/module.yml",
		"*/*/manifest.yml",
	}

	var files []string
	for _, pattern := range filePatterns {
		matches, err := filepath.Glob(filepath.Join(path, pattern))
		if err != nil {
			return err
		}
		files = append(files, matches...)
	}

	if len(files) == 0 {
		return fmt.Errorf("no light modules found")
	}

	for _, file := range files {
		rel, _ := filepath.Rel(path, file)
		dest := filepath.Join(dirModulesGenerated, rel)
		err := (&devtools.CopyTask{
			Source:  file,
			Dest:    dest,
			Mode:    0644,
			DirMode: 0755,
		}).Execute()
		if err != nil {
			return err
		}
	}
	return nil
}

// packageLightModules customizes packaging to add light modules
func packageLightModules() error {
	prepareLightModules("module")

	var (
		moduleTarget = "module"
		module       = devtools.PackageFile{
			Mode:   0644,
			Source: dirModulesGenerated,
		}
	)

	for _, args := range devtools.Packages {
		pkgType := args.Types[0]
		switch pkgType {
		case devtools.TarGz, devtools.Zip, devtools.Docker:
			args.Spec.Files[moduleTarget] = module
		case devtools.Deb, devtools.RPM:
			args.Spec.Files["/usr/share/{{.BeatName}}/"+moduleTarget] = module
		case devtools.DMG:
			args.Spec.Files["/Library/Application Support/{{.BeatVendor}}/{{.BeatName}}/"+moduleTarget] = module
		default:
			return fmt.Errorf("unhandled package type: %v", pkgType)
		}
	}
	return nil
}

// -----------------------------------------------------------------------------
// - Install the IBM redistributable client package
var (
	deps = map[string]func() error{
		"linux/amd64": installIBMCLinuxAMD64,
	}
)

func installIBMCLinuxAMD64() error {
	URL := "https://public.dhe.ibm.com/ibmdl/export/pub/software/websphere/messaging/mqdev/redist"
	RDTAR := "IBM-MQC-Redist-LinuxX64.tar.gz"
	VRMF := "9.1.3.0"

	// Create the directory for IBM Client
	err := sh.RunV("mkdir", "/opt/mqm")
	if err != nil {
		return errors.Wrap(err, "error while adding IBM C client")
	}

	pkgURL := fmt.Sprintf("%v/%v-%v", URL, VRMF, RDTAR)
	name := fmt.Sprintf("/opt/mqm/%v-%v", VRMF, RDTAR)
	// curl to get the IBM tar with the IBM C Client
	err = sh.RunV("curl", "-o", name, "-LO", pkgURL)
	if err != nil {
		return errors.Wrap(err, "error while adding IBM C client")
	}
	// Untar the IBM client
	err = sh.RunV("tar", "-zxvf", name, "-C", "/opt/mqm/")
	if err != nil {
		return errors.Wrap(err, "error while adding IBM C client")
	}
	return nil
}
