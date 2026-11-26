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

	"github.com/google/go-github/v68/github"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
)

// PR label sets match elastic-vault-github-plugin-prod release PRs (e.g. #48155, #49435).
var (
	releasePRLabels   = []string{"release", "Team:Automation", "skip-changelog"}
	ffReleasePRLabels = []string{"release", "docs", "in progress", "Team:Automation", "skip-changelog"}
)

// Feature-freeze merge-timing labels (number = RM merge order).
const (
	mergeLabelFFDay        = "merge:1-ff-day"
	mergeLabelAfterBranch  = "merge:2-after-branch"
	mergeLabelAfterImages  = "merge:3-after-images"
	mergeLabelAfterRelease = "merge:4-after-release"
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

// patchVersionBumpLabels labels the post-release version bump (ingest-dev prepare-next-patch).
func patchVersionBumpLabels() []string {
	return append(append([]string{}, releasePRLabels...), mergeLabelAfterRelease)
}

// checkRequirements validates prerequisites before running a release workflow
func checkRequirements(cfg *ReleaseConfig) error {
	if err := cfg.EnsureParsedVersions(); err != nil {
		return err
	}

	v := cfg.NextReleaseVersion
	if (v.Major() == 6 || v.Major() == 7 || v.Major() == 8) && v.Patch() == 0 {
		return fmt.Errorf("minor releases for version %d.x are deprecated and blocked", v.Major())
	}

	repo, err := openRepo(".")
	if err != nil {
		return err
	}

	clean, err := repo.isClean()
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
//
// Next-patch bumps on the release branch are intentionally omitted here; they
// are created only by RunPatchRelease so patch and minor/FF stay independent.
func RunMajorMinorRelease(cfg *ReleaseConfig) error {
	fmt.Println("=== Starting Major/Minor Release Workflow ===")

	if err := ensureLatestRelease(cfg); err != nil {
		return err
	}

	if err := validateReleaseConfig(cfg); err != nil {
		return err
	}

	if err := checkRequirements(cfg); err != nil {
		return err
	}

	repo, err := openRepo(".")
	if err != nil {
		return err
	}

	if err := ensureMajorMinorNextReleaseMatchesBase(repo, cfg); err != nil {
		return err
	}

	releaseBranch := cfg.ReleaseBranch

	fmt.Printf("Creating release branch: %s\n", releaseBranch)
	if err := repo.ensureBranchFrom(cfg.BaseBranch, releaseBranch); err != nil {
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

	branchesToFinalize := []workflowPR{prA, prB, prC}

	if cfg.DryRun {
		fmt.Println("\nDRY RUN: Skipping push and PR creation")
		fmt.Printf("Release branch prepared: %s\n", releaseBranch)
		for _, item := range branchesToFinalize {
			fmt.Printf("Branch prepared: %s\n", item.branch)
		}
		warnEnsureReleaseIssueTracker(cfg, nil)
		return nil
	}

	if err := repo.checkoutBranch(releaseBranch); err != nil {
		return err
	}
	if err := repo.push("origin", cfg.GitHubToken); err != nil {
		return err
	}

	gh := newGitHubClient(cfg.GitHubToken)
	results, err := finalizeWorkflowPRs(repo, gh, cfg.GitHubToken, branchesToFinalize)
	if err != nil {
		return err
	}

	fmt.Printf("\n=== Major/Minor Release Workflow Complete ===\n")
	fmt.Printf("Release branch created: %s\n", releaseBranch)
	printWorkflowPRResults(results)
	fmt.Println("\nNote: Release notes PR should be created separately via .github/workflows/release-notes.yml")

	warnEnsureReleaseIssueTracker(cfg, prsFromWorkflowResults(results))
	return nil
}

func prepMainBackportAndVersion(repo *gitRepo, cfg *ReleaseConfig) (workflowPR, error) {
	branch := fmt.Sprintf("ff-prep-main-%s", cfg.NextRelease)
	fmt.Printf("\n--- Preparing PR-A: backport rule + version %s on %s ---\n", cfg.NextProjectMinorVersion, cfg.BaseBranch)

	if err := repo.ensureBranchFrom(cfg.BaseBranch, branch); err != nil {
		return workflowPR{}, err
	}
	if err := UpdateMergify(cfg.ReleaseBranch); err != nil {
		return workflowPR{}, err
	}
	if err := UpdateVersion(cfg.NextProjectMinorVersion); err != nil {
		return workflowPR{}, err
	}
	commitMsg := fmt.Sprintf("[Release][%s] Prepare main for %s and mergify backport-%s", cfg.NextRelease, cfg.NextProjectMinorVersion, cfg.ReleaseBranch)
	if _, err := repo.commitAll(commitMsg, cfg.GitAuthorName, cfg.GitAuthorEmail); err != nil {
		return workflowPR{}, err
	}

	return workflowPR{
		branch: branch,
		base:   cfg.BaseBranch,
		opts: prOptions{
			Owner:     cfg.ProjectOwner,
			Repo:      cfg.ProjectRepo,
			Title:     fmt.Sprintf("[Release][%s] Prepare main for %s and mergify backport-%s", cfg.NextRelease, cfg.NextProjectMinorVersion, cfg.ReleaseBranch),
			Head:      branch,
			Base:      cfg.BaseBranch,
			Body:      prAMainBody(cfg),
			Reviewers: cfg.ProjectReviewers,
			Labels:    prAMainLabels(cfg.ReleaseBranch),
		},
	}, nil
}

func prepFFRelease(repo *gitRepo, cfg *ReleaseConfig) (workflowPR, error) {
	branch := fmt.Sprintf("ff-release-%s", cfg.NextRelease)
	fmt.Printf("\n--- Preparing PR-B: ff-release %s on %s ---\n", cfg.NextRelease, cfg.ReleaseBranch)

	if err := repo.ensureBranchFrom(cfg.ReleaseBranch, branch); err != nil {
		return workflowPR{}, err
	}
	if err := UpdateVersion(cfg.NextRelease); err != nil {
		return workflowPR{}, err
	}
	if err := updateDocsWithOptions(docsUpdateOptions{
		BaseBranch:     cfg.BaseBranch,
		CurrentVersion: cfg.NextRelease,
		ReleaseBranch:  cfg.ReleaseBranch,
		DocBranch:      "main",
	}); err != nil {
		return workflowPR{}, err
	}
	if err := UpdateTestEnv(cfg.LatestRelease); err != nil {
		return workflowPR{}, err
	}
	if err := runMakeUpdate(); err != nil {
		return workflowPR{}, err
	}
	commitMsg := fmt.Sprintf("[Release][%s] ff-release: update versions %s", cfg.NextRelease, cfg.NextRelease)
	if _, err := repo.commitAll(commitMsg, cfg.GitAuthorName, cfg.GitAuthorEmail); err != nil {
		return workflowPR{}, err
	}

	return workflowPR{
		branch: branch,
		base:   cfg.ReleaseBranch,
		opts: prOptions{
			Owner:     cfg.ProjectOwner,
			Repo:      cfg.ProjectRepo,
			Title:     fmt.Sprintf("[Release][%s] ff-release: update versions %s", cfg.NextRelease, cfg.NextRelease),
			Head:      branch,
			Base:      cfg.ReleaseBranch,
			Body:      prBReleaseBody(cfg),
			Reviewers: cfg.ProjectReviewers,
			Labels:    prBReleaseLabels(),
		},
	}, nil
}

func prepMainDocsAndTestEnv(repo *gitRepo, cfg *ReleaseConfig) (workflowPR, error) {
	branch := fmt.Sprintf("ff-prep-main-docs-env-%s", cfg.NextProjectMinorVersion)
	fmt.Printf("\n--- Preparing PR-C: docs + test env %s on %s ---\n", cfg.NextProjectMinorVersion, cfg.BaseBranch)

	if err := repo.ensureBranchFrom(cfg.BaseBranch, branch); err != nil {
		return workflowPR{}, err
	}
	// beats.mak prepare-next-dev-minor: update-docs BASE=main CURRENT=next RELEASE=main
	if err := updateDocsWithOptions(docsUpdateOptions{
		BaseBranch:     cfg.BaseBranch,
		CurrentVersion: cfg.NextProjectMinorVersion,
		ReleaseBranch:  cfg.BaseBranch,
		DocBranch:      "main",
	}); err != nil {
		return workflowPR{}, err
	}
	if err := UpdateTestEnv(cfg.LatestRelease); err != nil {
		return workflowPR{}, err
	}
	commitMsg := fmt.Sprintf("[Release][%s] Update docs for %s and test environment to latest release %s", cfg.NextRelease, cfg.NextProjectMinorVersion, cfg.LatestRelease)
	if _, err := repo.commitAll(commitMsg, cfg.GitAuthorName, cfg.GitAuthorEmail); err != nil {
		return workflowPR{}, err
	}

	return workflowPR{
		branch: branch,
		base:   cfg.BaseBranch,
		opts: prOptions{
			Owner:     cfg.ProjectOwner,
			Repo:      cfg.ProjectRepo,
			Title:     fmt.Sprintf("[Release][%s] Update docs for %s and test environment to latest release %s", cfg.NextRelease, cfg.NextProjectMinorVersion, cfg.LatestRelease),
			Head:      branch,
			Base:      cfg.BaseBranch,
			Body:      prCMainBody(cfg),
			Reviewers: cfg.ProjectReviewers,
			Labels:    prCMainLabels(cfg.ReleaseBranch),
		},
	}, nil
}

// prepPatchVersionBump prepares the post-release version + test-env bump to NEXT_RELEASE.
// Matches ingest-dev prepare-next-patch / create-prs-next-patch (grouped into one PR).
// Used only by RunPatchRelease (not by RunMajorMinorRelease).
func prepPatchVersionBump(repo *gitRepo, cfg *ReleaseConfig) (workflowPR, error) {
	branch := fmt.Sprintf("update-version-%s", cfg.NextRelease)
	fmt.Printf("\n--- Preparing patch version bump to %s on %s ---\n", cfg.NextRelease, cfg.ReleaseBranch)

	if err := repo.ensureBranchFrom(cfg.ReleaseBranch, branch); err != nil {
		return workflowPR{}, err
	}
	// beats.mak prepare-next-patch: update-version + update-project + update-test-env
	// (no updateStackVersion / update-docs). NEXT_RELEASE is the version to land.
	if err := UpdateVersion(cfg.NextRelease); err != nil {
		return workflowPR{}, err
	}
	if err := runMakeUpdate(); err != nil {
		return workflowPR{}, err
	}
	// LATEST = previous patch on this release line (cfg.LatestRelease), not global Agent latest.
	if err := UpdateTestEnv(cfg.LatestRelease); err != nil {
		return workflowPR{}, err
	}
	commitMsg := fmt.Sprintf("[Release][%s] Update version to %s and test environments", cfg.NextRelease, cfg.NextRelease)
	if _, err := repo.commitAll(commitMsg, cfg.GitAuthorName, cfg.GitAuthorEmail); err != nil {
		return workflowPR{}, err
	}

	return workflowPR{
		branch: branch,
		base:   cfg.ReleaseBranch,
		opts: prOptions{
			Owner:     cfg.ProjectOwner,
			Repo:      cfg.ProjectRepo,
			Title:     fmt.Sprintf("[Release][%s] Update version to %s and test environments", cfg.NextRelease, cfg.NextRelease),
			Head:      branch,
			Base:      cfg.ReleaseBranch,
			Body:      patchVersionBumpBody(cfg),
			Reviewers: cfg.ProjectReviewers,
			Labels:    patchVersionBumpLabels(),
		},
	}, nil
}

func prAMainBody(cfg *ReleaseConfig) string {
	return fmt.Sprintf(`## [Release][%s]

Prepares %s for the %s feature freeze.

- Adds Mergify backport rule for branch %s (label %s)
- Bumps libbeat/version/version.go to %s (next minor)

**Merge:** before release branch work is finalized.
`, cfg.NextRelease, cfg.BaseBranch, cfg.NextRelease, cfg.ReleaseBranch, backportLabel(cfg.ReleaseBranch), cfg.NextProjectMinorVersion)
}

func prBReleaseBody(cfg *ReleaseConfig) string {
	return fmt.Sprintf(`## [Release][%s]

Feature-freeze release branch updates for %s (version, docs, test env, make update).

**Merge:** as soon as the %s branch exists.
`, cfg.NextRelease, cfg.NextRelease, cfg.ReleaseBranch)
}

func prCMainBody(cfg *ReleaseConfig) string {
	return fmt.Sprintf(`## [Release][%s]

Updates documentation on %s for the next minor %s, and keeps test environments on the latest published Elastic Stack release %s.

**Merge:** once CI is green.
`, cfg.NextRelease, cfg.BaseBranch, cfg.NextProjectMinorVersion, cfg.LatestRelease)
}

func patchVersionBumpBody(cfg *ReleaseConfig) string {
	return fmt.Sprintf(`## [Release][%s]

Post-release version bump on %s to %s (ingest-dev prepare-next-patch).

- Bumps libbeat/version/version.go to %s
- Updates test environments so stack tags use LATEST=%s

**Merge:** after the patch release job opens this PR (merge promptly).
`, cfg.NextRelease, cfg.ReleaseBranch, cfg.NextRelease, cfg.NextRelease, cfg.LatestRelease)
}

// ensureMajorMinorNextReleaseMatchesBase checks out BASE_BRANCH
// and requires libbeat/version/version.go to equal NEXT_RELEASE. After the previous
// minor's prepare-next-dev version bump, BASE_BRANCH already carries the version being frozen.
func ensureMajorMinorNextReleaseMatchesBase(repo *gitRepo, cfg *ReleaseConfig) error {
	base := cfg.BaseBranch
	if base == "" {
		base = "main"
	}
	if err := repo.checkoutBranch(base); err != nil {
		return err
	}
	branchVersion, err := readBeatVersion()
	if err != nil {
		return err
	}
	if branchVersion != cfg.NextRelease {
		return fmt.Errorf(
			"NEXT_RELEASE=%s does not match version on %s (%s in libbeat/version/version.go); "+
				"set NEXT_RELEASE to the version already on %s (the minor being feature-frozen)",
			cfg.NextRelease, base, branchVersion, base,
		)
	}
	fmt.Printf("Verified NEXT_RELEASE=%s matches %s on branch %s\n", cfg.NextRelease, branchVersion, base)
	return nil
}

// RunPatchRelease executes the post-release patch bump on an existing release branch.
// NEXT_RELEASE is the version to land (NEW_VERSION from the version-bump orchestrator),
// matching ingest-dev prepare-next-patch. Docs and backport stay off this path.
func RunPatchRelease(cfg *ReleaseConfig) error {
	fmt.Println("=== Starting Patch Release Workflow ===")

	if err := validateReleaseConfig(cfg); err != nil {
		return err
	}

	if err := checkRequirements(cfg); err != nil {
		return err
	}

	repo, err := openRepo(".")
	if err != nil {
		return err
	}

	if cfg.ReleaseBranch == "" {
		cfg.ReleaseBranch = devtools.InferReleaseBranch(cfg.NextReleaseVersion)
	}

	if err := ensurePatchBranchReadyForBump(repo, cfg); err != nil {
		return err
	}

	prBump, err := prepPatchVersionBump(repo, cfg)
	if err != nil {
		return err
	}

	branchesToFinalize := []workflowPR{prBump}

	if cfg.DryRun {
		fmt.Println("\nDRY RUN: Skipping push and PR creation")
		for _, item := range branchesToFinalize {
			fmt.Printf("Branch prepared: %s\n", item.branch)
		}
		warnEnsureReleaseIssueTracker(cfg, nil)
		return nil
	}

	gh := newGitHubClient(cfg.GitHubToken)
	results, err := finalizeWorkflowPRs(repo, gh, cfg.GitHubToken, branchesToFinalize)
	if err != nil {
		return err
	}

	fmt.Printf("\n=== Patch Release Workflow Complete ===\n")
	printWorkflowPRResults(results)
	fmt.Println("\nNote: Release notes PR should be created separately via .github/workflows/release-notes.yml")

	warnEnsureReleaseIssueTracker(cfg, prsFromWorkflowResults(results))
	return nil
}

// ensurePatchBranchReadyForBump checks out the release branch and requires
// libbeat/version/version.go to equal the previous patch on the same line as
// NEXT_RELEASE (the version to land). Forces LatestRelease to that previous
// patch for test-env (branch-local), matching ingest-dev define-versions patch.
func ensurePatchBranchReadyForBump(repo *gitRepo, cfg *ReleaseConfig) error {
	if err := cfg.EnsureParsedVersions(); err != nil {
		return err
	}
	prev := devtools.InferLatestRelease(cfg.NextReleaseVersion)
	if prev == nil {
		return fmt.Errorf(
			"prepare-next-patch requires NEXT_RELEASE with patch > 0 (got %s)",
			cfg.NextRelease,
		)
	}
	cfg.LatestReleaseVersion = prev
	cfg.LatestRelease = prev.String()

	if err := repo.checkoutBranch(cfg.ReleaseBranch); err != nil {
		return err
	}
	branchVersion, err := readBeatVersion()
	if err != nil {
		return err
	}
	if branchVersion == cfg.NextRelease {
		return fmt.Errorf(
			"NEXT_RELEASE=%s is already on branch %s; nothing to bump",
			cfg.NextRelease, cfg.ReleaseBranch,
		)
	}
	if branchVersion != cfg.LatestRelease {
		return fmt.Errorf(
			"NEXT_RELEASE=%s expects version %s on branch %s (got %s in libbeat/version/version.go); "+
				"set NEXT_RELEASE to the version to land (NEW_VERSION from version-bump)",
			cfg.NextRelease, cfg.LatestRelease, cfg.ReleaseBranch, branchVersion,
		)
	}
	fmt.Printf("Verified branch %s is at %s; will bump to NEXT_RELEASE=%s\n", cfg.ReleaseBranch, branchVersion, cfg.NextRelease)
	return nil
}

type workflowPR struct {
	branch string
	base   string
	opts   prOptions
}

type workflowPRResult struct {
	item workflowPR
	pr   *github.PullRequest
}

func finalizeWorkflowPRs(repo *gitRepo, gh *gitHubClient, token string, items []workflowPR) ([]workflowPRResult, error) {
	results := make([]workflowPRResult, 0, len(items))
	for i, item := range items {
		pr, err := finalizePR(repo, gh, token, item.branch, item.base, item.opts)
		if err != nil {
			return results, fmt.Errorf("failed to finalize PR %d/%d: %w", i+1, len(items), err)
		}
		results = append(results, workflowPRResult{item: item, pr: pr})
	}
	return results, nil
}

func printWorkflowPRResults(results []workflowPRResult) {
	for i, result := range results {
		fmt.Println(formatWorkflowPRLine(i+1, result))
	}
}

func prsFromWorkflowResults(results []workflowPRResult) []*github.PullRequest {
	var prs []*github.PullRequest
	for _, result := range results {
		if result.pr != nil {
			prs = append(prs, result.pr)
		}
	}
	return prs
}

func formatWorkflowPRLine(index int, result workflowPRResult) string {
	if result.pr != nil {
		return fmt.Sprintf("PR %d: %s (%s)", index, result.pr.GetHTMLURL(), prDisplayState(result.pr))
	}
	return fmt.Sprintf(
		"PR %d: skipped (no related open/merged PR for %s → %s)",
		index, result.item.branch, result.item.base,
	)
}

// finalizePR pushes a branch when it has new commits and creates or reuses an open PR.
// When the branch has nothing new to push, it still resolves a related open or merged PR
// so workflow summaries always list every expected slot.
func finalizePR(repo *gitRepo, gh *gitHubClient, token, branchName, baseBranch string, opts prOptions) (*github.PullRequest, error) {
	if err := repo.checkoutBranch(branchName); err != nil {
		return nil, err
	}

	existingPR, found, err := gh.findOpenPR(opts.Owner, opts.Repo, opts.Head, opts.Base)
	if err != nil {
		return nil, err
	}
	if found {
		if err := gh.ensurePRLabels(opts.Owner, opts.Repo, existingPR.GetNumber(), opts.Labels); err != nil {
			return nil, err
		}
		return existingPR, nil
	}

	ahead, err := repo.hasCommitsAheadOf(baseBranch)
	if err != nil {
		return nil, err
	}
	if !ahead {
		fmt.Printf("No new commits on %s compared to %s; skipping push and PR creation\n", branchName, baseBranch)
		related, found, err := gh.findRelatedPR(opts.Owner, opts.Repo, opts.Head, opts.Base, opts.Title)
		if err != nil {
			return nil, err
		}
		if found {
			fmt.Printf("Related PR #%d (%s): %s\n", related.GetNumber(), prDisplayState(related), related.GetHTMLURL())
			return related, nil
		}
		return nil, nil
	}

	if err := repo.push("origin", token); err != nil {
		return nil, err
	}

	return gh.createPR(opts)
}
