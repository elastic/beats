package synthetic_suite

import (
	"fmt"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/otiai10/copy"
	"io/ioutil"
)

type Config struct {
	Schedule string                 `config:"schedule"`
	Params   map[string]interface{} `config:"params"`
	RawConfig *common.Config
	Source *Source `config:"source"`
}

type Source struct {
	Local      *LocalSource  `config:"local"`
	Github     *GithubSource `config:"github"`
	ZipURL     *ZipURLSource `config:"zip_url"`
	ActiveMemo ISource       // cache for selected source
}

func (s *Source) active() ISource {
	logp.Warn("IN ACTIVE!!!")
	if s.ActiveMemo != nil {
		return s.ActiveMemo
	}

	if s.Local != nil {
		s.ActiveMemo = s.Local
	} else if s.Github != nil {
		s.ActiveMemo = s.Github
	} else if s.ZipURL != nil {
		s.ActiveMemo = s.ZipURL
	}

	return s.ActiveMemo
}

func (s *Source) Validate() error {
	if  s.active() == nil {
		return fmt.Errorf("no valid source specified! Choose one of local, github, zip_url")
	}
	return nil
}

type ISource interface {
	Fetch() error
	Workdir() string
}

type BaseSource struct {
	Type	 string					`config:"type"`
}

type PollingSource struct {
	CheckEvery int `config:"check_every"`
	BaseSource
}

type LocalSource struct {
	Path     string                 `config:"path"`
	BaseSource
}

func (l *LocalSource) Fetch() error {
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

func (l *LocalSource) Workdir() string {
	panic("implement me")
}

// GithubSource handles configs for github repos, using the API defined here:
// https://docs.github.com/en/free-pro-team@latest/rest/reference/repos#download-a-repository-archive-tar.
type GithubSource struct {
	Owner	string `config:"owner"`
	Repo string `config:"repo"`
	Ref string `config:"ref"`
	UrlBase string `config:"string"`
	PollingSource
}

func (g *GithubSource) Fetch() error {
	panic("implement me")
}

func (g *GithubSource) Workdir() string {
	panic("implement me")
}

type ZipURLSource struct {
	Url string `config:"url"`
	Headers map[string]string `config:"headers"`
	PollingSource
}

func (z *ZipURLSource) Fetch() error {
	panic("implement me")
}

func (z *ZipURLSource) Workdir() string {
	panic("implement me")
}

