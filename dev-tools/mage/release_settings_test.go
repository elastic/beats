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
)

func TestInferLatestRelease(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
		wantErr bool
	}{
		{name: "valid version", version: "9.2.1", want: "9.2.0"},
		{name: "patch version is 0", version: "9.2.0", want: ""},
		{name: "invalid format", version: "9.2", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseReleaseVersion(tt.version)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, VersionString(InferLatestRelease(parsed)))
		})
	}
}

func TestInferReleaseBranch(t *testing.T) {
	parsed, err := ParseReleaseVersion("9.2.1")
	require.NoError(t, err)
	require.Equal(t, "9.2", InferReleaseBranch(parsed))
}

func TestInferNextProjectMinorVersion(t *testing.T) {
	parsed, err := ParseReleaseVersion("9.5.0")
	require.NoError(t, err)
	require.Equal(t, "9.6.0", VersionString(InferNextProjectMinorVersion(parsed)))
}

func TestLoadReleaseSettingsRequiresNextForRelease(t *testing.T) {
	t.Setenv("NEXT_RELEASE", "")
	s, err := LoadReleaseSettings()
	require.NoError(t, err)
	require.Error(t, s.RequireRelease())
}

func TestLoadReleaseSettingsFromEnv(t *testing.T) {
	t.Setenv("NEXT_RELEASE", "9.5.0")
	t.Setenv("DRY_RUN", "true")
	t.Setenv("PROJECT_OWNER", "elastic")
	t.Setenv("PROJECT_REPO", "beats")

	s, err := LoadReleaseSettings()
	require.NoError(t, err)
	require.NoError(t, s.RequireRelease())
	require.Equal(t, "9.5.0", s.NextRelease)
	require.NotNil(t, s.NextReleaseVersion)
	require.Equal(t, "9.5.0", s.NextReleaseVersion.String())
	require.Equal(t, "9.5", s.ReleaseBranch)
	require.Equal(t, "9.6.0", s.NextProjectMinorVersion)
	require.True(t, s.DryRun)
}
