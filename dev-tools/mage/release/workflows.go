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
	ffReleasePRLabels = []string{"release", "docs", "in progress", "Team:Automation", "skip-changelog"}
)

// Feature-freeze merge-timing labels (number = RM merge order).
const (
	mergeLabelFFDay        = "merge:1-ff-day"
	mergeLabelAfterBranch  = "merge:2-after-branch"
	mergeLabelAfterImages  = "merge:3-after-images"
	mergeLabelAfterRelease = "merge:4-after-release"
)

// Patch-release merge-timing labels.
const (
	mergeLabelBeforeBuild = "merge:1-before-build"
)

func backportLabel(releaseBranch string) string {
	return fmt.Sprintf("backport-%s", releaseBranch)
}

func prAMainLabels(releaseBranch string) []string {
	return []string{"release", "impact:critical", backportLabel(releaseBranch), "skip-changelog", "Team:Automation", mergeLabelFFDay}
}

func prBReleaseLabels() []string {
	return append(append([]string{}, ffReleasePRLabels...), mergeLabelAfterBranch)
}

func prCMainLabels(releaseBranch string) []string {
	return []string{"release", "docs", "in progress", backportLabel(releaseBranch), "skip-changelog", "Team:Automation", mergeLabelAfterImages}
}

func prDNextPatchLabels() []string {
	return append(append([]string{}, releasePRLabels...), mergeLabelAfterRelease)
}

func patchBeforeBuildPRLabels() []string {
	// Same label set as ff-release / former docs PR (includes docs + in progress).
	return append(append([]string{}, patchDocsPRLabels...), mergeLabelBeforeBuild)
}

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
// 1. Creates the release branch from BASE_BRANCH
// 2. Opens PR-A on main (backport rule + next minor version)
// 3. Opens PR-B on release branch (ff-release)
// 4. Opens PR-C on main (docs + test env for next minor)
// 5. Opens PR-D on release branch (next patch prep)
func RunMajorMinorRelease(cfg *ReleaseConfig) error {
	fmt.Println("=== Starting Major/Minor Release Workflow ===")

	if err := cfg.EnsureLatestRelease(); err != nil {
		return err
	}

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

	if err := ensureMajorMinorCurrentReleaseMatchesBase(repo, cfg); err != nil {
		return err
	}

	releaseBranch := cfg.ReleaseBranch

	fmt.Printf("Creating release branch: %s\n", releaseBranch)
	if err := repo.EnsureBranchFrom(cfg.BaseBranch, releaseBranch); err != nil {
		return err
	}

	prA, err := prepMainBackportAndVersion(repo, cfg)
	if err != nil {
		return err
	}
	prB, err := prepFFRelease(repo, cfg)
	if err != nil {
		return err
	}
	prC, err := prepMainDocsAndTestEnv(repo, cfg)
	if err != nil {
		return err
	}
	prD, err := prepNextPatchOnReleaseBranch(repo, cfg)
	if err != nil {
		return err
	}

	branchesToFinalize := []workflowPR{prA, prB, prC, prD}

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
	fmt.Println("\nNote: Release notes PR should be created separately via .github/workflows/release-notes.yml")

	return nil
}

func prepMainBackportAndVersion(repo *GitRepo, cfg *ReleaseConfig) (workflowPR, error) {
	branch := fmt.Sprintf("ff-prep-main-%s", cfg.CurrentRelease)
	fmt.Printf("\n--- Preparing PR-A: backport rule + version %s on %s ---\n", cfg.NextProjectMinorVersion, cfg.BaseBranch)

	if err := repo.EnsureBranchFrom(cfg.BaseBranch, branch); err != nil {
		return workflowPR{}, err
	}
	if err := UpdateMergify(cfg.ReleaseBranch); err != nil {
		return workflowPR{}, err
	}
	if err := UpdateVersion(cfg.NextProjectMinorVersion); err != nil {
		return workflowPR{}, err
	}
	commitMsg := fmt.Sprintf("[Release %s] Prepare main for %s and mergify backport-%s", cfg.CurrentRelease, cfg.NextProjectMinorVersion, cfg.ReleaseBranch)
	if _, err := repo.CommitAll(commitMsg, cfg.GitAuthorName, cfg.GitAuthorEmail); err != nil {
		return workflowPR{}, err
	}

	return workflowPR{
		branch: branch,
		base:   cfg.BaseBranch,
		opts: PROptions{
			Owner:     cfg.ProjectOwner,
			Repo:      cfg.ProjectRepo,
			Title:     fmt.Sprintf("[Release %s] Prepare main for %s and mergify backport-%s", cfg.CurrentRelease, cfg.NextProjectMinorVersion, cfg.ReleaseBranch),
			Head:      branch,
			Base:      cfg.BaseBranch,
			Body:      prAMainBody(cfg),
			Reviewers: cfg.ProjectReviewers,
			Labels:    prAMainLabels(cfg.ReleaseBranch),
		},
	}, nil
}

