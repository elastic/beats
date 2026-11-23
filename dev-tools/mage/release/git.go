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
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/storage/filesystem"
)

const (
	// pushTimeout bounds native git push so Buildkite jobs fail fast instead of
	// hanging until the step timeout (go-git push stalls on reference clones).
	pushTimeout = 10 * time.Minute
	// pushWaitDelay bounds CombinedOutput after context cancel when a git
	// helper (for example git-remote-http) keeps an I/O pipe open.
	pushWaitDelay = 10 * time.Second
	// worktreeCmdTimeout bounds git checkout/status/add/commit/merge-base used
	// instead of go-git on Buildkite reference clones. go-git Status/Checkout
	// can take tens of minutes on large trees with object alternates.
	worktreeCmdTimeout = 10 * time.Minute
)

// gitRepo wraps go-git Repository with helper methods.
type gitRepo struct {
	repo *git.Repository
	path string
}

// openRepo opens a git repository at the specified path.
func openRepo(path string) (*gitRepo, error) {
	repo, err := git.PlainOpen(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	// Buildkite reference clones borrow objects outside the repository's .git directory.
	storage, ok := repo.Storer.(*filesystem.Storage)
	if !ok {
		return nil, fmt.Errorf("open repository: unexpected storage type %T", repo.Storer)
	}
	root := filepath.VolumeName(storage.Filesystem().Root()) + string(filepath.Separator)
	repo.Storer = filesystem.NewStorageWithOptions(storage.Filesystem(), cache.NewObjectLRUDefault(), filesystem.Options{
		AlternatesFS: osfs.New(root),
	})

	return &gitRepo{
		repo: repo,
		path: path,
	}, nil
}

// branchExists reports whether a local branch exists.
func (g *gitRepo) branchExists(branchName string) (bool, error) {
	_, err := g.repo.Reference(plumbing.NewBranchReferenceName(branchName), true)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, plumbing.ErrReferenceNotFound) {
		return false, nil
	}
	return false, fmt.Errorf("failed to check branch %s: %w", branchName, err)
}

// createBranch creates a new branch from the current HEAD.
// It is idempotent when the branch already exists locally.
func (g *gitRepo) createBranch(branchName string) error {
	exists, err := g.branchExists(branchName)
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

// ensureBranchFrom checks out baseBranch and creates or checks out branchName from that point.
func (g *gitRepo) ensureBranchFrom(baseBranch, branchName string) error {
	if err := g.checkoutBranch(baseBranch); err != nil {
		return fmt.Errorf("failed to checkout base branch %s: %w", baseBranch, err)
	}

	exists, err := g.branchExists(branchName)
	if err != nil {
		return err
	}
	if exists {
		return g.checkoutBranch(branchName)
	}

	if err := g.createBranch(branchName); err != nil {
		return err
	}
	return g.checkoutBranch(branchName)
}

// runGit runs git in the repository path with a deadline (Buildkite-safe).
func (g *gitRepo) runGit(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.WaitDelay = pushWaitDelay
	cmd.Dir = g.path
	return cmd.CombinedOutput()
}

// checkoutBranch checks out an existing branch via the git CLI.
// It is idempotent when the branch is already checked out. go-git Checkout is
// avoided: with Buildkite reference clones it can take tens of minutes on large
// trees (see ingest-release-smoke beats timeout on observability-robots).
func (g *gitRepo) checkoutBranch(branchName string) error {
	currentBranch, err := g.getCurrentBranch()
	if err == nil && currentBranch == branchName {
		fmt.Printf("Already on branch: %s\n", branchName)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), worktreeCmdTimeout)
	defer cancel()

	output, err := g.runGit(ctx, "checkout", branchName)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("failed to checkout branch %s: timed out after %s: %w", branchName, worktreeCmdTimeout, err)
		}
		return fmt.Errorf("failed to checkout branch %s: %w: %s", branchName, err, strings.TrimSpace(string(output)))
	}

	fmt.Printf("Checked out branch: %s\n", branchName)
	return nil
}

// commitAll stages all changes and creates a commit via the git CLI.
// It is idempotent and returns committed=false when there is nothing to commit.
// go-git AddGlob/Status/Commit are avoided for the same reference-clone cost as
// checkoutBranch.
func (g *gitRepo) commitAll(message, authorName, authorEmail string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), worktreeCmdTimeout)
	defer cancel()

	if output, err := g.runGit(ctx, "add", "-A"); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return false, fmt.Errorf("failed to stage changes: timed out after %s: %w", worktreeCmdTimeout, err)
		}
		return false, fmt.Errorf("failed to stage changes: %w: %s", err, strings.TrimSpace(string(output)))
	}

	statusOut, err := g.runGit(ctx, "status", "--porcelain")
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return false, fmt.Errorf("failed to get status: timed out after %s: %w", worktreeCmdTimeout, err)
		}
		return false, fmt.Errorf("failed to get status: %w: %s", err, strings.TrimSpace(string(statusOut)))
	}
	if len(strings.TrimSpace(string(statusOut))) == 0 {
		fmt.Println("No changes to commit")
		return false, nil
	}

	commitOut, err := g.runGit(ctx,
		"-c", "user.name="+authorName,
		"-c", "user.email="+authorEmail,
		"commit",
		"--author", authorName+" <"+authorEmail+">",
		"-m", message,
	)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return false, fmt.Errorf("failed to commit: timed out after %s: %w", worktreeCmdTimeout, err)
		}
		return false, fmt.Errorf("failed to commit: %w: %s", err, strings.TrimSpace(string(commitOut)))
	}

	hashOut, err := g.runGit(ctx, "rev-parse", "HEAD")
	if err != nil {
		return false, fmt.Errorf("failed to read commit hash: %w: %s", err, strings.TrimSpace(string(hashOut)))
	}
	fmt.Printf("Created commit: %s\n", strings.TrimSpace(string(hashOut)))
	return true, nil
}

