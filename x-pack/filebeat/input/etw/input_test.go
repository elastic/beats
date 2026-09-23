// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package etw

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/windows"

	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/common/backoff"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/libbeat/reader/etw"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

type mockSessionOperator struct {
	// Fields to store function implementations that tests can customize
	newSessionFunc              func(config config) (*etw.Session, error)
	attachToExistingSessionFunc func(session *etw.Session) error
	createRealtimeSessionFunc   func(session *etw.Session) error
	startConsumerFunc           func(session *etw.Session) error
	stopSessionFunc             func(session *etw.Session) error

	// resets counts resetSession calls, one per reconnect attempt.
	resets atomic.Int32
}

func (m *mockSessionOperator) resetSession(*etw.Session) {
	m.resets.Add(1)
}

func (m *mockSessionOperator) newSession(config config) (*etw.Session, error) {
	if m.newSessionFunc != nil {
		return m.newSessionFunc(config)
	}
	return nil, nil
}

func (m *mockSessionOperator) attachToExistingSession(session *etw.Session) error {
	if m.attachToExistingSessionFunc != nil {
		return m.attachToExistingSessionFunc(session)
	}
	return nil
}

func (m *mockSessionOperator) createRealtimeSession(session *etw.Session) error {
	if m.createRealtimeSessionFunc != nil {
		return m.createRealtimeSessionFunc(session)
	}
	return nil
}

func (m *mockSessionOperator) startConsumer(session *etw.Session) error {
	if m.startConsumerFunc != nil {
		return m.startConsumerFunc(session)
	}
	return nil
}

func (m *mockSessionOperator) stopSession(session *etw.Session) error {
	if m.stopSessionFunc != nil {
		return m.stopSessionFunc(session)
	}
	return nil
}

// statusUpdate is one UpdateStatus call as seen by recordingReporter.
type statusUpdate struct {
	status status.Status
	msg    string
}

// recordingReporter captures every status update so tests can check the
// exact sequence the input reported.
type recordingReporter struct {
	mu      sync.Mutex
	updates []statusUpdate
}

func (r *recordingReporter) UpdateStatus(s status.Status, msg string) {
	r.mu.Lock()
	r.updates = append(r.updates, statusUpdate{status: s, msg: msg})
	r.mu.Unlock()
}

func (r *recordingReporter) statuses() []status.Status {
	r.mu.Lock()
	out := make([]status.Status, len(r.updates))
	for i, u := range r.updates {
		out[i] = u.status
	}
	r.mu.Unlock()
	return out
}

// lastMsg returns the message of the most recent update, or "" if none.
func (r *recordingReporter) lastMsg() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.updates) == 0 {
		return ""
	}
	return r.updates[len(r.updates)-1].msg
}

func Test_RunEtwInput_NewSessionError(t *testing.T) {
	// Mocks
	mockOperator := &mockSessionOperator{}

	// Setup the mock behavior for NewSession
	mockOperator.newSessionFunc = func(config config) (*etw.Session, error) {
		return nil, fmt.Errorf("failed creating session '%s'", config.SessionName)
	}

	// Setup input
	reporter := &recordingReporter{}
	inputCtx := input.Context{
		Cancelation:     t.Context(),
		Logger:          logptest.NewTestingLogger(t, ""),
		MetricsRegistry: monitoring.NewRegistry(),
	}.WithStatusReporter(reporter)

	etwInput := &etwInput{
		config: config{
			ProviderName:    "Microsoft-Windows-Provider",
			SessionName:     "MySession",
			TraceLevel:      "verbose",
			MatchAnyKeyword: 0xffffffffffffffff,
			MatchAllKeyword: 0,
		},
		operator: mockOperator,
		metrics: newInputMetrics(
			"test", inputCtx.MetricsRegistry, logptest.NewTestingLogger(t, "")),
	}

	// Run test
	err := etwInput.Run(inputCtx, nil)
	assert.EqualError(t, err, "error initializing ETW session: failed creating session 'MySession'")
	assert.Equal(t, []status.Status{status.Starting, status.Failed}, reporter.statuses(),
		"input should report Starting then Failed when the session cannot be initialized")
	assert.Equal(t, "failed to initialize ETW session: failed creating session 'MySession'", reporter.lastMsg(),
		"Failed message should carry the underlying error")
}

func Test_RunEtwInput_AttachToExistingSessionError(t *testing.T) {
	// Mocks
	mockOperator := &mockSessionOperator{}

	// Setup the mock behavior for NewSession
	mockOperator.newSessionFunc = func(config config) (*etw.Session, error) {
		mockSession := &etw.Session{
			Name:       "MySession",
			Realtime:   true,
			NewSession: false,
		}
		return mockSession, nil
	}
	// Setup the mock behavior for AttachToExistingSession. Wrap a real
	// Windows error so we can check the code is surfaced in the status.
	mockOperator.attachToExistingSessionFunc = func(session *etw.Session) error {
		return fmt.Errorf("session is not running: %w", etw.ERROR_WMI_INSTANCE_NOT_FOUND)
	}

	// Setup input
	reporter := &recordingReporter{}
	inputCtx := input.Context{
		Cancelation:     t.Context(),
		Logger:          logptest.NewTestingLogger(t, ""),
		MetricsRegistry: monitoring.NewRegistry(),
	}.WithStatusReporter(reporter)

	etwInput := &etwInput{
		config: config{
			ProviderName:    "Microsoft-Windows-Provider",
			SessionName:     "MySession",
			TraceLevel:      "verbose",
			MatchAnyKeyword: 0xffffffffffffffff,
			MatchAllKeyword: 0,
		},
		operator: mockOperator,
	}

	// Run test
	err := etwInput.Run(inputCtx, nil)
	assert.ErrorContains(t, err, "unable to retrieve handler: session is not running")
	assert.Equal(t, []status.Status{status.Starting, status.Configuring, status.Failed}, reporter.statuses(),
		"input should fail while configuring when attach fails")
	assert.Contains(t, reporter.lastMsg(), `failed to attach to session "MySession"`,
		"Failed message should name the session")
	assert.Contains(t, reporter.lastMsg(), fmt.Sprintf("(windows error %d)", uint32(etw.ERROR_WMI_INSTANCE_NOT_FOUND)),
		"Failed message should include the Windows error code")
}

