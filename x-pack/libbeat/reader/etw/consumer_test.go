// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package etw

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStartConsumer_OpenTraceError(t *testing.T) {
	// Backup original functions and defer restoration
	originalOpenTraceFunc := OpenTraceFunc
	defer func() {
		OpenTraceFunc = originalOpenTraceFunc
	}()

	// Mock implementations
	OpenTraceFunc = func(elf *EventTraceLogfile) (uint64, error) {
		return 0, ERROR_ACCESS_DENIED // Mock a valid session handler
	}

	// Create a Session instance
	session := &Session{
		Name:           "TestSession",
		Realtime:       false,
		BufferCallback: uintptr(0),
		Callback:       uintptr(0),
	}

	// Test StartConsumer
	err := session.StartConsumer()

	// Assertions
	assert.EqualError(t, err, "access denied when opening trace: Access is denied.")
}

func TestStartConsumer_ProcessTraceError(t *testing.T) {
	// Backup original functions and defer restoration
	originalOpenTraceFunc := OpenTraceFunc
	originalProcessTraceFunc := ProcessTraceFunc
	defer func() {
		OpenTraceFunc = originalOpenTraceFunc
		ProcessTraceFunc = originalProcessTraceFunc
	}()

	// Mock implementations
	OpenTraceFunc = func(elf *EventTraceLogfile) (uint64, error) {
		return 12345, nil // Mock a valid session handler
	}

	ProcessTraceFunc = func(handleArray *uint64, handleCount uint32, startTime *FileTime, endTime *FileTime) error {
		return ERROR_INVALID_PARAMETER
	}

	// Create a Session instance
	session := &Session{
		Name:           "TestSession",
		Realtime:       true,
		BufferCallback: uintptr(0),
		Callback:       uintptr(0),
	}

	// Test StartConsumer
	err := session.StartConsumer()

	// Assertions
	assert.EqualError(t, err, "failed to process trace: The parameter is incorrect.")
}

func TestStartConsumer_Success(t *testing.T) {
	// Backup original functions and defer restoration
	originalOpenTraceFunc := OpenTraceFunc
	originalProcessTraceFunc := ProcessTraceFunc
	defer func() {
		OpenTraceFunc = originalOpenTraceFunc
		ProcessTraceFunc = originalProcessTraceFunc
	}()

	// Mock implementations
	OpenTraceFunc = func(elf *EventTraceLogfile) (uint64, error) {
		return 12345, nil // Mock a valid session handler
	}

	ProcessTraceFunc = func(handleArray *uint64, handleCount uint32, startTime *FileTime, endTime *FileTime) error {
		return nil
	}

	// Create a Session instance
	session := &Session{
		Name:           "TestSession",
		Realtime:       true,
		BufferCallback: uintptr(0),
		Callback:       uintptr(0),
	}

	// Test StartConsumer
	err := session.StartConsumer()

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, uint64(12345), session.TraceHandler, "TraceHandler should be set to the mock value")
}
