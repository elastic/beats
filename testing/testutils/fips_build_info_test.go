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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckFIPSBuildInfo(t *testing.T) {
	tests := []struct {
		name       string
		godebug    string
		wantFIPSOn bool
	}{
		{name: "fips140=on", godebug: "fips140=on,tlsmlkem=0", wantFIPSOn: true},
		{name: "fips140=on last", godebug: "tlsmlkem=0,fips140=on", wantFIPSOn: true},
		{name: "fips140=only is not fips140=on", godebug: "fips140=only,tlsmlkem=0", wantFIPSOn: false},
		{name: "fips140=off", godebug: "fips140=off", wantFIPSOn: false},
		{name: "no fips140 entry", godebug: "tlsmlkem=0", wantFIPSOn: false},
		{name: "empty", godebug: "", wantFIPSOn: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			settings := []debug.BuildSetting{
				{Key: "-tags", Value: "requirefips"},
				{Key: "GOFIPS140", Value: "v1.0.0"},
				{Key: "DefaultGODEBUG", Value: tc.godebug},
			}

			result := CheckFIPSBuildInfo(settings)

			require.True(t, result.TagsFound)
			require.True(t, result.TagsHaveRequireFIPS)
			require.True(t, result.GOFIPS140Found)
			require.True(t, result.GOFIPS140IsCertified)
			require.True(t, result.DefaultGODEBUGFound)
			require.Equal(t, tc.wantFIPSOn, result.DefaultGODEBUGHasFIPSOn)
		})
	}
}

func TestCheckFIPSBuildInfo_MissingSettings(t *testing.T) {
	result := CheckFIPSBuildInfo(nil)

	require.False(t, result.TagsFound)
	require.False(t, result.GOFIPS140Found)
	require.False(t, result.DefaultGODEBUGFound)
	require.False(t, result.DefaultGODEBUGHasFIPSOn)
}
