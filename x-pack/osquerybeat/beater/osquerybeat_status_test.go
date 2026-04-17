// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/beatmonitoring"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/beats/v7/libbeat/management/status"
	agentconfig "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/osqd"
)

// Mock osqueryd implementation for testing
type mockOsqueryd struct {
	checkErr error
	runErr   error
	runDelay time.Duration
}

func (m *mockOsqueryd) Check(ctx context.Context) error {
	return m.checkErr
}

func (m *mockOsqueryd) Run(ctx context.Context, flags osqd.Flags) error {
	if m.runDelay > 0 {
		select {
		case <-time.After(m.runDelay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return m.runErr
}

func (m *mockOsqueryd) SocketPath() string {
	return "/tmp/test-osquery.sock"
}

func (m *mockOsqueryd) DataPath() string {
	return "/tmp/test-osquery-data"
}

type statusEvent struct {
	Status  status.Status
	Message string
}

type testManager struct {
	events   []statusEvent
	started  bool
	stopped  bool
	startErr error
	diagHook map[string]management.DiagnosticHook
	mx       sync.Mutex
}

type diagnosticsQueryExecutor struct {
	rows []map[string]interface{}
	err  error
}

func (e *diagnosticsQueryExecutor) Query(context.Context, string, time.Duration) ([]map[string]interface{}, error) {
	return e.rows, e.err
}

func (m *testManager) UpdateStatus(s status.Status, msg string) {
	m.mx.Lock()
	defer m.mx.Unlock()
	m.events = append(m.events, statusEvent{Status: s, Message: msg})
}
func (m *testManager) Start() error {
	m.started = true
	return m.startErr
}
func (m *testManager) PreInit() error                      { return nil }
func (m *testManager) PostInit()                           {}
func (m *testManager) Stop()                               { m.stopped = true }
func (m *testManager) SetPayload(map[string]any)           {}
func (m *testManager) Enabled() bool                       { return true }
func (m *testManager) AgentInfo() management.AgentInfo     { return management.AgentInfo{} }
func (m *testManager) SetStopCallback(func())              {}
func (m *testManager) CheckRawConfig(*agentconfig.C) error { return nil }
func (m *testManager) RegisterAction(management.Action)    {}
func (m *testManager) UnregisterAction(management.Action)  {}
func (m *testManager) RegisterDiagnosticHook(name, _ string, _ string, _ string, hook management.DiagnosticHook) {
	if m.diagHook == nil {
		m.diagHook = make(map[string]management.DiagnosticHook)
	}
	m.diagHook[name] = hook
}

func newStatusTestBeater(t *testing.T, overrides ...func(*osquerybeat)) (*osquerybeat, *beat.Beat, *testManager) {
	t.Helper()

	mgr := &testManager{}
	b := &beat.Beat{
		Manager:    mgr,
		Registry:   reload.NewRegistry(),
		Monitoring: beatmonitoring.NewMonitoring(),
	}
	b.Info.Paths = newTestBeatPaths(t)

	cfg := agentconfig.NewConfig()
	beater, err := New(b, cfg)
	require.NoError(t, err)

	ob, ok := beater.(*osquerybeat)
	require.True(t, ok)
	for _, override := range overrides {
		override(ob)
	}
	return ob, b, mgr
}

// TestOsquerybeatStatusReporting_Lifecycle tests the full lifecycle status reporting
// when osqueryd is available and runs successfully.
func TestOsquerybeatStatusReporting_Lifecycle(t *testing.T) {
	mgr := &testManager{}
	b := &beat.Beat{
		Manager:    mgr,
		Registry:   reload.NewRegistry(),
		Monitoring: beatmonitoring.NewMonitoring(),
	}
	b.Info.Paths = newTestBeatPaths(t)

	cfg := agentconfig.NewConfig()
	beater, err := New(b, cfg)
	require.NoError(t, err)

	// Inject mock osqueryd that simulates successful startup
	ob, ok := beater.(*osquerybeat)
	require.True(t, ok)
	ob.osquerydFactory = func(socketPath string, opts ...osqd.Option) (osqd.Runner, error) {
		return &mockOsqueryd{
			checkErr: nil,                    // Check succeeds
			runErr:   context.Canceled,       // Run until cancelled
			runDelay: 100 * time.Millisecond, // Simulate startup delay
		}, nil
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- beater.Run(b)
	}()

	// Wait for osqueryd to reach Running state
	assert.Eventually(t, func() bool {
		mgr.mx.Lock()
		defer mgr.mx.Unlock()
		for _, event := range mgr.events {
			if event.Status == status.Running {
				return true
			}
		}
		return false
	}, 5*time.Second, 50*time.Millisecond, "should reach Running status")

	t.Log("Reached Running state, stopping beat...")

	// Stop the beat
	beater.Stop()

	// Wait for Run() to complete
	runCompleted := false
	var runErr error
	assert.Eventually(t, func() bool {
		select {
		case runErr = <-errCh:
			runCompleted = true
			return true
		default:
			return false
		}
	}, 5*time.Second, 50*time.Millisecond, "Run() should complete")

	require.True(t, runCompleted, "Run() should have completed")
	t.Logf("Run completed with error: %v", runErr)

	// Wait for final status to be recorded
	assert.Eventually(t, func() bool {
		mgr.mx.Lock()
		defer mgr.mx.Unlock()
		if len(mgr.events) == 0 {
			return false
		}
		lastEvent := mgr.events[len(mgr.events)-1]
		return lastEvent.Status == status.Stopped || lastEvent.Status == status.Stopping
	}, 1*time.Second, 50*time.Millisecond, "should have final status")

	// Log all events
	mgr.mx.Lock()
	for i, event := range mgr.events {
		t.Logf("Event %d: Status=%v, Message=%s", i, event.Status, event.Message)
	}
	eventCount := len(mgr.events)
	events := make([]statusEvent, len(mgr.events))
	copy(events, mgr.events)
	mgr.mx.Unlock()

	// Validate lifecycle: Configuring -> Running -> Stopping -> Stopped
	require.GreaterOrEqual(t, eventCount, 2, "should have at least Configuring and Stopped")

	// First status should be Configuring
	assert.Equal(t, status.Configuring, events[0].Status, "first status should be Configuring")

	// Should have Running status at some point
	hasRunning := false
	for _, event := range events {
		if event.Status == status.Running {
			hasRunning = true
			break
		}
	}
	assert.True(t, hasRunning, "should have Running status")

	// Last status should be Stopped
	lastEvent := events[eventCount-1]
	assert.Equal(t, status.Stopped, lastEvent.Status, "last status should be Stopped")
}

func newTestBeatPaths(t *testing.T) *paths.Path {
	t.Helper()
	root := t.TempDir()
	p := paths.New()
	if err := p.InitPaths(&paths.Path{
		Home:   root,
		Config: root,
		Data:   root,
		Logs:   root,
	}); err != nil {
		t.Fatalf("failed to init beat paths: %v", err)
	}
	return p
}

// TestOsquerybeatStatusReporting_CheckFailure tests status reporting when osqueryd check fails.
func TestOsquerybeatStatusReporting_CheckFailure(t *testing.T) {
	mgr := &testManager{}
	b := &beat.Beat{
		Manager:    mgr,
		Registry:   reload.NewRegistry(),
		Monitoring: beatmonitoring.NewMonitoring(),
	}
	b.Info.Paths = newTestBeatPaths(t)

	cfg := agentconfig.NewConfig()
	beater, err := New(b, cfg)
	require.NoError(t, err)

	// Inject mock osqueryd that fails the check
	ob, ok := beater.(*osquerybeat)
	require.True(t, ok)
	ob.osquerydFactory = func(socketPath string, opts ...osqd.Option) (osqd.Runner, error) {
		return &mockOsqueryd{
			checkErr: assert.AnError, // Check fails
		}, nil
	}

	err = beater.Run(b)
	require.Error(t, err)

	// Should have Failed status
	mgr.mx.Lock()
	defer mgr.mx.Unlock()

	require.GreaterOrEqual(t, len(mgr.events), 1, "should have at least one status event")

	// Last event should be Failed
	lastEvent := mgr.events[len(mgr.events)-1]
	assert.Equal(t, status.Failed, lastEvent.Status, "should report Failed status on check failure")
	assert.Contains(t, lastEvent.Message, "Failed to check osqueryd")
}

// TestOsquerybeatStatusReporting_CreateOsquerydFailure tests status reporting when osqueryd creation fails.
func TestOsquerybeatStatusReporting_CreateOsquerydFailure(t *testing.T) {
	mgr := &testManager{}
	b := &beat.Beat{
		Manager:    mgr,
		Registry:   reload.NewRegistry(),
		Monitoring: beatmonitoring.NewMonitoring(),
	}
	b.Info.Paths = newTestBeatPaths(t)

	cfg := agentconfig.NewConfig()
	beater, err := New(b, cfg)
	require.NoError(t, err)

	// Inject factory that fails to create osqueryd
	ob, ok := beater.(*osquerybeat)
	require.True(t, ok)
	ob.osquerydFactory = func(socketPath string, opts ...osqd.Option) (osqd.Runner, error) {
		return nil, assert.AnError // Factory fails
	}

	err = beater.Run(b)
	require.Error(t, err)

	// Should have Failed status
	mgr.mx.Lock()
	defer mgr.mx.Unlock()

	require.GreaterOrEqual(t, len(mgr.events), 1, "should have at least one status event")

	// Last event should be Failed
	lastEvent := mgr.events[len(mgr.events)-1]
	assert.Equal(t, status.Failed, lastEvent.Status, "should report Failed status on creation failure")
	assert.Contains(t, lastEvent.Message, "Failed to create osqueryd")
}

// TestOsquerybeatStatusReporting_ManagerStartFailure tests status reporting when manager start fails.
func TestOsquerybeatStatusReporting_ManagerStartFailure(t *testing.T) {
	mgr := &testManager{
		startErr: assert.AnError, // Manager.Start() will fail
	}
	b := &beat.Beat{
		Manager:    mgr,
		Registry:   reload.NewRegistry(),
		Monitoring: beatmonitoring.NewMonitoring(),
	}
	b.Info.Paths = newTestBeatPaths(t)

	cfg := agentconfig.NewConfig()
	beater, err := New(b, cfg)
	require.NoError(t, err)

	// Inject mock osqueryd that works fine
	ob, ok := beater.(*osquerybeat)
	require.True(t, ok)
	ob.osquerydFactory = func(socketPath string, opts ...osqd.Option) (osqd.Runner, error) {
		return &mockOsqueryd{
			checkErr: nil,
		}, nil
	}

	err = beater.Run(b)
	require.Error(t, err)

	// Should have Failed status
	mgr.mx.Lock()
	defer mgr.mx.Unlock()

	require.GreaterOrEqual(t, len(mgr.events), 1, "should have at least one status event")

	// Last event should be Failed
	lastEvent := mgr.events[len(mgr.events)-1]
	assert.Equal(t, status.Failed, lastEvent.Status, "should report Failed status on manager start failure")
	assert.Contains(t, lastEvent.Message, "Failed to start manager")
}

func TestOsquerybeatRegistersScheduledProfilesDiagnostics(t *testing.T) {
	mgr := &testManager{}
	b := &beat.Beat{Manager: mgr}
	ob := &osquerybeat{
		qp: newQueryProfiler(logp.NewLogger("test")),
	}
	ob.setDiagnosticsQueryExecutor(&diagnosticsQueryExecutor{
		rows: []map[string]interface{}{
			{
				"name":              "pack_test_query",
				"query":             "select * from users limit 1",
				"executions":        int64(3),
				"last_executed":     int64(1730000000),
				"output_size":       int64(900),
				"wall_time_ms":      int64(120),
				"last_wall_time_ms": int64(40),
				"user_time":         int64(30),
				"last_user_time":    int64(10),
				"system_time":       int64(6),
				"last_system_time":  int64(2),
				"average_memory":    int64(5000),
				"last_memory":       int64(6000),
			},
		},
	})

	ob.registerDiagnosticHooks(b)

	hook, ok := mgr.diagHook["scheduled_query_profiles"]
	require.True(t, ok, "expected scheduled profiles diagnostics hook")

	var payload map[string]interface{}
	err := json.Unmarshal(hook(), &payload)
	require.NoError(t, err)

	count, ok := payload["count"].(float64)
	require.True(t, ok)
	//nolint:testifylint // We're comparing integers from a JSON
	assert.Equal(t, float64(1), count)

	profiles, ok := payload["osquery_schedule"].([]interface{})
	require.True(t, ok)
	require.Len(t, profiles, 1)

	p0, ok := profiles[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "select * from users limit 1", p0["query"])

	liveCount, ok := payload["live_query_profiles_count"].(float64)
	require.True(t, ok)
	//nolint:testifylint // We're comparing integers from a JSON
	assert.Equal(t, float64(0), liveCount)

	liveProfiles, ok := payload["live_query_profiles"].([]interface{})
	require.True(t, ok)
	assert.Empty(t, liveProfiles)
}

// TestOsquerybeatStatusReporting_RuntimeResolutionFailure tests status reporting
// when custom osquery runtime resolution fails before osqueryd runner creation.
func TestOsquerybeatStatusReporting_RuntimeResolutionFailure(t *testing.T) {
	ob, b, mgr := newStatusTestBeater(t, func(ob *osquerybeat) {
		platformCfg := &config.InstallPlatformConfig{
			AMD64: &config.InstallArtifactConfig{
				ArtifactURL: "https://example.org/osquery.tar.gz",
				SHA256:      "bad",
			},
			ARM64: &config.InstallArtifactConfig{
				ArtifactURL: "https://example.org/osquery.tar.gz",
				SHA256:      "bad",
			},
		}
		ob.osqueryInstallConfig = config.InstallConfig{
			Linux:   platformCfg,
			Darwin:  platformCfg,
			Windows: platformCfg,
		}
	})

	err := ob.Run(b)
	require.Error(t, err)

	mgr.mx.Lock()
	defer mgr.mx.Unlock()

	require.GreaterOrEqual(t, len(mgr.events), 1, "should have at least one status event")
	lastEvent := mgr.events[len(mgr.events)-1]
	assert.Equal(t, status.Failed, lastEvent.Status, "should report Failed status on runtime resolution failure")
	assert.Contains(t, lastEvent.Message, "Failed to resolve osquery runtime")
}
