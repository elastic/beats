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
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/google/go-github/v68/github"
)

// globalReleaseTrackerURL is the ingest-wide release tracker linked from every Beats release issue.
const globalReleaseTrackerURL = "https://github.com/elastic/ingest-dev/issues/8866"

var (
	prCheckboxLineRE = regexp.MustCompile(`(?m)^- \[([ xX])\] (https://github\.com/[^/]+/[^/]+/pull/\d+)\s*$`)
	prURLNumberRE    = regexp.MustCompile(`https://github\.com/[^/]+/[^/]+/pull/(\d+)`)
)

// warnEnsureReleaseIssueTracker runs EnsureReleaseIssueTracker but never fails the
// caller. Release workflows use this so a tracker outage cannot abort PR creation.
func warnEnsureReleaseIssueTracker(cfg *ReleaseConfig, workflowPRs []*github.PullRequest) {
	if err := EnsureReleaseIssueTracker(cfg, workflowPRs); err != nil {
		fmt.Printf("Warning: ensure release issue tracker failed (release workflow continues): %v\n", err)
		fmt.Println("Re-run with: mage release:ensureIssueTracker")
	}
}

// EnsureReleaseIssueTracker creates or updates the Beats release checklist issue for
// cfg.CurrentRelease. It links related Beats PRs (workflow PRs plus open/merged PRs
// with label "release" that mention the same version). Updates are idempotent: missing
// PR links are appended; existing checklist and checkbox state are preserved.
// Standalone mage release:ensureIssueTracker surfaces errors; runMajorMinor/runPatch
// wrap this with warnEnsureReleaseIssueTracker so failures are non-blocking.
func EnsureReleaseIssueTracker(cfg *ReleaseConfig, workflowPRs []*github.PullRequest) error {
	if cfg == nil {
		return fmt.Errorf("release config is required")
	}
	if cfg.CurrentRelease == "" {
		return fmt.Errorf("CURRENT_RELEASE is required")
	}

	title := releaseIssueTitle(cfg.CurrentRelease)
	seedURLs := prURLsFromPullRequests(workflowPRs)

	if cfg.DryRun {
		fmt.Printf("\nDRY RUN: Would ensure issue tracker %q\n", title)
		fmt.Printf("Seed PRs from workflow: %d\n", len(seedURLs))
		fmt.Printf("Global tracker: %s\n", globalReleaseTrackerURL)
		return nil
	}

	if cfg.GitHubToken == "" {
		return fmt.Errorf("GITHUB_TOKEN is required to ensure the release issue tracker")
	}

	gh := NewGitHubClient(cfg.GitHubToken)
	owner := cfg.ProjectOwner
	repo := cfg.ProjectRepo

	discovered, err := gh.ListReleaseLabeledPRsForVersion(owner, repo, cfg.CurrentRelease)
	if err != nil {
		return fmt.Errorf("discover release-labeled PRs for %s: %w", cfg.CurrentRelease, err)
	}
	prURLs := mergePRURLs(seedURLs, prURLsFromPullRequests(discovered))

	existing, found, err := gh.FindIssueByTitle(owner, repo, title)
	if err != nil {
		return err
	}

	if !found {
		body := buildReleaseIssueBody(cfg.CurrentRelease, isMajorMinorRelease(cfg.CurrentRelease), prURLs, nil)
		issue, err := gh.CreateIssue(owner, repo, title, body, []string{"release"})
		if err != nil {
			return err
		}
		fmt.Printf("Created release issue tracker #%d: %s\n", issue.GetNumber(), issue.GetHTMLURL())
		return nil
	}

	updatedBody, changed := mergeReleaseIssueBody(existing.GetBody(), cfg.CurrentRelease, prURLs)
	if !changed {
		fmt.Printf("Release issue tracker already up to date: #%d %s\n", existing.GetNumber(), existing.GetHTMLURL())
		return nil
	}

	if err := gh.UpdateIssueBody(owner, repo, existing.GetNumber(), updatedBody); err != nil {
		return err
	}
	if err := gh.AddLabels(owner, repo, existing.GetNumber(), []string{"release"}); err != nil {
		fmt.Printf("Warning: failed to ensure release label on issue #%d: %v\n", existing.GetNumber(), err)
	}
	fmt.Printf("Updated release issue tracker #%d: %s\n", existing.GetNumber(), existing.GetHTMLURL())
	return nil
}

