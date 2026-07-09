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
	"strings"

	"github.com/google/go-github/v68/github"
)

// checkRequirements validates prerequisites before running a release workflow
func checkRequirements(cfg *ReleaseConfig) error {
	// Block deprecated releases (6.x, 7.x, 8.x minor releases)
	version := cfg.CurrentRelease
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return fmt.Errorf("invalid version format: %s", version)
	}

	major := parts[0]
	patch := ""
	if len(parts) >= 3 {
		patch = parts[2]
	}

	// Block minor releases for versions 6.x, 7.x, 8.x
	if (major == "6" || major == "7" || major == "8") && patch == "0" {
		return fmt.Errorf("minor releases for version %s.x are deprecated and blocked", major)
	}

	// Check if repository is clean
	repo, err := OpenRepo(".")
	if err != nil {
		return err
	}

	clean, err := repo.IsClean()
	if err != nil {
		return err
	}
	if !clean {
		return fmt.Errorf("working directory is not clean. Please commit or stash changes first")
	}

	return nil
}

// RunMajorMinorRelease executes the major/minor release workflow (creates 1 PR)
func RunMajorMinorRelease(cfg *ReleaseConfig) error {
	fmt.Println("=== Starting Major/Minor Release Workflow ===")

	// Validate and check requirements
	if err := cfg.Validate(); err != nil {
		return err
	}

	if err := checkRequirements(cfg); err != nil {
		return err
	}

	repo, err := OpenRepo(".")
	if err != nil {
		return err
	}

	// Create release branch (e.g., "9.3")
	releaseBranch := cfg.ReleaseBranch
	fmt.Printf("Ensuring release branch: %s\n", releaseBranch)
	if err := repo.EnsureBranch(releaseBranch); err != nil {
		return err
	}

	// Create update branch from release branch
	updateBranch := fmt.Sprintf("update-version-%s", cfg.CurrentRelease)
	fmt.Printf("Ensuring update branch: %s\n", updateBranch)
	if err := repo.EnsureBranch(updateBranch); err != nil {
		return err
	}

	// Update files
	fmt.Println("Updating version files...")
	if err := UpdateVersion(cfg.CurrentRelease); err != nil {
		return err
	}

	if err := UpdateDocs(cfg.CurrentRelease); err != nil {
		return err
	}

	if cfg.LatestRelease != "" {
		if err := UpdateTestEnv(cfg.LatestRelease, cfg.CurrentRelease); err != nil {
			return err
		}
	}

	// Commit changes
	commitMsg := fmt.Sprintf("Update version to %s for release", cfg.CurrentRelease)
	if _, err := repo.CommitAll(commitMsg, cfg.GitAuthorName, cfg.GitAuthorEmail); err != nil {
		return err
	}

	// Push and create PR (skip in dry-run mode)
	if cfg.DryRun {
		fmt.Println("\nDRY RUN: Skipping push and PR creation")
		fmt.Printf("Branches prepared: %s, %s\n", releaseBranch, updateBranch)
		fmt.Println("Review changes with 'git diff'")
		return nil
	}

	// Push release branch first
	if err := repo.CheckoutBranch(releaseBranch); err != nil {
		return err
	}
	if err := repo.Push("origin"); err != nil {
		return err
	}

	gh := NewGitHubClient(cfg.GitHubToken)
	prBody := fmt.Sprintf(`## Release %s

This PR prepares the repository for the %s release.

### Changes
- Updated version to %s
- Updated documentation references
- Updated test environment configurations

cc @%s
`, cfg.CurrentRelease, cfg.CurrentRelease, cfg.CurrentRelease, strings.Join(cfg.ProjectReviewers, " @"))

	prOpts := PROptions{
		Owner:     cfg.ProjectOwner,
		Repo:      cfg.ProjectRepo,
		Title:     fmt.Sprintf("Release %s", cfg.CurrentRelease),
		Head:      updateBranch,
		Base:      releaseBranch,
		Body:      prBody,
		Draft:     false,
		Reviewers: cfg.ProjectReviewers,
		Labels:    []string{"release", "version"},
	}

	pr, err := finalizePR(repo, gh, updateBranch, releaseBranch, prOpts)
	if err != nil {
		return err
	}

	fmt.Printf("\n=== Major/Minor Release Workflow Complete ===\n")
	if pr != nil {
		fmt.Printf("PR created: %s\n", pr.GetHTMLURL())
	} else {
		fmt.Println("No PR created (release already up to date)")
	}

	return nil
}

