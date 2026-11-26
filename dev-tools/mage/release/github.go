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
	"net/http"
	"strings"

	"github.com/google/go-github/v68/github"
	"golang.org/x/oauth2"
)

// gitHubClient wraps the GitHub API client
type gitHubClient struct {
	client *github.Client
	ctx    context.Context
}

// prOptions holds options for creating a pull request
type prOptions struct {
	Owner     string
	Repo      string
	Title     string
	Head      string
	Base      string
	Body      string
	Draft     bool
	Reviewers []string
	Labels    []string
}

// newGitHubClient creates a new GitHub API client.
// An empty token uses unauthenticated public API access (sufficient for reading public releases).
func newGitHubClient(token string) *gitHubClient {
	ctx := context.Background()
	var httpClient *http.Client
	if token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		httpClient = oauth2.NewClient(ctx, ts)
	}

	return &gitHubClient{
		client: github.NewClient(httpClient),
		ctx:    ctx,
	}
}

// createPR creates a pull request, or returns an existing open PR with the same head and base.
func (gh *gitHubClient) createPR(opts prOptions) (*github.PullRequest, error) {
	existingPR, found, err := gh.findOpenPR(opts.Owner, opts.Repo, opts.Head, opts.Base)
	if err != nil {
		return nil, err
	}
	if found {
		fmt.Printf("Open PR already exists #%d: %s\n", existingPR.GetNumber(), existingPR.GetHTMLURL())
		if err := gh.ensurePRLabels(opts.Owner, opts.Repo, existingPR.GetNumber(), opts.Labels); err != nil {
			return nil, err
		}
		return existingPR, nil
	}

	newPR := &github.NewPullRequest{
		Title: &opts.Title,
		Head:  &opts.Head,
		Base:  &opts.Base,
		Body:  &opts.Body,
		Draft: &opts.Draft,
	}

	pr, _, err := gh.client.PullRequests.Create(gh.ctx, opts.Owner, opts.Repo, newPR)
	if err != nil {
		return nil, fmt.Errorf("failed to create PR: %w", err)
	}

	// Add reviewers if specified
	if len(opts.Reviewers) > 0 {
		reviewersReq := github.ReviewersRequest{
			Reviewers: opts.Reviewers,
		}
		_, _, err = gh.client.PullRequests.RequestReviewers(gh.ctx, opts.Owner, opts.Repo, pr.GetNumber(), reviewersReq)
		if err != nil {
			fmt.Printf("Warning: failed to add reviewers: %v\n", err)
		}
	}

	if err := gh.ensurePRLabels(opts.Owner, opts.Repo, pr.GetNumber(), opts.Labels); err != nil {
		return nil, err
	}

	fmt.Printf("Created PR #%d: %s\n", pr.GetNumber(), pr.GetHTMLURL())
	return pr, nil
}

// findOpenPR returns an open pull request for the given head and base branches, if one exists.
func (gh *gitHubClient) findOpenPR(owner, repo, head, base string) (*github.PullRequest, bool, error) {
	prs, err := gh.listPRsByHeadBase(owner, repo, head, base, "open")
	if err != nil {
		return nil, false, err
	}
	if len(prs) == 0 {
		return nil, false, nil
	}
	return prs[0], true, nil
}

// findRelatedPR returns the best matching PR for head→base (open preferred, then
// merged, then closed). When none match by head/base, it falls back to an exact
// title match against the same base branch so already-merged release PRs still resolve.
func (gh *gitHubClient) findRelatedPR(owner, repo, head, base, title string) (*github.PullRequest, bool, error) {
	prs, err := gh.listPRsByHeadBase(owner, repo, head, base, "all")
	if err != nil {
		return nil, false, err
	}
	if pr := pickRelatedPR(prs); pr != nil {
		return pr, true, nil
	}
	if title == "" {
		return nil, false, nil
	}
	return gh.findPRByExactTitle(owner, repo, base, title)
}

