package source

// GithubSource handles configs for github repos, using the API defined here:
// https://docs.github.com/en/free-pro-team@latest/rest/reference/repos#download-a-repository-archive-tar.
type GithubSource struct {
	Owner	string `config:"owner"`
	Repo string `config:"repo"`
	Ref string `config:"ref"`
	UrlBase string `config:"url_base"`
	PollingSource
}

func (g *GithubSource) Fetch() error {
	panic("implement me")
}

func (g *GithubSource) Workdir() string {
	panic("implement me")
}
