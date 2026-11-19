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
	"fmt"
	"os"
	"strings"

	"github.com/Masterminds/semver"
)

// ReleaseSettings holds configuration for release automation (feature freeze / patch).
// Loaded via LoadReleaseSettings(). NEXT_RELEASE is optional during general loads;
// release targets call RequireRelease() before running workflows.
//
// Versions are parsed once into *semver.Version fields. String fields are derived
// at that time for call sites that need serialized values (PR titles, file writes).
// Workflow logic should use the parsed fields, not re-parse strings.
type ReleaseSettings struct {
	LatestReleaseVersion *semver.Version
	NextProjectMinor     *semver.Version
	NextReleaseVersion   *semver.Version

	LatestRelease           string
	NextProjectMinorBranch  string
	NextProjectMinorVersion string
	NextRelease             string

	BaseBranch    string
	ReleaseBranch string

	ProjectOwner     string
	ProjectRepo      string
	GitHubToken      string
	ProjectReviewers []string

	GitAuthorName  string
	GitAuthorEmail string

	DryRun bool
}

// LoadReleaseSettings loads release settings with defaults, then applies environment overrides.
func LoadReleaseSettings() (*ReleaseSettings, error) {
	s := &ReleaseSettings{}
	s.setReleaseDefaults()
	if err := s.loadReleaseSettingsFromEnv(); err != nil {
		return nil, err
	}
	return s, nil
}

// RequireRelease ensures NEXT_RELEASE was provided so inferred release fields are set.
func (s *ReleaseSettings) RequireRelease() error {
	if s.NextRelease == "" && s.NextReleaseVersion == nil {
		return fmt.Errorf("NEXT_RELEASE environment variable is required")
	}
	return s.EnsureParsedVersions()
}

func (s *ReleaseSettings) setReleaseDefaults() {
	s.BaseBranch = "main"
	s.ProjectOwner = "elastic"
	s.ProjectRepo = "beats"
	s.GitAuthorName = "github-actions[bot]"
	s.GitAuthorEmail = "github-actions[bot]@users.noreply.github.com"
	s.ProjectReviewers = []string{"elastic/elastic-agent-release"}
	s.DryRun = false
}

func (s *ReleaseSettings) loadReleaseSettingsFromEnv() error {
	if v := os.Getenv("BASE_BRANCH"); v != "" {
		s.BaseBranch = v
	}
	if v := os.Getenv("PROJECT_OWNER"); v != "" {
		s.ProjectOwner = v
	}
	if v := os.Getenv("PROJECT_REPO"); v != "" {
		s.ProjectRepo = v
	}
	if v := os.Getenv("GITHUB_TOKEN"); v != "" {
		s.GitHubToken = v
	}
	if v := os.Getenv("GIT_AUTHOR_NAME"); v != "" {
		s.GitAuthorName = v
	}
	if v := os.Getenv("GIT_AUTHOR_EMAIL"); v != "" {
		s.GitAuthorEmail = v
	}
	if v := os.Getenv("DRY_RUN"); v != "" {
		s.DryRun = strings.EqualFold(v, "true")
	}
	if v := os.Getenv("PROJECT_REVIEWERS"); v != "" {
		s.ProjectReviewers = strings.Split(v, ",")
	}

	nextRelease := os.Getenv("NEXT_RELEASE")
	if nextRelease == "" {
		return nil
	}
	return s.populateFromNext(nextRelease)
}

// EnsureParsedVersions fills parsed version fields from NextRelease when missing.
// Production loads set these in populateFromNext; tests may set only the string.
// An explicitly set LatestRelease string is preserved when inference leaves it empty
// (minor releases with patch 0).
func (s *ReleaseSettings) EnsureParsedVersions() error {
	if s.NextReleaseVersion != nil {
		if s.LatestReleaseVersion == nil && s.LatestRelease != "" {
			return s.SetLatestReleaseFromString(s.LatestRelease)
		}
		return nil
	}
	if s.NextRelease == "" {
		return fmt.Errorf("NextRelease is required")
	}
	latestOverride := s.LatestRelease
	if err := s.populateFromNext(s.NextRelease); err != nil {
		return err
	}
	if latestOverride != "" && s.LatestReleaseVersion == nil {
		return s.SetLatestReleaseFromString(latestOverride)
	}
	return nil
}

