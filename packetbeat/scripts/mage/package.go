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

	"github.com/elastic/beats/dev-tools/mage"
	"github.com/elastic/beats/dev-tools/mage/target/build"
	"github.com/elastic/beats/dev-tools/mage/target/pkg"
)

func init() {
	mage.BeatDescription = "Packetbeat analyzes network traffic and sends the data to Elasticsearch."
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

	mg.Deps(Update.All)
	mg.Deps(build.CrossBuild, build.CrossBuildGoDaemon)
	mg.SerialDeps(mage.Package, pkg.PackageTest)
}

// customizePackaging modifies the device in the configuration files based on
// the target OS.
func customizePackaging() {
	var (
		configYml = mage.PackageFile{
			Mode:   0600,
			Source: "{{.PackageDir}}/{{.BeatName}}.yml",
			Config: true,
			Dep: func(spec mage.PackageSpec) error {
				c := configFileParams()
				c.ExtraVars["GOOS"] = spec.OS
				c.ExtraVars["GOARCH"] = spec.MustExpand("{{.GOARCH}}")
				return mage.Config(mage.ShortConfigType, c, spec.MustExpand("{{.PackageDir}}"))
			},
		}
		referenceConfigYml = mage.PackageFile{
			Mode:   0644,
			Source: "{{.PackageDir}}/{{.BeatName}}.reference.yml",
			Dep: func(spec mage.PackageSpec) error {
				c := configFileParams()
				c.ExtraVars["GOOS"] = spec.OS
				c.ExtraVars["GOARCH"] = spec.MustExpand("{{.GOARCH}}")
				return mage.Config(mage.ReferenceConfigType, c, spec.MustExpand("{{.PackageDir}}"))
			},
		}
	)

	for _, args := range mage.Packages {
		for _, pkgType := range args.Types {
			switch pkgType {
			case mage.TarGz, mage.Zip:
				args.Spec.ReplaceFile("{{.BeatName}}.yml", configYml)
				args.Spec.ReplaceFile("{{.BeatName}}.reference.yml", referenceConfigYml)
			case mage.Deb, mage.RPM, mage.DMG:
				args.Spec.ReplaceFile("/etc/{{.BeatName}}/{{.BeatName}}.yml", configYml)
				args.Spec.ReplaceFile("/etc/{{.BeatName}}/{{.BeatName}}.reference.yml", referenceConfigYml)
			case mage.Docker:
				args.Spec.ExtraVar("linux_capabilities", "cap_net_raw,cap_net_admin=eip")
			default:
				panic(errors.Errorf("unhandled package type: %v", pkgType))
			}

			// Match the first package type then continue.
			break
		}
	}
}
