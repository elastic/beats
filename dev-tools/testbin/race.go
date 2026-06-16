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

// RaceDetectorEnvVar is the environment variable that opts the beat-under-test
// into the race detector during Go integration tests. When set to a truthy
// value, Build compiles the beat test binary with -race so data races are
// detected while the beat runs under the integration framework.
//
// It is intentionally distinct from RACE_DETECTOR (which controls "go test
// -race"): CI sets RACE_DETECTOR=true for unit tests, and reusing it here would
// implicitly race-instrument every spawned beat in CI. Keeping a separate
// variable makes the beat-under-test instrumentation an explicit opt-in.
const RaceDetectorEnvVar = "INTEG_RACE_DETECTOR"

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
