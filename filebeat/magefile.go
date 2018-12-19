// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// +build mage

package main

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/pkg/errors"

	"github.com/elastic/beats/dev-tools/mage"
)

func init() {
	mage.BeatDescription = "Filebeat sends log files to Logstash or directly to Elasticsearch."
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

// BuildGoDaemon builds the go-daemon binary (use crossBuildGoDaemon).
func BuildGoDaemon() error {
	return mage.BuildGoDaemon()
}

// CrossBuild cross-builds the beat for all target platforms.
func CrossBuild() error {
	return mage.CrossBuild()
}

// CrossBuildXPack cross-builds the beat with XPack for all target platforms.
func CrossBuildXPack() error {
	return mage.CrossBuildXPack()
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
// Use VERSION_QUALIFIER to control the version qualifier.
func Package() {
	start := time.Now()
	defer func() { fmt.Println("package ran for", time.Since(start)) }()

	mage.UseElasticBeatPackaging()
	customizePackaging()

	mg.Deps(Update, prepareModulePackagingOSS, prepareModulePackagingXPack)
	mg.Deps(CrossBuild, CrossBuildXPack, CrossBuildGoDaemon)
	mg.SerialDeps(mage.Package, TestPackages)
}

// TestPackages tests the generated packages (i.e. file modes, owners, groups).
func TestPackages() error {
	return mage.TestPackages(mage.WithModules(), mage.WithModulesD())
}

// Update updates the generated files (aka make update).
func Update() error {
	if err := sh.Run("make", "update"); err != nil {
		return err
	}

	// XXX (andrewkroh on 2018-10-14): This is a temporary solution for enabling
	// X-Pack modules for Filebeat. Packaging for X-Pack will be fully migrated
	// to a magefile.go in the x-pack/filebeat directory and this will be
	// removed.
	return mage.Mage("../x-pack/filebeat", "update")
}

// Fields generates a fields.yml for the Beat.
func Fields() error {
	return mage.GenerateFieldsYAML("module")
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

// ExportDashboard exports a dashboard and writes it into the correct directory
//
// Required ENV variables:
// * MODULE: Name of the module
// * ID: Dashboard id
func ExportDashboard() error {
	return mage.ExportDashboard()
}

// -----------------------------------------------------------------------------
// Customizations specific to Filebeat.
// - Include modules directory in packages (minus _meta and test files).
// - Include modules.d directory in packages.

var (
	dirModuleGeneratedOSS     = filepath.Clean("build/package/modules-oss")
	dirModuleGeneratedXPack   = filepath.Clean("build/package/modules-x-pack")
	dirModulesDGeneratedXPack = filepath.Clean("build/packaging/modules.d-x-pack")
)

func replacePackageFileSource(args mage.OSPackageArgs, replacements map[string]string) {
	missing := make(map[string]struct{})
	for key := range replacements {
		missing[key] = struct{}{}
	}
	for key, contents := range args.Spec.Files {
		oldSource := args.Spec.Files[key].Source
		if newSource, found := replacements[oldSource]; found {
			contents.Source = newSource
			args.Spec.Files[key] = contents
			delete(missing, oldSource)
		}
	}
	if len(missing) > 0 {
		asList := make([]string, 0, len(missing))
		for path := range missing {
			asList = append(asList, path)
		}
		panic(errors.Errorf("the following file sources were not found for replacement: %v", asList))
	}
}

// customizePackaging modifies the package specs to add the modules and
// modules.d directory.
func customizePackaging() {
	var (
		moduleTarget = "module"
		module       = mage.PackageFile{
			Mode:   0644,
			Source: dirModuleGeneratedOSS,
		}
		moduleXPack = mage.PackageFile{
			Mode:   0644,
			Source: dirModuleGeneratedXPack,
		}

		modulesDTarget = "modules.d"
		modulesD       = mage.PackageFile{
			Mode:    0644,
			Source:  "modules.d",
			Config:  true,
			Modules: true,
		}
		modulesDXPack = mage.PackageFile{
			Mode:    0644,
			Source:  dirModulesDGeneratedXPack,
			Config:  true,
			Modules: true,
		}
	)

	for _, args := range mage.Packages {
		mods := module
		modsD := modulesD
		pkgType := args.Types[0]
		if args.Spec.License == "Elastic License" {
			mods = moduleXPack
			modsD = modulesDXPack
			replacePackageFileSource(args, map[string]string{
				"fields.yml":                  "../x-pack/{{.BeatName}}/fields.yml",
				"{{.BeatName}}.reference.yml": "../x-pack/{{.BeatName}}/{{.BeatName}}.reference.yml",
				"_meta/kibana.generated":      "../x-pack/{{.BeatName}}/build/kibana",
			})
			if pkgType != mage.Docker {
				replacePackageFileSource(args, map[string]string{
					"{{.BeatName}}.yml": "../x-pack/{{.BeatName}}/{{.BeatName}}.yml",
				})
			}
		}

		switch pkgType {
		case mage.TarGz, mage.Zip, mage.Docker:
			args.Spec.Files[moduleTarget] = mods
			args.Spec.Files[modulesDTarget] = modsD
		case mage.Deb, mage.RPM:
			args.Spec.Files["/usr/share/{{.BeatName}}/"+moduleTarget] = mods
			args.Spec.Files["/etc/{{.BeatName}}/"+modulesDTarget] = modsD
		case mage.DMG:
			args.Spec.Files["/Library/Application Support/{{.BeatVendor}}/{{.BeatName}}"+moduleTarget] = mods
			args.Spec.Files["/etc/{{.BeatName}}/"+modulesDTarget] = modsD
		default:
			panic(errors.Errorf("unhandled package type: %v", pkgType))
		}
	}
}

// prepareModulePackagingOSS copies the module dir to the build dir and excludes
// _meta and test files so that they are not included in packages.
func prepareModulePackagingOSS() error {
	if err := sh.Rm(dirModuleGeneratedOSS); err != nil {
		return err
	}

	copy := &mage.CopyTask{
		Source:  "module",
		Dest:    dirModuleGeneratedOSS,
		Mode:    0644,
		DirMode: 0755,
		Exclude: []string{
			"/_meta",
			"/test",
			"fields.go",
		},
	}
	return copy.Execute()
}

// prepareModulePackagingXPack generates modules and modules.d directories
// for an x-pack distribution, excluding _meta and test files so that they are
// not included in packages.
func prepareModulePackagingXPack() error {
	err := mage.Clean([]string{
		dirModuleGeneratedXPack,
		dirModulesDGeneratedXPack,
	})
	if err != nil {
		return err
	}

	for _, copyAction := range []struct {
		src, dst string
	}{
		{"module", dirModuleGeneratedXPack},
		{"../x-pack/filebeat/module", dirModuleGeneratedXPack},
		{"modules.d", dirModulesDGeneratedXPack},
		{"../x-pack/filebeat/modules.d", dirModulesDGeneratedXPack},
	} {
		err := (&mage.CopyTask{
			Source:  copyAction.src,
			Dest:    copyAction.dst,
			Mode:    0644,
			DirMode: 0755,
			Exclude: []string{
				"/_meta",
				"/test",
				"fields.go",
			},
		}).Execute()
		if err != nil {
			return err
		}
	}
	return nil
}
