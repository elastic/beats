package suitefactory

import (
	"context"
	"fmt"
	"github.com/elastic/beats/v7/heartbeat/beater"
	"github.com/elastic/beats/v7/heartbeat/monitors"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	b "github.com/elastic/beats/v7/metricbeat/module/beat"
	"github.com/elastic/beats/v7/x-pack/heartbeat/monitors/synthexec"
	"github.com/pkg/errors"
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
	suite := &SyntheticSuite{}
	err := config.Unpack(suite)
	if err != nil {
		return nil, fmt.Errorf("could not parse suite config: %w", err)
	}

	var suitePath string
	var suiteParams map[string]interface{}
	switch suite.Type {
	case "zipurl":
		unpacked := &ZipUrlSyntheticSuite{}
		err := config.Unpack(unpacked)
		if err != nil {
			return nil, fmt.Errorf("could not parse zip URL synthetic suite: %w", err)
		}
	case "github":
		unpacked := &GithubSyntheticSuite{}
		err := config.Unpack(unpacked)
		if err != nil {
			return nil, fmt.Errorf("could not parse github synthetic suite: %w", err)
		}
	case "local":
		unpacked := &LocalSyntheticSuite{}
		err := config.Unpack(unpacked)
		if err != nil {
			return nil, fmt.Errorf("could not parse local synthetic suite: %w", err)
		}
		suitePath = unpacked.Path
		suiteParams = unpacked.Params
	default:
		return nil, fmt.Errorf("suite type not specified! Expected 'local', 'github', or 'zipurl'")
	}

	logp.Info("Listing suite %s", suitePath)
	journeyNames, err := synthexec.ListJourneys(context.TODO(), suitePath, suiteParams)
	if err != nil {
		return nil, err
    }
	logp.Warn("POSTLIST")
	factory := monitors.NewFactory(b.Info, bt.scheduler, false)
	for _, name := range journeyNames {
		cfg, err := common.NewConfigFrom(map[string]interface{}{
			"type":         "browser",
			"path":         suiteReloader.WorkingPath(),
			"schedule":     suite.Schedule,
			"params":       suite.Params,
			"journey_name": name,
			"name":         name,
			"id":           name,
		})
		if err != nil {
				  return err
				  }
		created, err := factory.Create(b.Publisher, cfg)
		if err != nil {
				  return errors.Wrap(err, "could not create monitor")
				  }
		created.Start()
	}
}

func (s *StdSuiteFactory) CheckConfig(config *common.Config) error {
	return nil
}



