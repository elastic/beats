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

	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/windows"
)

func TestAttachToExistingSession_Error(t *testing.T) {
	// Mock implementation of controlTrace
	controlTrace := func(traceHandle uintptr,
		instanceName *uint16,
		properties *EventTraceProperties,
		controlCode uint32) error {
		return ERROR_WMI_INSTANCE_NOT_FOUND
	}

	// Create a Session instance
	session := &Session{
		Name:         "TestSession",
		properties:   &EventTraceProperties{},
		controlTrace: controlTrace,
	}

	err := session.AttachToExistingSession()
	assert.EqualError(t, err, "session is not running: The instance name passed was not recognized as valid by a WMI data provider.")
}

func TestAttachToExistingSession_Success(t *testing.T) {
	// Mock implementation of controlTrace
	controlTrace := func(traceHandle uintptr,
		instanceName *uint16,
		properties *EventTraceProperties,
		controlCode uint32) error {
		// Set a mock handler value
		properties.Wnode.Union1 = 12345
		return nil
	}

	// Create a Session instance with initialized Properties
	session := &Session{
		Name:         "TestSession",
		properties:   &EventTraceProperties{},
		controlTrace: controlTrace,
	}

	err := session.AttachToExistingSession()

	assert.NoError(t, err)
	assert.Equal(t, uintptr(12345), session.handler, "Handler should be set to the mock value")
}

func TestCreateRealtimeSession_StartTraceError(t *testing.T) {
	// Mock implementation of startTrace
	startTrace := func(traceHandle *uintptr,
		instanceName *uint16,
		properties *EventTraceProperties) error {
		return ERROR_ALREADY_EXISTS
	}

	// Create a Session instance
	session := &Session{
		Name:       "TestSession",
		properties: &EventTraceProperties{},
		startTrace: startTrace,
	}

	err := session.CreateRealtimeSession()
	assert.EqualError(t, err, "session already exists: Cannot create a file when that file already exists.")
}

func TestCreateRealtimeSession_EnableTraceError(t *testing.T) {
	// Mock implementations
	startTrace := func(traceHandle *uintptr,
		instanceName *uint16,
		properties *EventTraceProperties) error {
		*traceHandle = 12345 // Mock handler value
		return nil
	}

	enableTrace := func(traceHandle uintptr,
		providerId *windows.GUID,
		isEnabled uint32,
		level uint8,
		matchAnyKeyword uint64,
		matchAllKeyword uint64,
		enableProperty uint32,
		enableParameters *EnableTraceParameters) error {
		return ERROR_INVALID_PARAMETER
	}

	// Create a Session instance
	session := &Session{
		Name:        "TestSession",
		properties:  &EventTraceProperties{},
		startTrace:  startTrace,
		enableTrace: enableTrace,
		config: Config{
			Providers: []ProviderConfig{
				{
					Name: "Microsoft-Windows-Kernel-Process",
				},
			},
		},
	}

	err := session.CreateRealtimeSession()
	assert.EqualError(t, err, "invalid parameters when enabling session trace: The parameter is incorrect.")
}

func TestCreateRealtimeSession_Success(t *testing.T) {
	// Mock implementations
	startTrace := func(traceHandle *uintptr,
		instanceName *uint16,
		properties *EventTraceProperties) error {
		*traceHandle = 12345 // Mock handler value
		return nil
	}

	enableTrace := func(traceHandle uintptr,
		providerId *windows.GUID,
		isEnabled uint32,
		level uint8,
		matchAnyKeyword uint64,
		matchAllKeyword uint64,
		enableProperty uint32,
		enableParameters *EnableTraceParameters) error {
		return nil
	}

	// Create a Session instance
	session := &Session{
		Name:        "TestSession",
		properties:  &EventTraceProperties{},
		startTrace:  startTrace,
		enableTrace: enableTrace,
	}

	err := session.CreateRealtimeSession()

	assert.NoError(t, err)
	assert.Equal(t, uintptr(12345), session.handler, "Handler should be set to the mock value")
}

func TestStopSession_Error(t *testing.T) {
	// Mock implementation of closeTrace
	closeTrace := func(traceHandle uint64) error {
		return ERROR_INVALID_PARAMETER
	}

	// Create a Session instance
	session := &Session{
		Realtime:     true,
		NewSession:   true,
		handler:      1,     // Session handle from StartTrace
		traceHandler: 12345, // Example handler value
		properties:   &EventTraceProperties{},
		closeTrace:   closeTrace,
		controlTrace: func(uintptr, *uint16, *EventTraceProperties, uint32) error { return nil },
	}

	err := session.StopSession()
	assert.EqualError(t, err, "failed to close trace: The parameter is incorrect.")
}

