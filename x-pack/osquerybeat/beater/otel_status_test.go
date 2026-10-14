// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componentstatus"

	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/osqd"
	"github.com/elastic/beats/v7/x-pack/otel/oteltest"
	otelstatus "github.com/elastic/beats/v7/x-pack/otel/status"
	conf "github.com/elastic/elastic-agent-libs/config"
)

func TestOsqueryIdleStatusDoesNotMakeInputsReady(t *testing.T) {
	host := &oteltest.MockHost{}
	manager := &testManager{}
	factory := &osqueryInputRunnerFactory{}
	reporter := osqueryStatusReporter{manager: manager, inputs: factory}
	reporter.WaitingForInputs()
	require.Equal(t, status.Running, manager.events[len(manager.events)-1].Status, "Agent must consider the idle beat running")
	wrapped := otelstatus.StatusReporterFactory(otelstatus.NewGroupStatusReporter(host))(factory)
	cfg, err := conf.NewConfigFrom(map[string]any{"id": "first"})
	require.NoError(t, err, "configure the first input")
	input, err := wrapped.Create(nil, cfg)
	require.NoError(t, err, "create the first input reporter")
	input.Start()
	require.Empty(t, host.GetEvents(), "idle Agent status must not become daemon readiness for a new input")
}

func TestOsqueryInputReadinessAfterIdle(t *testing.T) {
	for _, scenario := range []string{"empty startup", "remove and re-add", "changed options"} {
		t.Run(scenario, func(t *testing.T) {
			thriftTimeout := 4
			if scenario == "remove and re-add" {
				thriftTimeout = 3 // Removal must invalidate readiness even when options do not change.
			}
			host := &oteltest.MockHost{}
			factory := &osqueryInputRunnerFactory{state: status.Configuring}
			reporter := osqueryStatusReporter{manager: &testManager{}, inputs: factory}
			wrapped := otelstatus.StatusReporterFactory(otelstatus.NewGroupStatusReporter(host))(factory)
			create := func(id string) cfgfile.Runner {
				cfg, err := conf.NewConfigFrom(map[string]any{"id": id})
				require.NoError(t, err, "configure input identity")
				input, err := wrapped.Create(nil, cfg)
				require.NoError(t, err, "create input reporter")
				input.Start()
				return input
			}
			callbacks := make(chan func(), 2)
			cleanup := make(chan struct{})
			var releaseOnce sync.Once
			release := func() { releaseOnce.Do(func() { close(cleanup) }) }
			run := func(ctx context.Context, _ osqd.Flags, _ config.ExtensionsConfig, _ <-chan []config.InputConfig, ready, _ func()) error {
				callbacks <- ready
				<-ctx.Done()
				<-cleanup
				return ctx.Err()
			}
			runner, ctx, cancel, done := recoveryRunner(t, run, func(r *osqueryRunner) {
				r.reportStatus = reporter.UpdateStatus
				r.reportWaiting = reporter.WaitingForInputs
			})
			t.Cleanup(release)
			reporter.WaitingForInputs()
			if scenario != "empty startup" {
				require.NoError(t, runner.Update(ctx, recoveryInput("old", 3)), "start the original daemon")
				old := create("old")
				ready := waitRecoveryValue(t, callbacks)
				ready()
				require.Eventually(t, func() bool {
					return len(host.GetEvents()) > 0 && host.GetEvent().Status() == componentstatus.StatusOK
				}, time.Second, time.Millisecond, "original daemon must reach readiness")
				if scenario == "remove and re-add" {
					require.NoError(t, runner.Update(ctx, nil), "removal must invalidate readiness before returning")
				}
				old.Stop()
				// The original run is deliberately held in cleanup. A new reporter
				// must not inherit its readiness, even before replacement startup.
				require.NoError(t, runner.Update(ctx, recoveryInput("new", thriftTimeout)), "replacement update must not wait for cleanup")
				ready() // A stale callback cannot restore the original run's readiness.
			} else {
				require.NoError(t, runner.Update(ctx, recoveryInput("new", thriftTimeout)), "first input must leave idle status before returning")
			}
			before := len(host.GetEvents())
			create("new")
			for _, event := range host.GetEvents()[before:] {
				require.NotEqual(t, componentstatus.StatusOK, event.Status(), "new input must not inherit idle or old-run Running")
			}
			release()
			ready := waitRecoveryValue(t, callbacks)
			for _, event := range host.GetEvents()[before:] {
				require.NotEqual(t, componentstatus.StatusOK, event.Status(), "startup alone must not report configuration readiness")
			}
			ready()
			require.Eventually(t, func() bool {
				return len(host.GetEvents()) > before && host.GetEvent().Status() == componentstatus.StatusOK
			}, time.Second, time.Millisecond, "current daemon readiness must restore healthy input status")
			require.NoError(t, runner.Update(ctx, recoveryInput("same-run addition", thriftTimeout)), "adding inputs with unchanged options must retain daemon readiness")
			create("same-run addition")
			require.Equal(t, componentstatus.StatusOK, host.GetEvent().Status(), "healthy additions must not wait for the once-only readiness callback to fire again")
			cancel()
			require.ErrorIs(t, waitRecoveryValue(t, done), context.Canceled, "runner must stop cleanly")
		})
	}
}

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
