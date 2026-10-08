// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componentstatus"

	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/x-pack/otel/oteltest"
	otelstatus "github.com/elastic/beats/v7/x-pack/otel/status"
	conf "github.com/elastic/elastic-agent-libs/config"
)

func TestOsqueryInputStatusFollowsDaemonRecovery(t *testing.T) {
	host := &oteltest.MockHost{}
	factory := &osqueryInputRunnerFactory{}
	wrapped := otelstatus.StatusReporterFactory(otelstatus.NewGroupStatusReporter(host))(factory)
	create := func(id string) cfgfile.Runner {
		cfg, err := conf.NewConfigFrom(map[string]any{"id": id})
		require.NoError(t, err, "configure input status identity")
		runner, err := wrapped.Create(nil, cfg)
		require.NoError(t, err, "create wrapped status runner")
		runner.Start()
		return runner
	}
	first := create("result")
	create("response")
	factory.UpdateStatus(status.Degraded, "osqueryd exited; restarting")
	require.Equal(t, componentstatus.StatusRecoverableError, host.GetEvent().Status(), "daemon failure must reach receiver health")
	create("added-during-backoff")
	inputState := func(id string) string {
		inputs, ok := host.GetEvent().Attributes().Get("inputs")
		require.True(t, ok, "receiver health must retain input attributes")
		input, ok := inputs.Map().Get(id)
		require.True(t, ok, "receiver health must identify input %s", id)
		state, ok := input.Map().Get("status")
		require.True(t, ok, "input health must contain status")
		return state.Str()
	}
	for _, id := range []string{"result", "response", "added-during-backoff"} {
		require.Equal(t, componentstatus.StatusRecoverableError.String(), inputState(id), "input %s must inherit daemon degradation", id)
	}
	first.Stop()
	factory.UpdateStatus(status.Running, "Osqueryd configuration applied")
	require.Equal(t, componentstatus.StatusOK, host.GetEvent().Status(), "replacement readiness must restore receiver health")
	require.Equal(t, componentstatus.StatusStopped.String(), inputState("result"), "removed inputs must not be revived by recovery")
	for _, id := range []string{"response", "added-during-backoff"} {
		require.Equal(t, componentstatus.StatusOK.String(), inputState(id), "active input %s must recover", id)
	}
}
