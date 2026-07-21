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
	"strings"
	"testing"

	"github.com/google/go-github/v68/github"
)

func TestReleaseIssueTitle(t *testing.T) {
	got := releaseIssueTitle("9.4.1")
	want := "[RELEASE 9.4.1] Instructions & Checklist"
	if got != want {
		t.Fatalf("releaseIssueTitle() = %q, want %q", got, want)
	}
}

func TestIsMajorMinorRelease(t *testing.T) {
	if !isMajorMinorRelease("9.5.0") {
		t.Fatal("expected 9.5.0 to be treated as major/minor")
	}
	if isMajorMinorRelease("9.5.1") {
		t.Fatal("expected 9.5.1 not to be treated as major/minor")
	}
}

func TestVersionMentioned(t *testing.T) {
	if !versionMentioned("[Release] Update version to 9.4.1", "9.4.1") {
		t.Fatal("exact version in title should match")
	}
	if versionMentioned("[Release] Update version to 9.4.10", "9.4.1") {
		t.Fatal("9.4.1 must not match 9.4.10")
	}
	if !versionMentioned("Add Beats 9.4.1 release notes", "9.4.1") {
		t.Fatal("docs PR title should match")
	}
}

func TestBuildReleaseIssueBodyMajorMinorSection(t *testing.T) {
	minorBody := buildReleaseIssueBody("9.5.0", true, nil, nil)
	if !strings.Contains(minorBody, "### Only For Major/Minor Releases") {
		t.Fatal("minor release body should include major/minor section")
	}
	if !strings.Contains(minorBody, globalReleaseTrackerURL) {
		t.Fatal("body should link the global tracker")
	}

	patchBody := buildReleaseIssueBody("9.5.1", false, nil, nil)
	if strings.Contains(patchBody, "### Only For Major/Minor Releases") {
		t.Fatal("patch body should omit major/minor section")
	}
}

func TestMergePRURLsSortsAndDedups(t *testing.T) {
	got := mergePRURLs(
		[]string{"https://github.com/elastic/beats/pull/20", "https://github.com/elastic/beats/pull/10"},
		[]string{"https://github.com/elastic/beats/pull/10", "https://github.com/elastic/beats/pull/15"},
	)
	want := []string{
		"https://github.com/elastic/beats/pull/10",
		"https://github.com/elastic/beats/pull/15",
		"https://github.com/elastic/beats/pull/20",
	}
	if len(got) != len(want) {
		t.Fatalf("mergePRURLs() len = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("mergePRURLs()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestExtractPRCheckboxes(t *testing.T) {
	body := `
## PRs

- [x] https://github.com/elastic/beats/pull/10
- [ ] https://github.com/elastic/beats/pull/11
`
	got := extractPRCheckboxes(body)
	if !got["https://github.com/elastic/beats/pull/10"] {
		t.Fatal("checked PR should be true")
	}
	if got["https://github.com/elastic/beats/pull/11"] {
		t.Fatal("unchecked PR should be false")
	}
}

func TestMergeReleaseIssueBodyAddsMissingPRsAndPreservesChecks(t *testing.T) {
	existing := `Global tracker: https://github.com/elastic/ingest-dev/issues/8866

# Release Checklist

- [x] Find the Beats release PRs

## PRs

- [x] https://github.com/elastic/beats/pull/10
- [ ] https://github.com/elastic/beats/pull/11
`
	updated, changed := mergeReleaseIssueBody(existing, "9.4.1", []string{
		"https://github.com/elastic/beats/pull/11",
		"https://github.com/elastic/beats/pull/12",
	})
	if !changed {
		t.Fatal("adding a missing PR should mark the body changed")
	}
	if !strings.Contains(updated, "- [x] https://github.com/elastic/beats/pull/10") {
		t.Fatal("existing checked PR must stay checked")
	}
	if !strings.Contains(updated, "- [ ] https://github.com/elastic/beats/pull/11") {
		t.Fatal("existing unchecked PR stays unchecked")
	}
	if !strings.Contains(updated, "- [ ] https://github.com/elastic/beats/pull/12") {
		t.Fatal("new PR should be appended unchecked")
	}
	if !strings.Contains(updated, "- [x] Find the Beats release PRs") {
		t.Fatal("non-PR checklist state must be preserved")
	}

	_, changedAgain := mergeReleaseIssueBody(updated, "9.4.1", []string{
		"https://github.com/elastic/beats/pull/10",
		"https://github.com/elastic/beats/pull/11",
		"https://github.com/elastic/beats/pull/12",
	})
	if changedAgain {
		t.Fatal("second merge with same PRs should be a no-op")
	}
}

func TestMergeReleaseIssueBodyAddsTrackerWhenMissing(t *testing.T) {
	existing := `# Release Checklist

## PRs

- [ ] https://github.com/elastic/beats/pull/10
`
	updated, changed := mergeReleaseIssueBody(existing, "9.4.1", []string{"https://github.com/elastic/beats/pull/10"})
	if !changed {
		t.Fatal("missing global tracker should trigger an update")
	}
	if !strings.HasPrefix(strings.TrimSpace(updated), "Global tracker: "+globalReleaseTrackerURL) {
		t.Fatalf("tracker link should be prepended, got prefix %q", updated[:min(80, len(updated))])
	}
}

func TestEnsureReleaseIssueTrackerDryRun(t *testing.T) {
	cfg := &ReleaseConfig{
		CurrentRelease: "9.4.1",
		DryRun:         true,
	}
	pr := &github.PullRequest{HTMLURL: github.Ptr("https://github.com/elastic/beats/pull/99")}
	if err := EnsureReleaseIssueTracker(cfg, []*github.PullRequest{pr}); err != nil {
		t.Fatalf("dry-run ensureIssueTracker should not call GitHub: %v", err)
	}
}

func TestPrURLsFromPullRequestsSkipsNil(t *testing.T) {
	got := prURLsFromPullRequests([]*github.PullRequest{
		nil,
		{HTMLURL: github.Ptr("https://github.com/elastic/beats/pull/1")},
	})
	if len(got) != 1 || got[0] != "https://github.com/elastic/beats/pull/1" {
		t.Fatalf("unexpected URLs: %v", got)
	}
}
