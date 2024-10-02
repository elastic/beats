// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package etw

import (
	"fmt"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/windows"
)

// TestSetSessionName tests the setSessionName function with various configurations.
func TestSetSessionName(t *testing.T) {
	testCases := []struct {
		name         string
		config       Config
		expectedName string
	}{
		{
			name: "ProviderNameSet",
			config: Config{
				ProviderName: "Provider1",
			},
			expectedName: "Elastic-Provider1",
		},
		{
			name: "SessionNameSet",
			config: Config{
				SessionName: "Session1",
			},
			expectedName: "Session1",
		},
		{
			name: "LogFileSet",
			config: Config{
				Logfile: "LogFile1.etl",
			},
			expectedName: "LogFile1.etl",
		},
		{
			name: "FallbackToProviderGUID",
			config: Config{
				ProviderGUID: "12345",
			},
			expectedName: "Elastic-12345",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sessionName := setSessionName(tc.config)
			assert.Equal(t, tc.expectedName, sessionName, "The session name should be correctly determined")
		})
	}
}

func mockGUIDFromProviderName(providerName string) (windows.GUID, error) {
	// Return a mock GUID regardless of the input
	return windows.GUID{Data1: 0x12345678, Data2: 0x1234, Data3: 0x5678, Data4: [8]byte{0x9A, 0xBC, 0xDE, 0xF0, 0x12, 0x34, 0x56, 0x78}}, nil
}

func TestSetSessionGUID_ProviderName(t *testing.T) {
	// Defer restoration of original function
	t.Cleanup(func() {
		guidFromProviderNameFunc = guidFromProviderName
	})

	// Replace with mock function
	guidFromProviderNameFunc = mockGUIDFromProviderName

	conf := Config{ProviderName: "Provider1"}
	expectedGUID := windows.GUID{Data1: 0x12345678, Data2: 0x1234, Data3: 0x5678, Data4: [8]byte{0x9A, 0xBC, 0xDE, 0xF0, 0x12, 0x34, 0x56, 0x78}}

	guid, err := setSessionGUID(conf)
	assert.NoError(t, err)
	assert.Equal(t, expectedGUID, guid, "The GUID should match the mock GUID")
}

func TestSetSessionGUID_ProviderGUID(t *testing.T) {
	// Example GUID string
	guidString := "{12345678-1234-5678-1234-567812345678}"

	// Configuration with a set ProviderGUID
	conf := Config{ProviderGUID: guidString}

	// Expected GUID based on the GUID string
	expectedGUID := windows.GUID{Data1: 0x12345678, Data2: 0x1234, Data3: 0x5678, Data4: [8]byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78}}

	guid, err := setSessionGUID(conf)

	assert.NoError(t, err)
	assert.Equal(t, expectedGUID, guid, "The GUID should match the expected value")
}

func TestGetTraceLevel(t *testing.T) {
	testCases := []struct {
		name         string
		level        string
		expectedCode uint8
	}{
		{"CriticalLevel", "critical", TRACE_LEVEL_CRITICAL},
		{"ErrorLevel", "error", TRACE_LEVEL_ERROR},
		{"WarningLevel", "warning", TRACE_LEVEL_WARNING},
		{"InformationLevel", "information", TRACE_LEVEL_INFORMATION},
		{"VerboseLevel", "verbose", TRACE_LEVEL_VERBOSE},
		{"DefaultLevel", "unknown", TRACE_LEVEL_INFORMATION}, // Default case
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := getTraceLevel(tc.level)
			assert.Equal(t, tc.expectedCode, result, "Trace level code should match the expected value")
		})
	}
}

func TestNewSessionProperties(t *testing.T) {
	testCases := []struct {
		name         string
		sessionName  string
		expectedSize uint32
	}{
		{"EmptyName", "", 2 + uint32(unsafe.Sizeof(EventTraceProperties{}))},
		{"NormalName", "Session1", 18 + uint32(unsafe.Sizeof(EventTraceProperties{}))},
		// Additional test cases can be added here
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			props := newSessionProperties(tc.sessionName)

			assert.Equal(t, tc.expectedSize, props.Wnode.BufferSize, "BufferSize should match expected value")
			assert.Equal(t, windows.GUID{}, props.Wnode.Guid, "GUID should be empty")
			assert.Equal(t, uint32(1), props.Wnode.ClientContext, "ClientContext should be 1")
			assert.Equal(t, uint32(WNODE_FLAG_TRACED_GUID), props.Wnode.Flags, "Flags should match WNODE_FLAG_TRACED_GUID")
			assert.Equal(t, uint32(EVENT_TRACE_REAL_TIME_MODE), props.LogFileMode, "LogFileMode should be set to real-time")
			assert.Equal(t, uint32(0), props.LogFileNameOffset, "LogFileNameOffset should be 0")
			assert.Equal(t, uint32(64), props.BufferSize, "BufferSize should be 64")
			assert.Equal(t, uint32(unsafe.Sizeof(EventTraceProperties{})), props.LoggerNameOffset, "LoggerNameOffset should be the size of EventTraceProperties")
		})
	}
}

