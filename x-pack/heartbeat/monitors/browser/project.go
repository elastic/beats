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
	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
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

func (p *Project) String() string {
	panic("implement me")
}

func (p *Project) Fetch() error {
	return p.projectCfg.Source.Active().Fetch()
}

func (p *Project) Workdir() string {
	return p.projectCfg.Source.Active().Workdir()
}

func (p *Project) Params() map[string]interface{} {
	return p.projectCfg.Params
}

func (p *Project) FilterJourneys() synthexec.FilterJourneyConfig {
	return p.projectCfg.FilterJourneys
}

func (p *Project) Fields() stdfields.StdMonitorFields {
	sFields := stdfields.StdMonitorFields{
		Name: p.projectCfg.Name,
		ID:   p.projectCfg.Id,
		Type: "browser",
	}

	if p.projectCfg.Source.Local != nil || p.projectCfg.Source.ZipUrl != nil {
		sFields.IsLegacyBrowserSource = true
	}

	return sFields
}

func (p *Project) Close() error {
	if p.projectCfg.Source.ActiveMemo != nil {
		p.projectCfg.Source.ActiveMemo.Close()
	}

	return nil
}

func (p *Project) extraArgs() []string {
	extraArgs := p.projectCfg.SyntheticsArgs
	if len(p.projectCfg.PlaywrightOpts) > 0 {
		s, err := json.Marshal(p.projectCfg.PlaywrightOpts)
		if err != nil {
			// This should never happen, if it was parsed as a config it should be serializable
			logp.L().Warn("could not serialize playwright options '%v': %w", p.projectCfg.PlaywrightOpts, err)
		} else {
			extraArgs = append(extraArgs, "--playwright-options", string(s))
		}
	}
	if p.projectCfg.IgnoreHTTPSErrors {
		extraArgs = append(extraArgs, "--ignore-https-errors")
	}
	if p.projectCfg.Sandbox {
		extraArgs = append(extraArgs, "--sandbox")
	}
	if p.projectCfg.Screenshots != "" {
		extraArgs = append(extraArgs, "--screenshots", p.projectCfg.Screenshots)
	}
	if p.projectCfg.Throttling != nil {
		switch t := p.projectCfg.Throttling.(type) {
		case bool:
			if !t {
				extraArgs = append(extraArgs, "--no-throttling")
			}
		case string:
			extraArgs = append(extraArgs, "--throttling", fmt.Sprintf("%v", p.projectCfg.Throttling))
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

func (p *Project) jobs() []jobs.Job {
	var j jobs.Job
	isScript := p.projectCfg.Source.Inline != nil
	if isScript {
		src := p.projectCfg.Source.Inline.Script
		j = synthexec.InlineJourneyJob(context.TODO(), src, p.Params(), p.Fields(), p.extraArgs()...)
	} else {
		j = func(event *beat.Event) ([]jobs.Job, error) {
			err := p.Fetch()
			if err != nil {
				return nil, fmt.Errorf("could not fetch for project job: %w", err)
			}
			sj, err := synthexec.ProjectJob(context.TODO(), p.Workdir(), p.Params(), p.FilterJourneys(), p.Fields(), p.extraArgs()...)
			if err != nil {
				return nil, err
			}
			return sj(event)
		}
	}
	return []jobs.Job{j}
}

func (p *Project) plugin() plugin.Plugin {
	return plugin.Plugin{
		Jobs:      p.jobs(),
		DoClose:   p.Close,
		Endpoints: 1,
	}
}
