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
