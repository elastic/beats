// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
//go:build linux || darwin || synthetics

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
)

type SourceJob struct {
	rawCfg     *config.C
	browserCfg *Config
	ctx        context.Context
	cancel     context.CancelFunc
}

func NewSourceJob(rawCfg *config.C) (*SourceJob, error) {
	// Global browser context to cancel all jobs
	// on close
	ctx, cancel := context.WithCancel(context.Background())

	s := &SourceJob{
		rawCfg:     rawCfg,
		browserCfg: DefaultConfig(),
		ctx:        ctx,
		cancel:     cancel,
	}
	err := rawCfg.Unpack(s.browserCfg)
	if err != nil {
		return nil, ErrBadConfig(err)
	}

	return s, nil
}

func ErrBadConfig(err error) error {
	return fmt.Errorf("could not parse browser config: %w", err)
}

func (sj *SourceJob) String() string {
	panic("implement me")
}

func (sj *SourceJob) Fetch() error {
	return sj.browserCfg.Source.Active().Fetch()
}

func (sj *SourceJob) Workdir() string {
	return sj.browserCfg.Source.Active().Workdir()
}

func (sj *SourceJob) Params() map[string]interface{} {
	return sj.browserCfg.Params
}

func (sj *SourceJob) FilterJourneys() synthexec.FilterJourneyConfig {
	return sj.browserCfg.FilterJourneys
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
	if sj.browserCfg.Source.ActiveMemo != nil {
		sj.browserCfg.Source.ActiveMemo.Close()
	}

	// Cancel running jobs ctxs
	sj.cancel()

	return nil
}

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
		if err != nil {
			// This should never happen, if it was parsed as a config it should be serializable
			logp.L().Warnf("could not serialize playwright options '%v': %w", sj.browserCfg.PlaywrightOpts, err)
		} else {
			extraArgs = append(extraArgs, "--playwright-options", string(s))
		}
	}
	if sj.browserCfg.IgnoreHTTPSErrors {
		extraArgs = append(extraArgs, "--ignore-https-errors")
	}
	if sj.browserCfg.Sandbox {
		extraArgs = append(extraArgs, "--sandbox")
	}
	if sj.browserCfg.Screenshots != "" {
		extraArgs = append(extraArgs, "--screenshots", sj.browserCfg.Screenshots)
	}
	if sj.browserCfg.Throttling != nil {
		switch t := sj.browserCfg.Throttling.(type) {
		case bool:
			if !t {
				extraArgs = append(extraArgs, "--no-throttling")
			}
		case string:
			extraArgs = append(extraArgs, "--throttling", fmt.Sprintf("%v", sj.browserCfg.Throttling))
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

	isScript := sj.browserCfg.Source.Inline != nil
	ctx := context.WithValue(sj.ctx, synthexec.SynthexecTimeout, sj.browserCfg.Timeout+30*time.Second)
	sFields := sj.StdFields()

	if isScript {
		src := sj.browserCfg.Source.Inline.Script
		j = synthexec.InlineJourneyJob(ctx, src, sj.Params(), sFields, sj.extraArgs(sFields.Origin != "")...)
	} else {
		j = func(event *beat.Event) ([]jobs.Job, error) {
			err := sj.Fetch()
			if err != nil {
				return nil, fmt.Errorf("could not fetch for browser source job: %w", err)
			}

			sj, err := synthexec.ProjectJob(ctx, sj.Workdir(), sj.Params(), sj.FilterJourneys(), sFields, sj.extraArgs(sFields.Origin != "")...)
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
