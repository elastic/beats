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
	"os"
	"strings"
	"testing"
)

// SkipIfFIPSOnly will mark the passed test as skipped if GODEBUG=fips140=only is detected.
// If GODEBUG=fips140=on, go may call non-compliant algorithms and the test does not need to be skipped.
func SkipIfFIPSOnly(t *testing.T, msg string) {
	// NOTE: This only checks env var; at the time of writing fips140 can only be set via env
	// other GODEBUG settings can be set via embedded comments or in go.mod, we may need to account for this in the future.
	s := os.Getenv("GODEBUG")
	if strings.Contains(s, "fips140=only") {
		t.Skip("GODEBUG=fips140=only detected, skipping test:", msg)
	}
}
