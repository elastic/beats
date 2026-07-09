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

package testutils

import (
	"runtime/debug"
	"strings"

	"github.com/stretchr/testify/assert"
)

// fipsCheckT is the subset of *testing.T that CheckFIPSBuildInfo needs. It's kept as
// an interface (rather than *testing.T directly) so this package's own tests can
// exercise the failure paths with a fake, instead of failing the enclosing test run.
type fipsCheckT interface {
	Helper()
	Errorf(format string, args ...any)
	Failed() bool
	FailNow()
}

// CheckFIPSBuildInfo asserts that a binary's build settings (as reported by
// debug/buildinfo) contain the markers of a FIPS-compliant build: the
// "requirefips" build tag, a GOFIPS140 setting referencing the certified
// module version, and a DefaultGODEBUG setting that enables fips140=on at
// runtime. All markers are checked before the test is failed, so a single
// run reports every missing marker instead of just the first.
func CheckFIPSBuildInfo(t fipsCheckT, settings []debug.BuildSetting) {
	t.Helper()

	var foundTags, foundFIPS, foundFIPSDefault bool
	for _, setting := range settings {
		switch setting.Key {
		case "-tags":
			foundTags = true
			assert.Contains(t, setting.Value, "requirefips")
		case "GOFIPS140":
			foundFIPS = true
			assert.True(t, strings.HasPrefix(setting.Value, "v1.0.0"), "GOFIPS140 must reference the certified module version, got %q", setting.Value)
		case "DefaultGODEBUG":
			for entry := range strings.SplitSeq(setting.Value, ",") {
				if key, val, ok := strings.Cut(entry, "="); ok && key == "fips140" && val == "on" {
					foundFIPSDefault = true
					break
				}
			}
		}
	}

	assert.True(t, foundTags, "did not find -tags within binary version information")
	assert.True(t, foundFIPS, "did not find GOFIPS140 within binary version information")
	assert.True(t, foundFIPSDefault, "did not find fips140=on in DefaultGODEBUG — binary will not enforce FIPS mode at runtime (check GOFIPS140 env at build time)")

	if t.Failed() {
		t.FailNow()
	}
}
