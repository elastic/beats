// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package etw

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetHandler_Error(t *testing.T) {
	// Backup and defer restoration of the original function
	originalFunc := ControlTraceFunc
	t.Cleanup(func() {
		ControlTraceFunc = originalFunc
	})

	// Mock implementation
	ControlTraceFunc = func(traceHandle uintptr,
		instanceName *uint16,
		properties *EventTraceProperties,
		controlCode uint32) error {
		return ERROR_WMI_INSTANCE_NOT_FOUND
	}

	// Create a Session instance
	session := &Session{
		Name:       "TestSession",
		Properties: &EventTraceProperties{},
	}

	err := session.GetHandler()
	assert.EqualError(t, err, "session is not running")
}

func TestGetHandler_Success(t *testing.T) {
	// Backup original function and defer restoration
	originalFunc := ControlTraceFunc
	t.Cleanup(func() {
		ControlTraceFunc = originalFunc
	})

	// Mock implementation
	ControlTraceFunc = func(traceHandle uintptr,
		instanceName *uint16,
		properties *EventTraceProperties,
		controlCode uint32) error {
		// Set a mock handler value
		properties.Wnode.Union1 = 12345
		return nil
	}

	// Create a Session instance with initialized Properties
	session := &Session{
		Name:       "TestSession",
		Properties: &EventTraceProperties{},
	}

	err := session.GetHandler()

	assert.NoError(t, err)
	assert.Equal(t, uintptr(12345), session.Handler, "Handler should be set to the mock value")
}

func TestCreateRealtimeSession_StartTraceError(t *testing.T) {
	// Backup original functions and defer restoration
	originalStartTraceFunc := StartTraceFunc
	t.Cleanup(func() {
		StartTraceFunc = originalStartTraceFunc
	})

	// Mock implementations
	StartTraceFunc = func(traceHandle *uintptr,
		instanceName *uint16,
		properties *EventTraceProperties) error {
		return ERROR_ALREADY_EXISTS
	}

	// Create a Session instance
	session := &Session{
		Name:       "TestSession",
		Properties: &EventTraceProperties{},
	}

	err := session.CreateRealtimeSession()
	assert.EqualError(t, err, "session already exists: Cannot create a file when that file already exists.")
}

func TestCreateRealtimeSession_EnableTraceError(t *testing.T) {
	// Backup original functions and defer restoration
	originalStartTraceFunc := StartTraceFunc
	originalEnableTraceFunc := EnableTraceFunc
	t.Cleanup(func() {
		StartTraceFunc = originalStartTraceFunc
		EnableTraceFunc = originalEnableTraceFunc
	})

	// Mock implementations
	StartTraceFunc = func(traceHandle *uintptr,
		instanceName *uint16,
		properties *EventTraceProperties) error {
		*traceHandle = 12345 // Mock handler value
		return nil
	}

	EnableTraceFunc = func(traceHandle uintptr,
		providerId *GUID,
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
		Name:       "TestSession",
		Properties: &EventTraceProperties{},
	}

	err := session.CreateRealtimeSession()
	assert.EqualError(t, err, "invalid parameters when enabling session trace: The parameter is incorrect.")
}

func TestCreateRealtimeSession_Success(t *testing.T) {
	// Backup original functions and defer restoration
	originalStartTraceFunc := StartTraceFunc
	originalEnableTraceFunc := EnableTraceFunc
	t.Cleanup(func() {
		StartTraceFunc = originalStartTraceFunc
		EnableTraceFunc = originalEnableTraceFunc
	})

	// Mock implementations
	StartTraceFunc = func(traceHandle *uintptr,
		instanceName *uint16,
		properties *EventTraceProperties) error {
		*traceHandle = 12345 // Mock handler value
		return nil
	}

	EnableTraceFunc = func(traceHandle uintptr,
		providerId *GUID,
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
		Name:       "TestSession",
		Properties: &EventTraceProperties{},
	}

	err := session.CreateRealtimeSession()

	assert.NoError(t, err)
	assert.Equal(t, uintptr(12345), session.Handler, "Handler should be set to the mock value")
}

func TestStopSession_Error(t *testing.T) {
	// Backup original function and defer restoration
	originalFunc := CloseTraceFunc
	t.Cleanup(func() {
		CloseTraceFunc = originalFunc
	})

	// Override with the mock function
	CloseTraceFunc = func(traceHandle uint64) error {
		return ERROR_INVALID_PARAMETER
	}

	// Create a Session instance
	session := &Session{
		Realtime:     true,
		NewSession:   true,
		TraceHandler: 12345, // Example handler value
		Properties:   &EventTraceProperties{},
	}

	err := session.StopSession()
	assert.EqualError(t, err, "failed to close trace: The parameter is incorrect.")
}

func TestStopSession_Success(t *testing.T) {
	// Backup original function and defer restoration
	originalCloseTraceFunc := CloseTraceFunc
	originalControlTraceFunc := ControlTraceFunc
	t.Cleanup(func() {
		CloseTraceFunc = originalCloseTraceFunc
		ControlTraceFunc = originalControlTraceFunc
	})

	// Override with the mock function
	CloseTraceFunc = func(traceHandle uint64) error {
		return nil
	}

	// Mock implementation
	ControlTraceFunc = func(traceHandle uintptr,
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
		TraceHandler: 12345, // Example handler value
		Properties:   &EventTraceProperties{},
	}

	err := session.StopSession()
	assert.NoError(t, err)
}
