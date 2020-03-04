package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"sort"
	"strings"
	"text/tabwriter"

	"golang.org/x/net/context" // "context"

	"golang.org/x/oauth2"

	"github.com/google/go-github/v29/github"
	"github.com/mattn/go-shellwords"
	"github.com/reviewdog/errorformat/fmts"
	"github.com/xanzy/go-gitlab"

	"github.com/reviewdog/reviewdog"
	"github.com/reviewdog/reviewdog/cienv"
	"github.com/reviewdog/reviewdog/commands"
	"github.com/reviewdog/reviewdog/project"
	githubservice "github.com/reviewdog/reviewdog/service/github"
	"github.com/reviewdog/reviewdog/service/github/githubutils"
	gitlabservice "github.com/reviewdog/reviewdog/service/gitlab"
)

const usageMessage = "" +
	`Usage:	reviewdog [flags]
	reviewdog accepts any compiler or linter results from stdin and filters
	them by diff for review. reviewdog also can posts the results as a comment to
	GitHub if you use reviewdog in CI service.`

type option struct {
	version          bool
	diffCmd          string
	diffStrip        int
	efms             strslice
	f                string // errorformat name
	list             bool   // list supported errorformat name
	name             string // tool name which is used in comment
	ci               string
	conf             string
	runners          string
	reporter         string
	level            string
	guessPullRequest bool
	tee              bool
}

// flags doc
const (
	diffCmdDoc          = `diff command (e.g. "git diff"). diff flag is ignored if you pass "ci" flag`
	diffStripDoc        = "strip NUM leading components from diff file names (equivalent to 'patch -p') (default is 1 for git diff)"
	efmsDoc             = `list of errorformat (https://github.com/reviewdog/errorformat)`
	fDoc                = `format name (run -list to see supported format name) for input. It's also used as tool name in review comment if -name is empty`
	listDoc             = `list supported pre-defined format names which can be used as -f arg`
	nameDoc             = `tool name in review comment. -f is used as tool name if -name is empty`
	ciDoc               = `[deprecated] reviewdog automatically get necessary data. See also -reporter for migration`
	confDoc             = `config file path`
	runnersDoc          = `comma separated runners name to run in config file. default: run all runners`
	levelDoc            = `report level currently used for github-pr-check reporter ("info","warning","error").`
	guessPullRequestDoc = `guess Pull Request ID by branch name and commit SHA`
	teeDoc              = `enable "tee"-like mode which outputs tools's output as is while reporting results to -reporter. Useful for debugging as well.`
	reporterDoc         = `reporter of reviewdog results. (local, github-check, github-pr-check, github-pr-review, gitlab-mr-discussion, gitlab-mr-commit)
	"local" (default)
		Report results to stdout.

	"github-check" (experimental)
		Report results to GitHub Check. It works both for Pull Requests and commits
		and this reporter reports all results regardless of it's new result or not.
		You can see report results in GitHub PullRequest Check tab for Pull Request.

		There are two options to use this reporter.

		Option 1) Run reviewdog from GitHub Actions w/ secrets.GITHUB_TOKEN
			Note that it reports result to GitHub Actions log consle for Pull
			Requests from fork repository due to GitHub Actions restriction.
			https://help.github.com/en/articles/virtual-environments-for-github-actions#github_token-secret

			Set REVIEWDOG_GITHUB_API_TOKEN with secrets.GITHUB_TOKEN. e.g.
					REVIEWDOG_GITHUB_API_TOKEN: ${{ secrets.GITHUB_TOKEN }}

		Option 2) Install reviewdog GitHub Apps
			1. Install reviewdog Apps. https://github.com/apps/reviewdog
			2. Set REVIEWDOG_TOKEN or run reviewdog CLI in trusted CI providers.
			You can get token from https://reviewdog.app/gh/<owner>/<repo-name>.
			$ export REVIEWDOG_TOKEN="xxxxx"

			Note: Token is not required if you run reviewdog in Travis CI.

	"github-pr-check" (experimental)
		Same as github-check reporter but it only supports Pull Requests and
		reports only new results which is in diff.

	"github-pr-review"
		Report results to GitHub review comments.

		1. Set REVIEWDOG_GITHUB_API_TOKEN environment variable.
		Go to https://github.com/settings/tokens and create new Personal access token with repo scope.

		For GitHub Enterprise:
			$ export GITHUB_API="https://example.githubenterprise.com/api/v3"

	"gitlab-mr-discussion"
		Report results to GitLab MergeRequest discussion.

		1. Set REVIEWDOG_GITLAB_API_TOKEN environment variable.
		Go to https://gitlab.com/profile/personal_access_tokens

		For self hosted GitLab:
			$ export GITLAB_API="https://example.gitlab.com/api/v4"

	"gitlab-mr-commit"
		Same as gitlab-mr-discussion, but report results to GitLab comments for
		each commits in Merge Requests.

	For GitHub Enterprise and self hosted GitLab, set
	REVIEWDOG_INSECURE_SKIP_VERIFY to skip verifying SSL (please use this at your own risk)
		$ export REVIEWDOG_INSECURE_SKIP_VERIFY=true

	For non-local reporters, reviewdog automatically get necessary data from
	environment variable in CI service (GitHub Actions, Travis CI, Circle CI, drone.io, GitLab CI).
	You can set necessary data with following environment variable manually if
	you want (e.g. run reviewdog in Jenkins).

		$ export CI_PULL_REQUEST=14 # Pull Request number (e.g. 14)
		$ export CI_COMMIT="$(git rev-parse @)" # SHA1 for the current build
		$ export CI_REPO_OWNER="haya14busa" # repository owner
		$ export CI_REPO_NAME="reviewdog" # repository name
`
)

