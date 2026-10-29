// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package otelmanager

import (
	"context"
	"net"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func TestUnitsManagerAppliesUnitsAfterPostInit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		agent, inputs, m := startUnitsManager(t)
		agent.setUnits(outputUnit(), inputUnit(t, "a", "@every 1m", 1))
		synctest.Wait()

		time.Sleep(time.Hour)
		synctest.Wait()
		assert.Empty(t, inputs.reloads(), "units must not be applied before the beat is ready")

		m.PostInit()
		synctest.Wait()
		assert.Equal(t, [][]string{{"a"}}, inputs.reloads(), "units received before the beat was ready should be applied once it is")
	})
}

func TestUnitsManagerDebouncesChanges(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		agent, inputs, _ := startReadyUnitsManager(t, outputUnit(), inputUnit(t, "a", "@every 1m", 1))
		require.Equal(t, [][]string{{"a"}}, inputs.reloads())

		// Changes keep postponing the reload until they stop for unitsDebounce.
		agent.setUnits(outputUnit(), inputUnit(t, "a", "@every 1m", 1), inputUnit(t, "b", "@every 1m", 1))
		synctest.Wait()
		time.Sleep(unitsDebounce / 2)
		agent.setUnits(outputUnit(), inputUnit(t, "a", "@every 1m", 1), inputUnit(t, "b", "@every 1m", 1), inputUnit(t, "c", "@every 1m", 1))
		synctest.Wait()
		time.Sleep(unitsDebounce - time.Nanosecond)
		synctest.Wait()
		assert.Len(t, inputs.reloads(), 1, "changes should not be applied before they stop for unitsDebounce")

		time.Sleep(time.Nanosecond)
		synctest.Wait()
		assert.Equal(t, [][]string{{"a"}, {"a", "b", "c"}}, inputs.reloads(), "changes should be applied together")
	})
}

func TestUnitsManagerAppliesModifiedAndRemovedUnits(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		agent, inputs, _ := startReadyUnitsManager(t, outputUnit(), inputUnit(t, "a", "@every 1m", 1), inputUnit(t, "b", "@every 1m", 1))
		applyUnits(agent, outputUnit(), inputUnit(t, "a", "@every 5m", 2))
		assert.Equal(t, []string{"@every 5m"}, inputs.last(), "the modified config should be applied")
		applyUnits(agent, outputUnit())

		assert.Equal(t, [][]string{{"a", "b"}, {"a"}, nil}, inputs.reloads())
	})
}

func TestUnitsManagerSkipsUnchangedConfigs(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		agent, inputs, _ := startReadyUnitsManager(t, outputUnit(), inputUnit(t, "a", "@every 1m", 1))

		// A log level change modifies the unit, but not its config.
		unit := inputUnit(t, "a", "@every 1m", 1)
		unit.LogLevel = proto.UnitLogLevel_TRACE
		applyUnits(agent, outputUnit(), unit)
		assert.Len(t, inputs.reloads(), 1, "inputs should not be reloaded when their configs don't change")
	})
}

func TestUnitsManagerStopStopsInputs(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		_, inputs, m := startReadyUnitsManager(t, outputUnit(), inputUnit(t, "a", "@every 1m", 1))
		m.Stop()
		assert.Equal(t, [][]string{{"a"}, nil}, inputs.reloads(), "Stop should stop the inputs")
	})
}

func TestUnitsManagerRendersConfig(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		_, _, m := startReadyUnitsManager(t, outputUnit(), inputUnit(t, "a", "@every 1m", 1))

		// No output reloader is registered, but the output is still part of the
		// rendered config.
		rendered := string(m.renderedConfig())
		assert.Contains(t, rendered, "@every 1m")
		assert.Contains(t, rendered, "pretty: true")
	})
}

// startReadyUnitsManager starts a UnitsManager, sends it units and calls
// PostInit, so the units are applied.
func startReadyUnitsManager(t *testing.T, units ...*proto.UnitExpected) (*unitsAgent, *recordingInputs, *UnitsManager) {
	agent, inputs, m := startUnitsManager(t)
	agent.setUnits(units...)
	synctest.Wait()
	m.PostInit()
	synctest.Wait()
	return agent, inputs, m
}

// applyUnits sends units to the manager and waits until it applies them.
func applyUnits(agent *unitsAgent, units ...*proto.UnitExpected) {
	agent.setUnits(units...)
	synctest.Wait()
	time.Sleep(unitsDebounce)
	synctest.Wait()
}

// startUnitsManager starts a UnitsManager connected to a fake agent, without
// calling PostInit. Everything runs over in-memory connections, so it works in
// a synctest bubble. The manager is stopped when the test ends.
func startUnitsManager(t *testing.T) (*unitsAgent, *recordingInputs, *UnitsManager) {
	agent := newUnitsAgent(t)
	agentClient := client.NewV2("pipe", "token", client.VersionInfo{Name: "test"},
		client.WithGRPCDialOptions(
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithContextDialer(agent.listener.dial),
		))

	inputs := &recordingInputs{}
	registry := reload.NewRegistry()
	registry.MustRegisterInput(inputs)

	m := NewUnitsManager(agentClient, registry, scheduleTransform, logptest.NewTestingLogger(t, ""))
	require.NoError(t, m.PreInit())
	t.Cleanup(m.Stop)
	return agent, inputs, m
}

