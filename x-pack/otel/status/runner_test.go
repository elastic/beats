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
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/pkg/config"
	"github.com/elastic/beats/v7/x-pack/otel/oteltest"
)

type statusRunner struct {
	reporter status.StatusReporter
}

func (r *statusRunner) Start()                                      {}
func (r *statusRunner) Stop()                                       {}
func (r *statusRunner) String() string                              { return "statusRunner" }
func (r *statusRunner) SetStatusReporter(rep status.StatusReporter) { r.reporter = rep }

type statusRunnerFactory struct{}

func (statusRunnerFactory) Create(beat.PipelineConnector, *config.C) (cfgfile.Runner, error) {
	return &statusRunner{}, nil
}

func (statusRunnerFactory) CheckConfig(*config.C) error { return nil }

func mustConfig(t *testing.T, fields map[string]any) *config.C {
	t.Helper()
	c, err := config.NewConfigFrom(fields)
	require.NoError(t, err, "creating the runner config")
	return c
}

func TestStatusFactoryReportsConfiguredRunners(t *testing.T) {
	m := &oteltest.MockHost{}
	reporter := newGroupStatusReporter(m, 0) // report every change
	factory := StatusReporterFactory(reporter)(statusRunnerFactory{})

	withID := mustConfig(t, map[string]any{"id": "with-id", "type": "test"})
	withoutID := mustConfig(t, map[string]any{"type": "test", "paths": []string{"/a"}})
	unconfigured := mustConfig(t, map[string]any{"id": "unconfigured", "type": "test"})
	cfgfile.SetRunnerConfigs(factory, "test", []*config.C{withID, withoutID})

	for _, c := range []*config.C{withID, withoutID} {
		runner, err := factory.Create(nil, c)
		require.NoError(t, err, "creating a configured runner")
		r, ok := runner.(*statusRunner)
		require.True(t, ok, "the factory should return the runner itself")
		require.NotNil(t, r.reporter, "a configured runner should get a reporter")
		r.reporter.UpdateStatus(status.Running, "")
	}
	statuses := inputStatuses(t, m)
	assert.Len(t, statuses, 2, "both configured runners should be reported")
	assert.Contains(t, statuses, "with-id", "a runner with an id should be reported by its id")
	for id, s := range statuses {
		assert.Equal(t, componentstatus.StatusOK.String(), s, "runner %s should report its status", id)
	}

	runner, err := factory.Create(nil, unconfigured)
	require.NoError(t, err, "creating an unconfigured runner")
	r, ok := runner.(*statusRunner)
	require.True(t, ok, "the factory should return the runner itself")
	assert.Nil(t, r.reporter, "an unconfigured runner should get no reporter")
	assert.NotContains(t, inputStatuses(t, m), "unconfigured", "an unconfigured runner should not be reported")
}
