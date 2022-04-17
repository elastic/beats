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

	"github.com/magefile/mage/mg"

	devtools "github.com/menderesk/beats/v7/dev-tools/mage"
)

func init() {
	devtools.BeatDescription = "Ping remote services for availability and log " +
		"results to Elasticsearch or send to Logstash."
	devtools.BeatServiceName = "heartbeat-elastic"
}

// CustomizePackaging modifies the package specs to add the modules and
// modules.d directory. You must declare a dependency on either
// PrepareModulePackagingOSS or PrepareModulePackagingXPack.
func CustomizePackaging() {
	mg.Deps(dashboards)

	monitorsDTarget := "monitors.d"
	unixMonitorsDir := "/etc/{{.BeatName}}/monitors.d"
	monitorsD := devtools.PackageFile{
		Mode:   0644,
		Source: devtools.OSSBeatDir("monitors.d"),
	}

	for _, args := range devtools.Packages {
		pkgType := args.Types[0]
		switch pkgType {
		case devtools.Docker:
			args.Spec.ExtraVar("linux_capabilities", "cap_net_raw+eip")
			args.Spec.Files[monitorsDTarget] = monitorsD
		case devtools.TarGz, devtools.Zip:
			args.Spec.Files[monitorsDTarget] = monitorsD
		case devtools.Deb, devtools.RPM:
			args.Spec.Files[unixMonitorsDir] = monitorsD
		}
	}
}

func dashboards() error {
	// Heartbeat doesn't have any dashboards so just create the empty directory.
	return os.MkdirAll("build/kibana", 0755)
}

// Fields generates a fields.yml for the Beat.
func Fields() error {
	err := devtools.GenerateFieldsYAML(devtools.OSSBeatDir())
	if err != nil {
		return err
	}
	return devtools.GenerateFieldsGo("fields.yml", "include/fields.go")
}
