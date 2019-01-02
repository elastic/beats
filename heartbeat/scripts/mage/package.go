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
	mage.BeatDescription = "Ping remote services for availability and log " +
		"results to Elasticsearch or send to Logstash."
	mage.BeatServiceName = "heartbeat-elastic"
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

// customizePackaging modifies the package specs include a modules.d directory.
func customizePackaging() {
	monitorsDTarget := "monitors.d"
	unixMonitorsDir := "/etc/{{.BeatName}}/monitors.d"
	monitorsD := mage.PackageFile{
		Mode:   0644,
		Source: mage.OSSBeatDir("monitors.d"),
	}

	for _, args := range mage.Packages {
		for _, pkgType := range args.Types {
			switch pkgType {
			case mage.Docker:
				args.Spec.ExtraVar("linux_capabilities", "cap_net_raw=eip")
				args.Spec.Files[monitorsDTarget] = monitorsD
			case mage.TarGz, mage.Zip:
				args.Spec.Files[monitorsDTarget] = monitorsD
			case mage.Deb, mage.RPM, mage.DMG:
				args.Spec.Files[unixMonitorsDir] = monitorsD
			default:
				panic(errors.Errorf("unknown package type: %v", pkgType))
			}

			break
		}
	}
}
