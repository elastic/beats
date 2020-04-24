package doghouse

import "github.com/reviewdog/reviewdog"

// CheckRequest represents doghouse GitHub check request.
type CheckRequest struct {
	// Commit SHA.
	// Required.
	SHA string `json:"sha,omitempty"`
	// PullRequest number.
	// Optional.
	PullRequest int `json:"pull_request,omitempty"`
	// Owner of the repository.
	// Required.
	Owner string `json:"owner,omitempty"`
	// Repository name.
	// Required.
	Repo string `json:"repo,omitempty"`

	// Branch name.
	// Optional.
	// DEPRECATED: No need to fill this field.
	Branch string `json:"branch,omitempty"`

	// Annotations associated with the repository's commit and Pull Request.
	Annotations []*Annotation `json:"annotations,omitempty"`

	// Name of the annotation tool.
	// Optional.
	Name string `json:"name,omitempty"`

	// Level is report level for this request.
	// One of ["info", "warning", "error"]. Default is "error".
	// Optional.
	Level string `json:"level"`

	// OutsideDiff represents whether it report results in outside diff or not as
	// annotations. It's useful only when PullRequest != 0. If PullRequest is
	// empty, it will always report results all resutls including outside diff
	// (because there are no diff!).
	// Optional.
	OutsideDiff bool `json:"outside_diff"`
}

// CheckResponse represents doghouse GitHub check response.
type CheckResponse struct {
	// ReportURL is report URL of check run.
	// Optional.
	ReportURL string `json:"report_url,omitempty"`

	// CheckedResults is checked annotations result.
	// This field is expected to be filled for GitHub Actions integration and
	// filled when ReportURL is not available. i.e. reviewdog doesn't have write
	// permission to Check API.
	// It's also not expected to be passed over network via JSON.
	// TODO(haya14busa): Consider to move this type to this package to avoid
	// (cyclic) import.
	// Optional.
	CheckedResults []*reviewdog.FilteredCheck
}

// Annotation represents an annotation to file or specific line.
type Annotation struct {
	// Relative file path
	// Required.
	Path string `json:"path,omitempty"`
	// Line number.
	// Optional.
	Line int `json:"line,omitempty"`
	// Annotation message.
	// Required.
	Message string `json:"message,omitempty"`
	// Original error message of this annotation.
	// Optional.
	RawMessage string `json:"raw_message,omitempty"`
}