var opt = &option{}

func init() {
	flag.BoolVar(&opt.version, "version", false, "print version")
	flag.StringVar(&opt.diffCmd, "diff", "", diffCmdDoc)
	flag.IntVar(&opt.diffStrip, "strip", 1, diffStripDoc)
	flag.Var(&opt.efms, "efm", efmsDoc)
	flag.StringVar(&opt.f, "f", "", fDoc)
	flag.BoolVar(&opt.list, "list", false, listDoc)
	flag.StringVar(&opt.name, "name", "", nameDoc)
	flag.StringVar(&opt.ci, "ci", "", ciDoc)
	flag.StringVar(&opt.conf, "conf", "", confDoc)
	flag.StringVar(&opt.runners, "runners", "", runnersDoc)
	flag.StringVar(&opt.reporter, "reporter", "local", reporterDoc)
	flag.StringVar(&opt.level, "level", "error", levelDoc)
	flag.BoolVar(&opt.guessPullRequest, "guess", false, guessPullRequestDoc)
	flag.BoolVar(&opt.tee, "tee", false, teeDoc)
}

func usage() {
	fmt.Fprintln(os.Stderr, usageMessage)
	fmt.Fprintln(os.Stderr, "Flags:")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "See https://github.com/reviewdog/reviewdog for more detail.")
	os.Exit(2)
}

func main() {
	flag.Usage = usage
	flag.Parse()
	if err := run(os.Stdin, os.Stdout, opt); err != nil {
		fmt.Fprintf(os.Stderr, "reviewdog: %v\n", err)
		os.Exit(1)
	}
}

