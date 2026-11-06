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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// TestIsClean tests the IsClean function with various git states
func TestIsClean(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(string) error
		expectedClean bool
	}{
		{
			name: "clean repository",
			setup: func(dir string) error {
				return nil // No changes
			},
			expectedClean: true,
		},
		{
			name: "untracked file",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "newfile.txt"), []byte("content"), 0644)
			},
			expectedClean: false,
		},
		{
			name: "modified file",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "testfile.txt"), []byte("modified"), 0644)
			},
			expectedClean: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir := t.TempDir()
			ctx := context.Background()

			// Initialize git repo
			cmd := exec.CommandContext(ctx, "git", "init")
			cmd.Dir = tmpDir
			if err := cmd.Run(); err != nil {
				t.Skipf("git not available: %v", err)
			}

			// Configure git for testing
			cmd = exec.CommandContext(ctx, "git", "config", "user.name", "Test User")
			cmd.Dir = tmpDir
			if err := cmd.Run(); err != nil {
				t.Skipf("Failed to configure git user.name: %v", err)
			}
			cmd = exec.CommandContext(ctx, "git", "config", "user.email", "test@example.com")
			cmd.Dir = tmpDir
			if err := cmd.Run(); err != nil {
				t.Skipf("Failed to configure git user.email: %v", err)
			}

			// Create initial file and commit
			testFile := filepath.Join(tmpDir, "testfile.txt")
			if err := os.WriteFile(testFile, []byte("initial"), 0644); err != nil {
				t.Fatalf("Failed to write initial file: %v", err)
			}
			cmd = exec.CommandContext(ctx, "git", "add", ".")
			cmd.Dir = tmpDir
			if err := cmd.Run(); err != nil {
				t.Skipf("Failed to add files: %v", err)
			}
			cmd = exec.CommandContext(ctx, "git", "commit", "-m", "initial commit")
			cmd.Dir = tmpDir
			if err := cmd.Run(); err != nil {
				t.Skipf("Failed to create initial commit: %v", err)
			}

			// Run setup
			if err := tt.setup(tmpDir); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			// Test IsClean
			repo, err := OpenRepo(tmpDir)
			if err != nil {
				t.Fatalf("Failed to open repo: %v", err)
			}

			clean, err := repo.IsClean()
			if err != nil {
				t.Fatalf("IsClean failed: %v", err)
			}

			if clean != tt.expectedClean {
				t.Errorf("IsClean() = %v, want %v", clean, tt.expectedClean)
			}
		})
	}
}

// TestGetCurrentBranch tests the GetCurrentBranch function
func TestGetCurrentBranch(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	// Initialize git repo
	cmd := exec.CommandContext(ctx, "git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Configure git
	cmd = exec.CommandContext(ctx, "git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("Failed to configure git user.name: %v", err)
	}
	cmd = exec.CommandContext(ctx, "git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("Failed to configure git user.email: %v", err)
	}

	// Create initial commit
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	cmd = exec.CommandContext(ctx, "git", "add", ".")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("Failed to add files: %v", err)
	}
	cmd = exec.CommandContext(ctx, "git", "commit", "-m", "initial")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("Failed to create initial commit: %v", err)
	}

	// Test on default branch
	repo, err := OpenRepo(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open repo: %v", err)
	}

	branch, err := repo.GetCurrentBranch()
	if err != nil {
		t.Fatalf("GetCurrentBranch failed: %v", err)
	}

	// Should be master or main depending on git version
	if branch != "master" && branch != "main" {
		t.Errorf("GetCurrentBranch() = %v, want master or main", branch)
	}

	// Create and checkout new branch
	if err := repo.CreateBranch("test-branch"); err != nil {
		t.Fatalf("CreateBranch failed: %v", err)
	}
	if err := repo.CheckoutBranch("test-branch"); err != nil {
		t.Fatalf("CheckoutBranch failed: %v", err)
	}

	// Test on new branch
	branch, err = repo.GetCurrentBranch()
	if err != nil {
		t.Fatalf("GetCurrentBranch failed: %v", err)
	}

	if branch != "test-branch" {
		t.Errorf("GetCurrentBranch() = %v, want test-branch", branch)
	}
}

