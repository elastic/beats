package status

import (
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/elastic-agent-libs/config"
)

type statusFactory struct {
	f              cfgfile.RunnerFactory
	statusReporter status.StatusReporter
}

func (s *statusFactory) Create(p beat.PipelineConnector, config *config.C) (cfgfile.Runner, error) {
	runner, err := s.f.Create(p, config)
	if err != nil {
		return nil, err
	}
	if runnerWithStatus, ok := runner.(status.WithStatusReporter); ok {
		runnerWithStatus.SetStatusReporter(s.statusReporter)
	}
	return runner, nil
}

func (s *statusFactory) CheckConfig(config *config.C) error {
	return s.f.CheckConfig(config)
}

func StatusReporterFactory() cfgfile.FactoryWrapper {
	return func(f cfgfile.RunnerFactory) cfgfile.RunnerFactory {
		return &statusFactory{f: f}
	}
}
