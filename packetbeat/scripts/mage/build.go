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
	"strings"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
)

// NpcapVersion specifies the version of the OEM Npcap installer to bundle with
// the packetbeat executable. It is used to specify which npcap builder crossbuild
// image to use.
const NpcapVersion = "1.60"

// CrossBuild cross-builds the beat for all target platforms.
//
// On Windows platforms, if CrossBuild is invoked with the environment variables
// CI or NPCAP_LOCAL set to "true", a private cross-build image is selected that
// provides the OEM Npcap installer for the build. This behaviour requires access
// to the private image.
func CrossBuild() error {
	return devtools.CrossBuild(
		// Run all builds serially to try to address failures that might be caused
		// by concurrent builds. See https://github.com/elastic/beats/issues/24304.
		devtools.Serially(),

		devtools.ImageSelector(func(platform string) (string, error) {
			image, err := devtools.CrossBuildImage(platform)
			if err != nil {
				return "", err
			}
			if os.ExpandEnv("CI") != "true" && os.ExpandEnv("NPCAP_LOCAL") != "true" {
				return image, nil
			}
			if platform == "windows/amd64" || platform == "windows/386" {
				image = strings.ReplaceAll(image, "main", "npcap-"+NpcapVersion)
			}
			return image, nil
		}),
	)
}
