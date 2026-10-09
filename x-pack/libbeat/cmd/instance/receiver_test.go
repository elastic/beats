// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package instance

import (
	"context"
	"errors"
	"maps"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc/credentials"

	"github.com/elastic/beats/v7/filebeat/cmd"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/acker"
	"github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	"github.com/elastic/beats/v7/x-pack/otel/otelmanager"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// mockReceiverBeater is a minimal Beater that publishes a fixed number of
// events through the receiver's publisher pipeline and blocks until Stop. On
// stop it closes its client (stage one of the two-stage shutdown), modeling a
// well-behaved Beater that owns its inputs' shutdown (issue #49794).
type mockReceiverBeater struct {
	npub     int
	acked    *atomic.Int64
	initDone chan struct{}
	done     chan struct{}
	stopOnce sync.Once
}

func (m *mockReceiverBeater) Run(b *beat.Beat) error {
	client, err := b.Publisher.ConnectWith(beat.ClientConfig{
		EventListener: acker.RawCounting(func(n int) { m.acked.Add(int64(n)) }),
	})
	if err != nil {
		return err
	}

	for i := 0; i < m.npub; i++ {
		client.Publish(beat.Event{
			Timestamp: time.Now(),
			Fields:    mapstr.M{"n": i},
		})
	}
	close(m.initDone)

	<-m.done
	// Beater owns shutdown sequencing: close the client before Run returns so
	// the pipeline can drain and finalize acknowledgments on Disconnect.
	_ = client.Close()
	return nil
}

func (m *mockReceiverBeater) Stop() {
	m.stopOnce.Do(func() { close(m.done) })
}

// TestBeatReceiverStartShutdown exercises the full beat-receiver lifecycle end
// to end: it builds a real BeatReceiver backed by the slabqueue-pool publisher
// pipeline (NewForReceiver), starts it, publishes events, and shuts it down. It
// verifies that:
//   - Shutdown completes promptly (it is bounded by receiverPublisherCloseTimeout,
//     so it never hangs even though the output drains during disconnect), and
//   - every published event is acknowledged by the time Shutdown returns, which
//     proves the output stays running and drains acks while the pipeline is
//     being disconnected (issues #50104, #50105, #49794).
func TestBeatReceiverStartShutdown(t *testing.T) {
	const npub = 5
	acked := &atomic.Int64{}
	mb := &mockReceiverBeater{
		npub:     npub,
		acked:    acked,
		initDone: make(chan struct{}),
		done:     make(chan struct{}),
	}
	creator := func(*beat.Beat, *conf.C) (beat.Beater, error) { return mb, nil }

	cfg := map[string]any{"path.home": t.TempDir()}
	b, err := NewBeatForReceiver(
		cmd.FilebeatSettings("filebeat"),
		cfg,
		consumertest.NewNop(), // accepts every batch -> events get acknowledged
		"test-receiver",
		zapcore.NewNopCore(),
	)
	require.NoError(t, err, "building the receiver beat should succeed")

	var rs receiver.Settings
	rs.Logger = zap.NewNop()
	rs.ID = component.NewIDWithName(component.MustNewType("mockbeatreceiver"), "r1")

	br, err := NewBeatReceiver(t.Context(), b, creator, rs)
	require.NoError(t, err, "creating the beat receiver should succeed")

	// Start blocks in beater.Run, so run it in a goroutine.
	startErr := make(chan error, 1)
	go func() { startErr <- br.Start(componenttest.NewNopHost()) }()

	// Wait until the beater is running and has published its events.
	select {
	case <-mb.initDone:
	case <-time.After(30 * time.Second):
		t.Fatal("beater did not start")
	}

	// Shutdown must complete promptly. If the output were torn down before the
	// queue drained, this would block for the full close timeout (or forever);
	// the timeout guard here catches a hang.
	shutdownDone := make(chan error, 1)
	go func() { shutdownDone <- br.Shutdown(t.Context()) }()
	select {
	case err := <-shutdownDone:
		require.NoError(t, err, "Shutdown should not error")
	case <-time.After(30 * time.Second):
		t.Fatal("Shutdown hung — the output is likely not draining acknowledgments during disconnect")
	}

	// beater.Run (and therefore Start) must have returned after Stop.
	select {
	case err := <-startErr:
		require.NoError(t, err, "beater.Run should return cleanly")
	case <-time.After(10 * time.Second):
		t.Fatal("beater.Run did not return after Stop")
	}

	// Every published event must have been acknowledged by the time Shutdown
	// returned: this is the key end-to-end assertion that the output kept
	// consuming and acking while the pipeline was disconnected.
	assert.Equal(t, int64(npub), acked.Load(),
		"all published events must be acknowledged by the time Shutdown returns")
}

// fakeActionDiagExtension implements both otelmanager.DiagnosticExtension and
// otelmanager.ActionExtension, modeling elastic-agent's elasticdiagnostics
// extension for the purposes of testing that BeatReceiver.Start wires both
// into the beat's manager.
type fakeActionDiagExtension struct {
	mu                  sync.Mutex
	registeredDiagName  string
	registeredActionFor string
	unregisteredFor     string
	actionHandler       func(ctx context.Context, params map[string]any) (map[string]any, error)
}

func (f *fakeActionDiagExtension) Start(context.Context, component.Host) error { return nil }
func (f *fakeActionDiagExtension) Shutdown(context.Context) error              { return nil }

func (f *fakeActionDiagExtension) RegisterDiagnosticHook(name, _, _, _ string, _ func() []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.registeredDiagName = name
}

func (f *fakeActionDiagExtension) RegisterActionHandler(name string, handler func(ctx context.Context, params map[string]any) (map[string]any, error)) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.registeredActionFor = name
	f.actionHandler = handler
	return nil
}

