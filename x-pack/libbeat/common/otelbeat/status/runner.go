// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package status

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/elastic-agent-libs/config"

	"github.com/mitchellh/hashstructure"
)

type statusFactory struct {
	factory  cfgfile.RunnerFactory
	reporter RunnerReporter
}

func (s *statusFactory) Create(p beat.PipelineConnector, config *config.C) (cfgfile.Runner, error) {
	runner, err := s.factory.Create(p, config)
	if err != nil {
		return nil, err
	}
	if runnerWithStatus, ok := runner.(status.WithStatusReporter); ok {
		var h map[string]interface{}
		err := config.Unpack(&h)
		if err != nil {
			return nil, fmt.Errorf("could not unpack config: %w", err)
		}
		id, err := hashstructure.Hash(h, nil)
		if err != nil {
			return nil, fmt.Errorf("can not compute id from configuration: %w", err)
		}
		reporter := s.reporter.GetReporterForRunner(id)
		runnerWithStatus.SetStatusReporter(reporter)
	}
	return runner, nil
}

func (s *statusFactory) CheckConfig(config *config.C) error {
	return s.factory.CheckConfig(config)
}

func StatusReporterFactory(reporter RunnerReporter) cfgfile.FactoryWrapper {
	return func(f cfgfile.RunnerFactory) cfgfile.RunnerFactory {
		return &statusFactory{factory: f, reporter: reporter}
	}
}
