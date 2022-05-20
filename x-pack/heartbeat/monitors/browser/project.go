// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browser

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/heartbeat/monitors/plugin"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/x-pack/heartbeat/monitors/browser/synthexec"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type JourneyLister func(ctx context.Context, projectPath string, params mapstr.M) (journeyNames []string, err error)

type Project struct {
	rawCfg     *config.C
	projectCfg *Config
}

func NewProject(rawCfg *config.C) (*Project, error) {
	s := &Project{
		rawCfg:     rawCfg,
		projectCfg: DefaultConfig(),
	}
	err := rawCfg.Unpack(s.projectCfg)
	if err != nil {
		return nil, ErrBadConfig(err)
	}

	return s, nil
}

func ErrBadConfig(err error) error {
	return fmt.Errorf("could not parse project config: %w", err)
}

func (s *Project) String() string {
	panic("implement me")
}

func (s *Project) Fetch() error {
	return s.projectCfg.Source.Active().Fetch()
}

func (s *Project) Workdir() string {
	return s.projectCfg.Source.Active().Workdir()
}

func (s *Project) InlineSource() (string, bool) {
	if s.projectCfg.Source.Inline != nil {
		return s.projectCfg.Source.Inline.Script, true
	}
	return "", false
}

func (s *Project) Params() map[string]interface{} {
	return s.projectCfg.Params
}

func (s *Project) FilterJourneys() synthexec.FilterJourneyConfig {
	return s.projectCfg.FilterJourneys
}

func (s *Project) Fields() synthexec.StdProjectFields {
	_, isInline := s.InlineSource()
	return synthexec.StdProjectFields{
		Name:     s.projectCfg.Name,
		Id:       s.projectCfg.Id,
		IsInline: isInline,
		Type:     "browser",
	}
}

func (s *Project) Close() error {
	if s.projectCfg.Source.ActiveMemo != nil {
		s.projectCfg.Source.ActiveMemo.Close()
	}

	return nil
}

func (s *Project) extraArgs() []string {
	extraArgs := s.projectCfg.SyntheticsArgs
	if s.projectCfg.IgnoreHTTPSErrors {
		extraArgs = append(extraArgs, "--ignore-https-errors")
	}
	if s.projectCfg.Sandbox {
		extraArgs = append(extraArgs, "--sandbox")
	}
	if s.projectCfg.Screenshots != "" {
		extraArgs = append(extraArgs, "--screenshots", s.projectCfg.Screenshots)
	}
	if s.projectCfg.Throttling != nil {
		switch t := s.projectCfg.Throttling.(type) {
		case bool:
			if !t {
				extraArgs = append(extraArgs, "--no-throttling")
			}
		case string:
			extraArgs = append(extraArgs, "--throttling", fmt.Sprintf("%v", s.projectCfg.Throttling))
		case map[string]interface{}:
			j, err := json.Marshal(t)
			if err != nil {
				logp.L().Warnf("could not serialize throttling config to JSON: %s", err)
			} else {
				extraArgs = append(extraArgs, "--throttling", string(j))
			}
		}
	}

	return extraArgs
}

func (s *Project) jobs() []jobs.Job {
	var j jobs.Job
	if src, ok := s.InlineSource(); ok {
		j = synthexec.InlineJourneyJob(context.TODO(), src, s.Params(), s.Fields(), s.extraArgs()...)
	} else {
		j = func(event *beat.Event) ([]jobs.Job, error) {
			err := s.Fetch()
			if err != nil {
				return nil, fmt.Errorf("could not fetch for project job: %w", err)
			}
			sj, err := synthexec.ProjectJob(context.TODO(), s.Workdir(), s.Params(), s.FilterJourneys(), s.Fields(), s.extraArgs()...)
			if err != nil {
				return nil, err
			}
			return sj(event)
		}
	}
	return []jobs.Job{j}
}

func (s *Project) plugin() plugin.Plugin {
	return plugin.Plugin{
		Jobs:      s.jobs(),
		DoClose:   s.Close,
		Endpoints: 1,
	}
}
