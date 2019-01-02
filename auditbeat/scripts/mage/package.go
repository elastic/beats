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
	mage.BeatDescription = "Audit the activities of users and processes on your system."
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

	mg.SerialDeps(Update.All)
	mg.Deps(build.CrossBuild, build.CrossBuildGoDaemon)
	mg.SerialDeps(mage.Package, pkg.PackageTest)
}

// customizePackaging modifies the package specs to use templated config files
// instead of the defaults.
//
// Customizations specific to Auditbeat:
// - Include audit.rules.d directory in packages.
// - Generate OS specific config files.
func customizePackaging() {
	var (
		shortConfig = mage.PackageFile{
			Mode:   0600,
			Source: "{{.PackageDir}}/auditbeat.yml",
			Dep: func(spec mage.PackageSpec) error {
				return generateConfig(mage.ShortConfigType, spec)
			},
			Config: true,
		}
		referenceConfig = mage.PackageFile{
			Mode:   0644,
			Source: "{{.PackageDir}}/auditbeat.reference.yml",
			Dep: func(spec mage.PackageSpec) error {
				return generateConfig(mage.ReferenceConfigType, spec)
			},
		}
	)

	const (
		sampleRulesSource        = "{{.PackageDir}}/audit.rules.d/sample-rules.conf.disabled"
		defaultSampleRulesTarget = "audit.rules.d/sample-rules.conf.disabled"
	)
	sampleRules := mage.PackageFile{
		Mode:   0644,
		Source: sampleRulesSource,
		Dep: func(spec mage.PackageSpec) error {
			if spec.OS != "linux" {
				return errors.New("audit rules are for linux only")
			}

			// Origin rule file.
			params := map[string]interface{}{"ArchBits": archBits}
			origin := mage.OSSBeatDir(
				"module/auditd/_meta/audit.rules.d",
				spec.MustExpand("sample-rules-linux-{{call .ArchBits .GOARCH}}bit.conf", params),
			)

			if err := mage.Copy(origin, spec.MustExpand(sampleRulesSource)); err != nil {
				return errors.Wrap(err, "failed to copy sample rules")
			}
			return nil
		},
	}

	for _, args := range mage.Packages {
		for _, pkgType := range args.Types {
			sampleRulesTarget := defaultSampleRulesTarget

			switch pkgType {
			case mage.TarGz, mage.Zip:
				args.Spec.ReplaceFile("{{.BeatName}}.yml", shortConfig)
				args.Spec.ReplaceFile("{{.BeatName}}.reference.yml", referenceConfig)
			case mage.Deb, mage.RPM, mage.DMG:
				args.Spec.ReplaceFile("/etc/{{.BeatName}}/{{.BeatName}}.yml", shortConfig)
				args.Spec.ReplaceFile("/etc/{{.BeatName}}/{{.BeatName}}.reference.yml", referenceConfig)
				sampleRulesTarget = "/etc/{{.BeatName}}/" + defaultSampleRulesTarget
			case mage.Docker:
				args.Spec.ExtraVar("user", "root")
			default:
				panic(errors.Errorf("unhandled package type: %v", pkgType))
			}

			if args.OS == "linux" {
				args.Spec.Files[sampleRulesTarget] = sampleRules
			}
			break
		}
	}
}

func generateConfig(ct mage.ConfigFileType, spec mage.PackageSpec) error {
	args, err := configFileParams()
	if err != nil {
		return err
	}

	// PackageDir isn't exported but we can grab it's value this way.
	packageDir := spec.MustExpand("{{.PackageDir}}")
	args.ExtraVars["GOOS"] = spec.OS
	args.ExtraVars["GOARCH"] = spec.MustExpand("{{.GOARCH}}")
	return mage.Config(ct, args, packageDir)
}