// hasCommitsAheadOf reports whether HEAD has commits not reachable from baseBranch.
// Equal or behind returns false; ahead or diverged returns true.
// A missing base branch is treated as ahead (true).
// Uses the git CLI so mixed CLI commits and reference-clone object stores do not
// hit go-git MergeBase "object not found" failures.
func (g *gitRepo) hasCommitsAheadOf(baseBranch string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), worktreeCmdTimeout)
	defer cancel()

	if _, err := g.runGit(ctx, "show-ref", "--verify", "--quiet", "refs/heads/"+baseBranch); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return false, fmt.Errorf(
				"failed to verify base branch %s: timed out after %s: %w",
				baseBranch, worktreeCmdTimeout, err,
			)
		}
		// Missing local branch: treat as ahead (prior go-git ErrReferenceNotFound).
		return true, nil
	}

	headOut, err := g.runGit(ctx, "rev-parse", "HEAD")
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return false, fmt.Errorf("failed to get HEAD: timed out after %s: %w", worktreeCmdTimeout, err)
		}
		return false, fmt.Errorf("failed to get HEAD: %w: %s", err, strings.TrimSpace(string(headOut)))
	}
	baseOut, err := g.runGit(ctx, "rev-parse", baseBranch)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return false, fmt.Errorf(
				"failed to get base branch %s: timed out after %s: %w",
				baseBranch, worktreeCmdTimeout, err,
			)
		}
		return false, fmt.Errorf(
			"failed to get base branch %s: %w: %s",
			baseBranch, err, strings.TrimSpace(string(baseOut)),
		)
	}
	headHash := strings.TrimSpace(string(headOut))
	baseHash := strings.TrimSpace(string(baseOut))
	if headHash == baseHash {
		return false, nil
	}

	mbOut, err := g.runGit(ctx, "merge-base", "HEAD", baseBranch)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return false, fmt.Errorf(
				"failed to compare commits with %s: timed out after %s: %w",
				baseBranch, worktreeCmdTimeout, err,
			)
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			// No common ancestor → treat as ahead (diverged / unrelated).
			return true, nil
		}
		return false, fmt.Errorf(
			"failed to compare commits with %s: %w: %s",
			baseBranch, err, strings.TrimSpace(string(mbOut)),
		)
	}
	if strings.TrimSpace(string(mbOut)) == headHash {
		return false, nil
	}
	return true, nil
}

// githubHTTPSPushURL converts SSH-style GitHub remotes to HTTPS so
// token-authenticated pushes can use HTTPS. HTTPS/HTTP URLs are unchanged.
// Returns the URL to use and whether it was rewritten from SSH.
func githubHTTPSPushURL(remoteURL string) (string, bool) {
	u := strings.TrimSpace(remoteURL)
	switch {
	case strings.HasPrefix(u, "https://"), strings.HasPrefix(u, "http://"):
		return u, false
	case strings.HasPrefix(u, "git@github.com:"):
		return "https://github.com/" + strings.TrimPrefix(u, "git@github.com:"), true
	case strings.HasPrefix(u, "ssh://git@github.com"):
		rest := strings.TrimPrefix(u, "ssh://git@github.com")
		rest = strings.TrimPrefix(rest, ":22")
		rest = strings.TrimPrefix(rest, "/")
		return "https://github.com/" + rest, true
	default:
		return u, false
	}
}

// ensureHTTPSRemote rewrites an SSH GitHub remote to HTTPS when needed.
func (g *gitRepo) ensureHTTPSRemote(remoteName string) error {
	remote, err := g.repo.Remote(remoteName)
	if err != nil {
		return fmt.Errorf("failed to get remote %s: %w", remoteName, err)
	}
	urls := remote.Config().URLs
	if len(urls) == 0 {
		return fmt.Errorf("remote %s has no URLs", remoteName)
	}

	httpsURL, rewritten := githubHTTPSPushURL(urls[0])
	if !isTokenPushRemoteURL(httpsURL) {
		return fmt.Errorf("remote %s URL %q is not HTTPS or file://; token push requires an HTTPS remote", remoteName, urls[0])
	}
	if !rewritten {
		return nil
	}
	if err := g.setRemoteURL(remoteName, httpsURL); err != nil {
		return fmt.Errorf("failed to set HTTPS remote for token push: %w", err)
	}
	return nil
}

