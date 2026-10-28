// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package status

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componentstatus"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/common/diagnostics"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/x-pack/otel/oteltest"
	"github.com/elastic/elastic-agent-libs/config"
)

// fakeRunner reports Failed when started and, like Metricbeat modules, does
// not report that it stopped.
type fakeRunner struct {
	reporter status.StatusReporter
	stopped  bool
}

func (r *fakeRunner) Start()         { r.reporter.UpdateStatus(status.Failed, "invalid config") }
func (r *fakeRunner) Stop()          { r.stopped = true }
func (r *fakeRunner) String() string { return "fake" }

func (r *fakeRunner) SetStatusReporter(reporter status.StatusReporter) { r.reporter = reporter }

type plainRunner struct{}

func (plainRunner) Start()         {}
func (plainRunner) Stop()          {}
func (plainRunner) String() string { return "plain" }

type diagRunner struct{ fakeRunner }

func (*diagRunner) Diagnostics() []diagnostics.DiagnosticSetup {
	return []diagnostics.DiagnosticSetup{{Name: "fake"}}
}

type updatableFakeRunner struct {
	fakeRunner
	updated bool
}

func (r *updatableFakeRunner) Update(*config.C) error {
	r.updated = true
	return nil
}

func (r *updatableFakeRunner) isUpdated() bool { return r.updated }

type diagUpdatableRunner struct{ updatableFakeRunner }

func (*diagUpdatableRunner) Diagnostics() []diagnostics.DiagnosticSetup {
	return []diagnostics.DiagnosticSetup{{Name: "fake"}}
}

type fakeFactory struct{ runner cfgfile.Runner }

func (f fakeFactory) Create(beat.PipelineConnector, *config.C) (cfgfile.Runner, error) {
	return f.runner, nil
}

func (fakeFactory) CheckConfig(*config.C) error { return nil }

func createRunner(t *testing.T, host *oteltest.MockHost, runner cfgfile.Runner) cfgfile.Runner {
	t.Helper()
	factory := StatusReporterFactory(NewGroupStatusReporter(host))(fakeFactory{runner: runner})
	created, err := factory.Create(nil, config.MustNewConfigFrom(map[string]any{"id": "input-1"}))
	require.NoError(t, err, "creating the runner")
	return created
}

func TestStatusFactoryRemovesStoppedRunner(t *testing.T) {
	m := &oteltest.MockHost{}
	inner := &fakeRunner{}
	runner := createRunner(t, m, inner)

	runner.Start()
	require.Equal(t, componentstatus.StatusPermanentError, m.GetEvent().Status(), "the failed runner fails the group")

	runner.Stop()

	assert.True(t, inner.stopped, "the wrapped runner must be stopped")
	evt := m.GetEvent()
	assert.Equal(t, componentstatus.StatusOK, evt.Status(), "a stopped runner must not keep the group failed")
	inputs, ok := evt.Attributes().Get(InputStatusAttributesKey)
	require.True(t, ok, "expected the event to carry input statuses")
	assert.Zero(t, inputs.Map().Len(), "a stopped runner must not be reported")
}

func TestStatusFactoryKeepsRunnerInterfaces(t *testing.T) {
	tests := map[string]struct {
		runner        cfgfile.Runner
		wantDiag      bool
		wantUpdatable bool
	}{
		"status only":            {runner: &fakeRunner{}},
		"diagnostics":            {runner: &diagRunner{}, wantDiag: true},
		"updatable":              {runner: &updatableFakeRunner{}, wantUpdatable: true},
		"diagnostics, updatable": {runner: &diagUpdatableRunner{}, wantDiag: true, wantUpdatable: true},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			runner := createRunner(t, &oteltest.MockHost{}, tc.runner)

			unwrapper, ok := runner.(interface{ Unwrap() cfgfile.Runner })
			require.True(t, ok, "expected the runner to be wrapped")
			assert.Same(t, tc.runner, unwrapper.Unwrap(), "Unwrap must return the created runner")
			_, ok = runner.(status.WithStatusReporter)
			assert.True(t, ok, "the wrapper must accept a status reporter")

			diag, ok := runner.(diagnostics.DiagnosticReporter)
			assert.Equal(t, tc.wantDiag, ok, "the wrapper must implement DiagnosticReporter only if the runner does")
			if ok {
				assert.Len(t, diag.Diagnostics(), 1, "diagnostics must come from the runner")
			}

			updatable, ok := runner.(updatableRunner)
			assert.Equal(t, tc.wantUpdatable, ok, "the wrapper must be updatable only if the runner is")
			if ok {
				require.NoError(t, updatable.Update(config.NewConfig()), "updating the runner")
				inner, ok := tc.runner.(interface{ isUpdated() bool })
				require.True(t, ok, "expected an updatable fake runner")
				assert.True(t, inner.isUpdated(), "the update must reach the runner")
			}
		})
	}

	t.Run("runner without status is not wrapped", func(t *testing.T) {
		runner := createRunner(t, &oteltest.MockHost{}, plainRunner{})
		assert.Equal(t, plainRunner{}, runner, "a runner without status must be returned as is")
	})
}