func (f *fakeActionDiagExtension) UnregisterActionHandler(name string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.unregisteredFor = name
	f.actionHandler = nil
}

// fakeExtensionHost is a component.Host exposing a fixed set of extensions.
type fakeExtensionHost struct {
	extensions map[component.ID]component.Component
}

func (h *fakeExtensionHost) GetExtensions() map[component.ID]component.Component {
	return h.extensions
}

// fakeAction implements management.Action for exercising OtelManager.RegisterAction.
type fakeAction struct {
	name     string
	executed atomic.Bool
}

func (a *fakeAction) Name() string { return a.name }

func (a *fakeAction) Execute(_ context.Context, _ map[string]any) (map[string]any, error) {
	a.executed.Store(true)
	return map[string]any{"ok": true}, nil
}

// TestBeatReceiverStart_WiresActionAndDiagnosticExtensions verifies that Start
// discovers an extension implementing otelmanager.DiagnosticExtension and
// otelmanager.ActionExtension on the collector host and wires both into the
// beat's OtelManager, so that Fleet actions (e.g. osquery live queries) routed
// to elastic-agent can reach this receiver instance.
func TestBeatReceiverStart_WiresActionAndDiagnosticExtensions(t *testing.T) {
	mb := &mockReceiverBeater{
		npub:     0,
		acked:    &atomic.Int64{},
		initDone: make(chan struct{}),
		done:     make(chan struct{}),
	}
	creator := func(*beat.Beat, *conf.C) (beat.Beater, error) { return mb, nil }

	cfg := map[string]any{
		"path.home":               t.TempDir(),
		"management.otel.enabled": true,
	}
	defer management.SetUnderAgent(false) // reset global state set by NewBeatForReceiver
	b, err := NewBeatForReceiver(
		cmd.FilebeatSettings("filebeat"),
		cfg,
		consumertest.NewNop(),
		"test-receiver",
		zapcore.NewNopCore(),
	)
	require.NoError(t, err, "building the receiver beat should succeed")

	// With management.otel.enabled, NewBeatForReceiver's manager factory produces
	// an *otelmanager.OtelManager.
	require.IsType(t, &otelmanager.OtelManager{}, b.Manager)

	var rs receiver.Settings
	rs.Logger = zap.NewNop()
	rs.ID = component.NewIDWithName(component.MustNewType("mockbeatreceiver"), "r1")

	br, err := NewBeatReceiver(t.Context(), b, creator, rs)
	require.NoError(t, err, "creating the beat receiver should succeed")

	ext := &fakeActionDiagExtension{}
	host := &fakeExtensionHost{extensions: map[component.ID]component.Component{
		component.MustNewID("elastic_diagnostics"): ext,
	}}

	startErr := make(chan error, 1)
	go func() { startErr <- br.Start(host) }()

	select {
	case <-mb.initDone:
	case <-time.After(30 * time.Second):
		t.Fatal("beater did not start")
	}

	// The diagnostic hook is registered eagerly by Start itself.
	ext.mu.Lock()
	assert.Equal(t, "test-receiver", ext.registeredDiagName, "diagnostic hook should be registered under the receiver's component ID")
	ext.mu.Unlock()

	// The action extension is only set on the manager by Start; the actual
	// handler is registered once something (e.g. osquerybeat) calls
	// Manager.RegisterAction, which OtelManager forwards to the extension.
	act := &fakeAction{name: "osquery"}
	b.Manager.RegisterAction(act)

	ext.mu.Lock()
	assert.Equal(t, "test-receiver", ext.registeredActionFor, "action handler should be registered under the receiver's component ID")
	handler := ext.actionHandler
	ext.mu.Unlock()
	require.NotNil(t, handler, "action handler should have been registered with the extension")

	res, err := handler(t.Context(), map[string]any{"id": "abc"})
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"ok": true}, res)
	assert.True(t, act.executed.Load(), "invoking the registered handler should execute the underlying action")

	b.Manager.UnregisterAction(act)
	ext.mu.Lock()
	assert.Equal(t, "test-receiver", ext.unregisteredFor)
	assert.Nil(t, ext.actionHandler)
	ext.mu.Unlock()

	shutdownDone := make(chan error, 1)
	go func() { shutdownDone <- br.Shutdown(t.Context()) }()
	select {
	case err := <-shutdownDone:
		require.NoError(t, err, "Shutdown should not error")
	case <-time.After(30 * time.Second):
		t.Fatal("Shutdown hung")
	}

	select {
	case err := <-startErr:
		require.NoError(t, err, "beater.Run should return cleanly")
	case <-time.After(10 * time.Second):
		t.Fatal("beater.Run did not return after Stop")
	}
}