func prepFFRelease(repo *GitRepo, cfg *ReleaseConfig) (workflowPR, error) {
	branch := fmt.Sprintf("ff-release-%s", cfg.CurrentRelease)
	fmt.Printf("\n--- Preparing PR-B: ff-release %s on %s ---\n", cfg.CurrentRelease, cfg.ReleaseBranch)

	if err := repo.EnsureBranchFrom(cfg.ReleaseBranch, branch); err != nil {
		return workflowPR{}, err
	}
	if err := UpdateVersion(cfg.CurrentRelease); err != nil {
		return workflowPR{}, err
	}
	if err := UpdateDocsWithOptions(DocsUpdateOptions{
		BaseBranch:     cfg.BaseBranch,
		CurrentVersion: cfg.CurrentRelease,
		ReleaseBranch:  cfg.ReleaseBranch,
		DocBranch:      "main",
	}); err != nil {
		return workflowPR{}, err
	}
	if err := UpdateTestEnv(cfg.LatestRelease, cfg.CurrentRelease); err != nil {
		return workflowPR{}, err
	}
	if err := runMakeUpdate(); err != nil {
		return workflowPR{}, err
	}
	commitMsg := fmt.Sprintf("[Release %s] ff-release: update versions %s", cfg.CurrentRelease, cfg.CurrentRelease)
	if _, err := repo.CommitAll(commitMsg, cfg.GitAuthorName, cfg.GitAuthorEmail); err != nil {
		return workflowPR{}, err
	}

	return workflowPR{
		branch: branch,
		base:   cfg.ReleaseBranch,
		opts: PROptions{
			Owner:     cfg.ProjectOwner,
			Repo:      cfg.ProjectRepo,
			Title:     fmt.Sprintf("[Release %s] ff-release: update versions %s", cfg.CurrentRelease, cfg.CurrentRelease),
			Head:      branch,
			Base:      cfg.ReleaseBranch,
			Body:      prBReleaseBody(cfg),
			Reviewers: cfg.ProjectReviewers,
			Labels:    prBReleaseLabels(),
		},
	}, nil
}

func prepMainDocsAndTestEnv(repo *GitRepo, cfg *ReleaseConfig) (workflowPR, error) {
	branch := fmt.Sprintf("ff-prep-main-docs-env-%s", cfg.NextProjectMinorVersion)
	fmt.Printf("\n--- Preparing PR-C: docs + test env %s on %s ---\n", cfg.NextProjectMinorVersion, cfg.BaseBranch)

	if err := repo.EnsureBranchFrom(cfg.BaseBranch, branch); err != nil {
		return workflowPR{}, err
	}
	// beats.mak prepare-next-dev-minor: update-docs BASE=main CURRENT=next RELEASE=main
	if err := UpdateDocsWithOptions(DocsUpdateOptions{
		BaseBranch:     cfg.BaseBranch,
		CurrentVersion: cfg.NextProjectMinorVersion,
		ReleaseBranch:  cfg.BaseBranch,
		DocBranch:      "main",
	}); err != nil {
		return workflowPR{}, err
	}
	if err := UpdateTestEnv(cfg.LatestRelease, cfg.NextProjectMinorVersion); err != nil {
		return workflowPR{}, err
	}
	commitMsg := fmt.Sprintf("[Release %s] Update docs and test environment for %s", cfg.CurrentRelease, cfg.NextProjectMinorVersion)
	if _, err := repo.CommitAll(commitMsg, cfg.GitAuthorName, cfg.GitAuthorEmail); err != nil {
		return workflowPR{}, err
	}

	return workflowPR{
		branch: branch,
		base:   cfg.BaseBranch,
		opts: PROptions{
			Owner:     cfg.ProjectOwner,
			Repo:      cfg.ProjectRepo,
			Title:     fmt.Sprintf("[Release %s] Update docs and test environment for %s", cfg.CurrentRelease, cfg.NextProjectMinorVersion),
			Head:      branch,
			Base:      cfg.BaseBranch,
			Body:      prCMainBody(cfg),
			Reviewers: cfg.ProjectReviewers,
			Labels:    prCMainLabels(cfg.ReleaseBranch),
		},
	}, nil
}