// scheduleTransform turns an input unit into one input config holding the
// unit's schedule.
func scheduleTransform(cfg *proto.UnitExpectedConfig, _ *client.AgentInfo) ([]*reload.ConfigWithMeta, error) {
	c, err := conf.NewConfigFrom(map[string]any{"schedule": cfg.GetSource().AsMap()["schedule"]})
	if err != nil {
		return nil, err
	}
	return []*reload.ConfigWithMeta{{Config: c}}, nil
}

// recordingInputs records the input unit IDs of every reload, and the
// schedules of the last one.
type recordingInputs struct {
	mx        sync.Mutex
	unitIDs   [][]string
	schedules []string
}

func (r *recordingInputs) Reload(configs []*reload.ConfigWithMeta) error {
	r.mx.Lock()
	defer r.mx.Unlock()
	var ids, schedules []string
	for _, c := range configs {
		ids = append(ids, c.InputUnitID)
		schedule, err := c.Config.String("schedule", -1)
		if err != nil {
			return err
		}
		schedules = append(schedules, schedule)
	}
	r.unitIDs = append(r.unitIDs, ids)
	r.schedules = schedules
	return nil
}

func (r *recordingInputs) reloads() [][]string {
	r.mx.Lock()
	defer r.mx.Unlock()
	return slices.Clone(r.unitIDs)
}

func (r *recordingInputs) last() []string {
	r.mx.Lock()
	defer r.mx.Unlock()
	return slices.Clone(r.schedules)
}

// unitsAgent serves the control protocol, and sends the expected units
// whenever they change, as Elastic Agent does for receivers.
type unitsAgent struct {
	proto.UnimplementedElasticAgentServer

	listener *pipeListener

	mx      sync.Mutex
	units   []*proto.UnitExpected
	changed chan struct{}
}

func newUnitsAgent(t *testing.T) *unitsAgent {
	agent := &unitsAgent{listener: newPipeListener(), changed: make(chan struct{})}
	server := grpc.NewServer()
	proto.RegisterElasticAgentServer(server, agent)
	go func() { _ = server.Serve(agent.listener) }()
	t.Cleanup(server.Stop)
	return agent
}

func (a *unitsAgent) setUnits(units ...*proto.UnitExpected) {
	a.mx.Lock()
	defer a.mx.Unlock()
	a.units = units
	close(a.changed)
	a.changed = make(chan struct{})
}

func (a *unitsAgent) CheckinV2(server proto.ElasticAgent_CheckinV2Server) error {
	recvDone := make(chan error, 1)
	go func() {
		for {
			if _, err := server.Recv(); err != nil {
				recvDone <- err
				return
			}
		}
	}()
	for {
		a.mx.Lock()
		expected := &proto.CheckinExpected{Units: a.units}
		changed := a.changed
		a.mx.Unlock()
		if err := server.Send(expected); err != nil {
			return err
		}
		select {
		case <-changed:
		case <-recvDone:
			return nil
		}
	}
}

func (a *unitsAgent) Actions(server proto.ElasticAgent_ActionsServer) error {
	for {
		if _, err := server.Recv(); err != nil {
			return nil //nolint:nilerr // the stream ends when the client stops
		}
	}
}

// pipeListener is a listener for in-memory connections from dial.
type pipeListener struct {
	conns     chan net.Conn
	done      chan struct{}
	closeOnce sync.Once
}

func newPipeListener() *pipeListener {
	return &pipeListener{conns: make(chan net.Conn), done: make(chan struct{})}
}

func (l *pipeListener) dial(ctx context.Context, _ string) (net.Conn, error) {
	clientConn, serverConn := net.Pipe()
	select {
	case l.conns <- serverConn:
		return clientConn, nil
	case <-l.done:
		return nil, net.ErrClosed
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (l *pipeListener) Accept() (net.Conn, error) {
	select {
	case conn := <-l.conns:
		return conn, nil
	case <-l.done:
		return nil, net.ErrClosed
	}
}

func (l *pipeListener) Close() error {
	l.closeOnce.Do(func() { close(l.done) })
	return nil
}

func (l *pipeListener) Addr() net.Addr { return pipeAddr{} }

type pipeAddr struct{}

func (pipeAddr) Network() string { return "pipe" }
func (pipeAddr) String() string  { return "pipe" }

func outputUnit() *proto.UnitExpected {
	return &proto.UnitExpected{
		Id:             "output",
		Type:           proto.UnitType_OUTPUT,
		State:          proto.State_HEALTHY,
		ConfigStateIdx: 1,
		Config: &proto.UnitExpectedConfig{
			Id:   "default",
			Type: "console",
			Source: &structpb.Struct{Fields: map[string]*structpb.Value{
				"type":   structpb.NewStringValue("console"),
				"pretty": structpb.NewBoolValue(true),
			}},
		},
	}
}

func inputUnit(t *testing.T, id string, schedule string, configIdx uint64) *proto.UnitExpected {
	source, err := structpb.NewStruct(map[string]any{"id": id, "schedule": schedule})
	require.NoError(t, err)
	return &proto.UnitExpected{
		Id:             id,
		Type:           proto.UnitType_INPUT,
		ConfigStateIdx: configIdx,
		State:          proto.State_HEALTHY,
		Config: &proto.UnitExpectedConfig{
			Id:     id,
			Type:   "test",
			Source: source,
		},
	}
}