func Test_RunEtwInput_CreateRealtimeSessionError(t *testing.T) {
	// Mocks
	mockOperator := &mockSessionOperator{}

	// Setup the mock behavior for NewSession
	mockOperator.newSessionFunc = func(config config) (*etw.Session, error) {
		mockSession := &etw.Session{
			Name:       "MySession",
			Realtime:   true,
			NewSession: true,
		}
		return mockSession, nil
	}
	// Setup the mock behavior for AttachToExistingSession
	mockOperator.attachToExistingSessionFunc = func(session *etw.Session) error {
		return nil
	}
	// Setup the mock behavior for CreateRealtimeSession
	mockOperator.createRealtimeSessionFunc = func(session *etw.Session) error {
		return fmt.Errorf("mock error")
	}

	// Setup input
	reporter := &recordingReporter{}
	inputCtx := input.Context{
		Cancelation:     t.Context(),
		Logger:          logptest.NewTestingLogger(t, ""),
		MetricsRegistry: monitoring.NewRegistry(),
	}.WithStatusReporter(reporter)

	etwInput := &etwInput{
		config: config{
			ProviderName:    "Microsoft-Windows-Provider",
			SessionName:     "MySession",
			TraceLevel:      "verbose",
			MatchAnyKeyword: 0xffffffffffffffff,
			MatchAllKeyword: 0,
		},
		operator: mockOperator,
	}

	// Run test
	err := etwInput.Run(inputCtx, nil)
	assert.EqualError(t, err, "realtime session could not be created: mock error")
	assert.Equal(t, []status.Status{status.Starting, status.Configuring, status.Failed}, reporter.statuses(),
		"input should fail while configuring when session creation fails")
	assert.Equal(t, `failed to create realtime session "MySession" for provider Microsoft-Windows-Provider: mock error`,
		reporter.lastMsg(), "Failed message should name the session and provider")
}

func Test_RunEtwInput_CreateRealtimeSessionAlreadyExists(t *testing.T) {
	// When the session already exists we fall back to attaching. That is a
	// normal path and must not show up as Degraded or Failed.
	mockOperator := &mockSessionOperator{}
	mockOperator.newSessionFunc = func(config config) (*etw.Session, error) {
		return &etw.Session{Name: "MySession", Realtime: true, NewSession: true}, nil
	}
	mockOperator.createRealtimeSessionFunc = func(session *etw.Session) error {
		return fmt.Errorf("session already exists: %w", etw.ERROR_ALREADY_EXISTS)
	}
	attached := false
	mockOperator.attachToExistingSessionFunc = func(session *etw.Session) error {
		attached = true
		return nil
	}
	blockConsumerUntilStopped(mockOperator)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	reporter := &recordingReporter{}
	inputCtx := input.Context{
		Cancelation:     ctx,
		Logger:          logptest.NewTestingLogger(t, ""),
		MetricsRegistry: monitoring.NewRegistry(),
	}.WithStatusReporter(reporter)

	etwInput := &etwInput{
		config:   config{ProviderName: "Microsoft-Windows-Provider", SessionName: "MySession"},
		operator: mockOperator,
	}

	done := make(chan error, 1)
	go func() { done <- etwInput.Run(inputCtx, nil) }()
	waitForStatus(t, reporter, status.Running)
	cancel()
	assert.NoError(t, <-done, "Run should exit cleanly on cancellation")
	assert.True(t, attached, "input should attach when the session already exists")
	assert.Equal(t, []status.Status{status.Starting, status.Configuring, status.Running}, reporter.statuses(),
		"falling back to attach should still end up Running")
}

func Test_RunEtwInput_StartConsumerError(t *testing.T) {
	// Mocks
	mockOperator := &mockSessionOperator{}

	// Setup the mock behavior for NewSession
	mockOperator.newSessionFunc = func(config config) (*etw.Session, error) {
		mockSession := &etw.Session{
			Name:       "MySession",
			Realtime:   true,
			NewSession: true,
		}
		return mockSession, nil
	}
	// Setup the mock behavior for AttachToExistingSession
	mockOperator.attachToExistingSessionFunc = func(session *etw.Session) error {
		return nil
	}
	// Setup the mock behavior for CreateRealtimeSession
	mockOperator.createRealtimeSessionFunc = func(session *etw.Session) error {
		return nil
	}
	// Setup the mock behavior for StartConsumer. Access denied is the most
	// common real failure here, so use it to check the privilege hint.
	mockOperator.startConsumerFunc = func(session *etw.Session) error {
		return fmt.Errorf("access denied when opening trace: %w", etw.ERROR_ACCESS_DENIED)
	}
	// Setup the mock behavior for StopSession
	mockOperator.stopSessionFunc = func(session *etw.Session) error {
		return nil
	}

	// Setup cancellation
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	// Setup input
	reporter := &recordingReporter{}
	inputCtx := input.Context{
		Cancelation:     ctx,
		Logger:          logptest.NewTestingLogger(t, ""),
		MetricsRegistry: monitoring.NewRegistry(),
	}.WithStatusReporter(reporter)

	etwInput := &etwInput{
		config: config{
			ProviderName:    "Microsoft-Windows-Provider",
			SessionName:     "MySession",
			TraceLevel:      "verbose",
			MatchAnyKeyword: 0xffffffffffffffff,
			MatchAllKeyword: 0,
		},
		operator: mockOperator,
	}

	// Run test
	err := etwInput.Run(inputCtx, nil)
	assert.ErrorContains(t, err, "failed running ETW consumer: access denied when opening trace")
	assert.Equal(t, []status.Status{status.Starting, status.Configuring, status.Running, status.Failed}, reporter.statuses(),
		"consumer failure should move from Running to Failed")
	assert.Contains(t, reporter.lastMsg(), `ETW consumer for session "MySession" failed`,
		"Failed message should name the session")
	assert.Contains(t, reporter.lastMsg(), "Performance Log Users",
		"access denied should explain which privileges ETW needs")
}

