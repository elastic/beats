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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/windows"

	input "github.com/elastic/beats/v7/filebeat/input/v2"
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
		Cancelation:     nil,
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
		Cancelation:     nil,
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
		Cancelation:     nil,
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
	ctx := t.Context()

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
	// 'g' is a success. Spaces are ignored and are only there to make the
	// runs readable.
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