func TestGitHTTPSAuthExtraHeader(t *testing.T) {
	got := gitHTTPSAuthExtraHeader("test-token")
	wantPrefix := "Authorization: Basic "
	if !strings.HasPrefix(got, wantPrefix) {
		t.Fatalf("gitHTTPSAuthExtraHeader() = %q, want prefix %q", got, wantPrefix)
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(got, wantPrefix))
	if err != nil {
		t.Fatalf("decode auth payload: %v", err)
	}
	if string(decoded) != "x-access-token:test-token" {
		t.Fatalf("auth payload = %q, want %q", decoded, "x-access-token:test-token")
	}
	if strings.Contains(got, "Bearer") {
		t.Fatalf("gitHTTPSAuthExtraHeader() must not use Bearer: %q", got)
	}
}

func TestGitPushEnv(t *testing.T) {
	env := gitPushEnv([]string{"PATH=/usr/bin"}, "test-token")
	wantHeader := gitHTTPSAuthExtraHeader("test-token")
	got := map[string]string{}
	for _, entry := range env {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			t.Fatalf("malformed env entry %q", entry)
		}
		got[key] = value
	}
	if got["PATH"] != "/usr/bin" {
		t.Fatalf("PATH = %q, want /usr/bin", got["PATH"])
	}
	if got["GIT_CONFIG_COUNT"] != "1" {
		t.Fatalf("GIT_CONFIG_COUNT = %q, want 1", got["GIT_CONFIG_COUNT"])
	}
	if got["GIT_CONFIG_KEY_0"] != "http.extraHeader" {
		t.Fatalf("GIT_CONFIG_KEY_0 = %q, want http.extraHeader", got["GIT_CONFIG_KEY_0"])
	}
	if got["GIT_CONFIG_VALUE_0"] != wantHeader {
		t.Fatalf("GIT_CONFIG_VALUE_0 = %q, want %q", got["GIT_CONFIG_VALUE_0"], wantHeader)
	}
}

func TestPushRequiresToken(t *testing.T) {
	gitRepo, _ := createTestRepo(t)

	err := gitRepo.Push("origin", "")
	if err == nil {
		t.Error("Push() should fail without a token, got nil error")
	}
}

func TestPushRequiresOriginRemote(t *testing.T) {
	gitRepo, _ := createTestRepo(t)

	err := gitRepo.Push("upstream", "test-token")
	if err == nil {
		t.Fatal("Push() should reject non-origin remotes")
	}
	if !strings.Contains(err.Error(), "only supports origin") {
		t.Fatalf("Push() error = %v, want origin restriction", err)
	}
}

func TestPushToLocalRemote(t *testing.T) {
	gitRepo, workDir := createTestRepo(t)

	bareDir := t.TempDir()
	if _, err := git.PlainInit(bareDir, true); err != nil {
		t.Fatalf("failed to init bare remote: %v", err)
	}
	remoteURL := "file://" + bareDir
	if err := gitRepo.SetRemoteURL("origin", remoteURL); err != nil {
		t.Fatalf("SetRemoteURL() error = %v", err)
	}

	branch := "release-9.6"
	if err := gitRepo.CreateBranch(branch); err != nil {
		t.Fatalf("CreateBranch() error = %v", err)
	}
	if err := gitRepo.CheckoutBranch(branch); err != nil {
		t.Fatalf("CheckoutBranch() error = %v", err)
	}

	// Token is unused for file:// remotes but required by Push.
	if err := gitRepo.Push("origin", "test-token"); err != nil {
		t.Fatalf("Push() error = %v", err)
	}

	remoteHeads := runGitOutput(t, workDir, "ls-remote", "--heads", "origin", branch)
	if !strings.Contains(remoteHeads, "refs/heads/"+branch) {
		t.Fatalf("Push() did not create remote branch %s; ls-remote:\n%s", branch, remoteHeads)
	}
}