// mockStorageBeater is a minimal Beater that also implements
// backend.WithESStateStoreExtension so that BeatReceiver.Start's storage
// preflight path is exercised.
type mockStorageBeater struct {
	mockReceiverBeater
}

func (m *mockStorageBeater) WithESStateStoreExtension(_ backend.Registry) {}

// TestBeatReceiverStartFailureShutdownDoesNotHang is a regression test for the
// nil-runDone hang.
func TestBeatReceiverStartFailureShutdownDoesNotHang(t *testing.T) {
	mb := &mockStorageBeater{
		mockReceiverBeater: mockReceiverBeater{
			acked:    &atomic.Int64{},
			initDone: make(chan struct{}),
			done:     make(chan struct{}),
		},
	}
	creator := func(*beat.Beat, *conf.C) (beat.Beater, error) { return mb, nil }

	cfg := map[string]any{
		"path.home": t.TempDir(),
		// Reference a storage extension that will not be present in the host.
		// This causes BeatReceiver.Start to return an error before launching
		// beater.Run, leaving runDone nil on the buggy path.
		"storage": "elasticsearch_storage/missing",
	}
	b, err := NewBeatForReceiver(
		cmd.FilebeatSettings("filebeat"),
		cfg,
		consumertest.NewNop(),
		"test-receiver",
		zapcore.NewNopCore(),
	)
	require.NoError(t, err, "building the receiver beat should succeed")

	var rs receiver.Settings
	rs.Logger = zap.NewNop()
	rs.ID = component.NewIDWithName(component.MustNewType("mockbeatreceiver"), "r1")

	br, err := NewBeatReceiver(t.Context(), b, creator, rs)
	require.NoError(t, err, "creating the beat receiver should succeed")

	// Reproduce the async wrapper pattern used by all beat receivers.
	var wg sync.WaitGroup
	wg.Go(func() {
		// Swallow the error, exactly as the wrapper receivers do.
		_ = br.Start(componenttest.NewNopHost())
	})
	// Wait for the goroutine to complete. Start has failed and returned; on the
	// buggy path runDone is still nil at this point.
	wg.Wait()

	// Shutdown must complete promptly even though Start failed before launching
	// beater.Run. Use t.Context() (no deadline during test execution) so that a
	// nil runDone would block indefinitely — a bounded context would mask the
	// bug by releasing the select via ctx.Done().
	done := make(chan error, 1)
	go func() { done <- br.Shutdown(t.Context()) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Shutdown hung after BeatReceiver.Start failed — nil runDone not fixed")
	}
}