func Test_RunEtwInput_Success(t *testing.T) {
	// Mocks
	mockOperator := &mockSessionOperator{}

	// Setup the mock behavior for NewSession
	mockOperator.newSessionFunc = func(config config) (*etw.Session, error) {
		mockSession := &etw.Session{
			Name:       "MySession",
			Realtime:   true,
			NewSession: true,
		}
		return mockSession, nil
	}
	// Setup the mock behavior for AttachToExistingSession
	mockOperator.attachToExistingSessionFunc = func(session *etw.Session) error {
		return nil
	}
	// Setup the mock behavior for CreateRealtimeSession
	mockOperator.createRealtimeSessionFunc = func(session *etw.Session) error {
		return nil
	}
	// Setup the mock behavior for StartConsumer and StopSession
	blockConsumerUntilStopped(mockOperator)

	// Setup cancellation
	ctx, cancelFunc := context.WithCancel(t.Context())
	defer cancelFunc()

	// Setup input
	reporter := &recordingReporter{}
	inputCtx := input.Context{
		Cancelation:     ctx,
		Logger:          logptest.NewTestingLogger(t, ""),
		MetricsRegistry: monitoring.NewRegistry(),
	}.WithStatusReporter(reporter)

	etwInput := &etwInput{
		config: config{
			ProviderName:    "Microsoft-Windows-Provider",
			SessionName:     "MySession",
			TraceLevel:      "verbose",
			MatchAnyKeyword: 0xffffffffffffffff,
			MatchAllKeyword: 0,
		},
		operator: mockOperator,
	}

	// Run test
	done := make(chan error, 1)
	go func() { done <- etwInput.Run(inputCtx, nil) }()

	// Wait until the consumer is up, then cancel to test cleanup and exit.
	waitForStatus(t, reporter, status.Running)
	cancelFunc()
	assert.NoError(t, <-done, "Run should exit cleanly on cancellation")
	assert.Equal(t, []status.Status{status.Starting, status.Configuring, status.Running}, reporter.statuses(),
		"a healthy start should report Starting, Configuring, Running exactly once each")
}