func TestNewSession_ProviderName(t *testing.T) {
	// Defer restoration of original function
	t.Cleanup(func() {
		setSessionGUIDFunc = setSessionGUID
	})

	// Override setSessionGUIDFunc with mock
	setSessionGUIDFunc = func(conf Config) (windows.GUID, error) {
		return windows.GUID{
			Data1: 0x12345678,
			Data2: 0x1234,
			Data3: 0x5678,
			Data4: [8]byte{0x9A, 0xBC, 0xDE, 0xF0, 0x12, 0x34, 0x56, 0x78},
		}, nil
	}

	expectedGUID := windows.GUID{
		Data1: 0x12345678,
		Data2: 0x1234,
		Data3: 0x5678,
		Data4: [8]byte{0x9A, 0xBC, 0xDE, 0xF0, 0x12, 0x34, 0x56, 0x78},
	}

	conf := Config{
		ProviderName:    "Provider1",
		SessionName:     "Session1",
		TraceLevel:      "warning",
		MatchAnyKeyword: 0xffffffffffffffff,
		MatchAllKeyword: 0,
	}
	session, err := NewSession(conf)

	assert.NoError(t, err)
	assert.Equal(t, "Session1", session.Name, "SessionName should match expected value")
	assert.Equal(t, expectedGUID, session.GUID, "The GUID in the session should match the expected GUID")
	assert.Equal(t, uint8(3), session.traceLevel, "TraceLevel should be 3 (warning)")
	assert.Equal(t, true, session.NewSession)
	assert.Equal(t, true, session.Realtime)
	assert.NotNil(t, session.properties)
}

func TestNewSession_GUIDError(t *testing.T) {
	// Defer restoration of original function
	t.Cleanup(func() {
		setSessionGUIDFunc = setSessionGUID
	})

	// Override setSessionGUIDFunc with mock
	setSessionGUIDFunc = func(conf Config) (windows.GUID, error) {
		// Return an empty GUID and an error
		return windows.GUID{}, fmt.Errorf("mock error")
	}

	conf := Config{
		ProviderName:    "Provider1",
		SessionName:     "Session1",
		TraceLevel:      "warning",
		MatchAnyKeyword: 0xffffffffffffffff,
		MatchAllKeyword: 0,
	}
	session, err := NewSession(conf)

	assert.EqualError(t, err, "error when initializing session 'Session1': mock error")
	assert.Nil(t, session)

}

func TestNewSession_AttachSession(t *testing.T) {
	// Test case
	conf := Config{
		Session:         "Session1",
		SessionName:     "TestSession",
		TraceLevel:      "verbose",
		MatchAnyKeyword: 0xffffffffffffffff,
		MatchAllKeyword: 0,
	}
	session, err := NewSession(conf)

	assert.NoError(t, err)
	assert.Equal(t, "Session1", session.Name, "SessionName should match expected value")
	assert.Equal(t, false, session.NewSession)
	assert.Equal(t, true, session.Realtime)
	assert.NotNil(t, session.properties)
}

func TestNewSession_Logfile(t *testing.T) {
	// Test case
	conf := Config{
		Logfile:         "LogFile1.etl",
		TraceLevel:      "verbose",
		MatchAnyKeyword: 0xffffffffffffffff,
		MatchAllKeyword: 0,
	}
	session, err := NewSession(conf)

	assert.NoError(t, err)
	assert.Equal(t, "LogFile1.etl", session.Name, "SessionName should match expected value")
	assert.Equal(t, false, session.NewSession)
	assert.Equal(t, false, session.Realtime)
	assert.Nil(t, session.properties)
}

func TestStartConsumer_CallbackNull(t *testing.T) {
	// Create a Session instance
	session := &Session{
		Name:           "TestSession",
		Realtime:       false,
		BufferCallback: nil,
		Callback:       nil,
	}

	err := session.StartConsumer()
	assert.EqualError(t, err, "error loading callback")
}

func TestStartConsumer_OpenTraceError(t *testing.T) {
	// Mock implementation of openTrace
	openTrace := func(elf *EventTraceLogfile) (uint64, error) {
		return 0, ERROR_ACCESS_DENIED // Mock a valid session handler
	}

	// Create a Session instance
	session := &Session{
		Name:           "TestSession",
		Realtime:       false,
		BufferCallback: nil,
		Callback: func(*EventRecord) uintptr {
			return 1
		},
		openTrace: openTrace,
	}

	err := session.StartConsumer()
	assert.EqualError(t, err, "access denied when opening trace: Access is denied.")
}

func TestStartConsumer_ProcessTraceError(t *testing.T) {
	// Mock implementations
	openTrace := func(elf *EventTraceLogfile) (uint64, error) {
		return 12345, nil // Mock a valid session handler
	}

	processTrace := func(handleArray *uint64, handleCount uint32, startTime *FileTime, endTime *FileTime) error {
		return ERROR_INVALID_PARAMETER
	}

	// Create a Session instance
	session := &Session{
		Name:           "TestSession",
		Realtime:       true,
		BufferCallback: nil,
		Callback: func(*EventRecord) uintptr {
			return 1
		},
		openTrace:    openTrace,
		processTrace: processTrace,
	}

	err := session.StartConsumer()
	assert.EqualError(t, err, "failed to process trace: The parameter is incorrect.")
}

func TestStartConsumer_Success(t *testing.T) {
	// Mock implementations
	openTrace := func(elf *EventTraceLogfile) (uint64, error) {
		return 12345, nil // Mock a valid session handler
	}

	processTrace := func(handleArray *uint64, handleCount uint32, startTime *FileTime, endTime *FileTime) error {
		return nil
	}

	// Create a Session instance
	session := &Session{
		Name:           "TestSession",
		Realtime:       true,
		BufferCallback: nil,
		Callback: func(*EventRecord) uintptr {
			return 1
		},
		openTrace:    openTrace,
		processTrace: processTrace,
	}

	err := session.StartConsumer()
	assert.NoError(t, err)
	assert.Equal(t, uint64(12345), session.traceHandler, "traceHandler should be set to the mock value")
}
