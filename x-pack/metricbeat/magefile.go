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
	"github.com/pkg/errors"

	devtools "github.com/elastic/beats/dev-tools/mage"
)

func init() {
	devtools.BeatDescription = "Metricbeat is a lightweight shipper for metrics."
	devtools.BeatLicense = "Elastic License"
}

// Build builds the Beat binary.
func Build() error {
	return devtools.Build(devtools.DefaultBuildArgs())
}

// GolangCrossBuild build the Beat binary inside of the golang-builder.
// Do not use directly, use crossBuild instead.
func GolangCrossBuild() error {
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

// Clean cleans all generated files and build artifacts.
func Clean() error {
	return devtools.Clean()
}

// Package packages the Beat for distribution.
// Use SNAPSHOT=true to build snapshots.
// Use PLATFORMS to control the target platforms.
// Use BEAT_VERSION_QUALIFIER to control the version qualifier.
func Package() {
	start := time.Now()
	defer func() { fmt.Println("package ran for", time.Since(start)) }()

	devtools.UseElasticBeatXPackPackaging()
	customizePackaging()
	devtools.PackageKibanaDashboardsFromBuildDir()

	mg.Deps(Update, prepareModulePackaging)
	mg.Deps(CrossBuild, CrossBuildGoDaemon)
	mg.SerialDeps(devtools.Package, TestPackages)
}

// TestPackages tests the generated packages (i.e. file modes, owners, groups).
func TestPackages() error {
	return devtools.TestPackages(devtools.WithModulesD())
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
	mg.Deps(shortConfig, referenceConfig, dockerConfig, createDirModulesD)
}

// Update is an alias for running fields, dashboards, config.
func Update() {
	mg.SerialDeps(Fields, Dashboards, Config, prepareModulePackaging,
		devtools.GenerateModuleIncludeListGo)
}

// Fmt formats source code and adds file headers.
func Fmt() {
	mg.Deps(devtools.Format)
}

// Check runs fmt and update then returns an error if any modifications are found.
func Check() {
	mg.SerialDeps(devtools.Format, Update, devtools.Check)
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

	err := devtools.Clean([]string{
		dirModulesDGenerated,
	})
	if err != nil {
		return err
	}

	for _, copyAction := range []struct {
		src, dst string
	}{
		{devtools.OSSBeatDir("modules.d"), dirModulesDGenerated},
		{"modules.d", dirModulesDGenerated},
	} {
		err := (&devtools.CopyTask{
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
		devtools.OSSBeatDir("_meta/common.yml"),
		devtools.OSSBeatDir("_meta/setup.yml"),
		"{{ elastic_beats_dir }}/libbeat/_meta/config.yml",
	}

	for i, f := range configParts {
		configParts[i] = devtools.MustExpand(f)
	}

	configFile := devtools.BeatName + ".yml"
	devtools.MustFileConcat(configFile, 0640, configParts...)
	devtools.MustFindReplace(configFile, regexp.MustCompile("beatname"), devtools.BeatName)
	devtools.MustFindReplace(configFile, regexp.MustCompile("beat-index-prefix"), devtools.BeatIndexPrefix)
	return nil
}

func referenceConfig() error {
	const modulesConfigYml = "build/config.modules.yml"
	err := devtools.GenerateModuleReferenceConfig(modulesConfigYml, devtools.OSSBeatDir("module"), "module")
	if err != nil {
		return err
	}
	defer os.Remove(modulesConfigYml)

	var configParts = []string{
		devtools.OSSBeatDir("_meta/common.reference.yml"),
		modulesConfigYml,
		"{{ elastic_beats_dir }}/libbeat/_meta/config.reference.yml",
	}

	for i, f := range configParts {
		configParts[i] = devtools.MustExpand(f)
	}

	configFile := devtools.BeatName + ".reference.yml"
	devtools.MustFileConcat(configFile, 0640, configParts...)
	devtools.MustFindReplace(configFile, regexp.MustCompile("beatname"), devtools.BeatName)
	devtools.MustFindReplace(configFile, regexp.MustCompile("beat-index-prefix"), devtools.BeatIndexPrefix)
	return nil
}

func dockerConfig() error {
	var configParts = []string{
		devtools.OSSBeatDir("_meta/beat.docker.yml"),
		devtools.LibbeatDir("_meta/config.docker.yml"),
	}

	return devtools.FileConcat(devtools.BeatName+".docker.yml", 0600, configParts...)
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

		cp := devtools.CopyTask{
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

func customizePackaging() {
	var (
		archiveModulesDir = "modules.d"
		unixModulesDir    = "/etc/{{.BeatName}}/modules.d"

		modulesDir = devtools.PackageFile{
			Mode:    0644,
			Source:  dirModulesDGenerated,
			Config:  true,
			Modules: true,
		}
		windowsModulesDir = devtools.PackageFile{
			Mode:    0644,
			Source:  "{{.PackageDir}}/modules.d",
			Config:  true,
			Modules: true,
			Dep: func(spec devtools.PackageSpec) error {
				if err := devtools.Copy(dirModulesDGenerated, spec.MustExpand("{{.PackageDir}}/modules.d")); err != nil {
					return errors.Wrap(err, "failed to copy modules.d dir")
				}

				return devtools.FindReplace(
					spec.MustExpand("{{.PackageDir}}/modules.d/system.yml"),
					regexp.MustCompile(`- load`), `#- load`)
			},
		}
		windowsReferenceConfig = devtools.PackageFile{
			Mode:   0644,
			Source: "{{.PackageDir}}/metricbeat.reference.yml",
			Dep: func(spec devtools.PackageSpec) error {
				err := devtools.Copy("metricbeat.reference.yml",
					spec.MustExpand("{{.PackageDir}}/metricbeat.reference.yml"))
				if err != nil {
					return errors.Wrap(err, "failed to copy reference config")
				}

				return devtools.FindReplace(
					spec.MustExpand("{{.PackageDir}}/metricbeat.reference.yml"),
					regexp.MustCompile(`- load`), `#- load`)
			},
		}
	)

	for _, args := range devtools.Packages {
		switch args.OS {
		case "windows":
			args.Spec.Files[archiveModulesDir] = windowsModulesDir
			args.Spec.ReplaceFile("{{.BeatName}}.reference.yml", windowsReferenceConfig)
		default:
			pkgType := args.Types[0]
			switch pkgType {
			case devtools.TarGz, devtools.Zip, devtools.Docker:
				args.Spec.Files[archiveModulesDir] = modulesDir
			case devtools.Deb, devtools.RPM, devtools.DMG:
				args.Spec.Files[unixModulesDir] = modulesDir
			default:
				panic(errors.Errorf("unhandled package type: %v", pkgType))
			}
		}
	}
}
