// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package etw

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"
)

func isValidHandler(handler uint64) bool {
	if handler == 0 || handler == INVALID_PROCESSTRACE_HANDLE {
		return false
	}
	return true
}

// GetHandler queries the status of an existing ETW session to get its handler and properties.
func (s *Session) GetHandler() error {
	// Convert the session name to UTF16 for Windows API compatibility.
	sessionNamePtr, err := syscall.UTF16PtrFromString(s.Name)
	if err != nil {
		return fmt.Errorf("failed to convert session name")
	}

	// Query the current state of the ETW session.
	if err = ControlTraceFunc(
		0,
		sessionNamePtr,
		s.Properties,
		EVENT_TRACE_CONTROL_QUERY,
	); err != nil {
		// Handle specific errors related to the query operation.
		if errors.Is(err, ERROR_BAD_LENGTH) {
			return fmt.Errorf("bad length when querying handler: %w", err)
		} else if errors.Is(err, ERROR_INVALID_PARAMETER) {
			return fmt.Errorf("invalid parameters when querying handler: %w", err)
		} else if errors.Is(err, ERROR_WMI_INSTANCE_NOT_FOUND) {
			return fmt.Errorf("session is not running")
		}
		return fmt.Errorf("failed to get handler: %w", err)
	}

	// Get the session handler from the properties struct.
	s.Handler = uintptr(s.Properties.Wnode.Union1)

	return nil
}

// CreateRealtimeSession initializes and starts a new real-time ETW session.
func (s *Session) CreateRealtimeSession() error {
	// Convert the session name to UTF16 format for Windows API compatibility.
	sessionPtr, err := syscall.UTF16PtrFromString(s.Name)
	if err != nil {
		return fmt.Errorf("failed to convert session name")
	}

	// Start the ETW trace session.
	err = StartTraceFunc(
		&s.Handler,
		sessionPtr,
		s.Properties,
	)
	if err != nil {
		// Handle specific errors related to starting the trace session.
		if errors.Is(err, ERROR_ALREADY_EXISTS) {
			return fmt.Errorf("session already exists: %w", err)
		} else if errors.Is(err, ERROR_INVALID_PARAMETER) {
			return fmt.Errorf("invalid parameters when starting session trace: %w", err)
		}
		return fmt.Errorf("failed to start trace: %w", err)
	}

	// Set additional parameters for trace enabling.
	// See https://learn.microsoft.com/en-us/windows/win32/api/evntrace/ns-evntrace-enable_trace_parameters#members
	params := EnableTraceParameters{
		Version: 2, // ENABLE_TRACE_PARAMETERS_VERSION_2
	}

	// Enable the trace session with extended options.
	if err := EnableTraceFunc(
		s.Handler,
		(*GUID)(unsafe.Pointer(&s.GUID)),
		EVENT_CONTROL_CODE_ENABLE_PROVIDER,
		s.TraceLevel,
		s.MatchAnyKeyword,
		s.MatchAllKeyword,
		0,       // Asynchronous enablement with zero timeout
		&params, // Additional parameters
	); err != nil {
		// Handle specific errors related to enabling the trace session.
		if errors.Is(err, ERROR_INVALID_PARAMETER) {
			return fmt.Errorf("invalid parameters when enabling session trace: %w", err)
		} else if errors.Is(err, ERROR_TIMEOUT) {
			return fmt.Errorf("timeout value expired before the enable callback completed: %w", err)
		} else if errors.Is(err, ERROR_NO_SYSTEM_RESOURCES) {
			return fmt.Errorf("exceeded the number of trace sessions that can enable the provider: %w", err)
		}
		return fmt.Errorf("failed to enable trace: %w", err)
	}

	return nil
}

// StopSession closes the ETW session and associated handles if they were created.
func (s *Session) StopSession() error {
	if s.Realtime {
		if isValidHandler(s.TraceHandler) {
			// Attempt to close the trace and handle potential errors.
			if err := CloseTraceFunc(s.TraceHandler); err != nil && !errors.Is(err, ERROR_CTX_CLOSE_PENDING) {
				return fmt.Errorf("failed to close trace: %w", err)
			}
		}

		if s.NewSession {
			// If we created the session, send a control command to stop it.
			return ControlTraceFunc(
				s.Handler,
				nil,
				s.Properties,
				EVENT_TRACE_CONTROL_STOP,
			)
		}
	}

	return nil
}