// RunPatchRelease executes the patch release workflow (creates 2 PRs)
func RunPatchRelease(cfg *ReleaseConfig) error {
	fmt.Println("=== Starting Patch Release Workflow ===")

	if err := cfg.Validate(); err != nil {
		return err
	}

	if err := checkRequirements(cfg); err != nil {
		return err
	}

	// Define PRs to create
	prConfigs := []PRConfig{
		{
			BranchName: fmt.Sprintf("update-docs-version-%s", cfg.CurrentRelease),
			Title:      fmt.Sprintf("Update docs and version for %s", cfg.CurrentRelease),
			Body: fmt.Sprintf(`## Update Documentation and Version for %s

This PR updates documentation and version files for the %s patch release.

### Changes
- Updated version to %s
- Updated documentation references
`, cfg.CurrentRelease, cfg.CurrentRelease, cfg.CurrentRelease),
			Labels: []string{"release", "version", "docs"},
		},
		{
			BranchName: fmt.Sprintf("update-testing-env-%s", cfg.CurrentRelease),
			Title:      fmt.Sprintf("Update testing environment for %s", cfg.CurrentRelease),
			Body: fmt.Sprintf(`## Update Testing Environment for %s

This PR updates test environment configurations for the %s patch release.

### Changes
- Updated docker-compose files with new version
`, cfg.CurrentRelease, cfg.CurrentRelease),
			Labels: []string{"release", "testing"},
		},
	}

	// For each PR config, create branch, make changes, commit, and push
	repo, err := OpenRepo(".")
	if err != nil {
		return err
	}

	// PR 1: Docs and version
	fmt.Println("\n--- Preparing PR 1: Docs and Version ---")
	if err := repo.EnsureBranch(prConfigs[0].BranchName); err != nil {
		return err
	}

	if err := UpdateVersion(cfg.CurrentRelease); err != nil {
		return err
	}
	if err := UpdateDocs(cfg.CurrentRelease); err != nil {
		return err
	}

	if _, err := repo.CommitAll(fmt.Sprintf("Update docs and version for %s", cfg.CurrentRelease), cfg.GitAuthorName, cfg.GitAuthorEmail); err != nil {
		return err
	}

	// PR 2: Test environment (only if LATEST_RELEASE is set)
	testEnvHasChanges := false
	if cfg.LatestRelease != "" {
		fmt.Println("\n--- Preparing PR 2: Test Environment ---")
		if err := repo.CheckoutBranch(cfg.BaseBranch); err != nil {
			return err
		}
		if err := repo.EnsureBranch(prConfigs[1].BranchName); err != nil {
			return err
		}

		if err := UpdateTestEnv(cfg.LatestRelease, cfg.CurrentRelease); err != nil {
			return err
		}

		committed, err := repo.CommitAll(fmt.Sprintf("Update testing environment for %s", cfg.CurrentRelease), cfg.GitAuthorName, cfg.GitAuthorEmail)
		if err != nil {
			return err
		}
		if committed {
			testEnvHasChanges = true
		} else {
			ahead, err := repo.HasCommitsAheadOf(cfg.BaseBranch)
			if err != nil {
				return err
			}
			testEnvHasChanges = ahead
			if !testEnvHasChanges {
				fmt.Println("No test environment changes to commit")
			}
		}
	} else {
		fmt.Println("\nSkipping test environment PR (LATEST_RELEASE not set)")
	}

	// Determine which branches to push and which PRs to create
	branchesToFinalize := []struct {
		branch string
		base   string
		config PRConfig
	}{
		{
			branch: prConfigs[0].BranchName,
			base:   cfg.BaseBranch,
			config: prConfigs[0],
		},
	}

	if testEnvHasChanges {
		branchesToFinalize = append(branchesToFinalize, struct {
			branch string
			base   string
			config PRConfig
		}{
			branch: prConfigs[1].BranchName,
			base:   cfg.BaseBranch,
			config: prConfigs[1],
		})
	}

	// Push and create PRs (skip in dry-run mode)
	if cfg.DryRun {
		fmt.Println("\nDRY RUN: Skipping push and PR creation")
		branchNames := make([]string, 0, len(branchesToFinalize))
		for _, item := range branchesToFinalize {
			branchNames = append(branchNames, item.branch)
		}
		fmt.Printf("Branches prepared: %v\n", branchNames)
		return nil
	}

	gh := NewGitHubClient(cfg.GitHubToken)
	var prs []*github.PullRequest
	for i, item := range branchesToFinalize {
		baseBranch := item.config.Base
		if baseBranch == "" {
			if cfg.ReleaseBranch != "" {
				baseBranch = cfg.ReleaseBranch
			} else {
				baseBranch = cfg.BaseBranch
			}
		}

		opts := PROptions{
			Owner:     cfg.ProjectOwner,
			Repo:      cfg.ProjectRepo,
			Title:     item.config.Title,
			Head:      item.config.BranchName,
			Base:      baseBranch,
			Body:      item.config.Body,
			Draft:     false,
			Reviewers: cfg.ProjectReviewers,
			Labels:    item.config.Labels,
		}

		pr, err := finalizePR(repo, gh, item.branch, item.base, opts)
		if err != nil {
			return fmt.Errorf("failed to finalize PR %d/%d: %w", i+1, len(branchesToFinalize), err)
		}
		if pr != nil {
			prs = append(prs, pr)
		}
	}

	fmt.Printf("\n=== Patch Release Workflow Complete ===\n")
	for i, pr := range prs {
		fmt.Printf("PR %d: %s\n", i+1, pr.GetHTMLURL())
	}
	if len(prs) == 0 {
		fmt.Println("No PRs created (release already up to date)")
	}

	return nil
}

