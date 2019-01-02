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
	"time"

	"github.com/magefile/mage/mg"
	"github.com/pkg/errors"

	"github.com/elastic/beats/dev-tools/mage/target/build"
	"github.com/elastic/beats/dev-tools/mage/target/pkg"

	"github.com/elastic/beats/dev-tools/mage"
)

const (
	dirModuleGenerated   = "build/package/module"
	dirModulesDGenerated = "build/package/modules.d"
)

func init() {
	mage.BeatDescription = "Filebeat sends log files to Logstash or directly to Elasticsearch."
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
	}
	mage.PackageKibanaDashboardsFromBuildDir()
	customizePackaging()

	mg.Deps(Update.All, prepareModulePackaging)
	mg.Deps(build.CrossBuild, build.CrossBuildGoDaemon)
	mg.SerialDeps(mage.Package, pkg.PackageTest)
}

// customizePackaging modifies the package specs to add the modules and
// modules.d directory. You must declare a dependency on either
// PrepareModulePackagingOSS or PrepareModulePackagingXPack.
func customizePackaging() {
	var (
		moduleTarget = "module"
		module       = mage.PackageFile{
			Mode:   0644,
			Source: dirModuleGenerated,
		}

		modulesDTarget = "modules.d"
		modulesD       = mage.PackageFile{
			Mode:    0644,
			Source:  dirModulesDGenerated,
			Config:  true,
			Modules: true,
		}
	)

	for _, args := range mage.Packages {
		for _, pkgType := range args.Types {
			switch pkgType {
			case mage.TarGz, mage.Zip, mage.Docker:
				args.Spec.Files[moduleTarget] = module
				args.Spec.Files[modulesDTarget] = modulesD
			case mage.Deb, mage.RPM:
				args.Spec.Files["/usr/share/{{.BeatName}}/"+moduleTarget] = module
				args.Spec.Files["/etc/{{.BeatName}}/"+modulesDTarget] = modulesD
			case mage.DMG:
				args.Spec.Files["/Library/Application Support/{{.BeatVendor}}/{{.BeatName}}/"+moduleTarget] = module
				args.Spec.Files["/etc/{{.BeatName}}/"+modulesDTarget] = modulesD
			default:
				panic(errors.Errorf("unhandled package type: %v", pkgType))
			}
			break
		}
	}
}

// prepareModulePackaging generates build/package/modules and
// build/package/modules.d directories for use in packaging.
func prepareModulePackaging() error {
	switch SelectLogic {
	case mage.OSSProject:
		return _prepareModulePackaging([]struct{ Src, Dst string }{
			{mage.OSSBeatDir("module"), dirModuleGenerated},
			{mage.OSSBeatDir("modules.d"), dirModulesDGenerated},
		}...)
	case mage.XPackProject:
		return _prepareModulePackaging([]struct{ Src, Dst string }{
			{mage.OSSBeatDir("module"), dirModuleGenerated},
			{"module", dirModuleGenerated},
			{mage.OSSBeatDir("modules.d"), dirModulesDGenerated},
			{"modules.d", dirModulesDGenerated},
		}...)
	default:
		panic(mage.ErrUnknownProjectType)
	}
}

// _prepareModulePackaging generates build/package/modules and
// build/package/modules.d directories for use in packaging.
func _prepareModulePackaging(files ...struct{ Src, Dst string }) error {
	// This depends on the modules.d directory being up-to-date.
	mg.Deps(Update.ModulesD)

	// Clean any existing generated directories.
	if err := mage.Clean([]string{dirModuleGenerated, dirModulesDGenerated}); err != nil {
		return err
	}

	for _, copyAction := range files {
		err := (&mage.CopyTask{
			Source:  copyAction.Src,
			Dest:    copyAction.Dst,
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