func TestGithubHTTPSPushURL(t *testing.T) {
	tests := []struct {
		name      string
		in        string
		want      string
		rewritten bool
	}{
		{
			name:      "scp ssh",
			in:        "git@github.com:elastic/beats.git",
			want:      "https://github.com/elastic/beats.git",
			rewritten: true,
		},
		{
			name:      "ssh scheme",
			in:        "ssh://git@github.com/elastic/beats.git",
			want:      "https://github.com/elastic/beats.git",
			rewritten: true,
		},
		{
			name:      "ssh scheme with port",
			in:        "ssh://git@github.com:22/elastic/beats.git",
			want:      "https://github.com/elastic/beats.git",
			rewritten: true,
		},
		{
			name:      "https unchanged",
			in:        "https://github.com/elastic/beats.git",
			want:      "https://github.com/elastic/beats.git",
			rewritten: false,
		},
		{
			name:      "non-github ssh unchanged",
			in:        "git@gitlab.com:group/repo.git",
			want:      "git@gitlab.com:group/repo.git",
			rewritten: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, rewritten := githubHTTPSPushURL(tt.in)
			if got != tt.want || rewritten != tt.rewritten {
				t.Errorf("githubHTTPSPushURL(%q) = (%q, %v), want (%q, %v)",
					tt.in, got, rewritten, tt.want, tt.rewritten)
			}
		})
	}
}

func TestEnsureHTTPSRemote(t *testing.T) {
	t.Run("rewrites ssh to https", func(t *testing.T) {
		gitRepo, _ := createTestRepo(t)
		if err := gitRepo.SetRemoteURL("origin", "git@github.com:elastic/beats.git"); err != nil {
			t.Fatalf("SetRemoteURL() error = %v", err)
		}
		if err := gitRepo.ensureHTTPSRemote("origin"); err != nil {
			t.Fatalf("ensureHTTPSRemote() error = %v", err)
		}
		remote, err := gitRepo.repo.Remote("origin")
		if err != nil {
			t.Fatalf("Remote() error = %v", err)
		}
		want := "https://github.com/elastic/beats.git"
		if got := remote.Config().URLs[0]; got != want {
			t.Errorf("origin URL = %q, want %q", got, want)
		}
	})

	t.Run("leaves https unchanged", func(t *testing.T) {
		gitRepo, _ := createTestRepo(t)
		want := "https://github.com/elastic/beats.git"
		if err := gitRepo.SetRemoteURL("origin", want); err != nil {
			t.Fatalf("SetRemoteURL() error = %v", err)
		}
		if err := gitRepo.ensureHTTPSRemote("origin"); err != nil {
			t.Fatalf("ensureHTTPSRemote() error = %v", err)
		}
		remote, err := gitRepo.repo.Remote("origin")
		if err != nil {
			t.Fatalf("Remote() error = %v", err)
		}
		if got := remote.Config().URLs[0]; got != want {
			t.Errorf("origin URL = %q, want %q", got, want)
		}
	})

	t.Run("rejects non-https remote", func(t *testing.T) {
		gitRepo, _ := createTestRepo(t)
		if err := gitRepo.SetRemoteURL("origin", "git@gitlab.com:group/repo.git"); err != nil {
			t.Fatalf("SetRemoteURL() error = %v", err)
		}
		err := gitRepo.ensureHTTPSRemote("origin")
		if err == nil {
			t.Fatal("ensureHTTPSRemote() expected error for non-HTTPS remote")
		}
		if !strings.Contains(err.Error(), "not HTTPS") {
			t.Errorf("ensureHTTPSRemote() error = %v, want mention of not HTTPS", err)
		}
	})
}

