package gitlab

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/xanzy/go-gitlab"

	"github.com/reviewdog/reviewdog"
	"github.com/reviewdog/reviewdog/service/serviceutil"
)

var _ reviewdog.DiffService = &GitLabMergeRequestDiff{}

// GitLabMergeRequestDiff is a diff service for GitLab MergeRequest.
type GitLabMergeRequestDiff struct {
	cli      *gitlab.Client
	pr       int
	sha      string
	projects string

	// wd is working directory relative to root of repository.
	wd string
}

// NewGitLabMergeRequestDiff returns a new GitLabMergeRequestDiff service.
// itLabMergeRequestDiff service needs git command in $PATH.
func NewGitLabMergeRequestDiff(cli *gitlab.Client, owner, repo string, pr int, sha string) (*GitLabMergeRequestDiff, error) {
	workDir, err := serviceutil.GitRelWorkdir()
	if err != nil {
		return nil, fmt.Errorf("GitLabMergeRequestCommitCommenter needs 'git' command: %v", err)
	}
	return &GitLabMergeRequestDiff{
		cli:      cli,
		pr:       pr,
		sha:      sha,
		projects: owner + "/" + repo,
		wd:       workDir,
	}, nil
}

// Diff returns a diff of MergeRequest. It runs `git diff` locally instead of
// diff_url of GitLab Merge Request because diff of diff_url is not suited for
// comment API in a sense that diff of diff_url is equivalent to
// `git diff --no-renames`, we want diff which is equivalent to
// `git diff --find-renames`.
func (g *GitLabMergeRequestDiff) Diff(ctx context.Context) ([]byte, error) {
	mr, _, err := g.cli.MergeRequests.GetMergeRequest(g.projects, g.pr, nil)
	if err != nil {
		return nil, err
	}
	targetBranch, _, err := g.cli.Branches.GetBranch(mr.TargetProjectID, mr.TargetBranch, nil)
	if err != nil {
		return nil, err
	}
	return g.gitDiff(ctx, g.sha, targetBranch.Commit.ID)
}

func (g *GitLabMergeRequestDiff) gitDiff(_ context.Context, baseSha, targetSha string) ([]byte, error) {
	b, err := exec.Command("git", "merge-base", targetSha, baseSha).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get merge-base commit: %v", err)
	}
	mergeBase := strings.Trim(string(b), "\n")
	relArg := fmt.Sprintf("--relative=%s", g.wd)
	bytes, err := exec.Command("git", "diff", relArg, "--find-renames", mergeBase, baseSha).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run git diff: %v", err)
	}
	return bytes, nil
}

// Strip returns 1 as a strip of git diff.
func (g *GitLabMergeRequestDiff) Strip() int {
	return 1
}