func TestStopSession_Success(t *testing.T) {
	// Mock implementations
	closeTrace := func(traceHandle uint64) error {
		return nil
	}

	controlTrace := func(traceHandle uintptr,
		instanceName *uint16,
		properties *EventTraceProperties,
		controlCode uint32) error {
		// Set a mock handler value
		return nil
	}

	// Create a Session instance
	session := &Session{
		Realtime:     true,
		NewSession:   true,
		handler:      1,     // Session handle from StartTrace
		traceHandler: 12345, // Example handler value
		properties:   &EventTraceProperties{},
		closeTrace:   closeTrace,
		controlTrace: controlTrace,
	}

	err := session.StopSession()
	assert.NoError(t, err)
}

func TestStopSession_AlreadyStopped(t *testing.T) {
	// When another controller has already stopped the session, the STOP
	// control fails with ERROR_WMI_INSTANCE_NOT_FOUND. The session is in the
	// state we asked for, so that is not an error; anything else still is.
	tests := []struct {
		name    string
		stopErr error
		wantErr error
	}{
		{name: "session already gone", stopErr: ERROR_WMI_INSTANCE_NOT_FOUND, wantErr: nil},
		{name: "other stop failures are reported", stopErr: ERROR_ACCESS_DENIED, wantErr: ERROR_ACCESS_DENIED},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			closed := false
			session := &Session{
				Realtime:     true,
				NewSession:   true,
				handler:      1,
				traceHandler: 12345,
				properties:   &EventTraceProperties{},
				closeTrace: func(uint64) error {
					closed = true
					return nil
				},
				controlTrace: func(_ uintptr, _ *uint16, _ *EventTraceProperties, controlCode uint32) error {
					if controlCode == EVENT_TRACE_CONTROL_STOP {
						return test.stopErr
					}
					return nil
				},
			}

			err := session.StopSession()
			assert.ErrorIs(t, err, test.wantErr, "StopSession with STOP failing with %v", test.stopErr)
			assert.True(t, closed, "trace handle should be closed before the session is stopped")
		})
	}
}

func TestStopSession_NothingToStop(t *testing.T) {
	// Connecting failed before StartTrace or the attach QUERY succeeded, so
	// there is no session handle. StopSession is called as cleanup anyway and
	// must not issue controls against a session that does not exist.
	session := &Session{
		Realtime:   true,
		NewSession: true,
		properties: &EventTraceProperties{},
		closeTrace: func(uint64) error {
			t.Error("no trace was opened, so nothing should be closed")
			return nil
		},
		controlTrace: func(_ uintptr, _ *uint16, _ *EventTraceProperties, controlCode uint32) error {
			t.Errorf("no session was created, so control %d should not be sent", controlCode)
			return nil
		},
	}

	start := time.Now()
	assert.NoError(t, session.StopSession(), "stopping a session that was never created should succeed")
	assert.Less(t, time.Since(start), time.Second/2, "there was nothing to flush, so there should be no wait for flushed events")
}

func TestStopSession_NoWaitWhenSessionGone(t *testing.T) {
	// Another controller stopped the session, which is what brings the
	// reconnecting consumer here. The flush fails because the session is
	// gone, so there are no flushed events to wait for; waiting anyway would
	// add a second to every reconnect.
	closed := false
	session := &Session{
		Realtime:     true,
		NewSession:   true,
		handler:      1,
		traceHandler: 12345,
		properties:   &EventTraceProperties{},
		closeTrace: func(uint64) error {
			closed = true
			return nil
		},
		controlTrace: func(uintptr, *uint16, *EventTraceProperties, uint32) error {
			return ERROR_WMI_INSTANCE_NOT_FOUND
		},
	}

	start := time.Now()
	assert.NoError(t, session.StopSession(), "a session that is already gone is in the state we asked for")
	assert.Less(t, time.Since(start), time.Second/2, "a failed flush has nothing to wait for")
	assert.True(t, closed, "the trace handle should still be closed")
}

func TestStopSession_NoWaitWithoutConsumer(t *testing.T) {
	// The session exists but no trace was ever opened on it, so a flush has
	// nobody to deliver to and there is nothing to wait for.
	session := &Session{
		Realtime:   true,
		NewSession: true,
		handler:    1,
		properties: &EventTraceProperties{},
		closeTrace: func(uint64) error {
			t.Error("no trace was opened, so nothing should be closed")
			return nil
		},
		controlTrace: func(uintptr, *uint16, *EventTraceProperties, uint32) error { return nil },
	}

	start := time.Now()
	assert.NoError(t, session.StopSession(), "stopping a session without a consumer should succeed")
	assert.Less(t, time.Since(start), time.Second/2, "no consumer is processing, so there should be no wait for flushed events")
}