func run(r io.Reader, w io.Writer, opt *option) error {
	ctx := context.Background()

	if opt.version {
		fmt.Fprintln(w, commands.Version)
		return nil
	}

	if opt.list {
		return runList(w)
	}

	// TODO(haya14busa): clean up when removing -ci flag from next release.
	if opt.ci != "" {
		return errors.New(`-ci flag is deprecated.
See -reporter flag for migration and set -reporter="github-pr-review" or -reporter="github-pr-check" or -reporter="gitlab-mr-commit"`)
	}

	if opt.tee {
		r = io.TeeReader(r, w)
	}

	// assume it's project based run when both -efm and -f are not specified
	isProject := len(opt.efms) == 0 && opt.f == ""

	var cs reviewdog.CommentService
	var ds reviewdog.DiffService

	if isProject {
		cs = reviewdog.NewUnifiedCommentWriter(w)
	} else {
		cs = reviewdog.NewRawCommentWriter(w)
	}

	switch opt.reporter {
	default:
		return fmt.Errorf("unknown -reporter: %s", opt.reporter)
	case "github-check":
		return runDoghouse(ctx, r, w, opt, isProject, false)
	case "github-pr-check":
		return runDoghouse(ctx, r, w, opt, isProject, true)
	case "github-pr-review":
		if os.Getenv("REVIEWDOG_GITHUB_API_TOKEN") == "" {
			fmt.Fprintln(os.Stderr, "REVIEWDOG_GITHUB_API_TOKEN is not set")
			return nil
		}
		gs, isPR, err := githubService(ctx, opt)
		if err != nil {
			return err
		}
		if !isPR {
			fmt.Fprintln(os.Stderr, "reviewdog: this is not PullRequest build.")
			return nil
		}
		// If it's running in GitHub Actions and it's PR from forked repository,
		// replace comment writer to GitHubActionLogWriter to create annotations
		// instead of review comment because if it's PR from forked repository,
		// GitHub token doen't have write permission due to security concern and
		// cannot post results via Review API.
		if cienv.IsInGitHubAction() && isPRFromForkedRepo() {
			fmt.Fprintln(w, `reviewdog: This is Pull-Request from forked repository.
GitHub token doesn't have write permission of Review API, so reviewdog will
report results via logging command [1] and create annotations similar to
github-pr-check reporter as a fallback.

[1]: https://help.github.com/en/actions/automating-your-workflow-with-github-actions/development-tools-for-github-actions#logging-commands`)
			cs = githubutils.NewGitHubActionLogWriter(opt.level)
		} else {
			cs = reviewdog.MultiCommentService(gs, cs)
		}
		ds = gs
	case "gitlab-mr-discussion":
		build, cli, err := gitlabBuildWithClient()
		if err != nil {
			return err
		}
		if build.PullRequest == 0 {
			fmt.Fprintln(os.Stderr, "this is not MergeRequest build.")
			return nil
		}

		gc, err := gitlabservice.NewGitLabMergeRequestDiscussionCommenter(cli, build.Owner, build.Repo, build.PullRequest, build.SHA)
		if err != nil {
			return err
		}

		cs = reviewdog.MultiCommentService(gc, cs)
		ds, err = gitlabservice.NewGitLabMergeRequestDiff(cli, build.Owner, build.Repo, build.PullRequest, build.SHA)
		if err != nil {
			return err
		}
	case "gitlab-mr-commit":
		build, cli, err := gitlabBuildWithClient()
		if err != nil {
			return err
		}
		if build.PullRequest == 0 {
			fmt.Fprintln(os.Stderr, "this is not MergeRequest build.")
			return nil
		}

		gc, err := gitlabservice.NewGitLabMergeRequestCommitCommenter(cli, build.Owner, build.Repo, build.PullRequest, build.SHA)
		if err != nil {
			return err
		}

		cs = reviewdog.MultiCommentService(gc, cs)
		ds, err = gitlabservice.NewGitLabMergeRequestDiff(cli, build.Owner, build.Repo, build.PullRequest, build.SHA)
		if err != nil {
			return err
		}
	case "local":
		d, err := diffService(opt.diffCmd, opt.diffStrip)
		if err != nil {
			return err
		}
		ds = d
	}

	if isProject {
		conf, err := projectConfig(opt.conf)
		if err != nil {
			return err
		}
		return project.Run(ctx, conf, buildRunnersMap(opt.runners), cs, ds, opt.tee)
	}

	p, err := newParserFromOpt(opt)
	if err != nil {
		return err
	}

	app := reviewdog.NewReviewdog(toolName(opt), p, cs, ds)
	return app.Run(ctx, r)
}

