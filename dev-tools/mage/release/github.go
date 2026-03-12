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

	"github.com/google/go-github/v68/github"
	"golang.org/x/oauth2"
)

// GitHubClient wraps the GitHub API client
type GitHubClient struct {
	client *github.Client
	ctx    context.Context
}

// PROptions holds options for creating a pull request
type PROptions struct {
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

// NewGitHubClient creates a new GitHub API client
func NewGitHubClient(token string) *GitHubClient {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	return &GitHubClient{
		client: github.NewClient(tc),
		ctx:    ctx,
	}
}

// CreatePR creates a pull request
func (gh *GitHubClient) CreatePR(opts PROptions) (*github.PullRequest, error) {
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

	// Add labels if specified
	if len(opts.Labels) > 0 {
		err = gh.AddLabels(opts.Owner, opts.Repo, pr.GetNumber(), opts.Labels)
		if err != nil {
			fmt.Printf("Warning: failed to add labels: %v\n", err)
		}
	}

	fmt.Printf("Created PR #%d: %s\n", pr.GetNumber(), pr.GetHTMLURL())
	return pr, nil
}

// AddLabels adds labels to a pull request or issue
func (gh *GitHubClient) AddLabels(owner, repo string, number int, labels []string) error {
	_, _, err := gh.client.Issues.AddLabelsToIssue(gh.ctx, owner, repo, number, labels)
	if err != nil {
		return fmt.Errorf("failed to add labels: %w", err)
	}

	fmt.Printf("Added labels to #%d: %v\n", number, labels)
	return nil
}

// PRConfig holds configuration for a single PR in a multi-PR workflow
type PRConfig struct {
	BranchName string
	Title      string
	Body       string
	Labels     []string
}

// CreateMultiplePRs creates multiple PRs in sequence
// This is a Beats-specific function for workflows that create 2-3 PRs
func CreateMultiplePRs(cfg *ReleaseConfig, prConfigs []PRConfig) ([]*github.PullRequest, error) {
	gh := NewGitHubClient(cfg.GitHubToken)

	var prs []*github.PullRequest
	for i, prCfg := range prConfigs {
		opts := PROptions{
			Owner:     cfg.ProjectOwner,
			Repo:      cfg.ProjectRepo,
			Title:     prCfg.Title,
			Head:      prCfg.BranchName,
			Base:      cfg.BaseBranch,
			Body:      prCfg.Body,
			Draft:     false,
			Reviewers: cfg.ProjectReviewers,
			Labels:    prCfg.Labels,
		}

		pr, err := gh.CreatePR(opts)
		if err != nil {
			return prs, fmt.Errorf("failed to create PR %d/%d: %w", i+1, len(prConfigs), err)
		}

		prs = append(prs, pr)
	}

	fmt.Printf("Successfully created %d PRs\n", len(prs))
	return prs, nil
}