func releaseIssueTitle(version string) string {
	return fmt.Sprintf("[RELEASE %s] Instructions & Checklist", version)
}

func isMajorMinorRelease(version string) bool {
	parts := strings.Split(version, ".")
	return len(parts) >= 3 && parts[2] == "0"
}

func prURLsFromPullRequests(prs []*github.PullRequest) []string {
	var urls []string
	for _, pr := range prs {
		if pr == nil {
			continue
		}
		if u := pr.GetHTMLURL(); u != "" {
			urls = append(urls, u)
		}
	}
	return urls
}

// mergePRURLs returns a de-duplicated, number-sorted union of PR URL lists.
func mergePRURLs(lists ...[]string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, list := range lists {
		for _, u := range list {
			u = strings.TrimSpace(u)
			if u == "" {
				continue
			}
			if _, ok := seen[u]; ok {
				continue
			}
			seen[u] = struct{}{}
			out = append(out, u)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return prNumber(out[i]) < prNumber(out[j])
	})
	return out
}

func prNumber(url string) int {
	m := prURLNumberRE.FindStringSubmatch(url)
	if len(m) < 2 {
		return 0
	}
	n, _ := strconv.Atoi(m[1])
	return n
}

// extractPRCheckboxes returns PR URL -> checked from an issue body.
func extractPRCheckboxes(body string) map[string]bool {
	out := map[string]bool{}
	for _, m := range prCheckboxLineRE.FindAllStringSubmatch(body, -1) {
		checked := m[1] == "x" || m[1] == "X"
		out[m[2]] = checked
	}
	return out
}

func buildReleaseIssueBody(version string, majorMinor bool, prURLs []string, checked map[string]bool) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Global tracker: %s\n\n", globalReleaseTrackerURL))
	b.WriteString("# Release Checklist\n\n")
	b.WriteString("Checklist for the Beats release process. The Elastic Agent and Beats release documentation can be reviewed ")
	b.WriteString("[here](https://github.com/elastic/ingest-dev/blob/main/fleet-platform/releases.md).\n\n")
	b.WriteString("The release dates can be checked at https://ela.st/release-schedule.\n\n")
	b.WriteString("Ensure you have joined the `#mission-control` Slack channel and the `@agent-team` slack group to receive release reminders and coordination messages.\n\n")

	b.WriteString("## On Feature Freeze\n\n")
	b.WriteString("- [ ] Find the Beats [release](https://github.com/elastic/beats/labels/release) PRs for this version and link them from this issue.\n")
	b.WriteString("- [ ] Check which of the PRs can be merged immediately or must wait until the release day (should be mentioned in each PR).\n")
	b.WriteString("  - Note that the PR updating the test Docker image versions will not pass until they are published on the release day.\n")

	if majorMinor {
		b.WriteString("\n### Only For Major/Minor Releases\n\n")
		b.WriteString("- [ ] On `main`, find and merge the PR changing the version to the next minor as soon as it is green.\n")
		b.WriteString("- [ ] Find the PR in observability-dev adding the backport labels for the project and merge them: https://github.com/elastic/observability-dev/pulls. The list of Github labels is maintained [here](https://github.com/elastic/observability-dev/tree/main/.github/labels).\n")
		b.WriteString("- [ ] Find the PR in Beats updating the test environments to the next major/minor version, ensure that it is merged and backported to the newly created minor version branch. The build will not succeed in the new Beats minor branch until this is done. See the [example PR](https://github.com/elastic/beats/pull/35872) from 8.10.\n")
		b.WriteString("- [ ] Ensure there is a [branch protection rule](https://github.com/elastic/beats/settings/branches) for the new branch. `9.*`, `8.*` and `7.*` patterns may already exist.\n")
	}

	b.WriteString("\n## On the Day before the Release\n\n")
	b.WriteString("- [ ] Prepare the changelog. Follow the detailed instructions for [Beats changelog preparation](https://github.com/elastic/ingest-dev/blob/main/fleet-platform/releases.md#beats-changelog-preparation).\n")

	b.WriteString("\n## On the Release Date\n\n")
	b.WriteString("- [ ] Wait for the ping from #mission-control to merge all version bump PRs and trigger the [Beats Packaging jobs](https://github.com/elastic/ingest-dev/blob/main/fleet-platform/releases.md#preparing-a-release) to stage artifacts for the DRA (daily releasable artifacts) process.\n")
	b.WriteString("- [ ] Find all Beats [release](https://github.com/elastic/beats/labels/release) pending PRs for the current release and merge them.\n")
	b.WriteString("  - The tests for the PR updating the Docker image versions to the new release versions will need to be re-triggered once the images are published.\n")
	b.WriteString("- [ ] Merge the Changelog / release-notes PR.\n")
	b.WriteString("- [ ] [Forward port](https://github.com/elastic/ingest-dev/blob/main/fleet-platform/releases.md#changelog-forward-ports) the changelog.\n")

	b.WriteString("\n## PRs\n\n")
	b.WriteString(formatPRChecklist(prURLs, checked))
	b.WriteString("\n")
	return b.String()
}

