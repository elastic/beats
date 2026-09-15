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

// osqueryInputRunner is a minimal cfgfile.Runner that reports OTel status when
// started. It is used together with otelStatusFactoryWrapper so that per-input
// componentstatus events reach the OTel host without requiring the cfgfile
// manager infrastructure. osquerybeat manages its own scheduling via osqueryd
// rather than through cfgfile runners, so this shim bridges the gap.
type osqueryInputRunner struct {
	mu       sync.Mutex
	reporter status.StatusReporter
}

var _ cfgfile.Runner = (*osqueryInputRunner)(nil)
var _ status.WithStatusReporter = (*osqueryInputRunner)(nil)

// Start reports status.Running to the injected reporter. This is called once
// per input after the otelStatusFactoryWrapper has injected a sub-reporter,
// mirroring the pattern used by heartbeat monitors.
func (r *osqueryInputRunner) Start() {
	r.mu.Lock()
	reporter := r.reporter
	r.mu.Unlock()
	if reporter != nil {
		reporter.UpdateStatus(status.Running, "")
	}
}

func (r *osqueryInputRunner) Stop() {}

func (r *osqueryInputRunner) String() string { return "osqueryInputRunner" }

func (r *osqueryInputRunner) SetStatusReporter(reporter status.StatusReporter) {
	r.mu.Lock()
	r.reporter = reporter
	r.mu.Unlock()
}

// osqueryInputRunnerFactory is a cfgfile.RunnerFactory whose sole purpose is to
// be wrapped by otelStatusFactoryWrapper. Each Create call returns a new
// osqueryInputRunner; the wrapper then calls SetStatusReporter on it so that
// subsequent Start() calls report componentstatus events to the OTel host.
type osqueryInputRunnerFactory struct{}

var _ cfgfile.RunnerFactory = (*osqueryInputRunnerFactory)(nil)

func (*osqueryInputRunnerFactory) Create(_ beat.PipelineConnector, _ *conf.C) (cfgfile.Runner, error) {
	return &osqueryInputRunner{}, nil
}

func (*osqueryInputRunnerFactory) CheckConfig(_ *conf.C) error { return nil }
