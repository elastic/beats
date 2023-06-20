// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
//go:build linux || darwin

package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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

type SourceJob struct {
	rawCfg     *config.C
	projectCfg *Config
	ctx        context.Context
	cancel     context.CancelFunc
}

func NewSourceJob(rawCfg *config.C) (*SourceJob, error) {
	// Global project context to cancel all jobs
	// on close
	ctx, cancel := context.WithCancel(context.Background())

	s := &SourceJob{
		rawCfg:     rawCfg,
		projectCfg: DefaultConfig(),
		ctx:        ctx,
		cancel:     cancel,
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

func (sj *SourceJob) String() string {
	panic("implement me")
}

func (sj *SourceJob) Fetch() error {
	return sj.projectCfg.Source.Active().Fetch()
}

func (sj *SourceJob) Workdir() string {
	return sj.projectCfg.Source.Active().Workdir()
}

func (sj *SourceJob) Params() map[string]interface{} {
	return sj.projectCfg.Params
}

func (sj *SourceJob) FilterJourneys() synthexec.FilterJourneyConfig {
	return sj.projectCfg.FilterJourneys
}

func (sj *SourceJob) StdFields() stdfields.StdMonitorFields {
	sFields, err := stdfields.ConfigToStdMonitorFields(sj.rawCfg)
	// Should be impossible since outer monitor.go should run this same code elsewhere
	// TODO: Just pass stdfields in to remove second deserialize
	if err != nil {
		logp.L().Warnf("Could not deserialize monitor fields for browser, this should never happen: %s", err)
	}
	return sFields
}

func (sj *SourceJob) Close() error {
	if sj.projectCfg.Source.ActiveMemo != nil {
		sj.projectCfg.Source.ActiveMemo.Close()
	}

	// Cancel running jobs ctxs
	sj.cancel()

	return nil
}

func (sj *SourceJob) extraArgs() []string {
	extraArgs := sj.projectCfg.SyntheticsArgs
	if len(sj.projectCfg.PlaywrightOpts) > 0 {
		s, err := json.Marshal(sj.projectCfg.PlaywrightOpts)
		if err != nil {
			// This should never happen, if it was parsed as a config it should be serializable
			logp.L().Warn("could not serialize playwright options '%v': %w", sj.projectCfg.PlaywrightOpts, err)
		} else {
			extraArgs = append(extraArgs, "--playwright-options", string(s))
		}
	}
	if sj.projectCfg.IgnoreHTTPSErrors {
		extraArgs = append(extraArgs, "--ignore-https-errors")
	}
	if sj.projectCfg.Sandbox {
		extraArgs = append(extraArgs, "--sandbox")
	}
	if sj.projectCfg.Screenshots != "" {
		extraArgs = append(extraArgs, "--screenshots", sj.projectCfg.Screenshots)
	}
	if sj.projectCfg.Throttling != nil {
		switch t := sj.projectCfg.Throttling.(type) {
		case bool:
			if !t {
				extraArgs = append(extraArgs, "--no-throttling")
			}
		case string:
			extraArgs = append(extraArgs, "--throttling", fmt.Sprintf("%v", sj.projectCfg.Throttling))
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

func (sj *SourceJob) jobs() []jobs.Job {
	var j jobs.Job

	isScript := sj.projectCfg.Source.Inline != nil
	ctx := context.WithValue(sj.ctx, synthexec.SynthexecTimeout, sj.projectCfg.Timeout+30*time.Second)

	if isScript {
		src := sj.projectCfg.Source.Inline.Script
		j = synthexec.InlineJourneyJob(ctx, src, sj.Params(), sj.StdFields(), sj.extraArgs()...)
	} else {
		j = func(event *beat.Event) ([]jobs.Job, error) {
			err := sj.Fetch()
			if err != nil {
				return nil, fmt.Errorf("could not fetch for project job: %w", err)
			}
			sj, err := synthexec.ProjectJob(ctx, sj.Workdir(), sj.Params(), sj.FilterJourneys(), sj.StdFields(), sj.extraArgs()...)
			if err != nil {
				return nil, err
			}
			return sj(event)
		}
	}
	return []jobs.Job{j}
}

func (sj *SourceJob) plugin() plugin.Plugin {
	return plugin.Plugin{
		Jobs:      sj.jobs(),
		DoClose:   sj.Close,
		Endpoints: 1,
	}
}