func Test_RunEtwInput_ReconnectAfterAttachedSessionStops(t *testing.T) {
	// The input is attached to a session owned by someone else, who stops it
	// and restarts it a little later. ProcessTrace returns success when the
	// session stops, so the first startConsumer returns nil unprompted.
	// Attaching then fails until the owner has restarted the session.
	mockOperator := &mockSessionOperator{}
	mockOperator.newSessionFunc = func(config) (*etw.Session, error) {
		return &etw.Session{Name: "MySession", Realtime: true, NewSession: false}, nil
	}
	var attaches atomic.Int32
	mockOperator.attachToExistingSessionFunc = func(*etw.Session) error {
		// 1: initial attach. 2, 3: session still gone. 4: it is back.
		switch attaches.Add(1) {
		case 2, 3:
			return fmt.Errorf("session is not running: %w", etw.ERROR_WMI_INSTANCE_NOT_FOUND)
		}
		return nil
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	consumes := lostSessionConsumer(mockOperator, ctx)
	reporter := &recordingReporter{}
	inputCtx := input.Context{
		Cancelation:     ctx,
		Logger:          logptest.NewTestingLogger(t, ""),
		MetricsRegistry: monitoring.NewRegistry(),
	}.WithStatusReporter(reporter)

	etwInput := &etwInput{
		config:   config{Session: "MySession"},
		operator: mockOperator,
		backoff:  testBackoff(),
	}

	done := make(chan error, 1)
	go func() { done <- etwInput.Run(inputCtx, nil) }()

	want := []status.Status{
		status.Starting, status.Configuring, status.Running, // First connection.
		status.Degraded, // Session stopped by its owner.
		status.Degraded, // Reconnect attempt 2 could not attach.
		status.Degraded, // Reconnect attempt 3 could not attach.
		status.Running,  // Reattached.
	}
	waitForUpdateCount(t, reporter, len(want))
	cancel()
	assert.NoError(t, <-done, "Run should exit cleanly on cancellation after reconnecting")

	assert.Equal(t, want, reporter.statuses(), "input should stay Degraded while the session is gone and recover to Running")
	updates := reporter.updates
	assert.Equal(t, `ETW session "MySession" stopped; reconnecting (attempt 1)`, updates[3].msg,
		"losing the session should say so and count the attempt")
	assert.Equal(t, fmt.Sprintf(`failed to attach to session "MySession": session is not running: %v (windows error %d); reconnecting (attempt 2)`,
		etw.ERROR_WMI_INSTANCE_NOT_FOUND, uint32(etw.ERROR_WMI_INSTANCE_NOT_FOUND)), updates[4].msg,
		"a failed reconnect should carry the attach error and the next attempt number")
	assert.Equal(t, int32(4), attaches.Load(), "one initial attach plus one per reconnect attempt")
	assert.Equal(t, int32(2), consumes.Load(), "the consumer should only start once the session is back")
	assert.Equal(t, int32(3), mockOperator.resets.Load(), "the session should be reset before every reconnect attempt")
	assert.Equal(t, uint64(3), etwInput.metrics.reconnects.Get(), "reconnects_total should count every attempt")
}

func Test_RunEtwInput_RecreateOwnSessionAfterItStops(t *testing.T) {
	// The input created the session itself and someone stops it from
	// outside. Recreating it does not depend on anyone else, so the first
	// reconnect attempt succeeds.
	mockOperator := &mockSessionOperator{}
	mockOperator.newSessionFunc = func(config) (*etw.Session, error) {
		return &etw.Session{Name: "MySession", Realtime: true, NewSession: true}, nil
	}
	var creates, attaches atomic.Int32
	mockOperator.createRealtimeSessionFunc = func(*etw.Session) error {
		creates.Add(1)
		return nil
	}
	mockOperator.attachToExistingSessionFunc = func(*etw.Session) error {
		attaches.Add(1)
		return nil
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	consumes := lostSessionConsumer(mockOperator, ctx)
	reporter := &recordingReporter{}
	inputCtx := input.Context{
		Cancelation:     ctx,
		Logger:          logptest.NewTestingLogger(t, ""),
		MetricsRegistry: monitoring.NewRegistry(),
	}.WithStatusReporter(reporter)

	etwInput := &etwInput{
		config:   config{ProviderName: "Microsoft-Windows-Provider", SessionName: "MySession"},
		operator: mockOperator,
		backoff:  testBackoff(),
	}

	done := make(chan error, 1)
	go func() { done <- etwInput.Run(inputCtx, nil) }()

	want := []status.Status{status.Starting, status.Configuring, status.Running, status.Degraded, status.Running}
	waitForUpdateCount(t, reporter, len(want))
	cancel()
	assert.NoError(t, <-done, "Run should exit cleanly on cancellation after reconnecting")

	assert.Equal(t, want, reporter.statuses(), "a recreated session should go Degraded then straight back to Running")
	assert.Equal(t, int32(2), creates.Load(), "the session should be created again after it was stopped")
	assert.Equal(t, int32(0), attaches.Load(), "creating succeeded, so there should be no attach fallback")
	assert.Equal(t, int32(2), consumes.Load(), "the consumer should be started for each session")
	assert.Equal(t, int32(1), mockOperator.resets.Load(), "the session should be reset once per reconnect")
}

func Test_RunEtwInput_ReconnectPermanentErrorFails(t *testing.T) {
	// The session was lost and the reconnect attempt fails with an error that
	// waiting will not fix. Staying Degraded forever would hide a problem that
	// needs a person, so the input reports Failed and returns, exactly as it
	// would had the same error happened on the first pass.
	mockOperator := &mockSessionOperator{}
	mockOperator.newSessionFunc = func(config) (*etw.Session, error) {
		return &etw.Session{Name: "MySession", Realtime: true, NewSession: false}, nil
	}
	var attaches atomic.Int32
	mockOperator.attachToExistingSessionFunc = func(*etw.Session) error {
		if attaches.Add(1) == 1 {
			return nil
		}
		return fmt.Errorf("failed to get handler: %w", etw.ERROR_ACCESS_DENIED)
	}
	mockOperator.startConsumerFunc = func(*etw.Session) error { return nil } // Session lost at once.

	reporter := &recordingReporter{}
	inputCtx := input.Context{
		Cancelation:     t.Context(),
		Logger:          logptest.NewTestingLogger(t, ""),
		MetricsRegistry: monitoring.NewRegistry(),
	}.WithStatusReporter(reporter)

	etwInput := &etwInput{
		config:   config{Session: "MySession"},
		operator: mockOperator,
		backoff:  testBackoff(),
	}

	done := make(chan error, 1)
	go func() { done <- etwInput.Run(inputCtx, nil) }()
	select {
	case err := <-done:
		assert.ErrorContains(t, err, "unable to retrieve handler", "the permanent error should be returned to the runner")
		assert.ErrorIs(t, err, etw.ERROR_ACCESS_DENIED, "the underlying Windows error should be preserved")
	case <-time.After(5 * time.Second):
		t.Fatal("Run kept retrying a permanent error instead of failing")
	}
	assert.Equal(t, []status.Status{status.Starting, status.Configuring, status.Running, status.Degraded, status.Failed}, reporter.statuses(),
		"a permanent reconnect failure should end in Failed, not stay Degraded")
	assert.Contains(t, reporter.lastMsg(), "Performance Log Users",
		"the Failed message should carry the same detail a first-pass failure would")
	assert.Equal(t, int32(2), attaches.Load(), "there should be exactly one reconnect attempt")
}

func Test_RunEtwInput_ConnectFailureStopsSession(t *testing.T) {
	// Creating a session can fail after StartTrace has succeeded, for example
	// when a provider cannot be enabled. Without a stop, that half-configured
	// session stays running; the next attempt would then get
	// ERROR_ALREADY_EXISTS, attach to it, and never see an event.
	tests := []struct {
		name string
		// failOnCreate is the create call that fails: 1 is the first pass,
		// 2 is the first reconnect attempt.
		failOnCreate int32
		wantStatus   []status.Status
		wantStops    int32
	}{
		{
			name:         "first pass",
			failOnCreate: 1,
			wantStatus:   []status.Status{status.Starting, status.Configuring, status.Failed},
			wantStops:    1, // Teardown after the failed create.
		},
		{
			name:         "reconnect",
			failOnCreate: 2,
			wantStatus:   []status.Status{status.Starting, status.Configuring, status.Running, status.Degraded, status.Failed},
			wantStops:    2, // Cleanup after the loss, then teardown after the failed create.
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockOperator := &mockSessionOperator{}
			mockOperator.newSessionFunc = func(config) (*etw.Session, error) {
				return &etw.Session{Name: "MySession", Realtime: true, NewSession: true}, nil
			}
			var creates, stops atomic.Int32
			mockOperator.createRealtimeSessionFunc = func(*etw.Session) error {
				if creates.Add(1) == test.failOnCreate {
					return fmt.Errorf("failed to enable trace: %w", etw.ERROR_INVALID_PARAMETER)
				}
				return nil
			}
			mockOperator.attachToExistingSessionFunc = func(*etw.Session) error {
				t.Error("a create failure other than ERROR_ALREADY_EXISTS must not fall back to attach")
				return nil
			}
			mockOperator.startConsumerFunc = func(*etw.Session) error { return nil } // Session lost at once.
			mockOperator.stopSessionFunc = func(*etw.Session) error {
				stops.Add(1)
				return nil
			}

			reporter := &recordingReporter{}
			inputCtx := input.Context{
				Cancelation:     t.Context(),
				Logger:          logptest.NewTestingLogger(t, ""),
				MetricsRegistry: monitoring.NewRegistry(),
			}.WithStatusReporter(reporter)

			etwInput := &etwInput{
				config:   config{ProviderName: "Microsoft-Windows-Provider", SessionName: "MySession"},
				operator: mockOperator,
				backoff:  testBackoff(),
			}

			done := make(chan error, 1)
			go func() { done <- etwInput.Run(inputCtx, nil) }()
			select {
			case err := <-done:
				assert.ErrorContains(t, err, "realtime session could not be created", "the create error should be returned")
			case <-time.After(5 * time.Second):
				t.Fatal("Run did not return after the create failure")
			}
			assert.Equal(t, test.wantStatus, reporter.statuses(), "create failure should end in Failed")
			assert.Equal(t, test.wantStops, stops.Load(), "the session must be stopped after a failed create so nothing is left running")
		})
	}
}

func Test_RunEtwInput_LogfileDoesNotReconnect(t *testing.T) {
	// For an .etl file ProcessTrace returning means end of file. That is
	// the input finishing its job, not a lost session.
	tests := []struct {
		name        string
		consumerErr error
		wantErr     string
		wantStatus  []status.Status
	}{
		{
			name:       "end of file",
			wantStatus: []status.Status{status.Starting, status.Running},
		},
		{
			name:        "unreadable file",
			consumerErr: fmt.Errorf("invalid log source when opening trace: %w", etw.ERROR_BAD_PATHNAME),
			wantErr:     "failed running ETW consumer: invalid log source when opening trace",
			wantStatus:  []status.Status{status.Starting, status.Running, status.Failed},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockOperator := &mockSessionOperator{}
			mockOperator.newSessionFunc = func(config) (*etw.Session, error) {
				return &etw.Session{Name: `C:\logs\trace.etl`, Realtime: false}, nil
			}
			var connects atomic.Int32
			mockOperator.createRealtimeSessionFunc = func(*etw.Session) error {
				connects.Add(1)
				return nil
			}
			mockOperator.attachToExistingSessionFunc = func(*etw.Session) error {
				connects.Add(1)
				return nil
			}
			mockOperator.startConsumerFunc = func(*etw.Session) error { return test.consumerErr }

			reporter := &recordingReporter{}
			inputCtx := input.Context{
				Cancelation:     t.Context(),
				Logger:          logptest.NewTestingLogger(t, ""),
				MetricsRegistry: monitoring.NewRegistry(),
			}.WithStatusReporter(reporter)

			etwInput := &etwInput{
				config:   config{Logfile: `C:\logs\trace.etl`},
				operator: mockOperator,
				backoff:  testBackoff(),
			}

			done := make(chan error, 1)
			go func() { done <- etwInput.Run(inputCtx, nil) }()
			select {
			case err := <-done:
				if test.wantErr == "" {
					assert.NoError(t, err, "reaching the end of the file is a clean exit")
				} else {
					assert.ErrorContains(t, err, test.wantErr, "a consumer error on a file is fatal")
				}
			case <-time.After(5 * time.Second):
				t.Fatal("Run did not return after the file was consumed; it must not try to reconnect")
			}
			assert.Equal(t, test.wantStatus, reporter.statuses(), "file input should not report Configuring or Degraded")
			assert.Equal(t, int32(0), connects.Load(), "file input should neither create nor attach to a session")
			assert.Equal(t, int32(0), mockOperator.resets.Load(), "file input should never reset the session")
			assert.Equal(t, uint64(0), etwInput.metrics.reconnects.Get(), "file input should never count a reconnect")
		})
	}
}