func TestSetRemoteURL(t *testing.T) {
	gitRepo, _ := createTestRepo(t)

	err := gitRepo.SetRemoteURL("origin", "https://github.com/test/repo.git")
	if err != nil {
		t.Errorf("SetRemoteURL() error = %v", err)
		return
	}

	remote, err := gitRepo.repo.Remote("origin")
	if err != nil {
		t.Errorf("Failed to get remote: %v", err)
		return
	}
	if len(remote.Config().URLs) == 0 || remote.Config().URLs[0] != "https://github.com/test/repo.git" {
		t.Errorf("SetRemoteURL() URL = %v, want https://github.com/test/repo.git", remote.Config().URLs)
	}
}

// TestCreateAndCheckoutBranch tests branch creation and checkout
func TestCreateAndCheckoutBranch(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	// Initialize git repo
	cmd := exec.CommandContext(ctx, "git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Configure git
	cmd = exec.CommandContext(ctx, "git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("Failed to configure git user.name: %v", err)
	}
	cmd = exec.CommandContext(ctx, "git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("Failed to configure git user.email: %v", err)
	}

	// Create initial commit
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	cmd = exec.CommandContext(ctx, "git", "add", ".")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("Failed to add files: %v", err)
	}
	cmd = exec.CommandContext(ctx, "git", "commit", "-m", "initial")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("Failed to create initial commit: %v", err)
	}

	repo, err := OpenRepo(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open repo: %v", err)
	}

	// Create new branch
	branchName := "feature-branch"
	if err := repo.CreateBranch(branchName); err != nil {
		t.Fatalf("CreateBranch failed: %v", err)
	}

	// Checkout the branch
	if err := repo.CheckoutBranch(branchName); err != nil {
		t.Fatalf("CheckoutBranch failed: %v", err)
	}

	// Verify we're on the new branch
	currentBranch, err := repo.GetCurrentBranch()
	if err != nil {
		t.Fatalf("GetCurrentBranch failed: %v", err)
	}

	if currentBranch != branchName {
		t.Errorf("GetCurrentBranch() = %v, want %v", currentBranch, branchName)
	}

	// Creating the same branch again should be idempotent
	if err := repo.CreateBranch(branchName); err != nil {
		t.Fatalf("CreateBranch failed on second call: %v", err)
	}
}

// TestCommitAll tests the CommitAll function
func TestCommitAll(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	// Initialize git repo
	cmd := exec.CommandContext(ctx, "git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Configure git
	cmd = exec.CommandContext(ctx, "git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("Failed to configure git user.name: %v", err)
	}
	cmd = exec.CommandContext(ctx, "git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("Failed to configure git user.email: %v", err)
	}

	// Create initial commit
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("initial"), 0644); err != nil {
		t.Fatalf("Failed to write initial file: %v", err)
	}
	cmd = exec.CommandContext(ctx, "git", "add", ".")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("Failed to add files: %v", err)
	}
	cmd = exec.CommandContext(ctx, "git", "commit", "-m", "initial")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("Failed to create initial commit: %v", err)
	}

	repo, err := OpenRepo(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open repo: %v", err)
	}

	// Make a change
	if err := os.WriteFile(testFile, []byte("modified"), 0644); err != nil {
		t.Fatalf("Failed to write modified file: %v", err)
	}

	// Commit the change
	committed, err := repo.CommitAll("test commit", "Test User", "test@example.com")
	if err != nil {
		t.Fatalf("CommitAll failed: %v", err)
	}
	if !committed {
		t.Fatal("CommitAll should report a commit was created")
	}

	// Verify repo is clean after commit
	clean, err := repo.IsClean()
	if err != nil {
		t.Fatalf("IsClean failed: %v", err)
	}

	if !clean {
		t.Error("Repository should be clean after commit")
	}

	// Second commit attempt should be a no-op
	committed, err = repo.CommitAll("test commit", "Test User", "test@example.com")
	if err != nil {
		t.Fatalf("CommitAll failed on second call: %v", err)
	}
	if committed {
		t.Error("CommitAll should be idempotent when there are no changes")
	}
}