func (s *ReleaseSettings) populateFromNext(nextRelease string) error {
	parsed, err := ParseReleaseVersion(nextRelease)
	if err != nil {
		return fmt.Errorf("invalid NEXT_RELEASE: %w", err)
	}

	s.NextReleaseVersion = parsed
	s.NextRelease = parsed.String()

	s.LatestReleaseVersion = InferLatestRelease(parsed)
	s.LatestRelease = VersionString(s.LatestReleaseVersion)

	s.ReleaseBranch = InferReleaseBranch(parsed)

	s.NextProjectMinor = InferNextProjectMinorVersion(parsed)
	s.NextProjectMinorVersion = VersionString(s.NextProjectMinor)
	s.NextProjectMinorBranch = InferNextProjectMinorBranch(parsed)
	return nil
}

// ParseReleaseVersion parses a release version string as major.minor.patch with no
// prerelease or build metadata. A leading "v" is stripped.
func ParseReleaseVersion(nextRelease string) (*semver.Version, error) {
	tag := strings.TrimPrefix(strings.TrimSpace(nextRelease), "v")
	if len(strings.Split(tag, ".")) != 3 {
		return nil, fmt.Errorf("invalid version format: %s (expected major.minor.patch)", nextRelease)
	}
	parsed, err := semver.NewVersion(tag)
	if err != nil {
		return nil, fmt.Errorf("invalid version format: %s (expected major.minor.patch): %w", nextRelease, err)
	}
	if parsed.Prerelease() != "" || parsed.Metadata() != "" {
		return nil, fmt.Errorf("invalid version format: %s (expected major.minor.patch without prerelease/build metadata)", nextRelease)
	}
	return parsed, nil
}

// VersionString returns v.String(), or "" when v is nil.
func VersionString(v *semver.Version) string {
	if v == nil {
		return ""
	}
	return v.String()
}

// mustVersion builds a *semver.Version from major.minor.patch components.
// Components come from an already-parsed release version, so construction cannot fail.
func mustVersion(major, minor, patch int64) *semver.Version {
	v, err := semver.NewVersion(fmt.Sprintf("%d.%d.%d", major, minor, patch))
	if err != nil {
		panic(fmt.Sprintf("invalid constructed version %d.%d.%d: %v", major, minor, patch, err))
	}
	return v
}

// InferLatestRelease returns the previous patch version (patch - 1).
// For minor releases (patch == 0), returns nil; callers may resolve via GitHub.
func InferLatestRelease(v *semver.Version) *semver.Version {
	if v.Patch() == 0 {
		return nil
	}
	return mustVersion(v.Major(), v.Minor(), v.Patch()-1)
}

// InferReleaseBranch returns the major.minor release branch name.
func InferReleaseBranch(v *semver.Version) string {
	return fmt.Sprintf("%d.%d", v.Major(), v.Minor())
}

// InferNextProjectMinorVersion returns the next minor version as major.(minor+1).0.
func InferNextProjectMinorVersion(v *semver.Version) *semver.Version {
	return mustVersion(v.Major(), v.Minor()+1, 0)
}

// InferNextProjectMinorBranch returns the next minor branch as major.(minor+1).
func InferNextProjectMinorBranch(v *semver.Version) string {
	return fmt.Sprintf("%d.%d", v.Major(), v.Minor()+1)
}

// SetLatestReleaseFromString parses and stores a previously published release version.
func (s *ReleaseSettings) SetLatestReleaseFromString(raw string) error {
	parsed, err := ParseReleaseVersion(raw)
	if err != nil {
		return err
	}
	s.LatestReleaseVersion = parsed
	s.LatestRelease = parsed.String()
	return nil
}
