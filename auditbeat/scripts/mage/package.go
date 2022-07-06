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
	"errors"
	"fmt"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
)

// PackagingFlavor specifies the type of packaging (OSS vs X-Pack).
type PackagingFlavor uint8

// Packaging flavors.
const (
	OSSPackaging PackagingFlavor = iota
	XPackPackaging
)

// CustomizePackaging modifies the package specs to use templated config files
// instead of the defaults.
//
// Customizations specific to Auditbeat:
// - Include audit.rules.d directory in packages.
// - Add elastic-agent specific config to x-pack tar.gz package.
func CustomizePackaging(pkgFlavor PackagingFlavor) {
	var (
		shortConfig = devtools.PackageFile{
			Mode:   0o600,
			Source: "{{.PackageDir}}/auditbeat.yml",
			Dep: func(spec devtools.PackageSpec) error {
				return generateConfig(pkgFlavor, devtools.ShortConfigType, spec)
			},
			Config: true,
		}
		referenceConfig = devtools.PackageFile{
			Mode:   0o644,
			Source: "{{.PackageDir}}/auditbeat.reference.yml",
			Dep: func(spec devtools.PackageSpec) error {
				return generateConfig(pkgFlavor, devtools.ReferenceConfigType, spec)
			},
		}
	)

	const (
		sampleRulesSource        = "{{.PackageDir}}/audit.rules.d/sample-rules.conf.disabled"
		defaultSampleRulesTarget = "audit.rules.d/sample-rules.conf.disabled"
	)
	sampleRules := devtools.PackageFile{
		Mode:   0o644,
		Source: sampleRulesSource,
		Dep: func(spec devtools.PackageSpec) error {
			if spec.OS != "linux" {
				return errors.New("audit rules are for linux only")
			}

			// Origin rule file.
			params := map[string]interface{}{"ArchBits": archBits}
			origin := devtools.OSSBeatDir(
				"module/auditd/_meta/audit.rules.d",
				spec.MustExpand("sample-rules-linux-{{call .ArchBits .GOARCH}}bit.conf", params),
			)

			if err := devtools.Copy(origin, spec.MustExpand(sampleRulesSource)); err != nil {
				return fmt.Errorf("failed to copy sample rules: %w", err)
			}
			return nil
		},
	}

	for _, args := range devtools.Packages {
		if len(args.Types) == 0 {
			continue
		}

		sampleRulesTarget := defaultSampleRulesTarget

		switch pkgType := args.Types[0]; pkgType {
		case devtools.TarGz, devtools.Zip:
			args.Spec.ReplaceFile("{{.BeatName}}.yml", shortConfig)
			args.Spec.ReplaceFile("{{.BeatName}}.reference.yml", referenceConfig)

			// Add an Elastic Agent specific config to the Elastic licensed packages.
			if XPackPackaging == pkgFlavor {
				args.Spec.Files["{{.BeatName}}.elastic-agent.yml"] = devtools.PackageFile{
					Mode:   0o644,
					Source: "auditbeat.elastic-agent.yml",
				}
			}
		case devtools.Deb, devtools.RPM:
			args.Spec.ReplaceFile("/etc/{{.BeatName}}/{{.BeatName}}.yml", shortConfig)
			args.Spec.ReplaceFile("/etc/{{.BeatName}}/{{.BeatName}}.reference.yml", referenceConfig)
			sampleRulesTarget = "/etc/{{.BeatName}}/" + defaultSampleRulesTarget
		case devtools.Docker:
		default:
			panic(fmt.Errorf("unhandled package type: %v", pkgType))
		}

		if args.OS == "linux" {
			args.Spec.Files[sampleRulesTarget] = sampleRules
		}
	}
}

func generateConfig(pkgFlavor PackagingFlavor, ct devtools.ConfigFileType, spec devtools.PackageSpec) error {
	var args devtools.ConfigFileParams
	switch pkgFlavor {
	case OSSPackaging:
		args = OSSConfigFileParams()
	case XPackPackaging:
		args = XPackConfigFileParams()
	default:
		panic(fmt.Errorf("invalid packaging flavor (either oss or xpack): %v", pkgFlavor))
	}

	// PackageDir isn't exported but we can grab it's value this way.
	packageDir := spec.MustExpand("{{.PackageDir}}")
	args.ExtraVars["GOOS"] = spec.OS
	args.ExtraVars["GOARCH"] = spec.MustExpand("{{.GOARCH}}")
	return devtools.Config(ct, args, packageDir)
}