func TestHasCommitsAheadOf(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T, dir, baseBranch string)
		want     bool
		wantErr  bool
		baseName string
	}{
		{
			name:     "equal tips",
			baseName: "base",
			setup: func(t *testing.T, dir, baseBranch string) {
				runGit(t, dir, "branch", baseBranch)
			},
			want: false,
		},
		{
			name:     "ahead of base",
			baseName: "base",
			setup: func(t *testing.T, dir, baseBranch string) {
				runGit(t, dir, "branch", baseBranch)
				writeAndCommit(t, dir, "feature.txt", "feature", "feature commit")
			},
			want: true,
		},
		{
			name:     "behind base",
			baseName: "base",
			setup: func(t *testing.T, dir, baseBranch string) {
				runGit(t, dir, "branch", baseBranch)
				runGit(t, dir, "checkout", baseBranch)
				writeAndCommit(t, dir, "base-only.txt", "base", "base commit")
				runGit(t, dir, "checkout", "-")
			},
			want: false,
		},
		{
			name:     "diverged from base",
			baseName: "base",
			setup: func(t *testing.T, dir, baseBranch string) {
				runGit(t, dir, "branch", baseBranch)
				writeAndCommit(t, dir, "feature.txt", "feature", "feature commit")
				runGit(t, dir, "checkout", baseBranch)
				writeAndCommit(t, dir, "base-only.txt", "base", "base commit")
				runGit(t, dir, "checkout", "-")
			},
			want: true,
		},
		{
			name:     "missing base branch",
			baseName: "does-not-exist",
			setup:    func(t *testing.T, dir, baseBranch string) {},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := initGitRepo(t)
			tt.setup(t, dir, tt.baseName)

			repo, err := OpenRepo(dir)
			if err != nil {
				t.Fatalf("OpenRepo failed: %v", err)
			}

			got, err := repo.HasCommitsAheadOf(tt.baseName)
			if tt.wantErr {
				if err == nil {
					t.Fatal("HasCommitsAheadOf() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("HasCommitsAheadOf() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("HasCommitsAheadOf() = %v, want %v", got, tt.want)
			}
		})
	}
}

func createTestRepo(t *testing.T) (*GitRepo, string) {
	t.Helper()

	tmpDir := t.TempDir()

	repo, err := git.PlainInit(tmpDir, false)
	if err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}

	w, err := repo.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}

	testFile := filepath.Join(tmpDir, "README.md")
	err = os.WriteFile(testFile, []byte("# Test Repo"), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err = w.Add("README.md")
	if err != nil {
		t.Fatalf("failed to add file: %v", err)
	}

	_, err = w.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
		},
	})
	if err != nil {
		t.Fatalf("failed to create initial commit: %v", err)
	}

	headRef, err := repo.Head()
	if err != nil {
		t.Fatalf("failed to get HEAD: %v", err)
	}

	mainRef := plumbing.NewHashReference(plumbing.NewBranchReferenceName("main"), headRef.Hash())
	if err := repo.Storer.SetReference(mainRef); err != nil {
		t.Fatalf("failed to create main branch: %v", err)
	}
	if err := w.Checkout(&git.CheckoutOptions{Branch: plumbing.NewBranchReferenceName("main")}); err != nil {
		t.Fatalf("failed to checkout main: %v", err)
	}
	_ = repo.Storer.RemoveReference(plumbing.NewBranchReferenceName("master"))

	return &GitRepo{repo: repo, path: tmpDir}, tmpDir
}

func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()

	cmd := exec.CommandContext(ctx, "git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	runGit(t, dir, "config", "user.name", "Test User")
	runGit(t, dir, "config", "user.email", "test@example.com")
	writeAndCommit(t, dir, "README", "initial", "initial commit")
	return dir
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	_ = runGitOutput(t, dir, args...)
}

func runGitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	gitArgs := append([]string{"-c", "protocol.file.allow=always"}, args...)
	cmd := exec.CommandContext(context.Background(), "git", gitArgs...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
	return string(out)
}

func writeAndCommit(t *testing.T, dir, filename, content, message string) {
	t.Helper()
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", filename, err)
	}
	runGit(t, dir, "add", filename)
	runGit(t, dir, "commit", "-m", message)
}
