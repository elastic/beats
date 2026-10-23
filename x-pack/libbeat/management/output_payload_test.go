// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"errors"
	"maps"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

// outputPayloadFixture wires a BeatV2Manager to one input unit and one output
// unit so tests can observe which units receive an update.
type outputPayloadFixture struct {
	manager *BeatV2Manager
	input   *mockClientUnit
	output  *mockClientUnit
	units   map[unitKey]*agentUnit
}

// reloadFailures lets a test steer BeatV2Manager.reload down its error paths.
// The reload fixture reads it at call time, so tests can flip a knob after the
// manager is built.
type reloadFailures struct {
	outputReload   error
	inputTransform error
}

func newOutputPayloadFixture(t *testing.T) outputPayloadFixture {
	t.Helper()

	return newManagerPayloadFixture(t, &Config{Enabled: false}, reload.NewRegistry(), nil)
}

// newReloadPayloadFixture builds a fixture whose registry, client and config
// transform are complete enough to drive BeatV2Manager.reload end to end.
func newReloadPayloadFixture(t *testing.T, failures *reloadFailures) outputPayloadFixture {
	t.Helper()

	registry := reload.NewRegistry()
	registry.MustRegisterOutput(&mockOutput{
		ReloadFn: func(*reload.ConfigWithMeta) error { return failures.outputReload },
	})
	registry.MustRegisterInput(&mockReloadable{
		ReloadFn: func([]*reload.ConfigWithMeta) error { return nil },
	})

	// reloadInputs runs the Beat-specific config transform, whose default
	// fallback dereferences an AgentInfo this manager never receives.
	originalTransform := ConfigTransform
	t.Cleanup(func() { ConfigTransform = originalTransform })
	ConfigTransform.SetTransform(func(cfg *proto.UnitExpectedConfig, _ *client.AgentInfo) ([]*reload.ConfigWithMeta, error) {
		if failures.inputTransform != nil {
			return nil, failures.inputTransform
		}

		beatCfg, err := conf.NewConfigFrom(map[string]any{"type": "mock", "id": cfg.GetId()})
		if err != nil {
			return nil, err
		}
		return []*reload.ConfigWithMeta{{Config: beatCfg}}, nil
	})

	return newManagerPayloadFixture(t, &Config{Enabled: true}, registry,
		client.NewV2("", "", client.VersionInfo{}))
}

func newManagerPayloadFixture(
	t *testing.T,
	cfg *Config,
	registry *reload.Registry,
	agentClient client.V2,
) outputPayloadFixture {
	t.Helper()

	m, err := NewV2AgentManagerWithClient(cfg, registry, agentClient, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err, "manager should be created")

	manager, ok := m.(*BeatV2Manager)
	require.True(t, ok, "NewV2AgentManagerWithClient must return a BeatV2Manager")

	input := &mockClientUnit{
		id:       "input-unit",
		unitType: client.UnitTypeInput,
		expected: client.Expected{
			State:  client.UnitStateHealthy,
			Config: &proto.UnitExpectedConfig{Id: "input-unit", Type: "mock", Name: "mock"},
		},
	}
	output := &mockClientUnit{
		id:       "output-unit",
		unitType: client.UnitTypeOutput,
		expected: client.Expected{
			State:  client.UnitStateHealthy,
			Config: &proto.UnitExpectedConfig{Id: "output-unit", Type: "mock", Name: "mock"},
		},
	}

	manager.units[unitKey{client.UnitTypeInput, input.id}] = newAgentUnit(input, manager.logger)
	manager.units[unitKey{client.UnitTypeOutput, output.id}] = newAgentUnit(output, manager.logger)

	// Bring both units to a known state so later assertions only see updates
	// caused by the payload under test.
	manager.UpdateStatus(status.Running, "Running")
	require.Equal(t, 1, input.updateCalls, "input unit should reach the steady state")
	require.Equal(t, 1, output.updateCalls, "output unit should reach the steady state")

	units := make(map[unitKey]*agentUnit, len(manager.units))
	maps.Copy(units, manager.units)

	return outputPayloadFixture{manager: manager, input: input, output: output, units: units}
}

func TestSetOutputPayloadOnlyReachesOutputUnits(t *testing.T) {
	f := newOutputPayloadFixture(t)

	telemetry := map[string]any{
		"heartbeat": map[string]any{
			"scheduler": map[string]any{"jobs": map[string]any{"browser": map[string]any{"limit": 2}}},
		},
	}
	f.manager.SetOutputPayload(telemetry)

	assert.Equal(t, 1, f.input.updateCalls, "output payload must not reach input units")
	assert.Nil(t, f.input.reportedPayload, "input units must not carry output telemetry")

	assert.Equal(t, 2, f.output.updateCalls, "output payload should update the output unit")
	assert.Equal(t, telemetry, f.output.reportedPayload, "output unit should carry the telemetry")

	_, err := structpb.NewStruct(f.output.reportedPayload)
	assert.NoError(t, err, "output payload must be serializable by structpb.NewStruct")
}

