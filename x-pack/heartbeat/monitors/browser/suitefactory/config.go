package suitefactory

import "github.com/elastic/beats/v7/libbeat/common"

type SyntheticSuite struct {
	Type	 string					`config:"type"`
	Name     string                 `config:"id_prefix"`
	Schedule string                 `config:"schedule"`
	Params   map[string]interface{} `config:"params"`
	RawConfig *common.Config
}

type LocalSyntheticSuite struct {
	Path     string                 `config:"path"`
	SyntheticSuite
}

type PollingSyntheticSuite struct {
	CheckEvery int `config:"check_every"`
	SyntheticSuite
}

// GithubSyntheticSuite handles configs for github repos, using the API defined here:
// https://docs.github.com/en/free-pro-team@latest/rest/reference/repos#download-a-repository-archive-tar.
type GithubSyntheticSuite struct {
	Owner	string `config:"owner"`
	Repo string `config:"repo"`
	Ref string `config:"ref"`
	UrlBase string `config:"string"`
	PollingSyntheticSuite
}

type ZipUrlSyntheticSuite struct {
	Url string `config:"url""`
	Headers map[string]string `config:"headers"`
	PollingSyntheticSuite
}
