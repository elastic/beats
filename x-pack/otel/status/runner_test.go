// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package status

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componentstatus"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/x-pack/otel/oteltest"
	"github.com/elastic/elastic-agent-libs/config"
)

type failingFactory struct{ err error }

func (f failingFactory) Create(beat.PipelineConnector, *config.C) (cfgfile.Runner, error) {
	return nil, f.err
}

func (failingFactory) CheckConfig(*config.C) error { return nil }

func TestStatusFactoryReportsFailedCreate(t *testing.T) {
	host := &oteltest.MockHost{}
	reporter := NewGroupStatusReporter(host)
	wrapped := StatusReporterFactory(reporter)(failingFactory{err: errors.New("invalid schedule")})

	cfg, err := config.NewConfigFrom(map[string]any{"id": "bad-monitor"})
	require.NoError(t, err, "could not build test config")

	runner, err := wrapped.Create(nil, cfg)
	require.Error(t, err, "Create must still return the factory error")
	assert.Nil(t, runner, "no runner expected when Create fails")

	evt := host.GetEvent()
	require.NotNil(t, evt, "a failed Create must report a status")
	assert.Equal(t, componentstatus.StatusPermanentError, evt.Status(), "failed input must be reported as a permanent error")
	require.Error(t, evt.Err(), "status event must carry the error")
	assert.Equal(t, "invalid schedule", evt.Err().Error(), "status event must carry the Create error")

	inputs, ok := evt.Attributes().Get(inputStatusAttributesKey)
	require.True(t, ok, "status event must list the per-input statuses")
	_, ok = inputs.Map().Get("bad-monitor")
	assert.True(t, ok, "the failed input must be reported under its own id")
}
