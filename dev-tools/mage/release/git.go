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
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

// GitRepo wraps go-git Repository with helper methods
type GitRepo struct {
	repo *git.Repository
	path string
}

// OpenRepo opens a git repository at the specified path
func OpenRepo(path string) (*GitRepo, error) {
	repo, err := git.PlainOpen(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	return &GitRepo{
		repo: repo,
		path: path,
	}, nil
}

// BranchExists reports whether a local branch exists.
func (g *GitRepo) BranchExists(branchName string) (bool, error) {
	_, err := g.repo.Reference(plumbing.NewBranchReferenceName(branchName), true)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, plumbing.ErrReferenceNotFound) {
		return false, nil
	}
	return false, fmt.Errorf("failed to check branch %s: %w", branchName, err)
}

// CreateBranch creates a new branch from the current HEAD.
// It is idempotent when the branch already exists locally.
func (g *GitRepo) CreateBranch(branchName string) error {
	exists, err := g.BranchExists(branchName)
	if err != nil {
		return err
	}
	if exists {
		fmt.Printf("Branch already exists: %s\n", branchName)
		return nil
	}

	headRef, err := g.repo.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD: %w", err)
	}

	refName := plumbing.NewBranchReferenceName(branchName)
	ref := plumbing.NewHashReference(refName, headRef.Hash())

	err = g.repo.Storer.SetReference(ref)
	if err != nil {
		return fmt.Errorf("failed to create branch %s: %w", branchName, err)
	}

	fmt.Printf("Created branch: %s\n", branchName)
	return nil
}

// EnsureBranchFrom checks out baseBranch and creates or checks out branchName from that point.
func (g *GitRepo) EnsureBranchFrom(baseBranch, branchName string) error {
	if err := g.CheckoutBranch(baseBranch); err != nil {
		return fmt.Errorf("failed to checkout base branch %s: %w", baseBranch, err)
	}

	exists, err := g.BranchExists(branchName)
	if err != nil {
		return err
	}
	if exists {
		return g.CheckoutBranch(branchName)
	}

	if err := g.CreateBranch(branchName); err != nil {
		return err
	}
	return g.CheckoutBranch(branchName)
}

// EnsureBranch checks out an existing local or remote branch, or creates it from HEAD.
func (g *GitRepo) EnsureBranch(branchName string) error {
	exists, err := g.BranchExists(branchName)
	if err != nil {
		return err
	}
	if exists {
		return g.CheckoutBranch(branchName)
	}

	remoteRef, err := g.repo.Reference(plumbing.NewRemoteReferenceName("origin", branchName), true)
	if err == nil {
		localRef := plumbing.NewHashReference(plumbing.NewBranchReferenceName(branchName), remoteRef.Hash())
		if err := g.repo.Storer.SetReference(localRef); err != nil {
			return fmt.Errorf("failed to create local branch %s from origin: %w", branchName, err)
		}
		fmt.Printf("Created local branch from origin: %s\n", branchName)
		return g.CheckoutBranch(branchName)
	}
	if !errors.Is(err, plumbing.ErrReferenceNotFound) {
		return fmt.Errorf("failed to check remote branch %s: %w", branchName, err)
	}

	if err := g.CreateBranch(branchName); err != nil {
		return err
	}
	return g.CheckoutBranch(branchName)
}

// CheckoutBranch checks out an existing branch.
// It is idempotent when the branch is already checked out.
func (g *GitRepo) CheckoutBranch(branchName string) error {
	currentBranch, err := g.GetCurrentBranch()
	if err == nil && currentBranch == branchName {
		fmt.Printf("Already on branch: %s\n", branchName)
		return nil
	}

	w, err := g.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	err = w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branchName),
	})
	if err != nil {
		return fmt.Errorf("failed to checkout branch %s: %w", branchName, err)
	}

	fmt.Printf("Checked out branch: %s\n", branchName)
	return nil
}