// isTokenPushRemoteURL reports whether the remote URL can be used with a
// token-authenticated git push (HTTPS for GitHub, file:// for local tests).
func isTokenPushRemoteURL(u string) bool {
	return strings.HasPrefix(u, "https://") ||
		strings.HasPrefix(u, "http://") ||
		strings.HasPrefix(u, "file://")
}

// gitHTTPSAuthExtraHeader returns the git http.extraHeader value for GitHub
// token auth. GitHub documents x-access-token as the Basic auth user name for
// App installation tokens; git-over-HTTPS rejects Bearer for those tokens.
func gitHTTPSAuthExtraHeader(token string) string {
	auth := base64.StdEncoding.EncodeToString([]byte("x-access-token:" + token))
	return "Authorization: Basic " + auth
}

// gitPushEnv adds process-local git config and disables interactive prompts so
// the token never appears in argv and a failed auth cannot hang on a TTY.
// An empty http.https://github.com/.extraHeader first clears headers that other
// git configuration may set for github.com, then the auth header is set.
func gitPushEnv(baseEnv []string, token string) []string {
	return append(append([]string{}, baseEnv...),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_CONFIG_COUNT=2",
		"GIT_CONFIG_KEY_0=http.https://github.com/.extraHeader",
		"GIT_CONFIG_VALUE_0=",
		"GIT_CONFIG_KEY_1=http.https://github.com/.extraHeader",
		"GIT_CONFIG_VALUE_1="+gitHTTPSAuthExtraHeader(token),
	)
}

// push pushes the current branch to the remote using the given GitHub token.
// It uses the git CLI so Buildkite reference clones can pack and upload; go-git
// Push hangs in that layout.
func (g *gitRepo) push(remoteName, token string) error {
	if token == "" {
		return fmt.Errorf("GitHub token is required for pushing")
	}
	if remoteName != "origin" {
		return fmt.Errorf("unsupported remote %q; release push only supports origin", remoteName)
	}

	if err := g.ensureHTTPSRemote(remoteName); err != nil {
		return err
	}

	currentBranch, err := g.getCurrentBranch()
	if err != nil {
		return err
	}

	fmt.Printf("Pushing branch %s to remote origin via git CLI...\n", currentBranch)

	ctx, cancel := context.WithTimeout(context.Background(), pushTimeout)
	defer cancel()

	// Auth is injected through GIT_CONFIG_* so the token is not passed as a
	// command-line argument. `git push origin HEAD` updates the remote branch
	// that matches the current local branch name.
	cmd := exec.CommandContext(ctx, "git", "push", "origin", "HEAD")
	cmd.WaitDelay = pushWaitDelay
	cmd.Dir = g.path
	cmd.Env = gitPushEnv(os.Environ(), token)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("failed to push: timed out after %s: %w", pushTimeout, err)
		}
		return fmt.Errorf("failed to push: %w: %s", err, strings.TrimSpace(string(output)))
	}

	fmt.Printf("Pushed branch %s to remote: origin\n", currentBranch)
	return nil
}

// setRemoteURL sets the URL for a remote.
func (g *gitRepo) setRemoteURL(remoteName, url string) error {
	_, err := g.repo.Remote(remoteName)
	if errors.Is(err, git.ErrRemoteNotFound) {
		_, err = g.repo.CreateRemote(&config.RemoteConfig{
			Name: remoteName,
			URLs: []string{url},
		})
		if err != nil {
			return fmt.Errorf("failed to create remote: %w", err)
		}
		fmt.Printf("Created remote %s: %s\n", remoteName, url)
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to get remote: %w", err)
	}

	err = g.repo.DeleteRemote(remoteName)
	if err != nil {
		return fmt.Errorf("failed to delete remote: %w", err)
	}

	_, err = g.repo.CreateRemote(&config.RemoteConfig{
		Name: remoteName,
		URLs: []string{url},
	})
	if err != nil {
		return fmt.Errorf("failed to recreate remote: %w", err)
	}

	fmt.Printf("Updated remote %s: %s\n", remoteName, url)
	return nil
}

// getCurrentBranch returns the name of the current branch
func (g *gitRepo) getCurrentBranch() (string, error) {
	headRef, err := g.repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	if !headRef.Name().IsBranch() {
		return "", fmt.Errorf("HEAD is not a branch")
	}

	return headRef.Name().Short(), nil
}

// isClean reports whether the working directory has no uncommitted changes.
// Uses git status --porcelain; go-git Status is too slow on Buildkite reference
// clones of large trees (tens of minutes observed on beats smoke).
func (g *gitRepo) isClean() (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), worktreeCmdTimeout)
	defer cancel()

	output, err := g.runGit(ctx, "status", "--porcelain")
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return false, fmt.Errorf("failed to get status: timed out after %s: %w", worktreeCmdTimeout, err)
		}
		return false, fmt.Errorf("failed to get status: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return len(strings.TrimSpace(string(output))) == 0, nil
}
