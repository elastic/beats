// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !windows

package hbreceiver

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"io"
	"net"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	gproto "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/elastic/beats/v7/x-pack/otel/oteltest"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
	"github.com/elastic/elastic-agent-libs/testing/certutil"
)

// TestReceiverWithControlProtocol checks that a receiver configured with
// management.agent.connection_secret gets its monitors from Elastic Agent over the
// control protocol, that changing one monitor leaves the others running, and
// that monitor status is still reported to the collector.
func TestReceiverWithControlProtocol(t *testing.T) {
	// Registered first so it runs after the receiver and the fake agent stop.
	t.Cleanup(func() { oteltest.VerifyNoLeaks(t) })

	target := startTCPTarget(t)

	// Version 1 runs monitors a and b, plus an invalid monitor. Version 2 keeps a,
	// replaces b with c, and removes the invalid monitor.
	unitsV1 := []*proto.UnitExpected{
		outputUnit(1),
		monitorUnit(t, "monitor-a", target, "@every 1s", 1),
		monitorUnit(t, "monitor-b", target, "@every 1s", 1),
		monitorUnit(t, "monitor-invalid", target, "not a schedule", 1),
	}
	unitsV2 := []*proto.UnitExpected{
		outputUnit(1),
		monitorUnit(t, "monitor-a", target, "@every 1s", 1),
		monitorUnit(t, "monitor-c", target, "@every 1s", 2),
	}

	agent := newFakeAgent(t)
	agent.setUnits(unitsV1)

	observedLogs, zapLogs := observer.New(zapcore.DebugLevel)
	factory := NewFactoryWithSettings(Settings{Home: t.TempDir()})
	settings := receiver.Settings{}
	settings.ID = component.NewIDWithName(factory.Type(), "r1")
	settings.Logger = zap.New(observedLogs)

	sink := new(consumertest.LogsSink)
	t.Cleanup(func() {
		if t.Failed() {
			for _, entry := range zapLogs.All() {
				t.Logf("%s %s %v", entry.Level, entry.Message, entry.ContextMap())
			}
		}
	})

	config := Config{
		Beatconfig: map[string]any{
			"logging":                            map[string]any{"level": "debug", "selectors": []string{"*"}},
			"path.home":                          t.TempDir(),
			"management.agent.connection_secret": agent.connectionSecret,
		},
	}
	rec, err := factory.CreateLogs(t.Context(), settings, &config, sink)
	require.NoError(t, err)
	diagExt := &fakeDiagnosticExtension{hooks: map[string]func() []byte{}}
	host := &hostWithExtension{ext: diagExt}
	require.NoError(t, rec.Start(t.Context(), host))
	t.Cleanup(func() { require.NoError(t, rec.Shutdown(t.Context())) })

	monitorNames := func() map[string]bool {
		names := map[string]bool{}
		for _, logs := range sink.AllLogs() {
			for _, rl := range logs.ResourceLogs().All() {
				for _, sl := range rl.ScopeLogs().All() {
					for _, lr := range sl.LogRecords().All() {
						if monitor, ok := lr.Body().Map().Get("monitor"); ok {
							if name, ok := monitor.Map().Get("name"); ok {
								names[name.Str()] = true
							}
						}
					}
				}
			}
		}
		return names
	}

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		names := monitorNames()
		assert.True(c, names["monitor-a"], "no events from monitor-a")
		assert.True(c, names["monitor-b"], "no events from monitor-b")
	}, 90*time.Second, time.Second)

	// The invalid monitor doesn't stop the others, and is reported as failed
	// through the collector. The receiver itself has a recoverable error, as the
	// monitor can be fixed or removed.
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		evt := host.GetEvent()
		if !assert.NotNil(c, evt) {
			return
		}
		assert.Equal(c, componentstatus.StatusRecoverableError, evt.Status())
		status, msg := inputStatus(evt, "synthetics/tcp-monitor-invalid")
		assert.Equal(c, componentstatus.StatusPermanentError.String(), status)
		assert.Contains(c, msg, "schedule")
	}, 30*time.Second, 100*time.Millisecond)

	// The rendered config lists the monitors and the output, as for a beat
	// process, though the output never reaches the beater.
	rendered := diagExt.run(t, "beat-rendered-config.yml")
	assert.Contains(t, rendered, "monitor-a")
	assert.Contains(t, rendered, "monitor-b")
	assert.Contains(t, rendered, "pretty: true")

	agent.setUnits(unitsV2)
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		assert.True(c, monitorNames()["monitor-c"], "no events from monitor-c")
	}, 90*time.Second, time.Second)

	// Of the monitors that ran, only monitor-b was stopped; monitor-a kept
	// running across the change.
	stopped := zapLogs.FilterMessageSnippet("Stopping runner: Monitor<").All()
	require.Len(t, stopped, 1)
	assert.Contains(t, stopped[0].Message, "monitor-b")

	// Removing the invalid monitor clears its failure.
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		evt := host.GetEvent()
		if !assert.NotNil(c, evt) {
			return
		}
		assert.Equal(c, componentstatus.StatusOK, evt.Status())
		status, _ := inputStatus(evt, "synthetics/tcp-monitor-invalid")
		assert.Equal(c, componentstatus.StatusStopped.String(), status)
	}, 30*time.Second, 100*time.Millisecond)
	assert.Empty(t, host.rejectedChanges(), "the collector would reject these status changes")

	// Monitor status goes to the collector, keyed by monitor ID. Heartbeat names
	// a monitor after its input unless it comes from the Synthetics app.
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		evt := host.GetEvent()
		if !assert.NotNil(c, evt) {
			return
		}
		inputs, ok := evt.Attributes().Get("inputs")
		if !assert.True(c, ok, "no input status in %v", evt.Attributes().AsRaw()) {
			return
		}
		for _, id := range []string{"synthetics/tcp-monitor-a", "synthetics/tcp-monitor-c"} {
			input, ok := inputs.Map().Get(id)
			if assert.True(c, ok, "no status for %s in %v", id, inputs.Map().AsRaw()) {
				status, _ := input.Map().Get("status")
				assert.Equal(c, componentstatus.StatusOK.String(), status.Str())
			}
		}
	}, 30*time.Second, 100*time.Millisecond)
}

