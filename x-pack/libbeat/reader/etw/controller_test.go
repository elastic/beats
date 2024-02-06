// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package etw

import (
	"testing"

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
		traceHandler: 12345, // Example handler value
		properties:   &EventTraceProperties{},
		closeTrace:   closeTrace,
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
		traceHandler: 12345, // Example handler value
		properties:   &EventTraceProperties{},
		closeTrace:   closeTrace,
		controlTrace: controlTrace,
	}

	err := session.StopSession()
	assert.NoError(t, err)
}
