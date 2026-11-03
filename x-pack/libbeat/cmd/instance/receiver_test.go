// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package instance

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
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

	"github.com/elastic/beats/v7/filebeat/cmd"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/acker"
	"github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	"github.com/elastic/beats/v7/x-pack/otel/otelmanager"
	oteltelemetry "github.com/elastic/beats/v7/x-pack/otel/telemetry"
	conf "github.com/elastic/elastic-agent-libs/config"
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
	diagHook            func() []byte
	registeredActionFor string
	unregisteredFor     string
	actionHandler       func(ctx context.Context, params map[string]any) (map[string]any, error)
}

func (f *fakeActionDiagExtension) Start(context.Context, component.Host) error { return nil }
func (f *fakeActionDiagExtension) Shutdown(context.Context) error              { return nil }

func (f *fakeActionDiagExtension) RegisterDiagnosticHook(name, _, _, _ string, hook func() []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.registeredDiagName = name
	f.diagHook = hook
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

	// The per-receiver beat_metrics.json must still carry the process-wide
	// system metrics even though receivers do not register them themselves.
	ext.mu.Lock()
	hook := ext.diagHook
	ext.mu.Unlock()
	require.NotNil(t, hook, "diagnostic hook should have been registered")
	var snapshot map[string]any
	require.NoError(t, json.Unmarshal(hook(), &snapshot), "hook output should be valid JSON")
	beatNS, ok := snapshot["beat"].(map[string]any)
	require.True(t, ok, "beat_metrics.json should have a beat namespace, got %v", snapshot)
	for _, k := range []string{"memstats", "cpu", "runtime", "info"} {
		assert.Contains(t, beatNS, k, "beat_metrics.json should contain beat.%s", k)
	}
	assert.Contains(t, snapshot, "system", "beat_metrics.json should contain the system namespace")
	// The shared system bridge reports itself as "beat" with no version.
	info, _ := beatNS["info"].(map[string]any)
	assert.Equal(t, b.Info.Version, info["version"], "beat.info must be the receiver's own, not the shared bridge's")
	assert.NotEqual(t, "beat", info["name"], "beat.info must be the receiver's own, not the shared bridge's")

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

func TestReceiverSystemMetrics(t *testing.T) {
	release, err := oteltelemetry.AcquireSystemBridge(componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)
	defer release()

	system := receiverSystemMetrics()
	beatNS, ok := system["beat"].(map[string]any)
	require.True(t, ok, "system metrics should have a beat namespace, got %v", system)
	assert.Contains(t, beatNS, "memstats", "beat.memstats")
	assert.Contains(t, system, "system", "system namespace")
}

// TestReceiverHTTPStatsIncludesSystemMetrics checks that a receiver with
// http.enabled still serves the system metrics on /stats even though it does not
// register them in its own registry.
func TestReceiverHTTPStatsIncludesSystemMetrics(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a unix socket")
	}
	// Unix socket paths are short; t.TempDir is too long on macOS.
	dir, err := os.MkdirTemp("/tmp", "hb")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	socket := filepath.Join(dir, "s.sock")

	mb := &mockReceiverBeater{acked: &atomic.Int64{}, initDone: make(chan struct{}), done: make(chan struct{})}
	creator := func(*beat.Beat, *conf.C) (beat.Beater, error) { return mb, nil }
	defer management.SetUnderAgent(false)
	b, err := NewBeatForReceiver(cmd.FilebeatSettings("filebeat"), map[string]any{
		"path.home":               t.TempDir(),
		"management.otel.enabled": true,
		"http.enabled":            true,
		"http.host":               "unix://" + socket,
	}, consumertest.NewNop(), "test-receiver", zapcore.NewNopCore())
	require.NoError(t, err)

	var rs receiver.Settings
	rs.Logger = zap.NewNop()
	rs.ID = component.NewIDWithName(component.MustNewType("mockbeatreceiver"), "r1")
	br, err := NewBeatReceiver(t.Context(), b, creator, rs)
	require.NoError(t, err)
	go func() { _ = br.Start(&fakeExtensionHost{}) }()
	defer func() { _ = br.Shutdown(t.Context()) }()

	client := &http.Client{Transport: &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socket)
		},
	}}
	resp, err := client.Get("http://unix/stats")
	require.NoError(t, err, "GET /stats")
	defer resp.Body.Close()
	var stats map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&stats), "decode /stats")

	beatNS, ok := stats["beat"].(map[string]any)
	require.True(t, ok, "/stats should have a beat namespace, got %v", stats)
	for _, k := range []string{"memstats", "cpu", "runtime", "info"} {
		assert.Contains(t, beatNS, k, "/stats should contain beat.%s", k)
	}
	assert.Contains(t, stats, "system", "/stats should contain the system namespace")
	info, _ := beatNS["info"].(map[string]any)
	assert.NotEqual(t, "beat", info["name"], "beat.info must be the receiver's own")
}