// fakeAgent serves the control protocol over mutual TLS on a unix socket, and
// sends the expected units whenever they change, as Elastic Agent does for
// receivers.
type fakeAgent struct {
	proto.UnimplementedElasticAgentServer

	connectionSecret string

	mx      sync.Mutex
	units   []*proto.UnitExpected
	changed chan struct{}
}

func newFakeAgent(t *testing.T) *fakeAgent {
	caKey, caCert, caPair, err := certutil.NewRootCA()
	require.NoError(t, err)
	serverCert, _, err := certutil.GenerateChildCert("localhost", nil, caKey, caCert)
	require.NoError(t, err)
	_, clientPair, err := certutil.GenerateChildCert("localhost", nil, caKey, caCert)
	require.NoError(t, err)

	pool := x509.NewCertPool()
	require.True(t, pool.AppendCertsFromPEM(caPair.Cert))
	server := grpc.NewServer(grpc.Creds(credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{*serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    pool,
		MinVersion:   tls.VersionTLS12,
	})))
	agent := &fakeAgent{changed: make(chan struct{})}
	proto.RegisterElasticAgentServer(server, agent)

	socket := filepath.Join(t.TempDir(), "units.sock")
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "unix", socket)
	require.NoError(t, err)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)

	info, err := gproto.Marshal(&proto.StartUpInfo{
		Addr:       "unix://" + socket,
		ServerName: "localhost",
		Token:      "token",
		CaCert:     caPair.Cert,
		PeerCert:   clientPair.Cert,
		PeerKey:    clientPair.Key,
		Services:   []proto.ConnInfoServices{proto.ConnInfoServices_CheckinV2},
		AgentInfo:  &proto.AgentInfo{Id: "agent-id", Version: "9.9.9"},
	})
	require.NoError(t, err)
	agent.connectionSecret = base64.StdEncoding.EncodeToString(info)
	return agent
}

func (a *fakeAgent) setUnits(units []*proto.UnitExpected) {
	a.mx.Lock()
	defer a.mx.Unlock()
	a.units = units
	close(a.changed)
	a.changed = make(chan struct{})
}

