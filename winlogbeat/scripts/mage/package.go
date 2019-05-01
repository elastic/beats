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

package mage

import (
	"fmt"
	"os"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/pkg/errors"

	"github.com/elastic/beats/dev-tools/mage"
	"github.com/elastic/beats/dev-tools/mage/target/build"
	"github.com/elastic/beats/dev-tools/mage/target/pkg"
)

const (
	dirModuleGenerated = "build/package/module"
)

func init() {
	mage.BeatDescription = "Winlogbeat ships Windows event logs to Elasticsearch or Logstash."

	mage.Platforms = mage.Platforms.Filter("windows")
}

// Package packages the Beat for distribution.
// Use SNAPSHOT=true to build snapshots.
// Use PLATFORMS to control the target platforms.
// Use VERSION_QUALIFIER to control the version qualifier.
func Package() {
	start := time.Now()
	defer func() { fmt.Println("package ran for", time.Since(start)) }()

	switch SelectLogic {
	case mage.OSSProject:
		mage.UseElasticBeatOSSPackaging()
	case mage.XPackProject:
		mage.UseElasticBeatXPackPackaging()
		customizePackaging()
	}
	mage.PackageKibanaDashboardsFromBuildDir()

	mg.Deps(Update.All)
	mg.Deps(build.CrossBuild, build.CrossBuildGoDaemon)
	mg.SerialDeps(mage.Package, pkg.PackageTest)
}

func customizePackaging() {
	// Skip if the module dir does not exist.
	// TODO: Remove this after the module dir is added.
	if _, err := os.Stat(mage.XPackBeatDir("module")); err != nil {
		return
	}

	mg.Deps(prepareModulePackaging)

	moduleDir := mage.PackageFile{
		Mode:    0644,
		Source:  dirModuleGenerated,
		Config:  true,
		Modules: true,
	}

	for _, args := range mage.Packages {
		for _, pkgType := range args.Types {
			switch pkgType {
			case mage.TarGz, mage.Zip, mage.Docker:
				args.Spec.Files["module"] = moduleDir
			case mage.Deb, mage.RPM, mage.DMG:
				args.Spec.Files["/etc/{{.BeatName}}/module"] = moduleDir
			default:
				panic(errors.Errorf("unhandled package type: %v", pkgType))
			}
		}
	}
}

// prepareModulePackaging generates build/package/module.
func prepareModulePackaging() error {
	// Clean any existing generated directories.
	if err := mage.Clean([]string{dirModuleGenerated}); err != nil {
		return err
	}

	return (&mage.CopyTask{
		Source:  mage.XPackBeatDir("module"),
		Dest:    dirModuleGenerated,
		Mode:    0644,
		DirMode: 0755,
		Exclude: []string{
			"/_meta",
			"/test",
			`\.go$`,
		},
	}).Execute()
}