func Test_RunEtwInput_CancelWhileWaitingToReconnect(t *testing.T) {
	// Shutdown during the backoff between reconnect attempts must return
	// promptly and cleanly, not report Failed and not wait out the backoff.
	mockOperator := &mockSessionOperator{}
	mockOperator.newSessionFunc = func(config) (*etw.Session, error) {
		return &etw.Session{Name: "MySession", Realtime: true, NewSession: false}, nil
	}
	mockOperator.startConsumerFunc = func(*etw.Session) error { return nil } // Session lost at once.

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	reporter := &recordingReporter{}
	inputCtx := input.Context{
		Cancelation:     ctx,
		Logger:          logptest.NewTestingLogger(t, ""),
		MetricsRegistry: monitoring.NewRegistry(),
	}.WithStatusReporter(reporter)

	etwInput := &etwInput{
		config:   config{Session: "MySession"},
		operator: mockOperator,
		// Long enough that the test can only pass if cancellation
		// interrupts the wait.
		backoff: backoff.NewEqualJitterBackoff(time.Hour, time.Hour),
	}

	done := make(chan error, 1)
	go func() { done <- etwInput.Run(inputCtx, nil) }()
	waitForStatus(t, reporter, status.Degraded)
	cancel()
	select {
	case err := <-done:
		assert.NoError(t, err, "cancellation while waiting to reconnect is a clean exit")
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after cancellation while waiting to reconnect")
	}
	assert.Equal(t, []status.Status{status.Starting, status.Configuring, status.Running, status.Degraded}, reporter.statuses(),
		"shutdown while Degraded must not report Failed")
	assert.Equal(t, int32(0), mockOperator.resets.Load(), "cancelled before the reconnect attempt, so no reset")
	assert.Equal(t, uint64(0), etwInput.metrics.reconnects.Get(), "an attempt that shutdown interrupted must not be counted")
}

