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
	"testing"

	"github.com/stretchr/testify/require"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
)

func TestWinlogbeatPackageArgs(t *testing.T) {
	originalPlatforms := append(devtools.BuildPlatformList(nil), devtools.Platforms...)
	originalSnapshot := devtools.Snapshot
	originalDevBuild := devtools.DevBuild
	t.Cleanup(func() {
		devtools.Platforms = originalPlatforms
		devtools.Snapshot = originalSnapshot
		devtools.DevBuild = originalDevBuild
	})

	testCases := []struct {
		name         string
		platformExpr string
		want         devtools.BuildPlatformList
	}{
		{
			name:         "drops non windows platforms from shared ci expression",
			platformExpr: "+all linux/amd64 windows/amd64 darwin/amd64",
			want:         devtools.NewPlatformList("windows/amd64"),
		},
		{
			name:         "keeps windows arm64 override",
			platformExpr: "+all windows/arm64",
			want:         devtools.NewPlatformList("windows/arm64"),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv("PLATFORMS", testCase.platformExpr)
			t.Setenv("PACKAGES", "")
			t.Setenv("SNAPSHOT", "")
			t.Setenv("DEV", "")

			args, err := winlogbeatPackageArgs()
			require.NoError(t, err, "winlogbeat package args should accept platform override %q", testCase.platformExpr)
			require.Equal(t, testCase.want, args.Platforms, "winlogbeat packaging should remain windows-only for %q", testCase.platformExpr)
		})
	}
}
