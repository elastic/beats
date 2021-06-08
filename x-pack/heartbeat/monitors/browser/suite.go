// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browser

import (
	"context"
	"fmt"

	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/heartbeat/monitors/plugin"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/x-pack/heartbeat/monitors/browser/synthexec"
)

type JourneyLister func(ctx context.Context, suitePath string, params common.MapStr) (journeyNames []string, err error)

type Suite struct {
	rawCfg   *common.Config
	suiteCfg *Config
}

func NewSuite(rawCfg *common.Config) (*Suite, error) {
	s := &Suite{
		rawCfg:   rawCfg,
		suiteCfg: DefaultConfig(),
	}
	err := rawCfg.Unpack(s.suiteCfg)
	if err != nil {
		return nil, ErrBadConfig(err)
	}

	return s, nil
}

func ErrBadConfig(err error) error {
	return fmt.Errorf("could not parse suite config: %w", err)
}

func (s *Suite) String() string {
	panic("implement me")
}

func (s *Suite) Fetch() error {
	return s.suiteCfg.Source.Active().Fetch()
}

func (s *Suite) Workdir() string {
	return s.suiteCfg.Source.Active().Workdir()
}

func (s *Suite) InlineSource() (string, bool) {
	if s.suiteCfg.Source.Inline != nil {
		return s.suiteCfg.Source.Inline.Script, true
	}
	return "", false
}

func (s *Suite) Params() map[string]interface{} {
	return s.suiteCfg.Params
}

func (s *Suite) Close() error {
	if s.suiteCfg.Source.ActiveMemo != nil {
		s.suiteCfg.Source.ActiveMemo.Close()
	}

	return nil
}

func (s *Suite) extraArgs() []string {
	extraArgs := s.suiteCfg.SyntheticsArgs
	if s.suiteCfg.Sandbox {
		extraArgs = append(extraArgs, "--sandbox")
	}

	return extraArgs
}

func (s *Suite) jobs() []jobs.Job {
	var j jobs.Job
	if src, ok := s.InlineSource(); ok {
		j = synthexec.InlineJourneyJob(context.TODO(), src, s.Params(), s.extraArgs()...)
	} else {
		j = func(event *beat.Event) ([]jobs.Job, error) {
			err := s.Fetch()
			if err != nil {
				return nil, fmt.Errorf("could not fetch for suite job: %w", err)
			}
			sj, err := synthexec.SuiteJob(context.TODO(), s.Workdir(), s.Params(), s.extraArgs()...)
			if err != nil {
				return nil, err
			}
			return sj(event)
		}
	}
	return []jobs.Job{j}
}

func (s *Suite) plugin() plugin.Plugin {
	return plugin.Plugin{
		Jobs:      s.jobs(),
		Close:     s.Close,
		Endpoints: 1,
	}
}