// lostSessionConsumer configures op so that startConsumer returns nil at once
// on its first call, as ProcessTrace does when another controller stops the
// session, and on later calls blocks until the input is cancelled and stops
// the session. It returns the count of startConsumer calls.
//
// The release is keyed on cancellation rather than on which stopSession call
// this is: the input's own cleanup stop after the loss must not release the
// consumer, and cancellation may land before or after the second
// startConsumer call, since Running is reported just ahead of it.
func lostSessionConsumer(op *mockSessionOperator, ctx context.Context) *atomic.Int32 {
	consumes := new(atomic.Int32)
	stopped := make(chan struct{})
	var closeStopped sync.Once
	op.startConsumerFunc = func(*etw.Session) error {
		if consumes.Add(1) == 1 {
			return nil
		}
		<-stopped
		return nil
	}
	op.stopSessionFunc = func(*etw.Session) error {
		if ctx.Err() != nil {
			closeStopped.Do(func() { close(stopped) })
		}
		return nil
	}
	return consumes
}

// testBackoff returns a backoff short enough that reconnect tests run in
// milliseconds rather than the seconds production waits.
func testBackoff() backoff.Backoff {
	return backoff.NewEqualJitterBackoff(time.Millisecond, 5*time.Millisecond)
}

// waitForUpdateCount blocks until reporter has recorded at least n status
// updates, failing the test if they do not show up in time.
func waitForUpdateCount(t *testing.T, reporter *recordingReporter, n int) {
	t.Helper()
	assert.Eventually(t, func() bool {
		return len(reporter.statuses()) >= n
	}, 5*time.Second, 10*time.Millisecond, "input never reported %d status updates, got %v", n, reporter.statuses())
}

// blockConsumerUntilStopped makes the mock consumer behave like the real
// one: startConsumer blocks until stopSession is called, so tests can
// exercise the cancellation path rather than having Run return at once.
func blockConsumerUntilStopped(op *mockSessionOperator) {
	stopped := make(chan struct{})
	op.startConsumerFunc = func(*etw.Session) error {
		<-stopped
		return nil
	}
	op.stopSessionFunc = func(*etw.Session) error {
		close(stopped)
		return nil
	}
}

// waitForStatus blocks until reporter has seen want, failing the test if it
// does not show up in time.
func waitForStatus(t *testing.T, reporter *recordingReporter, want status.Status) {
	t.Helper()
	assert.Eventually(t, func() bool {
		return slices.Contains(reporter.statuses(), want)
	}, 5*time.Second, 10*time.Millisecond, "input never reported %s", want)
}

func Test_eventHealth(t *testing.T) {
	// Each test feeds a string of events to eventHealth: 'b' is a failure,
	// 'g' is a success and 'r' is a reset, as done on reconnect. Spaces are
	// ignored and are only there to make the runs readable.
	degraded := func(n int) statusUpdate {
		return statusUpdate{status.Degraded, fmt.Sprintf("%d consecutive events could not be read; last error: bad", n)}
	}
	running := statusUpdate{status.Running, ""}

	tests := []struct {
		name     string
		failure  uint
		recovery uint
		events   string
		want     []statusUpdate
	}{
		{
			name:     "defaults: below failure threshold never degrades",
			failure:  3,
			recovery: 1,
			events:   "bb g bb g bb",
			want:     nil,
		},
		{
			name:     "defaults: degrades at threshold, once",
			failure:  3,
			recovery: 1,
			events:   "bbb bbbbb",
			want:     []statusUpdate{degraded(3)},
		},
		{
			name:     "defaults: one good event recovers",
			failure:  3,
			recovery: 1,
			events:   "bbb g",
			want:     []statusUpdate{degraded(3), running},
		},
		{
			name:     "defaults: alternating runs flap",
			failure:  3,
			recovery: 1,
			events:   "bbb g bbb g",
			want:     []statusUpdate{degraded(3), running, degraded(3), running},
		},
		{
			name:     "higher recovery threshold: one good event is not enough",
			failure:  3,
			recovery: 3,
			events:   "bbb g bbb g",
			want:     []statusUpdate{degraded(3)},
		},
		{
			name:     "higher recovery threshold: a failure resets the good run",
			failure:  3,
			recovery: 3,
			events:   "bbb gg b ggg",
			want:     []statusUpdate{degraded(3), running},
		},
		{
			name:     "good events while healthy report nothing",
			failure:  2,
			recovery: 1,
			events:   "gggggg",
			want:     nil,
		},
		{
			name:     "zero failure threshold disables reporting",
			failure:  0,
			recovery: 1,
			events:   "bbbbbbbbbb gggg",
			want:     nil,
		},
		{
			// Run reports Running itself after a reconnect. Without the
			// reset the stale degraded flag would swallow the second run.
			name:     "reset after Degraded lets the next failure run report again",
			failure:  3,
			recovery: 1,
			events:   "bbb r bbb",
			want:     []statusUpdate{degraded(3), degraded(3)},
		},
		{
			name:     "reset clears a partial failure run",
			failure:  3,
			recovery: 1,
			events:   "bb r bb",
			want:     nil,
		},
		{
			name:     "reset while healthy reports nothing",
			failure:  3,
			recovery: 1,
			events:   "gg r gg",
			want:     nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reporter := &recordingReporter{}
			h := &eventHealth{
				reporter:          reporter,
				failureThreshold:  test.failure,
				recoveryThreshold: test.recovery,
			}
			for _, ev := range test.events {
				switch ev {
				case 'b':
					h.failure(errors.New("bad"))
				case 'g':
					h.success()
				case 'r':
					h.reset()
				}
			}
			assert.Equal(t, test.want, reporter.updates,
				"events %q with failure=%d recovery=%d", test.events, test.failure, test.recovery)
		})
	}
}

