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

package release

import (
	"testing"

	"github.com/stretchr/testify/require"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
)

func TestValidateReleaseConfig(t *testing.T) {
	t.Run("missing current", func(t *testing.T) {
		err := ValidateReleaseConfig(&ReleaseConfig{DryRun: true, LatestRelease: "9.4.0"})
		require.Error(t, err)
	})
	t.Run("missing latest", func(t *testing.T) {
		err := ValidateReleaseConfig(&ReleaseConfig{CurrentRelease: "9.5.0", DryRun: true})
		require.Error(t, err)
		require.Contains(t, err.Error(), "LatestRelease")
	})
	t.Run("missing token when not dry-run", func(t *testing.T) {
		err := ValidateReleaseConfig(&ReleaseConfig{CurrentRelease: "9.5.0", LatestRelease: "9.4.0"})
		require.Error(t, err)
	})
	t.Run("valid dry-run", func(t *testing.T) {
		err := ValidateReleaseConfig(&ReleaseConfig{
			CurrentRelease: "9.5.0",
			LatestRelease:  "9.4.0",
			DryRun:         true,
		})
		require.NoError(t, err)
	})
}

func TestEnsureLatestRelease(t *testing.T) {
	orig := fetchLatestReleaseBefore
	t.Cleanup(func() { fetchLatestReleaseBefore = orig })

	fetchLatestReleaseBefore = func(token, owner, repo, current string) (string, error) {
		return "9.4.3", nil
	}

	cfg := &ReleaseConfig{CurrentRelease: "9.5.0", DryRun: true}
	require.NoError(t, EnsureLatestRelease(cfg))
	require.Equal(t, "9.4.3", cfg.LatestRelease)

	cfg.LatestRelease = "9.4.0"
	require.NoError(t, EnsureLatestRelease(cfg))
	require.Equal(t, "9.4.0", cfg.LatestRelease)
}

func TestLoadReleaseSettingsViaMage(t *testing.T) {
	t.Setenv("CURRENT_RELEASE", "9.5.0")
	t.Setenv("DRY_RUN", "true")

	settings, err := devtools.LoadReleaseSettings()
	require.NoError(t, err)
	require.NoError(t, settings.RequireRelease())
	require.Equal(t, "9.5.0", settings.CurrentRelease)
	require.Equal(t, "9.5.1", settings.NextRelease)
	require.True(t, settings.DryRun)
}
