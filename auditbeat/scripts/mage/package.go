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
	"github.com/pkg/errors"

	"github.com/elastic/beats/dev-tools/mage"
)

// CustomizePackaging modifies the package specs to use templated config files
// instead of the defaults.
//
// Customizations specific to Auditbeat:
// - Include audit.rules.d directory in packages.
func CustomizePackaging() {
	var (
		shortConfig = mage.PackageFile{
			Mode:   0600,
			Source: "{{.PackageDir}}/auditbeat.yml",
			Dep:    generateShortConfig,
			Config: true,
		}
		referenceConfig = mage.PackageFile{
			Mode:   0644,
			Source: "{{.PackageDir}}/auditbeat.reference.yml",
			Dep:    generateReferenceConfig,
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

func generateReferenceConfig(spec mage.PackageSpec) error {
	params := map[string]interface{}{
		"Reference": true,
		"ArchBits":  archBits,
	}
	return spec.ExpandFile(referenceConfigTemplate,
		"{{.PackageDir}}/auditbeat.reference.yml", params)
}

func generateShortConfig(spec mage.PackageSpec) error {
	params := map[string]interface{}{
		"Reference": false,
		"ArchBits":  archBits,
	}
	return spec.ExpandFile(shortConfigTemplate,
		"{{.PackageDir}}/auditbeat.yml", params)
}
