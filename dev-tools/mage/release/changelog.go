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
	"context"
	"fmt"
	"os/exec"
)

// PrepareChangelog generates changelog using the Python beats-changelog package
func PrepareChangelog(fromVersion, toCommit string) error {
	fmt.Printf("Generating changelog from %s to %s...\n", fromVersion, toCommit)

	// Check if beats-changelog is available
	_, err := exec.LookPath("beats-changelog")
	if err != nil {
		return fmt.Errorf("beats-changelog not found in PATH. Please install it first: %w", err)
	}

	// Run beats-changelog split command
	// Example: beats-changelog split --from v9.2.0 --to <commit>
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "beats-changelog", "split",
		"--from", "v"+fromVersion,
		"--to", toCommit,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run beats-changelog: %w\nOutput: %s", err, string(output))
	}

	fmt.Println(string(output))
	fmt.Println("Changelog generated successfully")

	return nil
}

// RunChangelog executes the complete changelog workflow
func RunChangelog(cfg *ReleaseConfig) error {
	fmt.Println("=== Starting Changelog Workflow ===")

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return err
	}

	// Open repository
	repo, err := OpenRepo(".")
	if err != nil {
		return err
	}

	// Check if working directory is clean
	clean, err := repo.IsClean()
	if err != nil {
		return err
	}
	if !clean {
		return fmt.Errorf("working directory is not clean. Please commit or stash changes first")
	}

	// Create changelog branch
	branchName := fmt.Sprintf("prepare-changelog-%s", cfg.CurrentRelease)
	if err := repo.CreateBranch(branchName); err != nil {
		return err
	}

	if err := repo.CheckoutBranch(branchName); err != nil {
		return err
	}

	// Generate changelog
	fromVersion := cfg.LatestRelease
	if fromVersion == "" {
		// For minor releases (e.g., 9.5.0), user must explicitly set LATEST_RELEASE
		// to the last patch of the previous minor version (e.g., 9.4.3)
		fmt.Println("WARNING: LATEST_RELEASE not set. For minor releases, please set LATEST_RELEASE explicitly.")
		fmt.Printf("WARNING: Using current release %s as starting point for changelog.\n", cfg.CurrentRelease)
		fromVersion = cfg.CurrentRelease
	}

	if err := PrepareChangelog(fromVersion, cfg.ChangelogToCommit); err != nil {
		return err
	}

	// Commit changes
	commitMsg := fmt.Sprintf("Update changelog for %s", cfg.CurrentRelease)
	if err := repo.CommitAll(commitMsg, cfg.GitAuthorName, cfg.GitAuthorEmail); err != nil {
		return err
	}

	// Push and create PR (skip in dry-run mode)
	if cfg.DryRun {
		fmt.Println("DRY RUN: Skipping push and PR creation")
		fmt.Printf("Branch created: %s\n", branchName)
		fmt.Println("Review changes with 'git diff'")
		return nil
	}

	if err := repo.Push("origin"); err != nil {
		return err
	}

	// Create PR
	gh := NewGitHubClient(cfg.GitHubToken)
	prBody := fmt.Sprintf(`## Changelog Updates for %s

This PR updates the changelog for the %s release.

Generated with beats-changelog from version %s.

Please review the changelog entries and merge when ready.
`, cfg.CurrentRelease, cfg.CurrentRelease, fromVersion)

	prOpts := PROptions{
		Owner:     cfg.ProjectOwner,
		Repo:      cfg.ProjectRepo,
		Title:     fmt.Sprintf("Update changelog for %s", cfg.CurrentRelease),
		Head:      branchName,
		Base:      cfg.ReleaseBranch,
		Body:      prBody,
		Draft:     false,
		Reviewers: cfg.ProjectReviewers,
		Labels:    []string{"changelog", "release"},
	}

	pr, err := gh.CreatePR(prOpts)
	if err != nil {
		return err
	}

	fmt.Printf("\n=== Changelog Workflow Complete ===\n")
	fmt.Printf("PR created: %s\n", pr.GetHTMLURL())

	return nil
}