func runList(w io.Writer) error {
	tabw := tabwriter.NewWriter(w, 0, 8, 0, '\t', 0)
	for _, f := range sortedFmts(fmts.DefinedFmts()) {
		fmt.Fprintf(tabw, "%s\t%s\t- %s\n", f.Name, f.Description, f.URL)
	}
	fmt.Fprintf(tabw, "%s\t%s\t- %s\n", "checkstyle", "checkstyle XML format", "http://checkstyle.sourceforge.net/")
	return tabw.Flush()
}

type byFmtName []*fmts.Fmt

func (p byFmtName) Len() int           { return len(p) }
func (p byFmtName) Less(i, j int) bool { return p[i].Name < p[j].Name }
func (p byFmtName) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func sortedFmts(fs fmts.Fmts) []*fmts.Fmt {
	r := make([]*fmts.Fmt, 0, len(fs))
	for _, f := range fs {
		r = append(r, f)
	}
	sort.Sort(byFmtName(r))
	return r
}

func diffService(s string, strip int) (reviewdog.DiffService, error) {
	cmds, err := shellwords.Parse(s)
	if err != nil {
		return nil, err
	}
	if len(cmds) < 1 {
		return nil, errors.New("diff command is empty")
	}
	cmd := exec.Command(cmds[0], cmds[1:]...)
	d := reviewdog.NewDiffCmd(cmd, strip)
	return d, nil
}

func newHTTPClient() *http.Client {
	tr := &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureSkipVerify()},
	}
	return &http.Client{Transport: tr}
}

func insecureSkipVerify() bool {
	return os.Getenv("REVIEWDOG_INSECURE_SKIP_VERIFY") == "true"
}

func githubService(ctx context.Context, opt *option) (gs *githubservice.GitHubPullRequest, isPR bool, err error) {
	token, err := nonEmptyEnv("REVIEWDOG_GITHUB_API_TOKEN")
	if err != nil {
		return nil, isPR, err
	}
	g, isPR, err := cienv.GetBuildInfo()
	if err != nil {
		return nil, isPR, err
	}

	client, err := githubClient(ctx, token)
	if err != nil {
		return nil, isPR, err
	}

	if !isPR {
		if !opt.guessPullRequest {
			return nil, false, nil
		}

		if g.Branch == "" && g.SHA == "" {
			return nil, false, nil
		}

		prID, err := getPullRequestIDByBranchOrCommit(ctx, client, g)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return nil, false, nil
		}
		g.PullRequest = prID
	}

	gs, err = githubservice.NewGitHubPullRequest(client, g.Owner, g.Repo, g.PullRequest, g.SHA)
	if err != nil {
		return nil, false, err
	}
	return gs, true, nil
}

func getPullRequestIDByBranchOrCommit(ctx context.Context, client *github.Client, info *cienv.BuildInfo) (int, error) {
	options := &github.SearchOptions{
		Sort:  "updated",
		Order: "desc",
	}

	query := []string{
		"type:pr",
		"state:open",
		fmt.Sprintf("repo:%s/%s", info.Owner, info.Repo),
	}
	if info.Branch != "" {
		query = append(query, fmt.Sprintf("head:%s", info.Branch))
	}
	if info.SHA != "" {
		query = append(query, info.SHA)
	}

	preparedQuery := strings.Join(query, " ")
	pullRequests, _, err := client.Search.Issues(ctx, preparedQuery, options)
	if err != nil {
		return 0, err
	}

	if *pullRequests.Total == 0 {
		return 0, fmt.Errorf("reviewdog: PullRequest not found, query: %s", preparedQuery)
	}

	return *pullRequests.Issues[0].Number, nil
}

func githubClient(ctx context.Context, token string) (*github.Client, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, newHTTPClient())
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	var err error
	client.BaseURL, err = githubBaseURL()
	return client, err
}

const defaultGitHubAPI = "https://api.github.com/"

func githubBaseURL() (*url.URL, error) {
	baseURL := os.Getenv("GITHUB_API")
	if baseURL == "" {
		baseURL = defaultGitHubAPI
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("GitHub base URL is invalid: %v, %v", baseURL, err)
	}
	return u, nil
}