func Test_errDetail(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "plain error",
			err:  errors.New("boom"),
			want: "boom",
		},
		{
			name: "wrapped windows error gets its code",
			err:  fmt.Errorf("session is not running: %w", etw.ERROR_WMI_INSTANCE_NOT_FOUND),
			want: fmt.Sprintf("session is not running: %v (windows error 4201)", etw.ERROR_WMI_INSTANCE_NOT_FOUND),
		},
		{
			name: "access denied explains required privileges",
			err:  fmt.Errorf("access denied when opening trace: %w", etw.ERROR_ACCESS_DENIED),
			want: fmt.Sprintf("access denied when opening trace: %v (windows error 5); ETW sessions require running as Administrator or membership in the Performance Log Users group",
				etw.ERROR_ACCESS_DENIED),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, errDetail(test.err), "errDetail(%v)", test.err)
		})
	}
}

func Test_statusMessage(t *testing.T) {
	se := &sessionError{
		status: `failed to attach to session "MySession": boom (windows error 5)`,
		err:    errors.New("unable to retrieve handler: boom"),
	}
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "session error reports its status text",
			err:  se,
			want: se.status,
		},
		{
			name: "wrapped session error is still found",
			err:  fmt.Errorf("outer: %w", se),
			want: se.status,
		},
		{
			name: "other errors fall back to errDetail",
			err:  fmt.Errorf("stopped: %w", etw.ERROR_WMI_INSTANCE_NOT_FOUND),
			want: errDetail(fmt.Errorf("stopped: %w", etw.ERROR_WMI_INSTANCE_NOT_FOUND)),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, statusMessage(test.err), "statusMessage(%v)", test.err)
		})
	}
	assert.EqualError(t, se, "unable to retrieve handler: boom", "the runner should see the short error, not the status text")
}