func prepNextPatchOnReleaseBranch(repo *GitRepo, cfg *ReleaseConfig) (workflowPR, error) {
	branch := fmt.Sprintf("ff-prep-next-patch-%s", cfg.NextRelease)
	fmt.Printf("\n--- Preparing next-patch prep: %s on %s ---\n", cfg.NextRelease, cfg.ReleaseBranch)

	if err := repo.EnsureBranchFrom(cfg.ReleaseBranch, branch); err != nil {
		return workflowPR{}, err
	}
	// beats.mak prepare-next-release: update-version + update-project + update-test-env
	// (no UpdateStackVersion / update-docs — matches elasticmachine PRs that only touch version.go + test-env).
	if err := UpdateVersion(cfg.NextRelease); err != nil {
		return workflowPR{}, err
	}
	if err := runMakeUpdate(); err != nil {
		return workflowPR{}, err
	}
	if err := UpdateTestEnv(cfg.CurrentRelease, cfg.NextRelease); err != nil {
		return workflowPR{}, err
	}
	commitMsg := fmt.Sprintf("[Release %s] Update version to %s and test environments", cfg.CurrentRelease, cfg.NextRelease)
	if _, err := repo.CommitAll(commitMsg, cfg.GitAuthorName, cfg.GitAuthorEmail); err != nil {
		return workflowPR{}, err
	}

	return workflowPR{
		branch: branch,
		base:   cfg.ReleaseBranch,
		opts: PROptions{
			Owner:     cfg.ProjectOwner,
			Repo:      cfg.ProjectRepo,
			Title:     fmt.Sprintf("[Release %s] Update version to %s and test environments", cfg.CurrentRelease, cfg.NextRelease),
			Head:      branch,
			Base:      cfg.ReleaseBranch,
			Body:      prDNextPatchBody(cfg),
			Reviewers: cfg.ProjectReviewers,
			Labels:    prDNextPatchLabels(),
		},
	}, nil
}

func prAMainBody(cfg *ReleaseConfig) string {
	return fmt.Sprintf(`## [Release %s]

Prepares %s for the %s feature freeze.

- Adds Mergify backport rule for branch %s (label %s)
- Bumps libbeat/version/version.go to %s (next minor)

**Merge:** before release branch work is finalized (%s).
`, cfg.CurrentRelease, cfg.BaseBranch, cfg.CurrentRelease, cfg.ReleaseBranch, backportLabel(cfg.ReleaseBranch), cfg.NextProjectMinorVersion, mergeLabelFFDay)
}

func prBReleaseBody(cfg *ReleaseConfig) string {
	return fmt.Sprintf(`## [Release %s]

Feature-freeze release branch updates for %s (version, docs, test env, make update).

**Merge:** as soon as the %s branch exists (%s).
`, cfg.CurrentRelease, cfg.CurrentRelease, cfg.ReleaseBranch, mergeLabelAfterBranch)
}

func prCMainBody(cfg *ReleaseConfig) string {
	return fmt.Sprintf(`## [Release %s]

Updates documentation and test environment on %s for the next minor %s.

**Merge:** after the %s branch is created (%s). CI may stay red until Docker images exist.
`, cfg.CurrentRelease, cfg.BaseBranch, cfg.NextProjectMinorVersion, cfg.ReleaseBranch, mergeLabelAfterImages)
}

func prDNextPatchBody(cfg *ReleaseConfig) string {
	return fmt.Sprintf(`## [Release %s]

Updates the %s branch after release of %s (former update-version + update-test-env).

- Bumps libbeat/version/version.go to %s
- Updates test environments so stack tags point at %s

**Merge:** after the release of %s (%s).
`, cfg.CurrentRelease, cfg.ReleaseBranch, cfg.CurrentRelease, cfg.NextRelease, cfg.CurrentRelease, cfg.CurrentRelease, mergeLabelAfterRelease)
}

// ensureMajorMinorCurrentReleaseMatchesBase checks out BASE_BRANCH
// and requires libbeat/version/version.go to equal CURRENT_RELEASE. After the previous
// minor's prepare-next-dev version bump, BASE_BRANCH already carries the version being frozen.
func ensureMajorMinorCurrentReleaseMatchesBase(repo *GitRepo, cfg *ReleaseConfig) error {
	base := cfg.BaseBranch
	if base == "" {
		base = "main"
	}
	if err := repo.CheckoutBranch(base); err != nil {
		return err
	}
	branchVersion, err := ReadBeatVersion()
	if err != nil {
		return err
	}
	if branchVersion != cfg.CurrentRelease {
		return fmt.Errorf(
			"CURRENT_RELEASE=%s does not match version on %s (%s in libbeat/version/version.go); "+
				"set CURRENT_RELEASE to the version already on %s (the minor being feature-frozen)",
			cfg.CurrentRelease, base, branchVersion, base,
		)
	}
	fmt.Printf("Verified CURRENT_RELEASE=%s matches %s on branch %s\n", cfg.CurrentRelease, branchVersion, base)
	return nil
}

