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
	"os"
	"os/exec"
	"path/filepath"
	"testing"
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

			// Initialize git repo
			cmd := exec.Command("git", "init")
			cmd.Dir = tmpDir
			if err := cmd.Run(); err != nil {
				t.Skipf("git not available: %v", err)
			}

			// Configure git for testing
			cmd = exec.Command("git", "config", "user.name", "Test User")
			cmd.Dir = tmpDir
			cmd.Run()
			cmd = exec.Command("git", "config", "user.email", "test@example.com")
			cmd.Dir = tmpDir
			cmd.Run()

			// Create initial file and commit
			testFile := filepath.Join(tmpDir, "testfile.txt")
			os.WriteFile(testFile, []byte("initial"), 0644)
			cmd = exec.Command("git", "add", ".")
			cmd.Dir = tmpDir
			cmd.Run()
			cmd = exec.Command("git", "commit", "-m", "initial commit")
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

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Configure git
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create initial commit
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("test"), 0644)
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "initial")
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

// TestPushRequiresToken tests that Push requires GITHUB_TOKEN
func TestPushRequiresToken(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Configure git
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create initial commit
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("test"), 0644)
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "initial")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("Failed to create initial commit: %v", err)
	}

	repo, err := OpenRepo(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open repo: %v", err)
	}

	// Unset GITHUB_TOKEN if it exists
	oldToken := os.Getenv("GITHUB_TOKEN")
	os.Unsetenv("GITHUB_TOKEN")
	defer func() {
		if oldToken != "" {
			os.Setenv("GITHUB_TOKEN", oldToken)
		}
	}()

	// Push should fail without token
	err = repo.Push("origin")
	if err == nil {
		t.Error("Push() should fail without GITHUB_TOKEN, got nil error")
	}
	if err != nil && err.Error() != "GITHUB_TOKEN environment variable is required for pushing" {
		t.Errorf("Push() error = %v, want 'GITHUB_TOKEN environment variable is required for pushing'", err)
	}
}

// TestCreateAndCheckoutBranch tests branch creation and checkout
func TestCreateAndCheckoutBranch(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Configure git
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create initial commit
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("test"), 0644)
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "initial")
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
}

// TestCommitAll tests the CommitAll function
func TestCommitAll(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Configure git
	exec.Command("git", "config", "user.name", "Test User").Dir = tmpDir
	exec.Command("git", "config", "user.email", "test@example.com").Dir = tmpDir

	// Create initial commit
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("initial"), 0644)
	exec.Command("git", "add", ".").Dir = tmpDir
	exec.Command("git", "commit", "-m", "initial").Dir = tmpDir

	repo, err := OpenRepo(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open repo: %v", err)
	}

	// Make a change
	os.WriteFile(testFile, []byte("modified"), 0644)

	// Commit the change
	err = repo.CommitAll("test commit", "Test User", "test@example.com")
	if err != nil {
		t.Fatalf("CommitAll failed: %v", err)
	}

	// Verify repo is clean after commit
	clean, err := repo.IsClean()
	if err != nil {
		t.Fatalf("IsClean failed: %v", err)
	}

	if !clean {
		t.Error("Repository should be clean after commit")
	}
}
