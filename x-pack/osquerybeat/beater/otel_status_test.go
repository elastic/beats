// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componentstatus"

	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/osqd"
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

func TestOsqueryInputStatusInheritsTerminalDaemonFailure(t *testing.T) {
	host := &oteltest.MockHost{}
	factory := &osqueryInputRunnerFactory{}
	wrapped := otelstatus.StatusReporterFactory(otelstatus.NewGroupStatusReporter(host))(factory)
	cfg, err := conf.NewConfigFrom(map[string]any{"id": "result"})
	require.NoError(t, err, "configure input status identity")
	runner, err := wrapped.Create(nil, cfg)
	require.NoError(t, err, "create input status runner")
	runner.Start()
	require.Empty(t, host.GetEvents(), "input must preserve receiver startup status until daemon readiness")
	factory.UpdateStatus(status.Failed, "osqueryd restart limit exceeded")
	require.Equal(t, componentstatus.StatusPermanentError, host.GetEvent().Status(), "exhausted recovery must reach receiver health")
	cfg, err = conf.NewConfigFrom(map[string]any{"id": "added-after-failure"})
	require.NoError(t, err, "configure input added after terminal failure")
	added, err := wrapped.Create(nil, cfg)
	require.NoError(t, err, "create late input status runner")
	added.Start()
	require.Equal(t, componentstatus.StatusPermanentError, host.GetEvent().Status(), "new input must inherit terminal daemon state")
}

func TestOsquerybeatStartupFailureReachesInputStatus(t *testing.T) {
	host := &oteltest.MockHost{}
	wantErr := errors.New("osqueryd startup check failed")
	ob, b, _ := newStatusTestBeater(t, func(ob *osquerybeat) {
		ob.config.Inputs = recoveryInput("policy", 3)
		ob.config.Inputs[0].ID = "result"
		ob.disableWatcher = true
		ob.otelStatusFactoryWrapper = otelstatus.StatusReporterFactory(otelstatus.NewGroupStatusReporter(host))
		ob.osquerydFactory = func(string, ...osqd.Option) (osqd.Runner, error) { return &mockOsqueryd{checkErr: wantErr}, nil }
	})
	require.ErrorIs(t, ob.Run(b), wantErr, "startup failure must remain terminal")
	require.Equal(t, componentstatus.StatusPermanentError, host.GetEvent().Status(), "fatal startup failure must reach existing input reporters")
	for _, event := range host.GetEvents() {
		require.NotEqual(t, componentstatus.StatusOK, event.Status(), "unready daemon must not report running")
	}
}
