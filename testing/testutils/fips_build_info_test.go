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
	"fmt"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// fakeT is a minimal fipsCheckT that records failures instead of stopping the
// goroutine, so failure paths can be asserted on directly.
type fakeT struct {
	failed bool
	errors []string
}

func (f *fakeT) Helper() {}

func (f *fakeT) Errorf(format string, args ...any) {
	f.failed = true
	f.errors = append(f.errors, fmt.Sprintf(format, args...))
}

func (f *fakeT) Failed() bool { return f.failed }

func (f *fakeT) FailNow() {}

func TestRequireFIPSBuildInfo(t *testing.T) {
	tests := []struct {
		name     string
		godebug  string
		wantPass bool
	}{
		{name: "fips140=on", godebug: "fips140=on,tlsmlkem=0", wantPass: true},
		{name: "fips140=on last", godebug: "tlsmlkem=0,fips140=on", wantPass: true},
		{name: "fips140=only is not fips140=on", godebug: "fips140=only,tlsmlkem=0", wantPass: false},
		{name: "fips140=off", godebug: "fips140=off", wantPass: false},
		{name: "no fips140 entry", godebug: "tlsmlkem=0", wantPass: false},
		{name: "empty", godebug: "", wantPass: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			settings := []debug.BuildSetting{
				{Key: "-tags", Value: "requirefips"},
				{Key: "GOFIPS140", Value: "v1.0.0"},
				{Key: "DefaultGODEBUG", Value: tc.godebug},
			}

			ft := &fakeT{}
			RequireFIPSBuildInfo(ft, settings)

			require.Equal(t, !tc.wantPass, ft.failed, "errors: %v", ft.errors)
		})
	}
}

func TestRequireFIPSBuildInfo_MissingSettings(t *testing.T) {
	ft := &fakeT{}
	RequireFIPSBuildInfo(ft, nil)

	require.True(t, ft.failed)
	require.Len(t, ft.errors, 3, "expected all three markers to be reported missing, got: %v", ft.errors)
	joined := strings.Join(ft.errors, "\n")
	require.Contains(t, joined, "did not find -tags within binary version information")
	require.Contains(t, joined, "did not find GOFIPS140 within binary version information")
	require.Contains(t, joined, "did not find fips140=on in DefaultGODEBUG — binary will not enforce FIPS mode at runtime (check GOFIPS140 env at build time)")
}
