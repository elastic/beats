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
	"path/filepath"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/pkg/errors"

	"github.com/elastic/beats/dev-tools/mage"
	"github.com/elastic/beats/dev-tools/mage/target/build"
	"github.com/elastic/beats/dev-tools/mage/target/pkg"
)

func init() {
	mage.BeatDescription = "Metricbeat is a lightweight shipper for metrics."
}

// Package packages the Beat for distribution.
// Use SNAPSHOT=true to build snapshots.
// Use PLATFORMS to control the target platforms.
// Use VERSION_QUALIFIER to control the version qualifier.
func Package() {
	start := time.Now()
	defer func() { fmt.Println("package ran for", time.Since(start)) }()

	switch mage.BeatProjectType {
	case mage.OSSProject:
		mage.UseElasticBeatOSSPackaging()
	case mage.XPackProject:
		mage.UseElasticBeatXPackPackaging()
	case mage.CommunityProject:
		mage.UseCommunityBeatPackaging()
	}
	mage.PackageKibanaDashboardsFromBuildDir()
	customizePackaging()

	mg.Deps(Update.All)
	mg.Deps(build.CrossBuild, build.CrossBuildGoDaemon)
	mg.SerialDeps(mage.Package, pkg.PackageTest)
}

// -----------------------------------------------------------------------------
// Customizations specific to Metricbeat.
// - Include modules.d directory in packages.
// - Disable system/load metricset for Windows.

// customizePackaging modifies the package specs to add the modules.d directory.
// And for Windows it comments out the system/load metricset because it's
// not supported.
func customizePackaging() {
	const shortConfigGlob = "module/*/_meta/config.yml"
	inputGlobs := []string{mage.OSSBeatDir(shortConfigGlob)}
	if mage.BeatProjectType == mage.XPackProject {
		inputGlobs = append(inputGlobs, mage.XPackBeatDir(shortConfigGlob))
	}

	var (
		modulesDTarget = "modules.d"
		modulesD       = mage.PackageFile{
			Mode:    0644,
			Source:  "{{.PackageDir}}/modules.d",
			Config:  true,
			Modules: true,
			Dep: func(spec mage.PackageSpec) error {
				packageDir := spec.MustExpand("{{.PackageDir}}")
				targetDir := filepath.Join(packageDir, "modules.d")
				return mage.GenerateDirModulesD(
					mage.InputGlobs(inputGlobs...),
					mage.OutputDir(targetDir),
					mage.SetTemplateVariable("GOOS", spec.OS),
					mage.SetTemplateVariable("GOARCH", mage.MustExpand("{{.GOARCH}}")),
					mage.EnableModule("system"),
				)
			},
		}
	)

	for _, args := range mage.Packages {
		for _, pkgType := range args.Types {
			switch pkgType {
			case mage.TarGz, mage.Zip, mage.Docker:
				args.Spec.Files[modulesDTarget] = modulesD
			case mage.Deb, mage.RPM:
				args.Spec.Files["/etc/{{.BeatName}}/"+modulesDTarget] = modulesD
			case mage.DMG:
				args.Spec.Files["/etc/{{.BeatName}}/"+modulesDTarget] = modulesD
			default:
				panic(errors.Errorf("unhandled package type: %v", pkgType))
			}

			break
		}
	}
}