func Test_buildEvent(t *testing.T) {
	tests := []struct {
		name     string
		event    etw.RenderedEtwEvent
		header   etw.EventHeader
		session  *etw.Session
		cfg      config
		expected mapstr.M
	}{
		{
			name: "TestStandardData",
			event: etw.RenderedEtwEvent{
				ProviderGUID: windows.GUID{
					Data1: 0x12345678,
					Data2: 0x1234,
					Data3: 0x1234,
					Data4: [8]byte{0x12, 0x34, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc},
				},
				ProcessID:   60,
				Opcode:      "foo",
				OpcodeRaw:   50,
				Keywords:    []string{"keyword1", "keyword2"},
				KeywordsRaw: 40,
				TaskRaw:     70,
				Task:        "TestTask",
				ThreadID:    80,
				Version:     90,
				Level:       "Critical",
				LevelRaw:    1, // Critical
				Channel:     "TestChannel",
				Properties: []etw.RenderedProperty{
					{
						Name:  "key",
						Value: "value",
					},
				},
			},
			header: etw.EventHeader{
				Size:          0,
				HeaderType:    0,
				Flags:         30,
				EventProperty: 30,
				TimeStamp:     133516441890350000,
				ProviderId: windows.GUID{
					Data1: 0x12345678,
					Data2: 0x1234,
					Data3: 0x1234,
					Data4: [8]byte{0x12, 0x34, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc},
				},
				EventDescriptor: etw.EventDescriptor{
					Id:      20,
					Channel: 10,
					Level:   1, // Critical
					Opcode:  50,
					Keyword: 40,
				},
				Time: 0,
				ActivityId: windows.GUID{
					Data1: 0x12345678,
					Data2: 0x1234,
					Data3: 0x1234,
					Data4: [8]byte{0x12, 0x34, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc},
				},
			},
			session: &etw.Session{
				Name: "Elastic-TestProvider",
			},
			cfg: config{
				ProviderName: "TestProvider",
			},
			expected: mapstr.M{
				"winlog": map[string]any{
					"activity_id": "{12345678-1234-1234-1234-123456789ABC}",
					"channel":     "TestChannel",
					"event_data": mapstr.M{
						"key": "value",
					},
					"flags":         []string{"NO_CPUTIME", "PRIVATE_SESSION", "STRING_ONLY", "TRACE_MESSAGE"},
					"flags_raw":     "0x1E",
					"keywords":      []string{"keyword1", "keyword2"},
					"keywords_raw":  "0x28",
					"opcode":        "foo",
					"opcode_raw":    "0x32",
					"process_id":    "60",
					"provider_guid": "{12345678-1234-1234-1234-123456789ABC}",
					"session":       "Elastic-TestProvider",
					"task":          "TestTask",
					"task_raw":      uint16(70),
					"thread_id":     "80",
					"version":       "90",
				},
				"event.code":     "20",
				"event.provider": "TestProvider",
				"event.severity": uint8(1),
				"log.level":      "critical",
			},
		},
		{
			// This case tests an unmapped severity, empty provider GUID and including logfile
			name: "TestAlternativeMetadata",
			event: etw.RenderedEtwEvent{
				ProviderGUID: windows.GUID{
					Data1: 0x12345678,
					Data2: 0x1234,
					Data3: 0x1234,
					Data4: [8]byte{0x12, 0x34, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc},
				},
				ProcessID:   60,
				Opcode:      "foo",
				OpcodeRaw:   50,
				Keywords:    []string{"keyword1", "keyword2"},
				KeywordsRaw: 40,
				TaskRaw:     70,
				Task:        "TestTask",
				ThreadID:    80,
				Version:     90,
				Channel:     "TestChannel",
				Properties: []etw.RenderedProperty{
					{
						Name:  "key",
						Value: "value",
					},
				},
			},
			header: etw.EventHeader{
				Size:          0,
				HeaderType:    0,
				Flags:         30,
				EventProperty: 30,
				TimeStamp:     133516441890350000,
				EventDescriptor: etw.EventDescriptor{
					Id:      20,
					Channel: 10,
					Opcode:  50,
					Keyword: 40,
				},
				Time: 0,
				ActivityId: windows.GUID{
					Data1: 0x12345678,
					Data2: 0x1234,
					Data3: 0x1234,
					Data4: [8]byte{0x12, 0x34, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc},
				},
			},
			session: &etw.Session{
				Name: "Elastic-TestProvider",
			},
			cfg: config{
				ProviderName: "TestProvider",
				Logfile:      "C:\\TestFile",
			},

			expected: mapstr.M{
				"winlog": map[string]any{
					"activity_id": "{12345678-1234-1234-1234-123456789ABC}",
					"channel":     "TestChannel",
					"event_data": mapstr.M{
						"key": "value",
					},
					"flags":         []string{"NO_CPUTIME", "PRIVATE_SESSION", "STRING_ONLY", "TRACE_MESSAGE"},
					"flags_raw":     "0x1E",
					"keywords":      []string{"keyword1", "keyword2"},
					"keywords_raw":  "0x28",
					"opcode":        "foo",
					"opcode_raw":    "0x32",
					"process_id":    "60",
					"provider_guid": "{12345678-1234-1234-1234-123456789ABC}",
					"session":       "Elastic-TestProvider",
					"task":          "TestTask",
					"task_raw":      uint16(70),
					"thread_id":     "80",
					"version":       "90",
				},
				"event.code":     "20",
				"event.provider": "TestProvider",
				"event.severity": uint8(0),
				"log.file.path":  "C:\\TestFile",
				"log.level":      "information",
			},
		},
	}

	for _, tt := range tests {
		//nolint:errcheck // Ignore error checks for simplicity in this test
		t.Run(tt.name, func(t *testing.T) {
			evt := buildEvent(tt.event, tt.header, tt.session, tt.cfg)
			assert.Equal(t, tt.expected["winlog"].(map[string]any)["activity_id"], evt.Fields["winlog"].(map[string]any)["activity_id"])
			assert.Equal(t, tt.expected["winlog"].(map[string]any)["channel"], evt.Fields["winlog"].(map[string]any)["channel"])
			assert.Equal(t, tt.expected["winlog"].(map[string]any)["event_data"], evt.Fields["winlog"].(map[string]any)["event_data"])
			assert.Equal(t, tt.expected["winlog"].(map[string]any)["flags"], evt.Fields["winlog"].(map[string]any)["flags"])
			assert.Equal(t, tt.expected["winlog"].(map[string]any)["flags_raw"], evt.Fields["winlog"].(map[string]any)["flags_raw"])
			assert.Equal(t, tt.expected["winlog"].(map[string]any)["keywords"], evt.Fields["winlog"].(map[string]any)["keywords"])
			assert.Equal(t, tt.expected["winlog"].(map[string]any)["keywords_raw"], evt.Fields["winlog"].(map[string]any)["keywords_raw"])
			assert.Equal(t, tt.expected["winlog"].(map[string]any)["opcode"], evt.Fields["winlog"].(map[string]any)["opcode"])
			assert.Equal(t, tt.expected["winlog"].(map[string]any)["process_id"], evt.Fields["winlog"].(map[string]any)["process_id"])
			assert.Equal(t, tt.expected["winlog"].(map[string]any)["provider_guid"], evt.Fields["winlog"].(map[string]any)["provider_guid"])
			assert.Equal(t, tt.expected["winlog"].(map[string]any)["session"], evt.Fields["winlog"].(map[string]any)["session"])
			assert.Equal(t, tt.expected["winlog"].(map[string]any)["task"], evt.Fields["winlog"].(map[string]any)["task"])
			assert.Equal(t, tt.expected["winlog"].(map[string]any)["task_raw"], evt.Fields["winlog"].(map[string]any)["task_raw"])
			assert.Equal(t, tt.expected["winlog"].(map[string]any)["thread_id"], evt.Fields["winlog"].(map[string]any)["thread_id"])
			mapEv := evt.Fields.Flatten()

			assert.Equal(t, tt.expected["winlog"].(map[string]any)["version"], strconv.Itoa(int(mapEv["winlog.version"].(uint8))))
			assert.Equal(t, tt.expected["event.code"], mapEv["event.code"])
			assert.Equal(t, tt.expected["event.provider"], mapEv["event.provider"])
			assert.Equal(t, tt.expected["event.severity"], mapEv["event.severity"])
			assert.Equal(t, tt.expected["log.file.path"], mapEv["log.file.path"])
			assert.Equal(t, tt.expected["log.level"], mapEv["log.level"])
		})
	}
}
