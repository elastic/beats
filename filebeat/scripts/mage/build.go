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
	"strings"

	"github.com/magefile/mage/mg"
	"go.uber.org/multierr"

	devtools "github.com/elastic/beats/v8/dev-tools/mage"
)

// declare journald dependencies for cross build target
var (
	journaldPlatforms = []devtools.PlatformDescription{
		devtools.Linux386, devtools.LinuxAMD64,
		devtools.LinuxARM64, devtools.LinuxARM5, devtools.LinuxARM6, devtools.LinuxARM7,
		devtools.LinuxMIPS, devtools.LinuxMIPSLE, devtools.LinuxMIPS64LE,
		devtools.LinuxPPC64LE,
		devtools.LinuxS390x,
	}

	journaldDeps = devtools.NewPackageInstaller().
			AddEach(journaldPlatforms, "libsystemd-dev").
			Add(devtools.Linux386, "libsystemd0", "libgcrypt20")
)

// GolangCrossBuild builds the Beat binary inside the golang-builder and then
// checks the binaries GLIBC requirements for RHEL compatability.
// Do not use directly, use crossBuild instead.
func GolangCrossBuild() error {
	return multierr.Combine(
		golangCrossBuild(),
		// Test the linked glibc version requirement of the binary.
		devtools.TestLinuxForCentosGLIBC(),
	)
}

// golangCrossBuild builds the Beat binary inside the golang-builder.
// Do not use directly, use crossBuild instead.
func golangCrossBuild() error {
	conf := devtools.DefaultGolangCrossBuildArgs()
	if devtools.Platform.GOOS == "linux" {
		mg.Deps(journaldDeps.Installer(devtools.Platform.Name))
		conf.ExtraFlags = append(conf.ExtraFlags, "-tags=withjournald")
	}
	return devtools.GolangCrossBuild(conf)
}

// CrossBuild cross-builds the beat for all target platforms.
func CrossBuild() error {
	return devtools.CrossBuild(devtools.ImageSelector(func(platform string) (string, error) {
		image, err := devtools.CrossBuildImage(platform)
		if err != nil {
			return "", err
		}
		// Normally linux/amd64 and linux/386 binaries are build using debian7
		// because it has an older glibc version that makes the binaries work on
		// RHEL 6, but debian7 does not have the systemd libraries needed for
		// the journald input.
		//
		// So use the debian8 image, but test the binary to ensure that the
		// linked glibc version requirement is still compatible with RHEL6.
		if platform == "linux/amd64" || platform == "linux/386" {
			image = strings.ReplaceAll(image, "main-debian7", "main-debian8")
		}
		return image, nil
	}))
}
