package synthetic_suite

import (
	"fmt"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/otiai10/copy"
	"io/ioutil"
)

type SuiteFetcher interface {
	Fetch() error
	Workdir() string
}

type BaseSuite struct {
	Type	 string					`config:"type"`
	Name     string                 `config:"id_prefix"`
	Schedule string                 `config:"schedule"`
	Params   map[string]interface{} `config:"params"`
	RawConfig *common.Config
}

type PollingSuite struct {
	CheckEvery int `config:"check_every"`
	SyntheticSuite
}

type LocalSuite struct {
	Path     string                 `config:"path"`
	SyntheticSuite
}

func (l LocalSuite) Fetch() error {
	dir, err := ioutil.TempDir("/tmp", "elastic-synthetics-")
	if err != nil {
		return fmt.Errorf("could not create tmp dir: %w", err)
	}

	err = copy.Copy(l.Path, dir)
	if err != nil {
		return fmt.Errorf("could not copy suite: %w", err)
	}
	return nil
}

func (l LocalSuite) Workdir() string {
	panic("implement me")
}

// GithubSuite handles configs for github repos, using the API defined here:
// https://docs.github.com/en/free-pro-team@latest/rest/reference/repos#download-a-repository-archive-tar.
type GithubSuite struct {
	Owner	string `config:"owner"`
	Repo string `config:"repo"`
	Ref string `config:"ref"`
	UrlBase string `config:"string"`
	PollingSuite
}

func (g GithubSuite) Fetch() error {
	panic("implement me")
}

func (g GithubSuite) Workdir() string {
	panic("implement me")
}

type ZipURLSuite struct {
	Url string `config:"url"`
	Headers map[string]string `config:"headers"`
	PollingSuite
}

func (z ZipURLSuite) Fetch() error {
	panic("implement me")
}

func (z ZipURLSuite) Workdir() string {
	panic("implement me")
}

