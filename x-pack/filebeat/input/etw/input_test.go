// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package etw

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/x-pack/libbeat/reader/etw"
	"github.com/elastic/elastic-agent-libs/logp"

	"golang.org/x/sys/windows"
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

func Test_RunEtwInput_NewSessionError(t *testing.T) {
	// Mocks
	mockOperator := &mockSessionOperator{}

	// Setup the mock behavior for NewSession
	mockOperator.newSessionFunc = func(config config) (*etw.Session, error) {
		return nil, fmt.Errorf("failed creating session '%s'", config.SessionName)
	}

	// Setup input
	inputCtx := input.Context{
		Cancelation: nil,
		Logger:      logp.NewLogger("test"),
	}

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
	assert.EqualError(t, err, "error initializing ETW session: failed creating session 'MySession'")
}

func Test_RunEtwInput_AttachToExistingSessionError(t *testing.T) {
	// Mocks
	mockOperator := &mockSessionOperator{}

	// Setup the mock behavior for NewSession
	mockOperator.newSessionFunc = func(config config) (*etw.Session, error) {
		mockSession := &etw.Session{
			Name:       "MySession",
			Realtime:   true,
			NewSession: false}
		return mockSession, nil
	}
	// Setup the mock behavior for AttachToExistingSession
	mockOperator.attachToExistingSessionFunc = func(session *etw.Session) error {
		return fmt.Errorf("mock error")
	}

	// Setup input
	inputCtx := input.Context{
		Cancelation: nil,
		Logger:      logp.NewLogger("test"),
	}

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
	assert.EqualError(t, err, "unable to retrieve handler: mock error")
}