// CommitAll stages all changes and creates a commit.
// It is idempotent and returns committed=false when there is nothing to commit.
func (g *GitRepo) CommitAll(message, authorName, authorEmail string) (bool, error) {
	w, err := g.repo.Worktree()
	if err != nil {
		return false, fmt.Errorf("failed to get worktree: %w", err)
	}

	// Add all changes
	err = w.AddGlob(".")
	if err != nil {
		return false, fmt.Errorf("failed to stage changes: %w", err)
	}

	status, err := w.Status()
	if err != nil {
		return false, fmt.Errorf("failed to get status: %w", err)
	}
	if status.IsClean() {
		fmt.Println("No changes to commit")
		return false, nil
	}

	// Create commit
	commit, err := w.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  authorName,
			Email: authorEmail,
			When:  time.Now(),
		},
	})
	if err != nil {
		return false, fmt.Errorf("failed to commit: %w", err)
	}

	fmt.Printf("Created commit: %s\n", commit.String())
	return true, nil
}

// HasCommitsAheadOf reports whether HEAD has commits not reachable from baseBranch.
// Equal or behind returns false; ahead or diverged returns true.
// A missing base branch is treated as ahead (true).
func (g *GitRepo) HasCommitsAheadOf(baseBranch string) (bool, error) {
	headRef, err := g.repo.Head()
	if err != nil {
		return false, fmt.Errorf("failed to get HEAD: %w", err)
	}

	baseRef, err := g.repo.Reference(plumbing.NewBranchReferenceName(baseBranch), true)
	if err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			return true, nil
		}
		return false, fmt.Errorf("failed to get base branch %s: %w", baseBranch, err)
	}

	if headRef.Hash() == baseRef.Hash() {
		return false, nil
	}

	headCommit, err := g.repo.CommitObject(headRef.Hash())
	if err != nil {
		return false, fmt.Errorf("failed to get HEAD commit: %w", err)
	}
	baseCommit, err := g.repo.CommitObject(baseRef.Hash())
	if err != nil {
		return false, fmt.Errorf("failed to get base commit %s: %w", baseBranch, err)
	}

	// HEAD has commits not on base iff HEAD is not a merge-base of (HEAD, base).
	// Behind/equal: merge-base is HEAD. Ahead/diverged: merge-base is not HEAD.
	bases, err := headCommit.MergeBase(baseCommit)
	if err != nil {
		return false, fmt.Errorf("failed to compare commits with %s: %w", baseBranch, err)
	}
	if len(bases) == 0 {
		return true, nil
	}
	for _, mb := range bases {
		if mb.Hash == headRef.Hash() {
			return false, nil
		}
	}
	return true, nil
}

// Push pushes the current branch to the remote
func (g *GitRepo) Push(remoteName string) error {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return fmt.Errorf("GITHUB_TOKEN environment variable is required for pushing")
	}

	// Get current branch to push only this branch
	currentBranch, err := g.GetCurrentBranch()
	if err != nil {
		return err
	}

	refSpec := config.RefSpec(fmt.Sprintf("refs/heads/%s:refs/heads/%s", currentBranch, currentBranch))

	err = g.repo.Push(&git.PushOptions{
		RemoteName: remoteName,
		RefSpecs:   []config.RefSpec{refSpec},
		Auth: &http.BasicAuth{
			Username: "git",
			Password: token,
		},
	})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("failed to push: %w", err)
	}

	fmt.Printf("Pushed branch %s to remote: %s\n", currentBranch, remoteName)
	return nil
}

// GetCurrentBranch returns the name of the current branch
func (g *GitRepo) GetCurrentBranch() (string, error) {
	headRef, err := g.repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	if !headRef.Name().IsBranch() {
		return "", fmt.Errorf("HEAD is not a branch")
	}

	return headRef.Name().Short(), nil
}

// IsClean checks if the working directory is clean (no uncommitted changes)
func (g *GitRepo) IsClean() (bool, error) {
	w, err := g.repo.Worktree()
	if err != nil {
		return false, fmt.Errorf("failed to get worktree: %w", err)
	}

	status, err := w.Status()
	if err != nil {
		return false, fmt.Errorf("failed to get status: %w", err)
	}

	return status.IsClean(), nil
}

// SetUpstream sets the upstream branch for the current branch
func (g *GitRepo) SetUpstream(remoteName, branchName string) error {
	currentBranch, err := g.GetCurrentBranch()
	if err != nil {
		return err
	}

	cfg, err := g.repo.Config()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	cfg.Branches[currentBranch] = &config.Branch{
		Name:   currentBranch,
		Remote: remoteName,
		Merge:  plumbing.NewBranchReferenceName(branchName),
	}

	err = g.repo.SetConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to set config: %w", err)
	}

	fmt.Printf("Set upstream for %s to %s/%s\n", currentBranch, remoteName, branchName)
	return nil
}
