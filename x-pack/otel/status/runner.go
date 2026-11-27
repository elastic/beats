// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package status

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/pkg/config"

	"github.com/gohugoio/hashstructure"
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
		inputId, err := getInputId(config)
		if err != nil {
			return nil, err
		}
		// Runners whose config no caller passed to SetRunnerConfigs, such as
		// the inputs of a Filebeat module, report no status.
		if statusReporter := s.reporter.GetReporterForRunner(inputId); statusReporter != nil {
			runnerWithStatus.SetStatusReporter(statusReporter)
		}
	}
	return runner, nil
}

// SetRunnerConfigs sets the runners that report a status for source to the
// runners of configs.
func (s *statusFactory) SetRunnerConfigs(source any, configs []*config.C) {
	ids := make([]string, 0, len(configs))
	for _, c := range configs {
		id, err := getInputId(c)
		if err != nil {
			// Create fails for this config as well, so it never runs
			continue
		}
		ids = append(ids, id)
	}
	s.reporter.SetRunners(source, ids)
}

// Unwrap returns the wrapped factory, so callers can use its optional
// interfaces, such as Heartbeat's config hash function.
func (s *statusFactory) Unwrap() cfgfile.RunnerFactory {
	return s.factory
}

func (s *statusFactory) CheckConfig(config *config.C) error {
	return s.factory.CheckConfig(config)
}

func StatusReporterFactory(reporter RunnerReporter) cfgfile.FactoryWrapper {
	return func(f cfgfile.RunnerFactory) cfgfile.RunnerFactory {
		return &statusFactory{factory: f, reporter: reporter}
	}
}

func getInputId(config *config.C) (string, error) {
	var h map[string]any
	err := config.Unpack(&h)
	if err != nil {
		return "", fmt.Errorf("could not unpack config: %w", err)
	}
	if inputId, ok := h["id"]; ok {
		return fmt.Sprintf("%v", inputId), nil
	}

	inputId, err := hashstructure.Hash(h, nil)
	if err != nil {
		return "", fmt.Errorf("can not compute id from configuration: %w", err)
	}

	return fmt.Sprintf("%x", inputId), nil
}
