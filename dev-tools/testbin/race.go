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

package testbin

import (
	"os"
	"strconv"
)

// RaceDetectorEnvVar is the single switch that turns on the race detector for
// Go integration tests. When truthy, both the integration test process and the
// beat-under-test are built with -race.
//
// It is intentionally separate from RACE_DETECTOR (which CI sets for unit
// tests), so that integration tests are not automatically run there.
const RaceDetectorEnvVar = "INTEG_RACE_DETECTOR"

// RaceDetectorEnabled reports whether RaceDetectorEnvVar requests race
// instrumentation of the beat-under-test. A missing or unparseable value is
// treated as disabled, matching how Build reads DEV and TEST_COVERAGE. It is
// the single source of truth shared by Build and the integration framework.
func RaceDetectorEnabled() bool {
	enabled, _ := strconv.ParseBool(os.Getenv(RaceDetectorEnvVar))
	return enabled
}

// RaceDetectorSupported reports whether the Go race detector (-race) is
// available for the given GOOS/GOARCH. The race detector only supports a
// subset of platforms; this is the intersection of the platforms supported by
// Beats and by Go, and is the single source of truth used by both the mage
// build system and the Go integration test binary builder.
//
// See https://go.dev/doc/articles/race_detector#Requirements.
func RaceDetectorSupported(goos, goarch string) bool {
	return goarch == "amd64" ||
		(goarch == "arm64" && (goos == "linux" || goos == "darwin"))
}
