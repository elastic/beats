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

type MockETWSessionOperator struct {
	// Fields to store function implementations that tests can customize
	NewSessionFunc              func(config config) (*etw.Session, error)
	AttachToExistingSessionFunc func(session *etw.Session) error
	CreateRealtimeSessionFunc   func(session *etw.Session) error
	StartConsumerFunc           func(session *etw.Session) error
	StopSessionFunc             func(session *etw.Session) error
}

func (m *MockETWSessionOperator) NewSession(config config) (*etw.Session, error) {
	if m.NewSessionFunc != nil {
		return m.NewSessionFunc(config)
	}
	return nil, nil
}

func (m *MockETWSessionOperator) AttachToExistingSession(session *etw.Session) error {
	if m.AttachToExistingSessionFunc != nil {
		return m.AttachToExistingSessionFunc(session)
	}
	return nil
}

func (m *MockETWSessionOperator) CreateRealtimeSession(session *etw.Session) error {
	if m.CreateRealtimeSessionFunc != nil {
		return m.CreateRealtimeSessionFunc(session)
	}
	return nil
}

func (m *MockETWSessionOperator) StartConsumer(session *etw.Session) error {
	if m.StartConsumerFunc != nil {
		return m.StartConsumerFunc(session)
	}
	return nil
}

func (m *MockETWSessionOperator) StopSession(session *etw.Session) error {
	if m.StopSessionFunc != nil {
		return m.StopSessionFunc(session)
	}
	return nil
}

