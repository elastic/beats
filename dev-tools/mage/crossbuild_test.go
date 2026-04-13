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

package mage

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// git runs a git command in dir and fails the test on error.
// It returns the combined stdout output.
func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	// Isolate from the user/system git configuration.
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %s failed: %s", strings.Join(args, " "), out)
	return strings.TrimSpace(string(out))
}

// initRepo creates a minimal git repository in a temp directory with one
// commit so that worktrees can be created from it.  Returns the resolved
// absolute path.
func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Resolve symlinks so path comparisons are deterministic
	// (e.g. /tmp may be a symlink on some systems).
	dir, err := filepath.EvalSymlinks(dir)
	require.NoError(t, err, "resolving temp dir symlinks")

	git(t, dir, "init")
	git(t, dir, "config", "user.email", "test@test")
	git(t, dir, "config", "user.name", "test")
	git(t, dir, "commit", "--allow-empty", "-m", "init")
	return dir
}

func TestGitWorktreeVolumes_RegularRepo(t *testing.T) {
	repo := initRepo(t)
	t.Chdir(repo)

	vols, err := gitWorktreeVolumes(repo)
	assert.NoError(t, err, "unexpected error for a regular repository")
	assert.Nil(t, vols, "expected no volumes for a regular repository, got: %v", vols)
}

func TestGitWorktreeVolumes_Worktree(t *testing.T) {
	repo := initRepo(t)
	worktree := filepath.Join(t.TempDir(), "wt")

	// Resolve the parent so the worktree path is also symlink-free.
	worktreeParent, err := filepath.EvalSymlinks(filepath.Dir(worktree))
	require.NoError(t, err, "resolving worktree parent symlinks")
	worktree = filepath.Join(worktreeParent, filepath.Base(worktree))

	git(t, repo, "worktree", "add", worktree)

	// gitWorktreeVolumes shells out to git, which needs CWD inside the worktree.
	t.Chdir(worktree)

	vols, err := gitWorktreeVolumes(worktree)
	require.NoError(t, err, "unexpected error for a worktree")

	require.Equal(t, 4, len(vols),
		"expected 4 elements (-v <path>:... -v <path>:...), got %d: %v", len(vols), vols)

	assert.Equal(t, "-v", vols[0])
	assert.Equal(t, "-v", vols[2])

	// The worktree-specific git dir should be under the main repo's
	// .git/worktrees/ directory.
	expectedGitDir := filepath.Join(repo, ".git", "worktrees", filepath.Base(worktree))
	assert.Equal(t, expectedGitDir+":"+expectedGitDir+":ro", vols[1],
		"worktree git dir mount mismatch")

	// The common git dir should be the main repo's .git directory.
	expectedCommonDir := filepath.Join(repo, ".git")
	assert.Equal(t, expectedCommonDir+":"+expectedCommonDir+":ro", vols[3],
		"common git dir mount mismatch")

	// Both mounts must be read-only.
	assert.True(t, strings.HasSuffix(vols[1], ":ro"), "worktree git dir mount should be read-only")
	assert.True(t, strings.HasSuffix(vols[3], ":ro"), "common git dir mount should be read-only")
}

func TestGitWorktreeVolumes_NoGit(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	vols, err := gitWorktreeVolumes(dir)
	assert.Error(t, err, "expected error when .git is missing, got volumes: %v", vols)
	assert.Nil(t, vols)
}
