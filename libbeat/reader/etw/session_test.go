// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build windows

package etw

import (
	"testing"
	"time"
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
				Providers: []ProviderConfig{
					{
						Name: "Provider1",
					},
				},
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
				Providers: []ProviderConfig{
					{
						GUID: "12345",
					},
				},
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

	conf := Config{Providers: []ProviderConfig{{Name: "Provider1"}}}
	expectedGUID := windows.GUID{Data1: 0x12345678, Data2: 0x1234, Data3: 0x5678, Data4: [8]byte{0x9A, 0xBC, 0xDE, 0xF0, 0x12, 0x34, 0x56, 0x78}}

	guid, err := getProviderGUID(conf.Providers[0])
	assert.NoError(t, err)
	assert.Equal(t, expectedGUID, guid, "The GUID should match the mock GUID")
}

func TestSetSessionGUID_ProviderGUID(t *testing.T) {
	// Example GUID string
	guidString := "{12345678-1234-5678-1234-567812345678}"

	// Configuration with a set ProviderGUID
	conf := Config{Providers: []ProviderConfig{{GUID: guidString}}}

	// Expected GUID based on the GUID string
	expectedGUID := windows.GUID{Data1: 0x12345678, Data2: 0x1234, Data3: 0x5678, Data4: [8]byte{0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78}}

	guid, err := getProviderGUID(conf.Providers[0])

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
			props := newSessionProperties(tc.sessionName, Config{})

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

func TestNewSession_AttachSession(t *testing.T) {
	// Test case
	conf := Config{
		Session:     "Session1",
		SessionName: "TestSession",
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
		Logfile: "LogFile1.etl",
	}
	session, err := NewSession(conf)

	assert.NoError(t, err)
	assert.Equal(t, "LogFile1.etl", session.Name, "SessionName should match expected value")
	assert.Equal(t, false, session.NewSession)
	assert.Equal(t, false, session.Realtime)
	assert.Nil(t, session.properties)
}

func TestSession_Reset(t *testing.T) {
	callback := func(*EventRecord) uintptr { return 0 }

	t.Run("realtime session gets fresh handles and properties", func(t *testing.T) {
		conf := Config{SessionName: "TestSession", BufferSize: 128}
		session, err := NewSession(conf)
		assert.NoError(t, err, "NewSession should not fail")
		session.Callback = callback

		// Simulate what StartTrace, ControlTrace, OpenTrace and StopSession
		// leave behind.
		session.handler = 12345
		session.traceHandler = 67890
		session.stopping = true
		session.properties.Wnode.Guid = windows.GUID{Data1: 0xdeadbeef}
		session.properties.Wnode.Union1 = 12345
		session.properties.NumberOfBuffers = 42
		before := session.properties

		session.Reset()

		assert.Equal(t, uintptr(0), session.handler, "session handle should be cleared")
		assert.Equal(t, uint64(0), session.traceHandler, "trace handle should be cleared")
		assert.False(t, session.stopping, "the stop from the previous run must not carry over or the next consumer would exit at once")
		assert.NotSame(t, before, session.properties, "properties should be rebuilt, not reused")
		assert.Equal(t, windows.GUID{}, session.properties.Wnode.Guid, "rebuilt properties should not carry the old session GUID")
		assert.Equal(t, uint64(0), session.properties.Wnode.Union1, "rebuilt properties should not carry the old handle")
		assert.Equal(t, uint32(0), session.properties.NumberOfBuffers, "rebuilt properties should not carry live buffer counts")
		assert.Equal(t, uint32(128), session.properties.BufferSize, "rebuilt properties should keep the configured buffer size")
		assert.Equal(t, before.Wnode.BufferSize, session.properties.Wnode.BufferSize, "rebuilt properties should be the same size")
		assert.NotNil(t, session.Callback, "Reset must keep the callback so NewCallback can dedupe it")
	})

	t.Run("logfile session keeps nil properties", func(t *testing.T) {
		session, err := NewSession(Config{Logfile: "LogFile1.etl"})
		assert.NoError(t, err, "NewSession should not fail")
		session.traceHandler = 67890

		session.Reset()

		assert.Equal(t, uint64(0), session.traceHandler, "trace handle should be cleared")
		assert.Nil(t, session.properties, "logfile sessions have no properties to rebuild")
	})
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

// stoppableSession returns a realtime session whose mocks record closed trace
// handles in *closed and run processTrace, so tests can drive the ordering of
// StartConsumer against StopSession.
func stoppableSession(closed *[]uint64, processTrace func(*uint64, uint32, *FileTime, *FileTime) error) *Session {
	return &Session{
		Name:       "TestSession",
		Realtime:   true,
		Callback:   func(*EventRecord) uintptr { return 1 },
		properties: &EventTraceProperties{},
		openTrace:  func(*EventTraceLogfile) (uint64, error) { return 12345, nil },
		closeTrace: func(h uint64) error {
			*closed = append(*closed, h)
			return nil
		},
		controlTrace: func(uintptr, *uint16, *EventTraceProperties, uint32) error { return nil },
		processTrace: processTrace,
	}
}

func TestStartConsumer_StopBeforeOpen(t *testing.T) {
	// StopSession ran before StartConsumer had a trace handle, so it had
	// nothing to close. StartConsumer must notice the stop once the trace is
	// open and close it itself, rather than block in ProcessTrace on a
	// session that nobody is going to end.
	var closed []uint64
	session := stoppableSession(&closed, func(*uint64, uint32, *FileTime, *FileTime) error {
		t.Error("ProcessTrace must not be called after StopSession")
		return nil
	})

	assert.NoError(t, session.StopSession(), "stopping before the trace is open should succeed")
	assert.Empty(t, closed, "there was no trace handle for StopSession to close yet")

	assert.NoError(t, session.StartConsumer(), "a consumer stopped before it opened should finish cleanly")
	assert.Equal(t, []uint64{12345}, closed, "StartConsumer should close the handle that StopSession could not")
}

func TestStartConsumer_AfterStopAndReset(t *testing.T) {
	// The reconnect sequence: the previous run was stopped, the session was
	// reset, and the consumer is started again. The old stop must not make
	// the new consumer exit before it has processed anything.
	var closed []uint64
	processed := 0
	session := stoppableSession(&closed, func(*uint64, uint32, *FileTime, *FileTime) error {
		processed++
		return nil
	})

	assert.NoError(t, session.StopSession(), "stopping the previous run should succeed")
	session.Reset()

	assert.NoError(t, session.StartConsumer(), "the consumer should run again after a reset")
	assert.Equal(t, 1, processed, "ProcessTrace should run for the new consumer")
	assert.Empty(t, closed, "nothing should have been closed while the new consumer ran")

	assert.NoError(t, session.StopSession(), "stopping the new run should succeed")
	assert.Equal(t, []uint64{12345}, closed, "StopSession should close the new consumer's handle")
}

func TestStartConsumer_StopWhileProcessing(t *testing.T) {
	// The common shutdown: ProcessTrace is blocked and StopSession, from
	// another goroutine, closes the trace to make it return. Under -race this
	// also checks that the two goroutines do not touch the handle unguarded.
	started := make(chan struct{})
	released := make(chan struct{})
	var closed []uint64
	session := stoppableSession(&closed, func(*uint64, uint32, *FileTime, *FileTime) error {
		close(started)
		<-released // ProcessTrace returns once the trace has been closed.
		return nil
	})
	closeTrace := session.closeTrace
	session.closeTrace = func(h uint64) error {
		err := closeTrace(h)
		close(released)
		return err
	}

	done := make(chan error, 1)
	go func() { done <- session.StartConsumer() }()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("StartConsumer never reached ProcessTrace")
	}

	assert.NoError(t, session.StopSession(), "StopSession while processing should succeed")
	select {
	case err := <-done:
		assert.NoError(t, err, "StartConsumer should return once the trace is closed")
	case <-time.After(5 * time.Second):
		t.Fatal("StartConsumer did not return after StopSession")
	}
	assert.Equal(t, []uint64{12345}, closed, "StopSession should have closed the open trace handle")
}

func TestStartConsumer_ConcurrentStop(t *testing.T) {
	// StartConsumer and StopSession run with no ordering between them, as
	// they do when the input is cancelled just as its consumer starts. Either
	// order must end with the trace closed exactly once and StartConsumer
	// returning; a stop that wins the race used to leave ProcessTrace blocked
	// for good. Under -race this also covers the handle access that the
	// ordered test above cannot, since waiting for ProcessTrace to start
	// serialises the two goroutines.
	released := make(chan struct{})
	var closed []uint64
	session := stoppableSession(&closed, func(*uint64, uint32, *FileTime, *FileTime) error {
		<-released // ProcessTrace returns once the trace has been closed.
		return nil
	})
	closeTrace := session.closeTrace
	session.closeTrace = func(h uint64) error {
		err := closeTrace(h)
		close(released)
		return err
	}

	done := make(chan error, 1)
	go func() { done <- session.StartConsumer() }()
	assert.NoError(t, session.StopSession(), "StopSession racing StartConsumer should succeed")
	select {
	case err := <-done:
		assert.NoError(t, err, "StartConsumer should return whichever side closed the trace")
	case <-time.After(5 * time.Second):
		t.Fatal("StartConsumer did not return; the stop was lost")
	}
	assert.Equal(t, []uint64{12345}, closed, "the trace handle should be closed exactly once")
}
