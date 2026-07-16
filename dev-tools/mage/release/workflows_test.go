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
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestRunMajorMinorReleaseDryRunBranches(t *testing.T) {
	origMakeUpdate := runMakeUpdate
	runMakeUpdate = func() error { return nil }
	t.Cleanup(func() { runMakeUpdate = origMakeUpdate })

	origFetch := fetchLatestReleaseBefore
	fetchLatestReleaseBefore = func(token, owner, repo, current string) (string, error) {
		return "9.4.3", nil
	}
	t.Cleanup(func() { fetchLatestReleaseBefore = origFetch })

	tmpDir := setupWorkflowTestRepo(t)

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Errorf("failed to restore cwd: %v", err)
		}
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	cfg := &ReleaseConfig{
		CurrentRelease:          "9.5.0",
		NextRelease:             "9.5.1",
		NextProjectMinorVersion: "9.6.0",
		NextProjectMinorBranch:  "9.6",
		BaseBranch:              "main",
		ReleaseBranch:           "9.5",
		DryRun:                  true,
		GitAuthorName:           "Test User",
		GitAuthorEmail:          "test@example.com",
	}

	if err := RunMajorMinorRelease(cfg); err != nil {
		t.Fatalf("RunMajorMinorRelease dry run failed: %v", err)
	}

	repo, err := OpenRepo(".")
	if err != nil {
		t.Fatalf("failed to open repo: %v", err)
	}

	wantBranches := []string{
		"main",
		"9.5",
		"ff-prep-main-9.5.0",
		"ff-release-9.5.0",
		"ff-prep-main-docs-env-9.6.0",
		"ff-prep-next-patch-9.5.1",
	}
	for _, branch := range wantBranches {
		exists, err := repo.BranchExists(branch)
		if err != nil {
			t.Fatalf("failed checking branch %s: %v", branch, err)
		}
		if !exists {
			t.Errorf("expected branch %s to exist after dry run", branch)
		}
	}

	// PR-C must not rewrite README /main/ → next minor branch.
	assertGitShowContains(t, tmpDir, "ff-prep-main-docs-env-9.6.0", "README.md", "/main/")
	assertGitShowNotContains(t, tmpDir, "ff-prep-main-docs-env-9.6.0", "README.md", "/9.6/")
	assertGitShowContains(t, tmpDir, "ff-prep-main-docs-env-9.6.0", "libbeat/docs/version.asciidoc", ":stack-version: 9.6.0")

	// PR-D bumps version.go only for docs-related paths (no stack-version churn).
	assertGitShowContains(t, tmpDir, "ff-prep-next-patch-9.5.1", "libbeat/version/version.go", `defaultBeatVersion = "9.5.1"`)
	assertGitShowContains(t, tmpDir, "ff-prep-next-patch-9.5.1", "libbeat/docs/version.asciidoc", ":stack-version: 9.4.3")
	assertGitShowContains(t, tmpDir, "ff-prep-next-patch-9.5.1", "testing/environments/latest.yml", "elasticsearch:9.5.0")
}

func assertGitShowContains(t *testing.T, dir, branch, file, want string) {
	t.Helper()
	out, err := exec.CommandContext(context.Background(), "git", "-C", dir, "show", branch+":"+file).CombinedOutput()
	if err != nil {
		t.Fatalf("git show %s:%s: %v (%s)", branch, file, err, out)
	}
	if !strings.Contains(string(out), want) {
		t.Errorf("%s:%s should contain %q, got:\n%s", branch, file, want, out)
	}
}

func assertGitShowNotContains(t *testing.T, dir, branch, file, forbid string) {
	t.Helper()
	out, err := exec.CommandContext(context.Background(), "git", "-C", dir, "show", branch+":"+file).CombinedOutput()
	if err != nil {
		t.Fatalf("git show %s:%s: %v (%s)", branch, file, err, out)
	}
	if strings.Contains(string(out), forbid) {
		t.Errorf("%s:%s should not contain %q, got:\n%s", branch, file, forbid, out)
	}
}

func TestMajorMinorPrepBranchNames(t *testing.T) {
	cfg := &ReleaseConfig{
		CurrentRelease:          "9.5.0",
		NextRelease:             "9.5.1",
		NextProjectMinorVersion: "9.6.0",
		ReleaseBranch:           "9.5",
		BaseBranch:              "main",
	}

	cases := []struct {
		name   string
		branch string
	}{
		{name: "PR-A", branch: fmt.Sprintf("ff-prep-main-%s", cfg.CurrentRelease)},
		{name: "PR-B", branch: fmt.Sprintf("ff-release-%s", cfg.CurrentRelease)},
		{name: "PR-C", branch: fmt.Sprintf("ff-prep-main-docs-env-%s", cfg.NextProjectMinorVersion)},
		{name: "PR-D", branch: fmt.Sprintf("ff-prep-next-patch-%s", cfg.NextRelease)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.branch == "" {
				t.Fatal("branch name should not be empty")
			}
		})
	}

	casesLabels := []struct {
		name   string
		labels []string
		want   string
	}{
		{name: "PR-A", labels: prAMainLabels(cfg.ReleaseBranch), want: mergeLabelFFDay},
		{name: "PR-B", labels: prBReleaseLabels(), want: mergeLabelAfterBranch},
		{name: "PR-C", labels: prCMainLabels(cfg.ReleaseBranch), want: mergeLabelAfterImages},
		{name: "PR-D", labels: prDNextPatchLabels(), want: mergeLabelAfterRelease},
	}
	for _, tc := range casesLabels {
		t.Run(tc.name+" labels", func(t *testing.T) {
			if !slices.Contains(tc.labels, tc.want) {
				t.Errorf("%s labels should include %q, got %v", tc.name, tc.want, tc.labels)
			}
		})
	}

	labelsA := prAMainLabels(cfg.ReleaseBranch)
	if !slices.Contains(labelsA, "backport-9.5") {
		t.Errorf("PR-A labels should include backport-9.5, got %v", labelsA)
	}
}

func setupWorkflowTestRepo(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	ctx := context.Background()

	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.CommandContext(ctx, "git", args...)
		cmd.Dir = tmpDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("git not available: %v (%s)", err, out)
		}
	}

	runGit("init", "-b", "main")
	runGit("config", "user.name", "Test User")
	runGit("config", "user.email", "test@example.com")

	files := map[string]string{
		"libbeat/version/version.go": `package version

const defaultBeatVersion = "9.4.3"
`,
		"libbeat/docs/version.asciidoc": `:stack-version: 9.4.3
:doc-branch: main
`,
		".mergify.yml": `pull_request_rules:
  - name: backport patches to 9.4 branch
    conditions:
      - merged
      - label=backport-9.4
    actions:
      backport:
        branches:
          - "9.4"
`,
		"testing/environments/latest.yml": `services:
  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:9.4.3
`,
		"metricbeat/docker-compose.yml": `image: docker.elastic.co/integrations-ci/beats-elasticsearch:${ELASTICSEARCH_VERSION:-9.4.3}-1
`,
		"README.md": "# Beats\n\nDocs: https://www.elastic.co/guide/en/beats/libbeat/main/index.html\n",
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("failed to create dir for %s: %v", path, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", path, err)
		}
	}

	runGit("add", ".")
	runGit("commit", "-m", "initial commit")

	return tmpDir
}