func Test_RunEtwInput_CreateRealtimeSessionError(t *testing.T) {
	// Mocks
	mockOperator := &mockSessionOperator{}

	// Setup the mock behavior for NewSession
	mockOperator.newSessionFunc = func(config config) (*etw.Session, error) {
		mockSession := &etw.Session{
			Name:       "MySession",
			Realtime:   true,
			NewSession: true}
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
	inputCtx := input.Context{
		Cancelation: nil,
		Logger:      logp.NewLogger("test"),
	}

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
}

func Test_RunEtwInput_StartConsumerError(t *testing.T) {
	// Mocks
	mockOperator := &mockSessionOperator{}

	// Setup the mock behavior for NewSession
	mockOperator.newSessionFunc = func(config config) (*etw.Session, error) {
		mockSession := &etw.Session{
			Name:       "MySession",
			Realtime:   true,
			NewSession: true}
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
	// Setup the mock behavior for StartConsumer
	mockOperator.startConsumerFunc = func(session *etw.Session) error {
		return fmt.Errorf("mock error")
	}
	// Setup the mock behavior for StopSession
	mockOperator.stopSessionFunc = func(session *etw.Session) error {
		return nil
	}

	// Setup cancellation
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	// Setup input
	inputCtx := input.Context{
		Cancelation: ctx,
		Logger:      logp.NewLogger("test"),
	}

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
	assert.EqualError(t, err, "failed to start consumer: mock error")
}

func Test_RunEtwInput_Success(t *testing.T) {
	// Mocks
	mockOperator := &mockSessionOperator{}

	// Setup the mock behavior for NewSession
	mockOperator.newSessionFunc = func(config config) (*etw.Session, error) {
		mockSession := &etw.Session{
			Name:       "MySession",
			Realtime:   true,
			NewSession: true}
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
	// Setup the mock behavior for StartConsumer
	mockOperator.startConsumerFunc = func(session *etw.Session) error {
		return nil
	}
	// Setup the mock behavior for StopSession
	mockOperator.stopSessionFunc = func(session *etw.Session) error {
		return nil
	}

	// Setup cancellation
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	// Setup input
	inputCtx := input.Context{
		Cancelation: ctx,
		Logger:      logp.NewLogger("test"),
	}

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
	go func() {
		err := etwInput.Run(inputCtx, nil)
		if err != nil {
			t.Errorf("Run() error = %v, wantErr %v", err, false)
		}
	}()

	// Simulate waiting for a condition
	time.Sleep(time.Millisecond * 100)
	cancelFunc() // Trigger cancellation to test cleanup and goroutine exit
}

func Test_buildEvent(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]any
		header   etw.EventHeader
		session  *etw.Session
		cfg      config
		expected map[string]any
	}{
		{
			name: "TestStandardData",
			data: map[string]any{
				"key": "value",
			},
			header: etw.EventHeader{
				Size:          0,
				HeaderType:    0,
				Flags:         30,
				EventProperty: 30,
				ThreadId:      80,
				ProcessId:     60,
				TimeStamp:     133516441890350000,
				ProviderId: windows.GUID{
					Data1: 0x12345678,
					Data2: 0x1234,
					Data3: 0x1234,
					Data4: [8]byte{0x12, 0x34, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc},
				},
				EventDescriptor: etw.EventDescriptor{
					Id:      20,
					Version: 90,
					Channel: 10,
					Level:   1, // Critical
					Opcode:  50,
					Task:    70,
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
				GUID: windows.GUID{
					Data1: 0x12345678,
					Data2: 0x1234,
					Data3: 0x1234,
					Data4: [8]byte{0x12, 0x34, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc},
				},
				Name: "Elastic-TestProvider",
			},
			cfg: config{
				ProviderName: "TestProvider",
			},

			expected: map[string]any{
				"activity_guid": "{12345678-1234-1234-1234-123456789ABC}",
				"channel":       uint8(10),
				"event_data": map[string]any{
					"key": "value",
				},
				"event_id":      uint16(20),
				"flags":         uint16(30),
				"keywords":      uint64(40),
				"level":         uint8(1),
				"logfile":       nil,
				"opcode":        uint8(50),
				"process_id":    uint32(60),
				"provider_guid": "{12345678-1234-1234-1234-123456789ABC}",
				"provider_name": "TestProvider",
				"session":       "Elastic-TestProvider",
				"severity":      "critical",
				"task":          uint16(70),
				"thread_id":     uint32(80),
				"timestamp":     "2024-02-05T22:03:09.035Z",
				"version":       uint8(90),
			},
		},
		{
			// This case tests an unmapped severity, empty provider GUID and including logfile
			name: "TestAlternativeMetadata",
			data: map[string]any{
				"key": "value",
			},
			header: etw.EventHeader{
				Size:          0,
				HeaderType:    0,
				Flags:         30,
				EventProperty: 30,
				ThreadId:      80,
				ProcessId:     60,
				TimeStamp:     133516441890350000,
				EventDescriptor: etw.EventDescriptor{
					Id:      20,
					Version: 90,
					Channel: 10,
					Level:   17, // Unknown
					Opcode:  50,
					Task:    70,
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
				GUID: windows.GUID{
					Data1: 0x12345678,
					Data2: 0x1234,
					Data3: 0x1234,
					Data4: [8]byte{0x12, 0x34, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc},
				},
				Name: "Elastic-TestProvider",
			},
			cfg: config{
				ProviderName: "TestProvider",
				Logfile:      "C:\\TestFile",
			},

			expected: map[string]any{
				"activity_guid": "{12345678-1234-1234-1234-123456789ABC}",
				"channel":       uint8(10),
				"event_data": map[string]any{
					"key": "value",
				},
				"event_id":      uint16(20),
				"flags":         uint16(30),
				"keywords":      uint64(40),
				"level":         uint8(17),
				"logfile":       "C:\\TestFile",
				"opcode":        uint8(50),
				"process_id":    uint32(60),
				"provider_guid": "{12345678-1234-1234-1234-123456789ABC}",
				"provider_name": "TestProvider",
				"session":       "Elastic-TestProvider",
				"severity":      "unknown",
				"task":          uint16(70),
				"thread_id":     uint32(80),
				"timestamp":     "2024-02-05T22:03:09.035Z",
				"version":       uint8(90),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			winlog := buildEvent(tt.data, tt.header, tt.session, tt.cfg)

			// Parse the expected time string to time.Time
			expectedTime, err := time.Parse(time.RFC3339, tt.expected["timestamp"].(string))
			if err != nil {
				t.Fatalf("Failed to parse expected time string: %v", err)
			}

			winlogTime := winlog["timestamp"].(time.Time)

			// Convert the expected time to the same time zone before comparison
			expectedTime = expectedTime.In(winlogTime.Location())

			assert.Equal(t, tt.expected["activity_guid"], winlog["activity_guid"])
			assert.Equal(t, tt.expected["channel"], winlog["channel"])
			assert.Equal(t, tt.expected["event_data"], winlog["event_data"])
			assert.Equal(t, tt.expected["event_id"], winlog["event_id"])
			assert.Equal(t, tt.expected["flags"], winlog["flags"])
			assert.Equal(t, tt.expected["keywords"], winlog["keywords"])
			assert.Equal(t, tt.expected["level"], winlog["level"])
			assert.Equal(t, tt.expected["logfile"], winlog["logfile"])
			assert.Equal(t, tt.expected["opcode"], winlog["opcode"])
			assert.Equal(t, tt.expected["process_id"], winlog["process_id"])
			assert.Equal(t, tt.expected["provider_guid"], winlog["provider_guid"])
			assert.Equal(t, tt.expected["provider_name"], winlog["provider_name"])
			assert.Equal(t, tt.expected["session"], winlog["session"])
			assert.Equal(t, tt.expected["severity"], winlog["severity"])
			assert.Equal(t, tt.expected["task"], winlog["task"])
			assert.Equal(t, tt.expected["thread_id"], winlog["thread_id"])
			assert.Equal(t, expectedTime, winlogTime)
			assert.Equal(t, tt.expected["version"], winlog["version"])

		})
	}
}

func Test_convertFileTimeToGoTime(t *testing.T) {
	tests := []struct {
		name     string
		fileTime uint64
		want     time.Time
	}{
		{
			name:     "TestZeroValue",
			fileTime: 0,
			want:     time.Time{},
		},
		{
			name:     "TestUnixEpoch",
			fileTime: 116444736000000000, // January 1, 1970 (Unix epoch)
			want:     time.Unix(0, 0),
		},
		{
			name:     "TestActualDate",
			fileTime: 133515900000000000, // February 05, 2024, 7:00:00 AM
			want:     time.Date(2024, 02, 05, 7, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertFileTimeToGoTime(tt.fileTime)
			if !got.Equal(tt.want) {
				t.Errorf("convertFileTimeToGoTime() = %v, want %v", got, tt.want)
			}
		})
	}
}
