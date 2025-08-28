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
	"go.uber.org/multierr"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
)

// GolangCrossBuild builds the Beat binary inside the golang-builder and then
// checks the binaries GLIBC requirements for RHEL compatibility.
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
	return devtools.GolangCrossBuild(conf)
}

// CrossBuild cross-builds the beat for all target platforms.
func CrossBuild() error {
	return devtools.CrossBuild(devtools.ImageSelector(devtools.CrossBuildImage))
}
