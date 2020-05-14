package cienv

import (
	"encoding/json"
	"errors"
	"os"
)

// https://help.github.com/en/articles/virtual-environments-for-github-actions#default-environment-variables
type GitHubEvent struct {
	PullRequest GitHubPullRequest `json:"pull_request"`
	Repository  struct {
		Owner struct {
			Login string `json:"login"`
		} `json:"owner"`
		Name string `json:"name"`
	} `json:"repository"`
	CheckSuite struct {
		After        string              `json:"after"`
		PullRequests []GitHubPullRequest `json:"pull_requests"`
	} `json:"check_suite"`
	HeadCommit struct {
		ID string `json:"id"`
	} `json:"head_commit"`
}

type GitHubPullRequest struct {
	Number int `json:"number"`
	Head   struct {
		Sha  string `json:"sha"`
		Ref  string `json:"ref"`
		Repo struct {
			Fork bool `json:"fork"`
		} `json:"repo"`
	} `json:"head"`
}

// LoadGitHubEvent loads GitHubEvent if it's running in GitHub Actions.
func LoadGitHubEvent() (*GitHubEvent, error) {
	eventPath := os.Getenv("GITHUB_EVENT_PATH")
	if eventPath == "" {
		return nil, errors.New("GITHUB_EVENT_PATH not found")
	}
	return loadGitHubEventFromPath(eventPath)
}

func loadGitHubEventFromPath(eventPath string) (*GitHubEvent, error) {
	f, err := os.Open(eventPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var event GitHubEvent
	if err := json.NewDecoder(f).Decode(&event); err != nil {
		return nil, err
	}
	return &event, nil
}

func getBuildInfoFromGitHubAction() (*BuildInfo, bool, error) {
	eventPath := os.Getenv("GITHUB_EVENT_PATH")
	if eventPath == "" {
		return nil, false, errors.New("GITHUB_EVENT_PATH not found")
	}
	return getBuildInfoFromGitHubActionEventPath(eventPath)
}
func getBuildInfoFromGitHubActionEventPath(eventPath string) (*BuildInfo, bool, error) {
	event, err := loadGitHubEventFromPath(eventPath)
	if err != nil {
		return nil, false, err
	}
	info := &BuildInfo{
		Owner:       event.Repository.Owner.Login,
		Repo:        event.Repository.Name,
		PullRequest: event.PullRequest.Number,
		Branch:      event.PullRequest.Head.Ref,
		SHA:         event.PullRequest.Head.Sha,
	}
	// For re-run check_suite event.
	if info.PullRequest == 0 && len(event.CheckSuite.PullRequests) > 0 {
		pr := event.CheckSuite.PullRequests[0]
		info.PullRequest = pr.Number
		info.Branch = pr.Head.Ref
		info.SHA = pr.Head.Sha
	}
	if info.SHA == "" {
		info.SHA = event.HeadCommit.ID
	}
	return info, info.PullRequest != 0, nil
}

// IsInGitHubAction returns true if reviewdog is running in GitHub Actions.
func IsInGitHubAction() bool {
	// https://help.github.com/en/articles/virtual-environments-for-github-actions#default-environment-variables
	return os.Getenv("GITHUB_ACTION") != ""
}
