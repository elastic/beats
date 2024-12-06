// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package etw

import (
	"errors"
	"fmt"
	"syscall"
)

// AttachToExistingSession queries the status of an existing ETW session.
// On success, it updates the Session's handler with the queried information.
func (s *Session) AttachToExistingSession() error {
	// Convert the session name to UTF16 for Windows API compatibility.
	sessionNamePtr, err := syscall.UTF16PtrFromString(s.Name)
	if err != nil {
		return fmt.Errorf("failed to convert session name: %w", err)
	}

	// Query the current state of the ETW session.
	err = s.controlTrace(0, sessionNamePtr, s.properties, EVENT_TRACE_CONTROL_QUERY)
	switch {
	case err == nil:
		// Get the session handler from the properties struct.
		s.handler = uintptr(s.properties.Wnode.Union1)

		return nil

	// Handle specific errors related to the query operation.
	case errors.Is(err, ERROR_BAD_LENGTH):
		return fmt.Errorf("bad length when querying handler: %w", err)
	case errors.Is(err, ERROR_INVALID_PARAMETER):
		return fmt.Errorf("invalid parameters when querying handler: %w", err)
	case errors.Is(err, ERROR_WMI_INSTANCE_NOT_FOUND):
		return fmt.Errorf("session is not running: %w", err)
	default:
		return fmt.Errorf("failed to get handler: %w", err)
	}
}

// CreateRealtimeSession initializes and starts a new real-time ETW session.
func (s *Session) CreateRealtimeSession() error {
	// Convert the session name to UTF16 format for Windows API compatibility.
	sessionPtr, err := syscall.UTF16PtrFromString(s.Name)
	if err != nil {
		return fmt.Errorf("failed to convert session name: %w", err)
	}

	// Start the ETW trace session.
	err = s.startTrace(&s.handler, sessionPtr, s.properties)
	switch {
	case err == nil:

	// Handle specific errors related to starting the trace session.
	case errors.Is(err, ERROR_ALREADY_EXISTS):
		return fmt.Errorf("session already exists: %w", err)
	case errors.Is(err, ERROR_INVALID_PARAMETER):
		return fmt.Errorf("invalid parameters when starting session trace: %w", err)
	default:
		return fmt.Errorf("failed to start trace: %w", err)
	}

	// Set additional parameters for trace enabling.
	// See https://learn.microsoft.com/en-us/windows/win32/api/evntrace/ns-evntrace-enable_trace_parameters#members
	params := EnableTraceParameters{
		Version: 2, // ENABLE_TRACE_PARAMETERS_VERSION_2
	}

	// Zero timeout means asynchronous enablement
	const timeout = 0

	// Enable the trace session with extended options.
	err = s.enableTrace(s.handler, &s.GUID, EVENT_CONTROL_CODE_ENABLE_PROVIDER, s.traceLevel, s.matchAnyKeyword, s.matchAllKeyword, timeout, &params)
	switch {
	case err == nil:
		return nil
	// Handle specific errors related to enabling the trace session.
	case errors.Is(err, ERROR_INVALID_PARAMETER):
		return fmt.Errorf("invalid parameters when enabling session trace: %w", err)
	case errors.Is(err, ERROR_TIMEOUT):
		return fmt.Errorf("timeout value expired before the enable callback completed: %w", err)
	case errors.Is(err, ERROR_NO_SYSTEM_RESOURCES):
		return fmt.Errorf("exceeded the number of trace sessions that can enable the provider: %w", err)
	default:
		return fmt.Errorf("failed to enable trace: %w", err)
	}
}

// StopSession closes the ETW session and associated handles if they were created.
func (s *Session) StopSession() error {
	if !s.Realtime {
		return nil
	}

	if isValidHandler(s.traceHandler) {
		// Attempt to close the trace and handle potential errors.
		if err := s.closeTrace(s.traceHandler); err != nil && !errors.Is(err, ERROR_CTX_CLOSE_PENDING) {
			return fmt.Errorf("failed to close trace: %w", err)
		}
	}

	if s.NewSession {
		// If we created the session, send a control command to stop it.
		return s.controlTrace(
			s.handler,
			nil,
			s.properties,
			EVENT_TRACE_CONTROL_STOP,
		)
	}

	return nil
}

func isValidHandler(handler uint64) bool {
	return handler != 0 && handler != INVALID_PROCESSTRACE_HANDLE
}
