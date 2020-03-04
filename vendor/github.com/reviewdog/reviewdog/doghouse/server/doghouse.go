package server

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-github/v29/github"

	"github.com/reviewdog/reviewdog"
	"github.com/reviewdog/reviewdog/diff"
	"github.com/reviewdog/reviewdog/doghouse"
	"github.com/reviewdog/reviewdog/service/github/githubutils"
)

// GitHub check runs API cannot handle too large requests.
// Set max number of filtered findings to be shown in check-run summary.
// ERROR:
//  https://api.github.com/repos/easymotion/vim-easymotion/check-runs: 422
//  Invalid request.
//  Only 65535 characters are allowed; 250684 were supplied. []
const maxFilteredFinding = 150

// > The Checks API limits the number of annotations to a maximum of 50 per API
// > request.
// https://developer.github.com/v3/checks/runs/#output-object
const maxAnnotationsPerRequest = 50

type Checker struct {
	req *doghouse.CheckRequest
	gh  checkerGitHubClientInterface
}

func NewChecker(req *doghouse.CheckRequest, gh *github.Client) *Checker {
	return &Checker{req: req, gh: &checkerGitHubClient{Client: gh}}
}

func (ch *Checker) Check(ctx context.Context) (*doghouse.CheckResponse, error) {
	var filediffs []*diff.FileDiff
	if ch.req.PullRequest != 0 {
		var err error
		filediffs, err = ch.pullRequestDiff(ctx, ch.req.PullRequest)
		if err != nil {
			return nil, fmt.Errorf("fail to parse diff: %v", err)
		}
	}

	results := annotationsToCheckResults(ch.req.Annotations)
	filtered := reviewdog.FilterCheck(results, filediffs, 1, "")

	check, err := ch.createCheck(ctx)
	if err != nil {
		// If this error is StatusForbidden (403) here, it means reviewdog is
		// running on GitHub Actions and has only read permission (because it's
		// running for Pull Requests from forked repository). If the token itself
		// is invalid, reviewdog should return an error earlier (e.g. when reading
		// Pull Requests diff), so it should be ok not to return error here and
		// return results instead.
		if err, ok := err.(*github.ErrorResponse); ok && err.Response.StatusCode == http.StatusForbidden {
			return &doghouse.CheckResponse{CheckedResults: filtered}, nil
		}
		return nil, fmt.Errorf("failed to create check: %v", err)
	}

	checkRun, err := ch.postCheck(ctx, check.GetID(), filtered)
	if err != nil {
		return nil, fmt.Errorf("failed to post result: %v", err)
	}
	res := &doghouse.CheckResponse{
		ReportURL: checkRun.GetHTMLURL(),
	}
	return res, nil
}

func (ch *Checker) postCheck(ctx context.Context, checkID int64, checks []*reviewdog.FilteredCheck) (*github.CheckRun, error) {
	filterByDiff := ch.req.PullRequest != 0 && !ch.req.OutsideDiff
	var annotations []*github.CheckRunAnnotation
	for _, c := range checks {
		if !c.InDiff && filterByDiff {
			continue
		}
		annotations = append(annotations, ch.toCheckRunAnnotation(c))
	}
	if err := ch.postAnnotations(ctx, checkID, annotations); err != nil {
		return nil, fmt.Errorf("failed to post annotations: %v", err)
	}

	conclusion := "success"
	if len(annotations) > 0 {
		conclusion = ch.conclusion()
	}
	opt := github.UpdateCheckRunOptions{
		Name:        ch.checkName(),
		Status:      github.String("completed"),
		Conclusion:  github.String(conclusion),
		CompletedAt: &github.Timestamp{Time: time.Now()},
		Output: &github.CheckRunOutput{
			Title:   github.String(ch.checkTitle()),
			Summary: github.String(ch.summary(checks)),
		},
	}
	return ch.gh.UpdateCheckRun(ctx, ch.req.Owner, ch.req.Repo, checkID, opt)
}

func (ch *Checker) createCheck(ctx context.Context) (*github.CheckRun, error) {
	opt := github.CreateCheckRunOptions{
		Name:    ch.checkName(),
		HeadSHA: ch.req.SHA,
		Status:  github.String("in_progress"),
	}
	return ch.gh.CreateCheckRun(ctx, ch.req.Owner, ch.req.Repo, opt)
}