// fakeHeadersSetter models the contrib headers_setter extension: a component
// implementing extensionauth.GRPCClient whose PerRPCCredentials resolve to a
// fixed set of headers.
type fakeHeadersSetter struct {
	headers  map[string]string
	credsErr error // returned by PerRPCCredentials
	metaErr  error // returned by GetRequestMetadata
}

func (f *fakeHeadersSetter) Start(context.Context, component.Host) error { return nil }
func (f *fakeHeadersSetter) Shutdown(context.Context) error              { return nil }

func (f *fakeHeadersSetter) PerRPCCredentials() (credentials.PerRPCCredentials, error) {
	if f.credsErr != nil {
		return nil, f.credsErr
	}
	return fakePerRPC{headers: f.headers, err: f.metaErr}, nil
}

type fakePerRPC struct {
	headers map[string]string
	err     error
}

func (f fakePerRPC) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return maps.Clone(f.headers), nil
}

func (fakePerRPC) RequireTransportSecurity() bool { return false }

// plainExtension does not implement extensionauth.GRPCClient.
type plainExtension struct{}

func (plainExtension) Start(context.Context, component.Host) error { return nil }
func (plainExtension) Shutdown(context.Context) error              { return nil }

// agentHeadersSetterID returns an ID shaped like the ones Elastic Agent
// generates, e.g. "headers_setter/_agent-component/default".
func agentHeadersSetterID(name string) component.ID {
	return component.NewIDWithName(component.MustNewType(headersSetterType), agentComponentPrefix+name)
}

