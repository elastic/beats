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
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	devtools "github.com/elastic/beats/v8/dev-tools/mage"
)

// CustomizePackaging modifies the device in the configuration files based on
// the target OS.
func CustomizePackaging() {
	var (
		configYml = devtools.PackageFile{
			Mode:   0o600,
			Source: "{{.PackageDir}}/{{.BeatName}}.yml",
			Config: true,
			Dep: func(spec devtools.PackageSpec) error {
				c := ConfigFileParams()
				c.ExtraVars["GOOS"] = spec.OS
				c.ExtraVars["GOARCH"] = spec.MustExpand("{{.GOARCH}}")
				return devtools.Config(devtools.ShortConfigType, c, spec.MustExpand("{{.PackageDir}}"))
			},
		}
		referenceConfigYml = devtools.PackageFile{
			Mode:   0o644,
			Source: "{{.PackageDir}}/{{.BeatName}}.reference.yml",
			Dep: func(spec devtools.PackageSpec) error {
				c := ConfigFileParams()
				c.ExtraVars["GOOS"] = spec.OS
				c.ExtraVars["GOARCH"] = spec.MustExpand("{{.GOARCH}}")
				return devtools.Config(devtools.ReferenceConfigType, c, spec.MustExpand("{{.PackageDir}}"))
			},
		}
		npcapNoticeTxt = devtools.PackageFile{
			Mode:   0o644,
			Source: "{{.PackageDir}}/NOTICE.txt",
			Dep: func(spec devtools.PackageSpec) error {
				repo, err := devtools.GetProjectRepoInfo()
				if err != nil {
					return err
				}

				notice, err := os.ReadFile(filepath.Join(repo.RootDir, "NOTICE.txt"))
				if err != nil {
					return err
				}

				if spec.OS == "windows" && spec.License == "Elastic License" {
					license, err := os.ReadFile(devtools.XPackBeatDir("npcap/installer/LICENSE"))
					if err != nil {
						return err
					}
					notice = append(notice, license...)
				}

				return os.WriteFile(devtools.CreateDir(spec.MustExpand("{{.PackageDir}}/NOTICE.txt")), notice, 0o644)
			},
		}
	)

	for _, args := range devtools.Packages {
		if len(args.Types) == 0 {
			continue
		}
		switch pkgType := args.Types[0]; pkgType {
		case devtools.TarGz, devtools.Zip:
			args.Spec.ReplaceFile("{{.BeatName}}.yml", configYml)
			args.Spec.ReplaceFile("{{.BeatName}}.reference.yml", referenceConfigYml)
			args.Spec.ReplaceFile("NOTICE.txt", npcapNoticeTxt)
		case devtools.Deb, devtools.RPM:
			args.Spec.ReplaceFile("/etc/{{.BeatName}}/{{.BeatName}}.yml", configYml)
			args.Spec.ReplaceFile("/etc/{{.BeatName}}/{{.BeatName}}.reference.yml", referenceConfigYml)
		case devtools.Docker:
			args.Spec.ExtraVar("linux_capabilities", "cap_net_raw,cap_net_admin+eip")
		default:
			panic(errors.Errorf("unhandled package type: %v", pkgType))
		}
	}
}
