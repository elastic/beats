// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
//go:build linux || darwin
// +build linux darwin

package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
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

type Project struct {
	rawCfg     *config.C
	projectCfg *Config
	ctx        context.Context
	cancel     context.CancelFunc
}

func NewProject(rawCfg *config.C) (*Project, error) {
	// Global project context to cancel all jobs
	// on close
	ctx, cancel := context.WithCancel(context.Background())

	s := &Project{
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

func (p *Project) StdFields() stdfields.StdMonitorFields {
	sFields, err := stdfields.ConfigToStdMonitorFields(p.rawCfg)
	// Should be impossible since outer monitor.go should run this same code elsewhere
	// TODO: Just pass stdfields in to remove second deserialize
	if err != nil {
		logp.L().Warnf("Could not deserialize monitor fields for browser, this should never happen: %s", err)
	}
	return sFields
}

func (p *Project) Close() error {
	if p.projectCfg.Source.ActiveMemo != nil {
		p.projectCfg.Source.ActiveMemo.Close()
	}

	// Cancel running jobs ctxs
	p.cancel()

	return nil
}

<<<<<<< HEAD:x-pack/heartbeat/monitors/browser/project.go
func (p *Project) extraArgs() []string {
	extraArgs := p.projectCfg.SyntheticsArgs
	if len(p.projectCfg.PlaywrightOpts) > 0 {
		s, err := json.Marshal(p.projectCfg.PlaywrightOpts)
=======
// Dev flags + expected number of params, math.MaxInt32 for variadic flags
var filterMap = map[string]int{
	"--dry-run":         0,
	"-h":                0,
	"--help":            0,
	"--inline":          1,
	"--match":           math.MaxInt32,
	"--outfd":           1,
	"--pause-on-error":  0,
	"--quiet-exit-code": 0,
	"-r":                math.MaxInt32,
	"--require":         math.MaxInt32,
	"--reporter":        1,
	"--tags":            math.MaxInt32,
	"-V":                0,
	"--version":         0,
	"--ws-endpoint":     1,
}

func (sj *SourceJob) extraArgs(uiOrigin bool) []string {
	extraArgs := []string{}

	if uiOrigin {
		extraArgs = filterDevFlags(sj.browserCfg.SyntheticsArgs, filterMap)
	} else {
		extraArgs = append(extraArgs, sj.browserCfg.SyntheticsArgs...)
	}

	if len(sj.browserCfg.PlaywrightOpts) > 0 {
		s, err := json.Marshal(sj.browserCfg.PlaywrightOpts)
>>>>>>> c6e955a2ff ([Heartbeat] filter dev flags inside synthetics args (#35788)):x-pack/heartbeat/monitors/browser/sourcejob.go
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

<<<<<<< HEAD:x-pack/heartbeat/monitors/browser/project.go
	isScript := p.projectCfg.Source.Inline != nil
	ctx := context.WithValue(p.ctx, synthexec.SynthexecTimeout, p.projectCfg.Timeout+30*time.Second)

	if isScript {
		src := p.projectCfg.Source.Inline.Script
		j = synthexec.InlineJourneyJob(ctx, src, p.Params(), p.StdFields(), p.extraArgs()...)
=======
	isScript := sj.browserCfg.Source.Inline != nil
	ctx := context.WithValue(sj.ctx, synthexec.SynthexecTimeout, sj.browserCfg.Timeout+30*time.Second)
	sFields := sj.StdFields()

	if isScript {
		src := sj.browserCfg.Source.Inline.Script
		j = synthexec.InlineJourneyJob(ctx, src, sj.Params(), sFields, sj.extraArgs(sFields.Origin != "")...)
>>>>>>> c6e955a2ff ([Heartbeat] filter dev flags inside synthetics args (#35788)):x-pack/heartbeat/monitors/browser/sourcejob.go
	} else {
		j = func(event *beat.Event) ([]jobs.Job, error) {
			err := p.Fetch()
			if err != nil {
				return nil, fmt.Errorf("could not fetch for project job: %w", err)
			}
<<<<<<< HEAD:x-pack/heartbeat/monitors/browser/project.go
			sj, err := synthexec.ProjectJob(ctx, p.Workdir(), p.Params(), p.FilterJourneys(), p.StdFields(), p.extraArgs()...)
=======

			sj, err := synthexec.ProjectJob(ctx, sj.Workdir(), sj.Params(), sj.FilterJourneys(), sFields, sj.extraArgs(sFields.Origin != "")...)
>>>>>>> c6e955a2ff ([Heartbeat] filter dev flags inside synthetics args (#35788)):x-pack/heartbeat/monitors/browser/sourcejob.go
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

type argsIterator struct {
	i    int
	args []string
	val  string
}

func (a *argsIterator) Next() bool {
	if a.i >= len(a.args) {
		return false
	}
	a.val = a.args[a.i]
	a.i++
	return true
}

func (a *argsIterator) Val() string {
	return a.val
}

func (a *argsIterator) Peek() (val string, ok bool) {
	if a.i >= len(a.args) {
		return "", false
	}

	val = a.args[a.i]
	ok = true

	return val, ok
}

// Iterate through list and filter dev flags + potential params
func filterDevFlags(args []string, filter map[string]int) []string {
	result := []string{}

	iter := argsIterator{i: 0, args: args}
	for {
		next := iter.Next()

		if !next {
			break
		}

		if pCount, ok := filter[iter.Val()]; ok {
		ParamsIter:
			for i := 0; i < pCount; i++ {
				// Found filtered flag, check if it has associated params
				if param, ok := iter.Peek(); ok && !strings.HasPrefix(param, "-") {
					iter.Next()
				} else {
					break ParamsIter
				}
			}
		} else {
			result = append(result, iter.Val())
		}
	}

	return result
}