func TestHeadersFromExtensions(t *testing.T) {
	const ua = "Elastic-Filebeat/9.4.0 (linux; amd64; Managed; Privileged)"

	tests := []struct {
		name       string
		extensions map[component.ID]component.Component
		want       map[string]string
	}{
		{
			name:       "no extensions",
			extensions: map[component.ID]component.Component{},
			want:       nil,
		},
		{
			name: "other extension types are ignored even if they implement GRPCClient",
			extensions: map[component.ID]component.Component{
				component.MustNewIDWithName("beatsauth", agentComponentPrefix+"default"): &fakeHeadersSetter{headers: map[string]string{"User-Agent": ua}},
			},
			want: nil,
		},
		{
			name: "headers_setter not generated by Elastic Agent is ignored",
			extensions: map[component.ID]component.Component{
				component.MustNewID(headersSetterType):                   &fakeHeadersSetter{headers: map[string]string{"User-Agent": "user"}},
				component.MustNewIDWithName(headersSetterType, "custom"): &fakeHeadersSetter{headers: map[string]string{"User-Agent": "user"}},
			},
			want: nil,
		},
		{
			name:       "headers_setter without GRPCClient is skipped",
			extensions: map[component.ID]component.Component{agentHeadersSetterID("default"): plainExtension{}},
			want:       nil,
		},
		{
			name: "all configured headers are returned",
			extensions: map[component.ID]component.Component{
				agentHeadersSetterID("default"): &fakeHeadersSetter{headers: map[string]string{"User-Agent": ua, "X-Elastic-Team": "obs"}},
			},
			want: map[string]string{"User-Agent": ua, "X-Elastic-Team": "obs"},
		},
		{
			name: "PerRPCCredentials error yields no headers",
			extensions: map[component.ID]component.Component{
				agentHeadersSetterID("default"): &fakeHeadersSetter{credsErr: errors.New("additional_auth missing")},
			},
			want: nil,
		},
		{
			name: "GetRequestMetadata error yields no headers",
			extensions: map[component.ID]component.Component{
				agentHeadersSetterID("default"): &fakeHeadersSetter{metaErr: errors.New("value_file unreadable")},
			},
			want: nil,
		},
		{
			name: "multiple headers_setter: first by ID is used",
			extensions: map[component.ID]component.Component{
				agentHeadersSetterID("b"): &fakeHeadersSetter{headers: map[string]string{"User-Agent": "from-b"}},
				agentHeadersSetterID("a"): &fakeHeadersSetter{headers: map[string]string{"User-Agent": "from-a"}},
			},
			want: map[string]string{"User-Agent": "from-a"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := headersFromExtensions(t.Context(), tc.extensions, logptest.NewTestingLogger(t, ""))
			assert.Equal(t, tc.want, got, "unexpected headers read from extensions")
		})
	}
}

// TestBeatReceiverStart_UsesHeadersSetterUserAgent checks that Start replaces
// beat.Info.UserAgent with the one from the headers_setter extension before
// the beater runs, which is where inputs and modules copy it from.
func TestBeatReceiverStart_UsesHeadersSetterUserAgent(t *testing.T) {
	const ua = "Elastic-Filebeat/9.4.0 (linux; amd64; Managed; Privileged)"

	mb := &mockReceiverBeater{
		acked:    &atomic.Int64{},
		initDone: make(chan struct{}),
		done:     make(chan struct{}),
	}
	creator := func(*beat.Beat, *conf.C) (beat.Beater, error) { return mb, nil }

	b, err := NewBeatForReceiver(
		cmd.FilebeatSettings("filebeat"),
		map[string]any{"path.home": t.TempDir()},
		consumertest.NewNop(),
		"test-receiver",
		zapcore.NewNopCore(),
	)
	require.NoError(t, err, "building the receiver beat should succeed")
	require.NotEqual(t, ua, b.Info.UserAgent, "test precondition: the extension UA must differ from the generated one")

	var rs receiver.Settings
	rs.Logger = zap.NewNop()
	rs.ID = component.NewIDWithName(component.MustNewType("mockbeatreceiver"), "r1")

	br, err := NewBeatReceiver(t.Context(), b, creator, rs)
	require.NoError(t, err, "creating the beat receiver should succeed")

	host := &fakeExtensionHost{extensions: map[component.ID]component.Component{
		agentHeadersSetterID("default"): &fakeHeadersSetter{headers: map[string]string{"User-Agent": ua}},
	}}

	startErr := make(chan error, 1)
	go func() { startErr <- br.Start(host) }()

	select {
	case <-mb.initDone:
	case <-time.After(30 * time.Second):
		t.Fatal("beater did not start")
	}
	assert.Equal(t, ua, b.Info.UserAgent, "beat.Info.UserAgent should be the extension's User-Agent, not the generated one")

	shutdownDone := make(chan error, 1)
	go func() { shutdownDone <- br.Shutdown(t.Context()) }()
	select {
	case err := <-shutdownDone:
		require.NoError(t, err, "Shutdown should not error")
	case <-time.After(30 * time.Second):
		t.Fatal("Shutdown hung")
	}
	select {
	case err := <-startErr:
		require.NoError(t, err, "beater.Run should return cleanly")
	case <-time.After(10 * time.Second):
		t.Fatal("beater.Run did not return after Stop")
	}
}
