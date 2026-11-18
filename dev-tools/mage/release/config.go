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
	"fmt"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
)

// ReleaseConfig is the release automation configuration loaded via mage ReleaseSettings.
type ReleaseConfig = devtools.ReleaseSettings

// fetchLatestReleaseBefore looks up the previous published release from GitHub.
// Tests may replace this to avoid network calls.
var fetchLatestReleaseBefore = func(token, owner, repo, current string) (string, error) {
	return newGitHubClient(token).latestReleaseBefore(owner, repo, current)
}

// ensureLatestRelease sets LatestRelease when unset by querying elastic/beats releases.
func ensureLatestRelease(c *ReleaseConfig) error {
	if c.LatestRelease != "" {
		return nil
	}
	if c.CurrentRelease == "" {
		return fmt.Errorf("CurrentRelease is required to resolve LatestRelease")
	}

	latest, err := fetchLatestReleaseBefore(c.GitHubToken, releasesLookupOwner, releasesLookupRepo, c.CurrentRelease)
	if err != nil {
		return fmt.Errorf("failed to resolve LatestRelease from GitHub: %w", err)
	}
	c.LatestRelease = latest
	fmt.Printf("Resolved LatestRelease from %s/%s: %s\n", releasesLookupOwner, releasesLookupRepo, latest)
	return nil
}

// validateReleaseConfig checks if the configuration is valid for release workflows.
func validateReleaseConfig(c *ReleaseConfig) error {
	if c.CurrentRelease == "" {
		return fmt.Errorf("CurrentRelease is required")
	}

	if c.LatestRelease == "" {
		return fmt.Errorf("LatestRelease is required (resolve via ensureLatestRelease)")
	}

	if !c.DryRun && c.GitHubToken == "" {
		return fmt.Errorf("GITHUB_TOKEN is required when not in dry-run mode")
	}

	return nil
}
