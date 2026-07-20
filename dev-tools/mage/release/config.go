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
	"os"
	"strconv"
	"strings"
)

// ReleaseConfig holds the configuration for release operations
type ReleaseConfig struct {
	// Version information
	CurrentRelease          string
	LatestRelease           string
	NextRelease             string
	NextProjectMinorVersion string
	NextProjectMinorBranch  string

	// Branch information
	BaseBranch    string
	ReleaseBranch string

	// GitHub configuration
	ProjectOwner     string
	ProjectRepo      string
	GitHubToken      string
	ProjectReviewers []string

	// Git author information
	GitAuthorName  string
	GitAuthorEmail string

	// Operational flags
	DryRun bool

	// Changelog configuration
	ChangelogToCommit string
}

// LoadConfigFromEnv loads release configuration from environment variables
func LoadConfigFromEnv() (*ReleaseConfig, error) {
	currentRelease := os.Getenv("CURRENT_RELEASE")

	// Validate required fields
	if currentRelease == "" {
		return nil, fmt.Errorf("CURRENT_RELEASE environment variable is required")
	}

	// Infer LatestRelease, NextRelease, and ReleaseBranch from CurrentRelease
	latestRelease, err := inferLatestRelease(currentRelease)
	if err != nil {
		return nil, fmt.Errorf("failed to infer LatestRelease: %w", err)
	}

	nextRelease, err := inferNextRelease(currentRelease)
	if err != nil {
		return nil, fmt.Errorf("failed to infer NextRelease: %w", err)
	}

	releaseBranch := inferReleaseBranch(currentRelease)

	nextProjectMinorVersion, err := inferNextProjectMinorVersion(currentRelease)
	if err != nil {
		return nil, fmt.Errorf("failed to infer NextProjectMinorVersion: %w", err)
	}
	nextProjectMinorBranch := inferNextProjectMinorBranch(currentRelease)

	cfg := &ReleaseConfig{
		CurrentRelease:          currentRelease,
		LatestRelease:           latestRelease,
		NextRelease:             nextRelease,
		NextProjectMinorVersion: nextProjectMinorVersion,
		NextProjectMinorBranch:  nextProjectMinorBranch,
		BaseBranch:              getEnvOrDefault("BASE_BRANCH", "main"),
		ReleaseBranch:           releaseBranch,
		ProjectOwner:      getEnvOrDefault("PROJECT_OWNER", "elastic"),
		ProjectRepo:       getEnvOrDefault("PROJECT_REPO", "beats"),
		GitHubToken:       os.Getenv("GITHUB_TOKEN"),
		GitAuthorName:     getEnvOrDefault("GIT_AUTHOR_NAME", "github-actions[bot]"),
		GitAuthorEmail:    getEnvOrDefault("GIT_AUTHOR_EMAIL", "github-actions[bot]@users.noreply.github.com"),
		DryRun:            getEnvOrDefault("DRY_RUN", "false") == "true",
		ChangelogToCommit: getEnvOrDefault("CHANGELOG_TO_COMMIT", "HEAD"),
	}

	// Parse reviewers (comma-separated)
	reviewers := getEnvOrDefault("PROJECT_REVIEWERS", "elastic/elastic-agent-release")
	cfg.ProjectReviewers = strings.Split(reviewers, ",")

	return cfg, nil
}

// getEnvOrDefault returns the value of an environment variable or a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// inferLatestRelease calculates the previous release version (patch - 1).
// For minor releases (patch == 0), returns empty string; callers should use EnsureLatestRelease.
func inferLatestRelease(currentRelease string) (string, error) {
	parts := strings.Split(currentRelease, ".")
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid version format: %s (expected major.minor.patch)", currentRelease)
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return "", fmt.Errorf("invalid patch version: %s", parts[2])
	}

	// For minor releases (e.g., 9.5.0), previous release is resolved via GitHub API.
	if patch == 0 {
		return "", nil
	}

	return fmt.Sprintf("%s.%s.%d", parts[0], parts[1], patch-1), nil
}

// inferNextRelease calculates the next release version (patch + 1)
func inferNextRelease(currentRelease string) (string, error) {
	parts := strings.Split(currentRelease, ".")
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid version format: %s (expected major.minor.patch)", currentRelease)
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return "", fmt.Errorf("invalid patch version: %s", parts[2])
	}

	return fmt.Sprintf("%s.%s.%d", parts[0], parts[1], patch+1), nil
}

// inferReleaseBranch extracts the major.minor version
func inferReleaseBranch(currentRelease string) string {
	parts := strings.Split(currentRelease, ".")
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	return ""
}

// inferNextProjectMinorVersion returns major.(minor+1).0 for the current release.
func inferNextProjectMinorVersion(currentRelease string) (string, error) {
	parts := strings.Split(currentRelease, ".")
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid version format: %s (expected major.minor.patch)", currentRelease)
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", fmt.Errorf("invalid minor version: %s", parts[1])
	}

	return fmt.Sprintf("%s.%d.0", parts[0], minor+1), nil
}

// inferNextProjectMinorBranch returns major.(minor+1) for Mergify label naming.
func inferNextProjectMinorBranch(currentRelease string) string {
	parts := strings.Split(currentRelease, ".")
	if len(parts) < 2 {
		return ""
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%s.%d", parts[0], minor+1)
}

// isMinorRelease reports whether version is a minor release (patch == 0).
func isMinorRelease(version string) bool {
	parts := strings.Split(version, ".")
	return len(parts) >= 3 && parts[2] == "0"
}

// fetchLatestReleaseBefore looks up the previous published release from GitHub.
// Tests may replace this to avoid network calls.
var fetchLatestReleaseBefore = func(token, owner, repo, current string) (string, error) {
	return NewGitHubClient(token).LatestReleaseBefore(owner, repo, current)
}

// EnsureLatestRelease sets LatestRelease when unset by querying elastic/beats releases.
// Patch releases typically already have LatestRelease from patch−1 inference.
func (c *ReleaseConfig) EnsureLatestRelease() error {
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

// Validate checks if the configuration is valid
func (c *ReleaseConfig) Validate() error {
	if c.CurrentRelease == "" {
		return fmt.Errorf("CurrentRelease is required")
	}

	if c.LatestRelease == "" {
		return fmt.Errorf("LatestRelease is required (resolve via EnsureLatestRelease)")
	}

	if !c.DryRun && c.GitHubToken == "" {
		return fmt.Errorf("GITHUB_TOKEN is required when not in dry-run mode")
	}

	return nil
}
