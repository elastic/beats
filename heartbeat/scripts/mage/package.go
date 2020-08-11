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

import devtools "github.com/elastic/beats/v7/dev-tools/mage"

const (
	dirModuleGenerated   = "build/package/module"
	dirModulesDGenerated = "build/package/modules.d"
)

// CustomizePackaging modifies the package specs to add the modules and
// modules.d directory. You must declare a dependency on either
// PrepareModulePackagingOSS or PrepareModulePackagingXPack.
func CustomizePackaging() {
	monitorsDTarget := "monitors.d"
	unixMonitorsDir := "/etc/{{.BeatName}}/monitors.d"
	monitorsD := devtools.PackageFile{
		Mode:   0644,
		Source: "monitors.d",
	}

	for _, args := range devtools.Packages {
		pkgType := args.Types[0]
		switch pkgType {
		case devtools.Docker:
			args.Spec.ExtraVar("linux_capabilities", "cap_net_raw=eip")
			args.Spec.Files[monitorsDTarget] = monitorsD
		case devtools.TarGz, devtools.Zip:
			args.Spec.Files[monitorsDTarget] = monitorsD
		case devtools.Deb, devtools.RPM, devtools.DMG:
			args.Spec.Files[unixMonitorsDir] = monitorsD
		}
	}
}

// Fields generates a fields.yml for the Beat.
func Fields() error {
	return devtools.GenerateFieldsYAML("monitors/active")
}
