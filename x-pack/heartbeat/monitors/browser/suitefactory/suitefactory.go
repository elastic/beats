package suitefactory

import (
	"github.com/elastic/beats/v7/heartbeat/beater"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/common"
)

func init() {
	sf := NewStdSuiteFactory()
	beater.RegisterSuiteFactory(sf)
}

func NewStdSuiteFactory() *StdSuiteFactory {
	return &StdSuiteFactory{}
}

type StdSuiteFactory struct {
	//
}

func (s *StdSuiteFactory) Create(p beat.PipelineConnector, config *common.Config) (cfgfile.Runner, error) {
	panic("implement me")
}

func (s *StdSuiteFactory) CheckConfig(config *common.Config) error {
	panic("implement me")
}