// RunPatchRelease executes the patch release workflow on an existing release branch:
// 1. Opens PR-A (docs only for CURRENT_RELEASE — before build; version already on branch)
// 2. Opens PR-B (next patch version + test env — after release; former prepare-next-release)
func RunPatchRelease(cfg *ReleaseConfig) error {
	fmt.Println("=== Starting Patch Release Workflow ===")

	if err := cfg.EnsureLatestRelease(); err != nil {
		return err
	}

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

	if cfg.ReleaseBranch == "" {
		cfg.ReleaseBranch = inferReleaseBranch(cfg.CurrentRelease)
	}

	if err := ensurePatchCurrentReleaseMatchesBranch(repo, cfg); err != nil {
		return err
	}

	prA, err := prepPatchBeforeBuild(repo, cfg)
	if err != nil {
		return err
	}
	prB, err := prepNextPatchOnReleaseBranch(repo, cfg)
	if err != nil {
		return err
	}

	branchesToFinalize := []workflowPR{prA, prB}

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
	fmt.Println("\nNote: Release notes PR should be created separately via .github/workflows/release-notes.yml")

	return nil
}

// ensurePatchCurrentReleaseMatchesBranch checks out the release branch and requires
// libbeat/version/version.go to equal CURRENT_RELEASE. After the previous patch's
// prepare-next-release merge, the branch already carries the version being released.
func ensurePatchCurrentReleaseMatchesBranch(repo *GitRepo, cfg *ReleaseConfig) error {
	if err := repo.CheckoutBranch(cfg.ReleaseBranch); err != nil {
		return err
	}
	branchVersion, err := ReadBeatVersion()
	if err != nil {
		return err
	}
	if branchVersion != cfg.CurrentRelease {
		return fmt.Errorf(
			"CURRENT_RELEASE=%s does not match version on branch %s (%s in libbeat/version/version.go); "+
				"set CURRENT_RELEASE to the version already on the release branch (the patch being released)",
			cfg.CurrentRelease, cfg.ReleaseBranch, branchVersion,
		)
	}
	fmt.Printf("Verified CURRENT_RELEASE=%s matches %s on branch %s\n", cfg.CurrentRelease, branchVersion, cfg.ReleaseBranch)
	return nil
}

func prepPatchBeforeBuild(repo *GitRepo, cfg *ReleaseConfig) (workflowPR, error) {
	branch := fmt.Sprintf("patch-release-%s", cfg.CurrentRelease)
	fmt.Printf("\n--- Preparing PR-A: docs for %s on %s ---\n", cfg.CurrentRelease, cfg.ReleaseBranch)

	if err := repo.EnsureBranchFrom(cfg.ReleaseBranch, branch); err != nil {
		return workflowPR{}, err
	}
	// Former prepare-patch-release / elasticmachine docs PR: docs + K8s only.
	// version.go is already CURRENT; test-env advances after release (PR-B).
	if err := UpdateDocsWithOptions(DocsUpdateOptions{
		BaseBranch:       cfg.ReleaseBranch,
		CurrentVersion:   cfg.CurrentRelease,
		ReleaseBranch:    cfg.ReleaseBranch,
		IncludeHeartbeat: true,
	}); err != nil {
		return workflowPR{}, err
	}
	commitMsg := fmt.Sprintf("[Release %s] Update docs versions %s", cfg.CurrentRelease, cfg.CurrentRelease)
	if _, err := repo.CommitAll(commitMsg, cfg.GitAuthorName, cfg.GitAuthorEmail); err != nil {
		return workflowPR{}, err
	}

	return workflowPR{
		branch: branch,
		base:   cfg.ReleaseBranch,
		opts: PROptions{
			Owner:     cfg.ProjectOwner,
			Repo:      cfg.ProjectRepo,
			Title:     fmt.Sprintf("[Release %s] Update docs versions %s", cfg.CurrentRelease, cfg.CurrentRelease),
			Head:      branch,
			Base:      cfg.ReleaseBranch,
			Body:      patchBeforeBuildPRBody(cfg.CurrentRelease),
			Reviewers: cfg.ProjectReviewers,
			Labels:    patchBeforeBuildPRLabels(),
		},
	}, nil
}

type workflowPR struct {
	branch string
	base   string
	opts   PROptions
}

func patchBeforeBuildPRBody(currentRelease string) string {
	return fmt.Sprintf(`## [Release %s]

Updates docs versions to %s (former prepare-patch-release docs PR).

- Updates :stack-version: / :doc-branch: and K8s manifests (including heartbeat)
- Does **not** bump libbeat/version/version.go (already %s on the release branch)
- Does **not** update test environments (that happens after release)

**Merge:** before the final Release build (%s).
`, currentRelease, currentRelease, currentRelease, mergeLabelBeforeBuild)
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
