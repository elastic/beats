package synthetic_suite

import (
	"context"
	"fmt"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)


type JourneyLister func(ctx context.Context, suitePath string, params common.MapStr) (journeyNames []string, err error)

var journeyListSingleton JourneyLister

func RegisterJourneyLister(jl JourneyLister) {
	journeyListSingleton = jl
}

type SyntheticSuite struct {
	rawCfg          *common.Config
	suiteCfg        *BaseSuite
	fetcher         SuiteFetcher
}

func NewSuite(rawCfg *common.Config) (ss *SyntheticSuite, err error) {
	if journeyListSingleton == nil {
		return nil, fmt.Errorf("synthetic monitoring is only supported with x-pack heartbeat")
	}

	ss = &SyntheticSuite{
		rawCfg: rawCfg,
	}

	err = rawCfg.Unpack(ss.suiteCfg)
	if err != nil {
		logp.Err("could not parse suite config: %s", err)
	}

	switch ss.suiteCfg.Type {
	case "local":
		ss.fetcher = LocalSuite{}
	case "github":
		ss.fetcher = GithubSuite{}
	case "zip_url":
		ss.fetcher = ZipURLSuite{}
	}

	err = ss.rawCfg.Unpack(&ss.fetcher)
	if err != nil {
		return nil, fmt.Errorf("could not parse local synthetic suite: %s", err)
	}

	return
}

func (s *SyntheticSuite) String() string {
	panic("implement me")
}

func (s *SyntheticSuite) Start() {
	panic("implement me")
}

func (s *SyntheticSuite) Stop() {
	panic("implement me")
}
