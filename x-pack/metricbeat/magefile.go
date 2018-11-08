// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build mage

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/generator/fields"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"

	"github.com/elastic/beats/dev-tools/mage"
)

func init() {
	mage.BeatDescription = "Metricbeat is a lightweight shipper for metrics."
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

// CrossBuildXPack cross-builds the beat with XPack for all target platforms.
func CrossBuildXPack() error {
	return mage.CrossBuildXPack()
}

// BuildGoDaemon builds the go-daemon binary (use crossBuildGoDaemon).
func BuildGoDaemon() error {
	return mage.BuildGoDaemon()
}

// CrossBuild cross-builds the beat for all target platforms.
func CrossBuild() error {
	return mage.CrossBuild()
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

	mage.UseElasticBeatPackaging()
	customizePackaging()

	mg.Deps(Update)
	mg.Deps(CrossBuild, CrossBuildXPack, CrossBuildGoDaemon)
	mg.SerialDeps(mage.Package, TestPackages)
}

// TestPackages tests the generated packages (i.e. file modes, owners, groups).
func TestPackages() error {
	return mage.TestPackages(mage.WithModulesD())
}

// Update updates the generated files (aka make update).
func Update() error {
	return sh.Run("make", "update")
}

// GoTestUnit executes the Go unit tests.
// Use TEST_COVERAGE=true to enable code coverage profiling.
// Use RACE_DETECTOR=true to enable the race detector.
func GoTestUnit(ctx context.Context) error {
	return mage.GoTest(ctx, mage.DefaultGoTestUnitArgs())
}

// GoTestIntegration executes the Go integration tests.
// Use TEST_COVERAGE=true to enable code coverage profiling.
// Use RACE_DETECTOR=true to enable the race detector.
func GoTestIntegration(ctx context.Context) error {
	return mage.GoTest(ctx, mage.DefaultGoTestIntegrationArgs())
}

// Fields generates a fields.yml for the Beat.
// $OUTPUT_FILE_PATH: Specify `fields.yml` output file path. Default: fields.yml
// $EXTRA_FIELDS_FOLDERS: Comma separated list of `/module` paths that must be included. Default: ../../metricbeat/module,module
func Fields() (err error) {
	conf := fieldsConf{}

	if conf.beatPath, err = os.Getwd(); err != nil {
		return errors.Wrap(err, "Error trying to get current working directory. Aborting")
	}

	if conf.beatsRootPath, err = mage.ElasticBeatsDir(); err != nil {
		return err
	}

	if conf.outputFilePath = os.Getenv("OUTPUT_FILE_PATH"); conf.outputFilePath == "" {
		conf.outputFilePath = "fields.yml"
	}

	if foldersEnv := os.Getenv("EXTRA_FIELDS_FOLDERS"); foldersEnv != "" {
		folders := strings.Split(foldersEnv, ",")
		conf.extraBeatsModulesPaths = folders
	} else {
		conf.extraBeatsModulesPaths = []string{"module", conf.beatsRootPath + "/metricbeat/module"}
	}

	return generateFields(conf)
}

// -----------------------------------------------------------------------------
// Customizations specific to Metricbeat.
// - Include modules.d directory in packages.
// - Disable system/load metricset for Windows.

// customizePackaging modifies the package specs to add the modules.d directory.
// And for Windows it comments out the system/load metricset because it's
// not supported.
func customizePackaging() {
	var (
		archiveModulesDir = "modules.d"
		unixModulesDir    = "/etc/{{.BeatName}}/modules.d"

		modulesDir = mage.PackageFile{
			Mode:   0644,
			Source: "modules.d",
			Config: true,
		}
		windowsModulesDir = mage.PackageFile{
			Mode:   0644,
			Source: "{{.PackageDir}}/modules.d",
			Config: true,
			Dep: func(spec mage.PackageSpec) error {
				if err := mage.Copy("modules.d", spec.MustExpand("{{.PackageDir}}/modules.d")); err != nil {
					return errors.Wrap(err, "failed to copy modules.d dir")
				}

				return mage.FindReplace(
					spec.MustExpand("{{.PackageDir}}/modules.d/system.yml"),
					regexp.MustCompile(`- load`), `#- load`)
			},
		}
		windowsReferenceConfig = mage.PackageFile{
			Mode:   0644,
			Source: "{{.PackageDir}}/metricbeat.reference.yml",
			Dep: func(spec mage.PackageSpec) error {
				err := mage.Copy("metricbeat.reference.yml",
					spec.MustExpand("{{.PackageDir}}/metricbeat.reference.yml"))
				if err != nil {
					return errors.Wrap(err, "failed to copy reference config")
				}

				return mage.FindReplace(
					spec.MustExpand("{{.PackageDir}}/metricbeat.reference.yml"),
					regexp.MustCompile(`- load`), `#- load`)
			},
		}
	)

	for _, args := range mage.Packages {
		switch args.OS {
		case "windows":
			args.Spec.Files[archiveModulesDir] = windowsModulesDir
			args.Spec.ReplaceFile("{{.BeatName}}.reference.yml", windowsReferenceConfig)
		default:
			pkgType := args.Types[0]
			switch pkgType {
			case mage.TarGz, mage.Zip:
				args.Spec.Files[archiveModulesDir] = modulesDir
			case mage.Deb, mage.RPM, mage.DMG:
				args.Spec.Files[unixModulesDir] = modulesDir
			default:
				panic(errors.Errorf("unhandled package type: %v", pkgType))
			}
		}
	}
}

type fieldsConf struct {
	// beatsRootPath is the root of the beats folder, usually in github.com/elastic/beats
	beatsRootPath string

	// beatPath is the path of the beat. In case of metricbeat it is in github.com/elastic/beats/metricbeat
	beatPath string

	// extraBeatsModulesPaths a list of beats paths that must be included
	extraBeatsModulesPaths []string

	outputFilePath string
}

func generateFields(c fieldsConf) error {
	name := filepath.Base(c.beatPath)

	err := validFieldFilesFound(c)
	if err != nil {
		return err
	}

	var fieldsFiles []*fields.YmlFile

	for _, fieldsFilePath := range c.extraBeatsModulesPaths {
		fieldsFile, err := fields.CollectModuleFiles(fieldsFilePath)
		if err != nil {
			return errors.Wrap(err, "Cannot collect fields.yml files")
		}

		fieldsFiles = append(fieldsFiles, fieldsFile...)
	}

	err = fields.Generate(c.beatsRootPath, c.beatPath, fieldsFiles, c.outputFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot generate global fields.yml file for %s: %+v\n", name, err)
		os.Exit(3)
	}

	outputPath, _ := filepath.Abs(c.outputFilePath)
	if err != nil {
		outputPath = c.outputFilePath
	}
	fmt.Fprintf(os.Stderr, "Generated fields.yml for %s to %s\n", name, outputPath)

	return nil
}

func validFieldFilesFound(c fieldsConf) error {
	if c.beatPath == "" {
		return errors.New("Beat path cannot be empty")
	}

	beatsRootPathInfo, err := getFileInfo(c.beatsRootPath)
	if err != nil {
		return errors.Wrap(err, "Error getting file info of elastic/beats")
	}

	beatPathInfo, err := getFileInfo(c.beatPath)
	if err != nil {
		return errors.Wrap(err, "Error getting file info of target Beat")
	}

	// If a community Beat does not have its own fields.yml file, it still requires
	// the fields coming from libbeat to generate e.g assets. In case of Elastic Beats,
	// it's not a problem because all of them has unique fields.yml files somewhere.
	if len(c.extraBeatsModulesPaths) == 0 && os.SameFile(beatsRootPathInfo, beatPathInfo) {
		if c.outputFilePath != "-" {
			return errors.New("No field files to collect")
		}
	}

	return nil
}

func closeFile(f *os.File) {
	if err := f.Close(); err != nil {
		log.Fatalf("Error trying to close file '%s'. %v", f.Name(), err)
	}
}

func getFileInfo(p string) (i os.FileInfo, err error) {
	var f *os.File
	if f, err = os.Open(p); err != nil {
		return nil, errors.Wrap(err, "Error opening path")
	}
	defer closeFile(f)

	if i, err = f.Stat(); err != nil {
		return nil, errors.Wrap(err, "Error getting file info")
	}

	return
}
