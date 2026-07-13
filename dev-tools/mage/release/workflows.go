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

// PR label sets match elastic-vault-github-plugin-prod release PRs (e.g. #48155, #49435).
var (
	releasePRLabels   = []string{"release", "Team:Automation", "skip-changelog"}
	patchDocsPRLabels = []string{"docs", "in progress", "release", "Team:Automation", "skip-changelog"}
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

// RunMajorMinorRelease executes the feature-freeze workflow:
// 1. Creates the release branch
// 2. Opens a version/docs PR for NEXT_RELEASE (prepare-next-release)
// 3. Opens a test-environment PR for NEXT_RELEASE
func RunMajorMinorRelease(cfg *ReleaseConfig) error {
	fmt.Println("=== Starting Major/Minor Release Workflow ===")

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

	releaseBranch := cfg.ReleaseBranch
	nextRelease := cfg.NextRelease

	fmt.Printf("Creating release branch: %s\n", releaseBranch)
	if err := repo.EnsureBranchFrom(cfg.BaseBranch, releaseBranch); err != nil {
		return err
	}

	versionBranch := fmt.Sprintf("update-version-next-%s", nextRelease)
	fmt.Printf("\n--- Creating PR 1: Version to NEXT_RELEASE ---\n")
	if err := repo.EnsureBranchFrom(releaseBranch, versionBranch); err != nil {
		return err
	}
	if err := UpdateVersion(nextRelease); err != nil {
		return err
	}
	if err := UpdateStackVersion(nextRelease); err != nil {
		return err
	}
	if err := RunMakeUpdate(); err != nil {
		return err
	}
	versionCommitMsg := fmt.Sprintf("[Release] Update version to %s", nextRelease)
	if _, err := repo.CommitAll(versionCommitMsg, cfg.GitAuthorName, cfg.GitAuthorEmail); err != nil {
		return err
	}

	testEnvBranch := fmt.Sprintf("update-testing-env-next-%s", nextRelease)
	fmt.Printf("\n--- Creating PR 2: Test Environments to NEXT_RELEASE ---\n")
	if err := repo.EnsureBranchFrom(releaseBranch, testEnvBranch); err != nil {
		return err
	}
	if err := UpdateTestEnv(cfg.CurrentRelease, nextRelease); err != nil {
		return err
	}
	testEnvCommitMsg := fmt.Sprintf("[Release] Update test environments for %s", nextRelease)
	if _, err := repo.CommitAll(testEnvCommitMsg, cfg.GitAuthorName, cfg.GitAuthorEmail); err != nil {
		return err
	}

	branchesToFinalize := []workflowPR{
		{
			branch: versionBranch,
			base:   releaseBranch,
			opts: PROptions{
				Owner:     cfg.ProjectOwner,
				Repo:      cfg.ProjectRepo,
				Title:     fmt.Sprintf("[Release] Update version to %s", nextRelease),
				Head:      versionBranch,
				Base:      releaseBranch,
				Body:      nextVersionPRBody(nextRelease, cfg.CurrentRelease),
				Reviewers: cfg.ProjectReviewers,
				Labels:    releasePRLabels,
			},
		},
		{
			branch: testEnvBranch,
			base:   releaseBranch,
			opts: PROptions{
				Owner:     cfg.ProjectOwner,
				Repo:      cfg.ProjectRepo,
				Title:     fmt.Sprintf("[Release] Update test environments for %s", nextRelease),
				Head:      testEnvBranch,
				Base:      releaseBranch,
				Body:      nextTestEnvPRBody(nextRelease, cfg.CurrentRelease),
				Reviewers: cfg.ProjectReviewers,
				Labels:    releasePRLabels,
			},
		},
	}

	if cfg.DryRun {
		fmt.Println("\nDRY RUN: Skipping push and PR creation")
		fmt.Printf("Release branch prepared: %s\n", releaseBranch)
		for _, item := range branchesToFinalize {
			fmt.Printf("Branch prepared: %s\n", item.branch)
		}
		return nil
	}

	if err := repo.CheckoutBranch(releaseBranch); err != nil {
		return err
	}
	if err := repo.Push("origin"); err != nil {
		return err
	}

	gh := NewGitHubClient(cfg.GitHubToken)
	var prs []*github.PullRequest
	for i, item := range branchesToFinalize {
		pr, err := finalizePR(repo, gh, item.branch, item.base, item.opts)
		if err != nil {
			return fmt.Errorf("failed to finalize PR %d/%d: %w", i+1, len(branchesToFinalize), err)
		}
		if pr != nil {
			prs = append(prs, pr)
		}
	}

	fmt.Printf("\n=== Major/Minor Release Workflow Complete ===\n")
	fmt.Printf("Release branch created: %s\n", releaseBranch)
	for i, pr := range prs {
		fmt.Printf("PR %d: %s\n", i+1, pr.GetHTMLURL())
	}
	if len(prs) == 0 {
		fmt.Println("No PRs created (release already up to date)")
	}
	fmt.Println("\nNote: Release notes PR should be created separately using release:runChangelog")

	return nil
}

// RunPatchRelease executes the patch release workflow (creates up to 3 PRs)
func RunPatchRelease(cfg *ReleaseConfig) error {
	fmt.Println("=== Starting Patch Release Workflow ===")

	if err := cfg.Validate(); err != nil {
		return err
	}

	if err := checkRequirements(cfg); err != nil {
		return err
	}

	if cfg.LatestRelease == "" {
		return fmt.Errorf("LATEST_RELEASE is required for patch releases")
	}

	repo, err := OpenRepo(".")
	if err != nil {
		return err
	}

	releaseBranch := cfg.ReleaseBranch
	if releaseBranch == "" {
		releaseBranch = inferReleaseBranch(cfg.CurrentRelease)
	}

	versionBranch := fmt.Sprintf("update-version-%s", cfg.CurrentRelease)
	fmt.Println("\n--- Creating PR 1: Version ---")
	if err := repo.EnsureBranchFrom(releaseBranch, versionBranch); err != nil {
		return err
	}
	if err := UpdateVersion(cfg.CurrentRelease); err != nil {
		return err
	}
	versionCommitMsg := "[Release] update version"
	if _, err := repo.CommitAll(versionCommitMsg, cfg.GitAuthorName, cfg.GitAuthorEmail); err != nil {
		return err
	}

	docsBranch := fmt.Sprintf("update-docs-%s", cfg.CurrentRelease)
	fmt.Println("\n--- Creating PR 2: Docs ---")
	if err := repo.EnsureBranchFrom(releaseBranch, docsBranch); err != nil {
		return err
	}
	if err := UpdateDocsWithOptions(DocsUpdateOptions{
		BaseBranch:     releaseBranch,
		CurrentVersion: cfg.CurrentRelease,
		ReleaseBranch:  releaseBranch,
	}); err != nil {
		return err
	}
	docsCommitMsg := "docs: update docs"
	if _, err := repo.CommitAll(docsCommitMsg, cfg.GitAuthorName, cfg.GitAuthorEmail); err != nil {
		return err
	}

	testEnvBranch := fmt.Sprintf("update-testing-env-%s", cfg.CurrentRelease)
	fmt.Println("\n--- Creating PR 3: Test Environment ---")
	if err := repo.EnsureBranchFrom(releaseBranch, testEnvBranch); err != nil {
		return err
	}
	if err := UpdateTestEnv(cfg.LatestRelease, cfg.CurrentRelease); err != nil {
		return err
	}
	testEnvCommitMsg := "[Release] update test environment"
	if _, err := repo.CommitAll(testEnvCommitMsg, cfg.GitAuthorName, cfg.GitAuthorEmail); err != nil {
		return err
	}

	branchesToFinalize := []workflowPR{
		{
			branch: versionBranch,
			base:   releaseBranch,
			opts: PROptions{
				Owner:     cfg.ProjectOwner,
				Repo:      cfg.ProjectRepo,
				Title:     fmt.Sprintf("[Release] update version %s", cfg.CurrentRelease),
				Head:      versionBranch,
				Base:      releaseBranch,
				Body:      patchVersionPRBody(cfg.CurrentRelease),
				Reviewers: cfg.ProjectReviewers,
				Labels:    releasePRLabels,
			},
		},
		{
			branch: docsBranch,
			base:   releaseBranch,
			opts: PROptions{
				Owner:     cfg.ProjectOwner,
				Repo:      cfg.ProjectRepo,
				Title:     fmt.Sprintf("docs: update docs versions %s", cfg.CurrentRelease),
				Head:      docsBranch,
				Base:      releaseBranch,
				Body:      patchDocsPRBody(cfg.CurrentRelease),
				Reviewers: cfg.ProjectReviewers,
				Labels:    patchDocsPRLabels,
			},
		},
		{
			branch: testEnvBranch,
			base:   releaseBranch,
			opts: PROptions{
				Owner:     cfg.ProjectOwner,
				Repo:      cfg.ProjectRepo,
				Title:     fmt.Sprintf("[Release] Update test environments for %s", cfg.CurrentRelease),
				Head:      testEnvBranch,
				Base:      releaseBranch,
				Body:      patchTestEnvPRBody(cfg.CurrentRelease),
				Reviewers: cfg.ProjectReviewers,
				Labels:    releasePRLabels,
			},
		},
	}

	if cfg.DryRun {
		fmt.Println("\nDRY RUN: Skipping push and PR creation")
		for _, item := range branchesToFinalize {
			fmt.Printf("Branch prepared: %s\n", item.branch)
		}
		return nil
	}

	gh := NewGitHubClient(cfg.GitHubToken)
	var prs []*github.PullRequest
	for i, item := range branchesToFinalize {
		pr, err := finalizePR(repo, gh, item.branch, item.base, item.opts)
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

// RunNextRelease executes prepare-next-release only (2 PRs onto the release branch).
func RunNextRelease(cfg *ReleaseConfig) error {
	return RunMajorMinorRelease(cfg)
}

type workflowPR struct {
	branch string
	base   string
	opts   PROptions
}

func nextVersionPRBody(nextRelease, currentRelease string) string {
	return fmt.Sprintf(`Updates references to the new release %s.

Merge after the release %s.
`, nextRelease, currentRelease)
}

func nextTestEnvPRBody(nextRelease, currentRelease string) string {
	return fmt.Sprintf(`Update test environment versions to the correct Elastic Stack version.

Merge only after the release of %s.
`, currentRelease)
}

func patchDocsPRBody(currentRelease string) string {
	return fmt.Sprintf(`Updates docs versions to %s.

Merge before the final Release build.
`, currentRelease)
}

func patchVersionPRBody(currentRelease string) string {
	return fmt.Sprintf(`Updates version to %s.

Merge before the final Release build.
`, currentRelease)
}

func patchTestEnvPRBody(currentRelease string) string {
	return fmt.Sprintf(`Updates test environments for %s.
`, currentRelease)
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
		gh.ensurePRLabels(opts.Owner, opts.Repo, existingPR.GetNumber(), opts.Labels)
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
