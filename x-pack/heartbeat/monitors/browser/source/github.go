// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package source

// GithubSource handles configs for github repos, using the API defined here:
// https://docs.github.com/en/free-pro-team@latest/rest/reference/repos#download-a-repository-archive-tar.
type GithubSource struct {
	Owner   string `config:"owner"`
	Repo    string `config:"repo"`
	Ref     string `config:"ref"`
	UrlBase string `config:"url_base"`
	PollingSource
}

func (g *GithubSource) Fetch() error {
	panic("implement me")
}

func (g *GithubSource) Workdir() string {
	panic("implement me")
}