func Test_RunEtwInput_NewSessionError(t *testing.T) {
	// Mocks
	mockOperator := &MockETWSessionOperator{}

	// Setup the mock behavior for NewSession
	mockOperator.NewSessionFunc = func(config config) (*etw.Session, error) {
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
	mockOperator := &MockETWSessionOperator{}

	// Setup the mock behavior for NewSession
	mockOperator.NewSessionFunc = func(config config) (*etw.Session, error) {
		mockSession := &etw.Session{
			Name:       "MySession",
			Realtime:   true,
			NewSession: false}
		return mockSession, nil
	}
	// Setup the mock behavior for AttachToExistingSession
	mockOperator.AttachToExistingSessionFunc = func(session *etw.Session) error {
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
	mockOperator := &MockETWSessionOperator{}

	// Setup the mock behavior for NewSession
	mockOperator.NewSessionFunc = func(config config) (*etw.Session, error) {
		mockSession := &etw.Session{
			Name:       "MySession",
			Realtime:   true,
			NewSession: true}
		return mockSession, nil
	}
	// Setup the mock behavior for AttachToExistingSession
	mockOperator.AttachToExistingSessionFunc = func(session *etw.Session) error {
		return nil
	}
	// Setup the mock behavior for CreateRealtimeSession
	mockOperator.CreateRealtimeSessionFunc = func(session *etw.Session) error {
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
	mockOperator := &MockETWSessionOperator{}

	// Setup the mock behavior for NewSession
	mockOperator.NewSessionFunc = func(config config) (*etw.Session, error) {
		mockSession := &etw.Session{
			Name:       "MySession",
			Realtime:   true,
			NewSession: true}
		return mockSession, nil
	}
	// Setup the mock behavior for AttachToExistingSession
	mockOperator.AttachToExistingSessionFunc = func(session *etw.Session) error {
		return nil
	}
	// Setup the mock behavior for CreateRealtimeSession
	mockOperator.CreateRealtimeSessionFunc = func(session *etw.Session) error {
		return nil
	}
	// Setup the mock behavior for StartConsumer
	mockOperator.StartConsumerFunc = func(session *etw.Session) error {
		return fmt.Errorf("mock error")
	}
	// Setup the mock behavior for StopSession
	mockOperator.StopSessionFunc = func(session *etw.Session) error {
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
	mockOperator := &MockETWSessionOperator{}

	// Setup the mock behavior for NewSession
	mockOperator.NewSessionFunc = func(config config) (*etw.Session, error) {
		mockSession := &etw.Session{
			Name:       "MySession",
			Realtime:   true,
			NewSession: true}
		return mockSession, nil
	}
	// Setup the mock behavior for AttachToExistingSession
	mockOperator.AttachToExistingSessionFunc = func(session *etw.Session) error {
		return nil
	}
	// Setup the mock behavior for CreateRealtimeSession
	mockOperator.CreateRealtimeSessionFunc = func(session *etw.Session) error {
		return nil
	}
	// Setup the mock behavior for StartConsumer
	mockOperator.StartConsumerFunc = func(session *etw.Session) error {
		return nil
	}
	// Setup the mock behavior for StopSession
	mockOperator.StopSessionFunc = func(session *etw.Session) error {
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

func Test_fillEventHeader(t *testing.T) {
	tests := []struct {
		name     string
		header   etw.EventHeader
		expected map[string]interface{}
	}{
		{
			name: "TestStandardHeader",
			header: etw.EventHeader{
				Size:          100,
				HeaderType:    10,
				Flags:         20,
				EventProperty: 30,
				ThreadId:      40,
				ProcessId:     50,
				TimeStamp:     133516441890350000,
				ProviderId: windows.GUID{
					Data1: 0x12345678,
					Data2: 0x1234,
					Data3: 0x1234,
					Data4: [8]byte{0x12, 0x34, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc},
				},
				EventDescriptor: etw.EventDescriptor{
					Id:      60,
					Version: 70,
					Channel: 80,
					Level:   1, // Critical
					Opcode:  90,
					Task:    100,
					Keyword: 110,
				},
				Time: 120,
				ActivityId: windows.GUID{
					Data1: 0x12345678,
					Data2: 0x1234,
					Data3: 0x1234,
					Data4: [8]byte{0x12, 0x34, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc},
				},
			},
			expected: map[string]interface{}{
				"size":           uint16(100),
				"type":           uint16(10),
				"flags":          uint16(20),
				"event_property": uint16(30),
				"thread_id":      uint32(40),
				"process_id":     uint32(50),
				"timestamp":      "2024-02-05T22:03:09.035Z",
				"provider_guid":  "{12345678-1234-1234-1234-123456789ABC}",
				"event_id":       uint16(60),
				"event_version":  uint8(70),
				"channel":        uint8(80),
				"level":          uint8(1),
				"severity":       "critical",
				"opcode":         uint8(90),
				"task":           uint16(100),
				"keyword":        uint64(110),
				"time":           int64(120),
				"activity_guid":  "{12345678-1234-1234-1234-123456789ABC}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := fillEventHeader(tt.header)
			assert.Equal(t, tt.expected["size"], header["size"])
			assert.Equal(t, tt.expected["type"], header["type"])
			assert.Equal(t, tt.expected["flags"], header["flags"])
			assert.Equal(t, tt.expected["event_property"], header["event_property"])
			assert.Equal(t, tt.expected["thread_id"], header["thread_id"])
			assert.Equal(t, tt.expected["process_id"], header["process_id"])
			assert.Equal(t, tt.expected["provider_guid"], header["provider_guid"])
			assert.Equal(t, tt.expected["event_id"], header["event_id"])
			assert.Equal(t, tt.expected["event_version"], header["event_version"])
			assert.Equal(t, tt.expected["channel"], header["channel"])
			assert.Equal(t, tt.expected["level"], header["level"])
			assert.Equal(t, tt.expected["severity"], header["severity"])
			assert.Equal(t, tt.expected["opcode"], header["opcode"])
			assert.Equal(t, tt.expected["task"], header["task"])
			assert.Equal(t, tt.expected["keyword"], header["keyword"])
			assert.Equal(t, tt.expected["time"], header["time"])
			assert.Equal(t, tt.expected["activity_guid"], header["activity_guid"])
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

func Test_fillEventMetadata(t *testing.T) {
	tests := []struct {
		name     string
		session  *etw.Session
		cfg      config
		expected map[string]interface{}
	}{
		// Test Provider Name and GUID from config
		{
			name: "TestProviderNameAndGUIDFromConfig",
			session: &etw.Session{
				GUID: windows.GUID{},
				Name: "SessionName",
			},
			cfg: config{
				ProviderName: "TestProvider",
				ProviderGUID: "{12345678-1234-1234-1234-123456789ABC}",
			},
			expected: map[string]interface{}{
				"provider_name": "TestProvider",
				"provider_guid": "{12345678-1234-1234-1234-123456789ABC}",
				"session":       "SessionName",
			},
		},
		// Test Provider GUID from session if not available in config
		{
			name: "TestProviderGUIDFromSession",
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
			expected: map[string]interface{}{
				"provider_name": "TestProvider",
				"provider_guid": "{12345678-1234-1234-1234-123456789ABC}",
				"session":       "Elastic-TestProvider",
			},
		},
		// Test Logfile and Session Information
		{
			name: "TestLogfileAndSessionInfo",
			session: &etw.Session{
				GUID: windows.GUID{},
				Name: "SessionName",
			},
			cfg: config{
				Logfile:     "C:\\Logs\\test.log",
				Session:     "TestSession",
				SessionName: "DifferentSessionName",
			},
			expected: map[string]interface{}{
				"logfile": "C:\\Logs\\test.log",
				"session": "TestSession",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fillEventMetadata(tt.session, tt.cfg)
			assert.Equal(t, tt.expected, result, "fillEventMetadata() should match the expected output")
		})
	}
}