func formatPRChecklist(prURLs []string, checked map[string]bool) string {
	if len(prURLs) == 0 {
		return "_No related Beats release PRs discovered yet._\n"
	}
	var b strings.Builder
	for _, u := range prURLs {
		mark := " "
		if checked != nil && checked[u] {
			mark = "x"
		}
		b.WriteString(fmt.Sprintf("- [%s] %s\n", mark, u))
	}
	return b.String()
}

// mergeReleaseIssueBody updates an existing issue body with missing tracker link and PR URLs.
// Checklist item checkboxes outside the PRs section are left untouched.
func mergeReleaseIssueBody(existingBody, version string, prURLs []string) (string, bool) {
	existingChecked := extractPRCheckboxes(existingBody)
	allURLs := mergePRURLs(prURLs, keys(existingChecked))

	if strings.TrimSpace(existingBody) == "" {
		return buildReleaseIssueBody(version, isMajorMinorRelease(version), allURLs, existingChecked), true
	}

	body := existingBody
	changed := false

	if !strings.Contains(body, globalReleaseTrackerURL) {
		body = fmt.Sprintf("Global tracker: %s\n\n%s", globalReleaseTrackerURL, body)
		changed = true
	}

	const prSection = "## PRs"
	idx := strings.Index(body, prSection)
	if idx < 0 {
		updated := strings.TrimRight(body, "\n") + "\n\n" + prSection + "\n\n" + formatPRChecklist(allURLs, existingChecked)
		return updated, true
	}

	before := body[:idx]
	oldPRBlock := strings.TrimPrefix(body[idx+len(prSection):], "\n")
	newPRBlock := formatPRChecklist(allURLs, existingChecked)

	oldURLs := keys(extractPRCheckboxes(oldPRBlock))
	if !sameStringSet(oldURLs, allURLs) || normalizePRSection(oldPRBlock) != normalizePRSection(newPRBlock) {
		changed = true
	}
	if !changed {
		return existingBody, false
	}

	updated := strings.TrimRight(before, "\n") + "\n\n" + prSection + "\n\n" + newPRBlock
	return updated, true
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func sameStringSet(a, b []string) bool {
	am := map[string]struct{}{}
	for _, s := range a {
		am[s] = struct{}{}
	}
	bm := map[string]struct{}{}
	for _, s := range b {
		bm[s] = struct{}{}
	}
	if len(am) != len(bm) {
		return false
	}
	for s := range am {
		if _, ok := bm[s]; !ok {
			return false
		}
	}
	return true
}

func normalizePRSection(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "\r\n", "\n"))
}

// versionMentioned reports whether text refers to the exact version (avoids 9.4.1 matching 9.4.10).
func versionMentioned(text, version string) bool {
	if version == "" || text == "" {
		return false
	}
	re := regexp.MustCompile(`(?i)(?:^|[^0-9])` + regexp.QuoteMeta(version) + `(?:[^0-9]|$)`)
	return re.MatchString(text)
}
