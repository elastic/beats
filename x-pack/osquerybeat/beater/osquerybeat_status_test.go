// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	agentconfig "github.com/elastic/elastic-agent-libs/config"

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
	mx       sync.Mutex
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
func (m *testManager) Stop()                               { m.stopped = true }
func (m *testManager) SetPayload(map[string]any)           {}
func (m *testManager) Enabled() bool                       { return true }
func (m *testManager) AgentInfo() client.AgentInfo         { return client.AgentInfo{} }
func (m *testManager) SetStopCallback(func())              {}
func (m *testManager) CheckRawConfig(*agentconfig.C) error { return nil }
func (m *testManager) RegisterAction(client.Action)        {}
func (m *testManager) UnregisterAction(client.Action)      {}
func (m *testManager) RegisterDiagnosticHook(string, string, string, string, client.DiagnosticHook) {
}

// TestOsquerybeatStatusReporting_Lifecycle tests the full lifecycle status reporting
// when osqueryd is available and runs successfully.
func TestOsquerybeatStatusReporting_Lifecycle(t *testing.T) {
	mgr := &testManager{}
	b := &beat.Beat{
		Manager:    mgr,
		Registry:   reload.NewRegistry(),
		Monitoring: beat.NewMonitoring(),
	}

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

// TestOsquerybeatStatusReporting_CheckFailure tests status reporting when osqueryd check fails.
func TestOsquerybeatStatusReporting_CheckFailure(t *testing.T) {
	mgr := &testManager{}
	b := &beat.Beat{
		Manager:    mgr,
		Registry:   reload.NewRegistry(),
		Monitoring: beat.NewMonitoring(),
	}

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
		Monitoring: beat.NewMonitoring(),
	}

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
		Monitoring: beat.NewMonitoring(),
	}

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