// TestSetOutputPayloadSuppressesIdenticalValues asserts the dedup outcome that
// consumers see. The suppression happens in agentUnit.UpdateState, not in the
// manager, so a unit whose payload drifted still recovers.
func TestSetOutputPayloadSuppressesIdenticalValues(t *testing.T) {
	f := newOutputPayloadFixture(t)

	f.manager.SetOutputPayload(map[string]any{"telemetry": map[string]any{"running": 1}})
	require.Equal(t, 2, f.output.updateCalls, "first output payload should be forwarded")

	f.manager.SetOutputPayload(map[string]any{"telemetry": map[string]any{"running": 1}})
	assert.Equal(t, 2, f.output.updateCalls, "an identical output payload should be suppressed")

	f.manager.SetOutputPayload(map[string]any{"telemetry": map[string]any{"running": 2}})
	assert.Equal(t, 3, f.output.updateCalls, "a changed output payload should be forwarded")
}

func TestSetOutputPayloadDoesNotMutateCallerMap(t *testing.T) {
	f := newOutputPayloadFixture(t)

	nested := map[string]any{"running": 1}
	payload := map[string]any{"telemetry": nested}
	f.manager.SetOutputPayload(payload)
	require.Equal(t, 2, f.output.updateCalls, "first output payload should be forwarded")

	nested["running"] = 999
	f.manager.SetOutputPayload(map[string]any{"telemetry": map[string]any{"running": 1}})

	assert.Equal(t, 2, f.output.updateCalls,
		"mutating the caller map must not alter the stored comparison snapshot")
}

func TestStatusUpdateRetainsOutputPayload(t *testing.T) {
	f := newOutputPayloadFixture(t)

	telemetry := map[string]any{"telemetry": map[string]any{"running": 1}}
	f.manager.SetOutputPayload(telemetry)
	require.Equal(t, 2, f.output.updateCalls, "first output payload should be forwarded")

	f.manager.UpdateStatus(status.Degraded, "Degraded")

	assert.Equal(t, client.UnitStateDegraded, f.output.reportedState, "status change should reach the output unit")
	assert.Equal(t, telemetry, f.output.reportedPayload, "status change must keep the output payload attached")
	assert.Equal(t, client.UnitStateDegraded, f.input.reportedState, "status change should reach input units")
	assert.Nil(t, f.input.reportedPayload, "status change must not leak output telemetry to input units")
}

// TestPayloadForKeepsTelemetryAcrossUnitChurn covers the unit add/modify paths,
// which reuse lockedPayloadFor and must not clear the output payload.
func TestPayloadForKeepsTelemetryAcrossUnitChurn(t *testing.T) {
	f := newOutputPayloadFixture(t)

	telemetry := map[string]any{"telemetry": map[string]any{"running": 1}}
	f.manager.SetOutputPayload(telemetry)

	f.manager.mx.Lock()
	defer f.manager.mx.Unlock()

	assert.Equal(t, telemetry, f.manager.lockedPayloadFor(client.UnitTypeOutput),
		"output units must keep telemetry through state transitions")
	assert.Nil(t, f.manager.lockedPayloadFor(client.UnitTypeInput),
		"input units must not receive output telemetry")
}

// schedulerTelemetry mirrors the shape Heartbeat publishes so the reload
// regressions assert on the real consumer contract.
func schedulerTelemetry() map[string]any {
	return map[string]any{
		"heartbeat": map[string]any{
			"scheduler": map[string]any{
				"jobs": map[string]any{
					"browser": map[string]any{
						"limit":   int64(2),
						"running": int64(1),
						"waiting": int64(0),
						"schedule_delay": map[string]any{
							"count":    uint64(0),
							"total_ms": uint64(0),
							"max_ms":   uint64(0),
						},
					},
				},
			},
		},
	}
}