// RunNextRelease executes the next release workflow (creates 2 PRs + backport PR)
func RunNextRelease(cfg *ReleaseConfig) error {
	fmt.Println("=== Starting Next Release Workflow ===")
	fmt.Println("Note: This workflow creates 2 PRs for version updates + 1 backport PR")

	if err := cfg.Validate(); err != nil {
		return err
	}

	// Implementation similar to RunPatchRelease but with additional backport PR
	// This is a placeholder for the full implementation
	fmt.Println("RunNextRelease - Full implementation pending")

	return fmt.Errorf("RunNextRelease not fully implemented yet")
}

// RunNextDevMinor executes the next dev minor workflow (creates 3 PRs)
func RunNextDevMinor(cfg *ReleaseConfig) error {
	fmt.Println("=== Starting Next Dev Minor Workflow ===")
	fmt.Println("Note: This workflow creates 3 PRs (version + docs + test-env)")

	if err := cfg.Validate(); err != nil {
		return err
	}

	// Implementation similar to RunPatchRelease but with 3 PRs
	// This is a placeholder for the full implementation
	fmt.Println("RunNextDevMinor - Full implementation pending")

	return fmt.Errorf("RunNextDevMinor not fully implemented yet")
}

// finalizePR pushes a branch when it has new commits and creates or reuses an open PR.
func finalizePR(repo *GitRepo, gh *GitHubClient, branchName, baseBranch string, opts PROptions) (*github.PullRequest, error) {
	if err := repo.CheckoutBranch(branchName); err != nil {
		return nil, err
	}

	existingPR, found, err := gh.FindOpenPR(opts.Owner, opts.Repo, opts.Head, opts.Base)
	if err != nil {
		return nil, err
	}
	if found {
		return existingPR, nil
	}

	ahead, err := repo.HasCommitsAheadOf(baseBranch)
	if err != nil {
		return nil, err
	}
	if !ahead {
		fmt.Printf("No new commits on %s compared to %s; skipping push and PR creation\n", branchName, baseBranch)
		return nil, nil
	}

	if err := repo.Push("origin"); err != nil {
		return nil, err
	}

	return gh.CreatePR(opts)
}
