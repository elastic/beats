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
	"testing"
	"time"

	"github.com/google/go-github/v68/github"
)

func TestPickRelatedPRPrefersOpenThenMerged(t *testing.T) {
	mergedAt := time.Now()
	open := &github.PullRequest{Number: github.Ptr(1), State: github.Ptr("open")}
	merged := &github.PullRequest{
		Number:   github.Ptr(2),
		State:    github.Ptr("closed"),
		Merged:   github.Ptr(true),
		MergedAt: &github.Timestamp{Time: mergedAt},
	}
	closed := &github.PullRequest{Number: github.Ptr(3), State: github.Ptr("closed")}

	got := pickRelatedPR([]*github.PullRequest{closed, merged, open})
	if got == nil || got.GetNumber() != 1 {
		t.Fatalf("expected open PR #1, got %#v", got)
	}

	got = pickRelatedPR([]*github.PullRequest{closed, merged})
	if got == nil || got.GetNumber() != 2 {
		t.Fatalf("expected merged PR #2, got %#v", got)
	}

	got = pickRelatedPR([]*github.PullRequest{closed})
	if got == nil || got.GetNumber() != 3 {
		t.Fatalf("expected closed PR #3, got %#v", got)
	}

	if pickRelatedPR(nil) != nil {
		t.Fatal("expected nil for empty input")
	}
}

func TestPRDisplayState(t *testing.T) {
	mergedAt := time.Now()
	cases := []struct {
		name string
		pr   *github.PullRequest
		want string
	}{
		{name: "nil", pr: nil, want: "unknown"},
		{name: "open", pr: &github.PullRequest{State: github.Ptr("open")}, want: "open"},
		{
			name: "merged",
			pr: &github.PullRequest{
				State:    github.Ptr("closed"),
				Merged:   github.Ptr(true),
				MergedAt: &github.Timestamp{Time: mergedAt},
			},
			want: "merged",
		},
		{name: "closed", pr: &github.PullRequest{State: github.Ptr("closed")}, want: "closed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := prDisplayState(tc.pr); got != tc.want {
				t.Fatalf("prDisplayState() = %q, want %q", got, tc.want)
			}
		})
	}
}