func TestReloadRetainsUnitPayloads(t *testing.T) {
	f := newReloadPayloadFixture(t, &reloadFailures{})

	global := map[string]any{"osquery": map[string]any{"version": "5.0"}}
	telemetry := schedulerTelemetry()
	f.manager.SetPayload(global)
	f.manager.SetOutputPayload(telemetry)

	f.manager.reload(f.units)

	assert.Equal(t, client.UnitStateHealthy, f.output.reportedState,
		"a successful reload should mark the output unit healthy")
	assert.Equal(t, map[string]any{
		"osquery":   global["osquery"],
		"heartbeat": telemetry["heartbeat"],
	}, f.output.reportedPayload,
		"reload must keep scheduler telemetry on the output unit")

	assert.Equal(t, client.UnitStateHealthy, f.input.reportedState,
		"a successful reload should mark the input unit healthy")
	assert.Equal(t, global, f.input.reportedPayload,
		"reload must keep the global payload on input units")
}

func TestReloadOutputFailureRetainsOutputPayload(t *testing.T) {
	f := newReloadPayloadFixture(t, &reloadFailures{outputReload: errors.New("output is unreachable")})

	telemetry := schedulerTelemetry()
	f.manager.SetOutputPayload(telemetry)

	f.manager.reload(f.units)

	assert.Equal(t, client.UnitStateFailed, f.output.reportedState,
		"a failed output reload should mark the output unit failed")
	assert.Equal(t, telemetry, f.output.reportedPayload,
		"a failed output reload must keep scheduler telemetry on the output unit")
}

func TestReloadInputFailureRetainsGlobalPayload(t *testing.T) {
	f := newReloadPayloadFixture(t, &reloadFailures{inputTransform: errors.New("bad input config")})

	global := map[string]any{"osquery": map[string]any{"version": "5.0"}}
	telemetry := schedulerTelemetry()
	f.manager.SetPayload(global)
	f.manager.SetOutputPayload(telemetry)

	f.manager.reload(f.units)

	assert.Equal(t, client.UnitStateFailed, f.input.reportedState,
		"an input config error should mark the input unit failed")
	assert.Equal(t, global, f.input.reportedPayload,
		"a failed input reload must keep the global payload on input units")
	assert.Equal(t, map[string]any{
		"osquery":   global["osquery"],
		"heartbeat": telemetry["heartbeat"],
	}, f.output.reportedPayload,
		"a failed input reload must keep scheduler telemetry on the output unit")
}

// TestSetOutputPayloadRestoresClearedUnitPayload pins why the manager must not
// short circuit on an identical payload: a unit that lost its payload has to
// recover on the next snapshot, while a unit that is already correct stays
// quiet because agentUnit.UpdateState is the authoritative dedup layer.
func TestSetOutputPayloadRestoresClearedUnitPayload(t *testing.T) {
	f := newOutputPayloadFixture(t)

	telemetry := schedulerTelemetry()
	f.manager.SetOutputPayload(telemetry)
	require.Equal(t, 2, f.output.updateCalls, "first output payload should be forwarded")
	require.Equal(t, telemetry, f.output.reportedPayload, "output unit should carry the telemetry")

	outputUnit := f.manager.units[unitKey{client.UnitTypeOutput, f.output.id}]
	require.NoError(t, outputUnit.UpdateState(status.Running, "Running", nil),
		"clearing the unit payload should succeed")
	require.Nil(t, f.output.reportedPayload, "the unit payload should now be cleared")
	clearedCalls := f.output.updateCalls

	f.manager.SetOutputPayload(telemetry)
	assert.Equal(t, clearedCalls+1, f.output.updateCalls,
		"an identical snapshot must restore a unit that lost its payload")
	assert.Equal(t, telemetry, f.output.reportedPayload,
		"the restored payload should be the retained telemetry")

	restoredCalls := f.output.updateCalls
	f.manager.SetOutputPayload(telemetry)
	assert.Equal(t, restoredCalls, f.output.updateCalls,
		"agentUnit dedup should suppress an identical delivery to an already-correct unit")
}

func TestOutputPayloadMergesWithGlobalPayload(t *testing.T) {
	f := newOutputPayloadFixture(t)

	global := map[string]any{
		"global": map[string]any{"value": 1},
		"shared": "global",
	}
	f.manager.SetPayload(global)
	assert.Equal(t, global, f.input.reportedPayload, "global payload should reach input units")

	f.manager.SetOutputPayload(map[string]any{
		"telemetry": map[string]any{"running": 1},
		"shared":    "output",
	})

	assert.Equal(t, map[string]any{
		"global":    map[string]any{"value": 1},
		"shared":    "output",
		"telemetry": map[string]any{"running": 1},
	}, f.output.reportedPayload, "output unit should see the merge with output keys taking precedence")

	assert.Equal(t, map[string]any{
		"global": map[string]any{"value": 1},
		"shared": "global",
	}, global, "merging must not mutate the caller's global payload")
	assert.Equal(t, global, f.input.reportedPayload, "input units should keep only the global payload")
}