func (ch *Checker) postAnnotations(ctx context.Context, checkID int64, annotations []*github.CheckRunAnnotation) error {
	opt := github.UpdateCheckRunOptions{
		Name: ch.checkName(),
		Output: &github.CheckRunOutput{
			Title:       github.String(ch.checkTitle()),
			Summary:     github.String(""), // Post summary with the last reqeust.
			Annotations: annotations[:min(maxAnnotationsPerRequest, len(annotations))],
		},
	}
	if _, err := ch.gh.UpdateCheckRun(ctx, ch.req.Owner, ch.req.Repo, checkID, opt); err != nil {
		return err
	}
	if len(annotations) > maxAnnotationsPerRequest {
		return ch.postAnnotations(ctx, checkID, annotations[maxAnnotationsPerRequest:])
	}
	return nil
}

func (ch *Checker) checkName() string {
	if ch.req.Name != "" {
		return ch.req.Name
	}
	return "reviewdog"
}

func (ch *Checker) checkTitle() string {
	if name := ch.checkName(); name != "reviewdog" {
		return fmt.Sprintf("reviewdog [%s] report", name)
	}
	return "reviewdog report"
}

// https://developer.github.com/v3/checks/runs/#parameters-1
func (ch *Checker) conclusion() string {
	switch strings.ToLower(ch.req.Level) {
	case "info", "warning":
		return "neutral"
	}
	return "failure"
}

// https://developer.github.com/v3/checks/runs/#annotations-object
func (ch *Checker) annotationLevel() string {
	switch strings.ToLower(ch.req.Level) {
	case "info":
		return "notice"
	case "warning":
		return "warning"
	case "failure":
		return "failure"
	}
	return "failure"
}

func (ch *Checker) summary(checks []*reviewdog.FilteredCheck) string {
	var lines []string
	lines = append(lines, "reported by [reviewdog](https://github.com/reviewdog/reviewdog) :dog:")

	var findings []*reviewdog.FilteredCheck
	var filteredFindings []*reviewdog.FilteredCheck
	for _, c := range checks {
		if c.InDiff {
			findings = append(findings, c)
		} else {
			filteredFindings = append(filteredFindings, c)
		}
	}
	lines = append(lines, ch.summaryFindings("Findings", findings)...)
	lines = append(lines, ch.summaryFindings("Filtered Findings", filteredFindings)...)

	return strings.Join(lines, "\n")
}

func (ch *Checker) summaryFindings(name string, checks []*reviewdog.FilteredCheck) []string {
	var lines []string
	lines = append(lines, "<details>")
	lines = append(lines, fmt.Sprintf("<summary>%s (%d)</summary>", name, len(checks)))
	lines = append(lines, "")
	for i, c := range checks {
		if i >= maxFilteredFinding {
			lines = append(lines, "... (Too many findings. Dropped some findings)")
			break
		}
		lines = append(lines, githubutils.LinkedMarkdownCheckResult(
			ch.req.Owner, ch.req.Repo, ch.req.SHA, c.CheckResult))
	}
	lines = append(lines, "</details>")
	return lines
}

func (ch *Checker) toCheckRunAnnotation(c *reviewdog.FilteredCheck) *github.CheckRunAnnotation {
	a := &github.CheckRunAnnotation{
		Path:            github.String(c.Path),
		StartLine:       github.Int(c.Lnum),
		EndLine:         github.Int(c.Lnum),
		AnnotationLevel: github.String(ch.annotationLevel()),
		Message:         github.String(c.Message),
	}
	if ch.req.Name != "" {
		a.Title = github.String(fmt.Sprintf("[%s] %s#L%d", ch.req.Name, c.Path, c.Lnum))
	}
	if s := strings.Join(c.Lines, "\n"); s != "" {
		a.RawDetails = github.String(s)
	}
	return a
}

func (ch *Checker) pullRequestDiff(ctx context.Context, pr int) ([]*diff.FileDiff, error) {
	d, err := ch.rawPullRequestDiff(ctx, pr)
	if err != nil {
		return nil, err
	}
	filediffs, err := diff.ParseMultiFile(bytes.NewReader(d))
	if err != nil {
		return nil, fmt.Errorf("fail to parse diff: %v", err)
	}
	return filediffs, nil
}

func (ch *Checker) rawPullRequestDiff(ctx context.Context, pr int) ([]byte, error) {
	d, err := ch.gh.GetPullRequestDiff(ctx, ch.req.Owner, ch.req.Repo, pr)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func annotationsToCheckResults(as []*doghouse.Annotation) []*reviewdog.CheckResult {
	cs := make([]*reviewdog.CheckResult, 0, len(as))
	for _, a := range as {
		cs = append(cs, &reviewdog.CheckResult{
			Path:    a.Path,
			Lnum:    a.Line,
			Message: a.Message,
			Lines:   strings.Split(a.RawMessage, "\n"),
		})
	}
	return cs
}

func min(x, y int) int {
	if x > y {
		return y
	}
	return x
}
