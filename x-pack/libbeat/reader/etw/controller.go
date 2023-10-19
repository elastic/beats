// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package etw

import (
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

// Create a real-time session
func (s Session) CreateRealtimeSession() error {
	sessionPtr, err := syscall.UTF16PtrFromString(s.Name)
	if err != nil {
		return fmt.Errorf("failed to convert session name '%s'", s.Name)
	}

	err = _StartTrace(
		&s.Handler,
		sessionPtr,
		s.Properties,
	)
	if err != nil {
		if err == ERROR_ALREADY_EXISTS {
			return fmt.Errorf("session already exists for '%s'", s.Name)
		} else if err == ERROR_INVALID_PARAMETER {
			return fmt.Errorf("invalid parameters for '%s'", s.Name)
		}
		return fmt.Errorf("failed to start trace (unknown reason) for '%s'", s.Name)
	}

	params := EnableTraceParameters{
		Version: 2, // ENABLE_TRACE_PARAMETERS_VERSION_2
	}

	if err := _EnableTraceEx2(
		s.Handler,
		(*GUID)(unsafe.Pointer(&s.GUID)),
		EVENT_CONTROL_CODE_ENABLE_PROVIDER,
		s.TraceLevel,
		s.MatchAnyKeyword,
		s.MatchAllKeyword,
		0,       // Timeout set to zero to enable the trace asynchronously
		&params, // More filters that may be initialized (ENABLE_TRACE_PARAMETERS)
	); err != nil {
		// Todo: Catch specific error cases
		return fmt.Errorf("failed to enable trace for '%s'", s.Name)
	}
	return nil
}

// Closes handles and session if created
func (s Session) StopSession() error {
	if s.Realtime {
		if isValidHandler(s.TraceHandler) {
			if err := _CloseTrace(s.TraceHandler); err != nil && err != ERROR_CTX_CLOSE_PENDING {
				return fmt.Errorf("failed to close trace for session '%s'", s.Name)
			}
		}

		if s.NewSession {
			// Here we could also have to call _EnableTraceEx2 to disable provider if started
			// If calling _ControlTrace without disabling the providers, ETW should disable all the providers for that session automatically
			return _ControlTrace(
				s.Handler,
				nil,
				s.Properties,
				EVENT_TRACE_CONTROL_STOP,
			)
		}
	}
	return nil
}