func (a *fakeAgent) CheckinV2(server proto.ElasticAgent_CheckinV2Server) error {
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

func (a *fakeAgent) Actions(server proto.ElasticAgent_ActionsServer) error {
	for {
		if _, err := server.Recv(); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}

// inputStatus returns the status and error the receiver reported for an input.
func inputStatus(evt *componentstatus.Event, id string) (string, string) {
	inputs, ok := evt.Attributes().Get("inputs")
	if !ok {
		return "", ""
	}
	input, ok := inputs.Map().Get(id)
	if !ok {
		return "", ""
	}
	status, _ := input.Map().Get("status")
	msg, _ := input.Map().Get("error")
	return status.Str(), msg.Str()
}

// hostWithExtension is a collectorHost with a diagnostics extension.
type hostWithExtension struct {
	collectorHost
	ext component.Component
}

func (h *hostWithExtension) GetExtensions() map[component.ID]component.Component {
	return map[component.ID]component.Component{component.MustNewID("elastic_diagnostics"): h.ext}
}

// fakeDiagnosticExtension records the diagnostic hooks registered with it.
type fakeDiagnosticExtension struct {
	component.StartFunc
	component.ShutdownFunc

	mx    sync.Mutex
	hooks map[string]func() []byte
}

func (e *fakeDiagnosticExtension) RegisterDiagnosticHook(_ string, _ string, filename string, _ string, hook func() []byte) {
	e.mx.Lock()
	defer e.mx.Unlock()
	e.hooks[filename] = hook
}

func (e *fakeDiagnosticExtension) run(t *testing.T, filename string) string {
	e.mx.Lock()
	hook, ok := e.hooks[filename]
	e.mx.Unlock()
	require.True(t, ok, "no diagnostic hook for %s", filename)
	return string(hook())
}

func startTCPTarget(t *testing.T) string {
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { listener.Close() })
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()
	return listener.Addr().String()
}

func outputUnit(idx uint64) *proto.UnitExpected {
	return &proto.UnitExpected{
		Id:             "output",
		Type:           proto.UnitType_OUTPUT,
		State:          proto.State_HEALTHY,
		ConfigStateIdx: idx,
		LogLevel:       proto.UnitLogLevel_DEBUG,
		Config: &proto.UnitExpectedConfig{
			Id:   "default",
			Type: "console",
			Name: "console",
			Source: &structpb.Struct{Fields: map[string]*structpb.Value{
				"type":   structpb.NewStringValue("console"),
				"pretty": structpb.NewBoolValue(true),
			}},
		},
	}
}

// monitorUnit builds an input unit for one synthetics TCP monitor, shaped like
// the units Elastic Agent sends for a Fleet synthetics policy.
func monitorUnit(t *testing.T, id string, host string, schedule string, idx uint64) *proto.UnitExpected {
	stream := map[string]any{
		"id":       id,
		"name":     id,
		"type":     "tcp",
		"hosts":    host,
		"schedule": schedule,
		"enabled":  true,
		"data_stream": map[string]any{
			"dataset": "tcp",
			"type":    "synthetics",
		},
	}
	streamSource, err := structpb.NewStruct(stream)
	require.NoError(t, err)
	source, err := structpb.NewStruct(map[string]any{
		"id":      "synthetics/tcp-" + id,
		"type":    "synthetics/tcp",
		"name":    id,
		"streams": []any{stream},
		"data_stream": map[string]any{
			"namespace": "default",
		},
	})
	require.NoError(t, err)
	return &proto.UnitExpected{
		Id:             "synthetics/tcp-" + id,
		Type:           proto.UnitType_INPUT,
		State:          proto.State_HEALTHY,
		ConfigStateIdx: idx,
		LogLevel:       proto.UnitLogLevel_DEBUG,
		Config: &proto.UnitExpectedConfig{
			Id:     "synthetics/tcp-" + id,
			Type:   "synthetics/tcp",
			Name:   id,
			Source: source,
			Streams: []*proto.Stream{{
				Id:     id,
				Source: streamSource,
			}},
		},
	}
}