func gitlabBuildWithClient() (*cienv.BuildInfo, *gitlab.Client, error) {
	token, err := nonEmptyEnv("REVIEWDOG_GITLAB_API_TOKEN")
	if err != nil {
		return nil, nil, err
	}

	g, _, err := cienv.GetBuildInfo()
	if err != nil {
		return nil, nil, err
	}

	client, err := gitlabClient(token)
	if err != nil {
		return nil, nil, err
	}

	if g.PullRequest == 0 {
		prNr, err := fetchMergeRequestIDFromCommit(client, g.Owner+"/"+g.Repo, g.SHA)
		if err != nil {
			return nil, nil, err
		}
		if prNr != 0 {
			g.PullRequest = prNr
		}
	}

	return g, client, err
}

func fetchMergeRequestIDFromCommit(cli *gitlab.Client, projectID, sha string) (id int, err error) {
	// https://docs.gitlab.com/ce/api/merge_requests.html#list-project-merge-requests
	opt := &gitlab.ListProjectMergeRequestsOptions{
		State:   gitlab.String("opened"),
		OrderBy: gitlab.String("updated_at"),
	}
	mrs, _, err := cli.MergeRequests.ListProjectMergeRequests(projectID, opt)
	if err != nil {
		return 0, err
	}
	for _, mr := range mrs {
		if mr.SHA == sha {
			return mr.IID, nil
		}
	}
	return 0, nil
}

func gitlabClient(token string) (*gitlab.Client, error) {
	client := gitlab.NewClient(newHTTPClient(), token)
	baseURL, err := gitlabBaseURL()
	if err != nil {
		return nil, err
	}
	if err := client.SetBaseURL(baseURL.String()); err != nil {
		return nil, err
	}
	return client, nil
}

const defaultGitLabAPI = "https://gitlab.com/api/v4"

func gitlabBaseURL() (*url.URL, error) {
	baseURL := os.Getenv("GITLAB_API")
	if baseURL == "" {
		baseURL = defaultGitLabAPI
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("GitLab base URL is invalid: %v, %v", baseURL, err)
	}
	return u, nil
}

func nonEmptyEnv(env string) (string, error) {
	v := os.Getenv(env)
	if v == "" {
		return "", fmt.Errorf("environment variable $%v is not set", env)
	}
	return v, nil
}

type strslice []string

func (ss *strslice) String() string {
	return fmt.Sprintf("%v", *ss)
}

func (ss *strslice) Set(value string) error {
	*ss = append(*ss, value)
	return nil
}

func projectConfig(path string) (*project.Config, error) {
	b, err := readConf(path)
	if err != nil {
		return nil, fmt.Errorf("fail to open config: %v", err)
	}
	conf, err := project.Parse(b)
	if err != nil {
		return nil, fmt.Errorf("config is invalid: %v", err)
	}
	return conf, nil
}

func readConf(conf string) ([]byte, error) {
	var conffiles []string
	if conf != "" {
		conffiles = []string{conf}
	} else {
		conffiles = []string{
			".reviewdog.yaml",
			".reviewdog.yml",
			"reviewdog.yaml",
			"reviewdog.yml",
		}
	}
	for _, f := range conffiles {
		bytes, err := ioutil.ReadFile(f)
		if err == nil {
			return bytes, nil
		}
	}
	return nil, errors.New(".reviewdog.yml not found")
}

func newParserFromOpt(opt *option) (reviewdog.Parser, error) {
	p, err := reviewdog.NewParser(&reviewdog.ParserOpt{FormatName: opt.f, Errorformat: opt.efms})
	if err != nil {
		return nil, fmt.Errorf("fail to create parser. use either -f or -efm: %v", err)
	}
	return p, err
}

func toolName(opt *option) string {
	name := opt.name
	if name == "" && opt.f != "" {
		name = opt.f
	}
	return name
}

func buildRunnersMap(runners string) map[string]bool {
	m := make(map[string]bool)
	for _, r := range strings.Split(runners, ",") {
		if name := strings.TrimSpace(r); name != "" {
			m[name] = true
		}
	}
	return m
}
