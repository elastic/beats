// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"sync"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/management/status"
	conf "github.com/elastic/elastic-agent-libs/config"
)

// osqueryInputRunner is a minimal cfgfile.Runner that reports OTel status for
// its shared daemon. It is used with otelStatusFactoryWrapper so that per-input
// componentstatus events reach the OTel host without requiring the cfgfile
// manager infrastructure. osquerybeat manages its own scheduling via osqueryd
// rather than through cfgfile runners, so this shim bridges the gap.
type osqueryInputRunner struct {
	mu       sync.Mutex
	reporter status.StatusReporter
	factory  *osqueryInputRunnerFactory
}

var _ cfgfile.Runner = (*osqueryInputRunner)(nil)
var _ status.WithStatusReporter = (*osqueryInputRunner)(nil)

// Start inherits the current daemon state, including when an input is added
// during restart backoff. The wrapper has already injected its sub-reporter.
func (r *osqueryInputRunner) Start() {
	r.factory.mu.Lock()
	defer r.factory.mu.Unlock()
	if r.factory.runners == nil {
		r.factory.runners = make(map[*osqueryInputRunner]struct{})
	}
	r.factory.runners[r] = struct{}{}
	state := r.factory.state
	if state == status.Unknown {
		state = status.Running
	}
	r.updateStatus(state, r.factory.message)
}

func (r *osqueryInputRunner) Stop() {
	r.factory.mu.Lock()
	defer r.factory.mu.Unlock()
	delete(r.factory.runners, r)
	r.updateStatus(status.Stopped, "")
}

func (r *osqueryInputRunner) updateStatus(state status.Status, message string) {
	r.mu.Lock()
	reporter := r.reporter
	r.mu.Unlock()
	if reporter != nil {
		reporter.UpdateStatus(state, message)
	}
}

func (r *osqueryInputRunner) String() string { return "osqueryInputRunner" }

func (r *osqueryInputRunner) SetStatusReporter(reporter status.StatusReporter) {
	r.mu.Lock()
	r.reporter = reporter
	r.mu.Unlock()
}

// osqueryInputRunnerFactory forwards shared daemon state to the active input
// reporters. OtelManager.UpdateStatus does not forward beat status to the host,
// so osquerybeat must update its existing per-input status bridge as well.
type osqueryInputRunnerFactory struct {
	mu      sync.Mutex
	state   status.Status
	message string
	runners map[*osqueryInputRunner]struct{}
}

var _ cfgfile.RunnerFactory = (*osqueryInputRunnerFactory)(nil)

func (f *osqueryInputRunnerFactory) Create(_ beat.PipelineConnector, _ *conf.C) (cfgfile.Runner, error) {
	return &osqueryInputRunner{factory: f}, nil
}

func (*osqueryInputRunnerFactory) CheckConfig(_ *conf.C) error { return nil }

func (f *osqueryInputRunnerFactory) UpdateStatus(state status.Status, message string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.state, f.message = state, message
	for runner := range f.runners {
		runner.updateStatus(state, message)
	}
}