func (gh *gitHubClient) listPRsByHeadBase(owner, repo, head, base, state string) ([]*github.PullRequest, error) {
	headQuery := head
	if !strings.Contains(head, ":") {
		headQuery = fmt.Sprintf("%s:%s", owner, head)
	}

	prs, _, err := gh.client.PullRequests.List(gh.ctx, owner, repo, &github.PullRequestListOptions{
		State: state,
		Head:  headQuery,
		Base:  base,
		ListOptions: github.ListOptions{
			PerPage: 30,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pull requests: %w", err)
	}
	return prs, nil
}

func (gh *gitHubClient) findPRByExactTitle(owner, repo, base, title string) (*github.PullRequest, bool, error) {
	query := fmt.Sprintf(`repo:%s/%s is:pr base:%s "%s" in:title`, owner, repo, base, title)
	result, _, err := gh.client.Search.Issues(gh.ctx, query, &github.SearchOptions{
		Sort:  "updated",
		Order: "desc",
		ListOptions: github.ListOptions{
			PerPage: 10,
		},
	})
	if err != nil {
		return nil, false, fmt.Errorf("failed to search pull requests by title: %w", err)
	}

	var matches []*github.PullRequest
	for _, issue := range result.Issues {
		if !issue.IsPullRequest() {
			continue
		}
		if issue.GetTitle() != title {
			continue
		}
		pr, _, err := gh.client.PullRequests.Get(gh.ctx, owner, repo, issue.GetNumber())
		if err != nil {
			return nil, false, fmt.Errorf("failed to get pull request #%d: %w", issue.GetNumber(), err)
		}
		matches = append(matches, pr)
	}
	if pr := pickRelatedPR(matches); pr != nil {
		return pr, true, nil
	}
	return nil, false, nil
}

// pickRelatedPR prefers open PRs, then merged, then other closed PRs.
func pickRelatedPR(prs []*github.PullRequest) *github.PullRequest {
	var open, merged, closed *github.PullRequest
	for _, pr := range prs {
		if pr == nil {
			continue
		}
		switch {
		case pr.GetState() == "open":
			if open == nil {
				open = pr
			}
		case pr.GetMerged() || pr.MergedAt != nil:
			if merged == nil {
				merged = pr
			}
		default:
			if closed == nil {
				closed = pr
			}
		}
	}
	if open != nil {
		return open
	}
	if merged != nil {
		return merged
	}
	return closed
}

// prDisplayState returns a short state label for workflow summaries.
func prDisplayState(pr *github.PullRequest) string {
	if pr == nil {
		return "unknown"
	}
	if pr.GetMerged() || pr.MergedAt != nil {
		return "merged"
	}
	state := pr.GetState()
	if state == "" {
		return "unknown"
	}
	return state
}

// mergeLabelDefs are auto-created when missing so merge-timing labels can be applied.
var mergeLabelDefs = map[string]struct {
	Color       string
	Description string
}{
	mergeLabelFFDay: {
		Color:       "B60205",
		Description: "Merge 1st: feature-freeze day (main)",
	},
	mergeLabelAfterBranch: {
		Color:       "D93F0B",
		Description: "Merge 2nd: ASAP after release branch exists",
	},
	mergeLabelAfterImages: {
		Color:       "FBCA04",
		Description: "Merge 3rd: after branch exists; may wait on Docker images",
	},
	mergeLabelAfterRelease: {
		Color:       "0E8A16",
		Description: "Merge after release: post-release version bump",
	},
}

func (gh *gitHubClient) ensurePRLabels(owner, repo string, number int, labels []string) error {
	if len(labels) == 0 {
		return nil
	}
	for _, label := range labels {
		if def, ok := mergeLabelDefs[label]; ok {
			if err := gh.ensureLabel(owner, repo, label, def.Color, def.Description); err != nil {
				return fmt.Errorf("failed to ensure label %q: %w", label, err)
			}
		}
	}
	if err := gh.addLabels(owner, repo, number, labels); err != nil {
		return fmt.Errorf("failed to add labels to PR #%d: %w", number, err)
	}
	return nil
}

// ensureLabel creates a repository label if it does not already exist.
func (gh *gitHubClient) ensureLabel(owner, repo, name, color, description string) error {
	_, resp, err := gh.client.Issues.GetLabel(gh.ctx, owner, repo, name)
	if err == nil {
		return nil
	}
	if resp == nil || resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("failed to get label %q: %w", name, err)
	}

	_, _, err = gh.client.Issues.CreateLabel(gh.ctx, owner, repo, &github.Label{
		Name:        new(name),
		Color:       new(color),
		Description: new(description),
	})
	if err != nil {
		return fmt.Errorf("failed to create label %q: %w", name, err)
	}
	fmt.Printf("Created label %q\n", name)
	return nil
}

// addLabels adds labels to a pull request or issue
func (gh *gitHubClient) addLabels(owner, repo string, number int, labels []string) error {
	_, _, err := gh.client.Issues.AddLabelsToIssue(gh.ctx, owner, repo, number, labels)
	if err != nil {
		return fmt.Errorf("failed to add labels: %w", err)
	}

	fmt.Printf("Added labels to #%d: %v\n", number, labels)
	return nil
}

// findIssueByTitle finds an issue with an exact title match (open preferred over closed).
func (gh *gitHubClient) findIssueByTitle(owner, repo, title string) (*github.Issue, bool, error) {
	query := fmt.Sprintf(`repo:%s/%s is:issue in:title "%s"`, owner, repo, title)
	result, _, err := gh.client.Search.Issues(gh.ctx, query, &github.SearchOptions{
		Sort:  "updated",
		Order: "desc",
		ListOptions: github.ListOptions{
			PerPage: 20,
		},
	})
	if err != nil {
		return nil, false, fmt.Errorf("failed to search issues for title %q: %w", title, err)
	}

	var closedMatch *github.Issue
	for _, issue := range result.Issues {
		if issue.IsPullRequest() {
			continue
		}
		if issue.GetTitle() != title {
			continue
		}
		if issue.GetState() == "open" {
			return issue, true, nil
		}
		if closedMatch == nil {
			closedMatch = issue
		}
	}
	if closedMatch != nil {
		return closedMatch, true, nil
	}
	return nil, false, nil
}

// createIssue creates a repository issue with the given labels.
func (gh *gitHubClient) createIssue(owner, repo, title, body string, labels []string) (*github.Issue, error) {
	req := &github.IssueRequest{
		Title: new(title),
		Body:  new(body),
	}
	if len(labels) > 0 {
		req.Labels = &labels
	}
	issue, _, err := gh.client.Issues.Create(gh.ctx, owner, repo, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create issue %q: %w", title, err)
	}
	return issue, nil
}

// updateIssueBody replaces the body of an existing issue.
func (gh *gitHubClient) updateIssueBody(owner, repo string, number int, body string) error {
	_, _, err := gh.client.Issues.Edit(gh.ctx, owner, repo, number, &github.IssueRequest{
		Body: new(body),
	})
	if err != nil {
		return fmt.Errorf("failed to update issue #%d body: %w", number, err)
	}
	return nil
}

// listReleaseLabeledPRsForVersion returns Beats PRs with label "release" whose title
// mentions the exact version. Open and recently updated closed/merged PRs are included.
func (gh *gitHubClient) listReleaseLabeledPRsForVersion(owner, repo, version string) ([]*github.PullRequest, error) {
	query := fmt.Sprintf(`repo:%s/%s is:pr label:release "%s"`, owner, repo, version)
	opts := &github.SearchOptions{
		Sort:  "updated",
		Order: "desc",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	var prs []*github.PullRequest
	seen := map[int]struct{}{}
	for {
		result, resp, err := gh.client.Search.Issues(gh.ctx, query, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to search release PRs for %s: %w", version, err)
		}
		for _, issue := range result.Issues {
			if !issue.IsPullRequest() {
				continue
			}
			if !versionMentioned(issue.GetTitle(), version) {
				continue
			}
			num := issue.GetNumber()
			if _, ok := seen[num]; ok {
				continue
			}
			seen[num] = struct{}{}
			prs = append(prs, &github.PullRequest{
				Number:  new(num),
				HTMLURL: new(issue.GetHTMLURL()),
				Title:   new(issue.GetTitle()),
				State:   new(issue.GetState()),
			})
		}
		if resp == nil || resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return prs, nil
}

// releasesLookupOwner and releasesLookupRepo are the canonical repo for published Beats releases.
// Forks typically lack release history, so workflows always resolve LatestRelease from here.
const (
	releasesLookupOwner = "elastic"
	releasesLookupRepo  = "beats"
)

// latestReleaseBefore returns the highest published release version with the same major
// that is strictly less than currentVersion (e.g. current 9.5.0 → 9.4.3).
func (gh *gitHubClient) latestReleaseBefore(owner, repo, currentVersion string) (string, error) {
	versions, err := gh.listReleaseVersions(owner, repo)
	if err != nil {
		return "", err
	}
	return selectLatestReleaseBefore(versions, currentVersion)
}

func (gh *gitHubClient) listReleaseVersions(owner, repo string) ([]string, error) {
	var versions []string
	opts := &github.ListOptions{PerPage: 100}
	for {
		releases, resp, err := gh.client.Repositories.ListReleases(gh.ctx, owner, repo, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list releases for %s/%s: %w", owner, repo, err)
		}
		for _, rel := range releases {
			tag := rel.GetTagName()
			if tag == "" {
				continue
			}
			versions = append(versions, tag)
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return versions, nil
}
