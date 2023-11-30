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

// StartConsumer initializes and starts the ETW event tracing session.
func (s *Session) StartConsumer() error {
	var elf EventTraceLogfile
	var err error

	// Configure EventTraceLogfile based on the session type (realtime or not).
	if !s.Realtime {
		elf.LogFileMode = PROCESS_TRACE_MODE_EVENT_RECORD
		logfilePtr, err := syscall.UTF16PtrFromString(s.Name)
		if err != nil {
			return fmt.Errorf("failed to convert logfile name")
		}
		elf.LogFileName = logfilePtr
	} else {
		elf.LogFileMode = PROCESS_TRACE_MODE_EVENT_RECORD | PROCESS_TRACE_MODE_REAL_TIME
		sessionPtr, err := syscall.UTF16PtrFromString(s.Name)
		if err != nil {
			return fmt.Errorf("failed to convert session name")
		}
		elf.LoggerName = sessionPtr
	}

	// Set callbacks and context for the session.
	elf.BufferCallback = s.BufferCallback
	elf.Callback = s.Callback
	elf.Context = 0

	// Open an ETW trace processing handle for consuming events
	// from an ETW real-time trace session or an ETW log file.
	s.TraceHandler, err = OpenTraceFunc(&elf)

	switch {
	case err == nil:

	// Handle specific errors for trace opening.
	case errors.Is(err, ERROR_BAD_PATHNAME):
		return fmt.Errorf("invalid log source when opening trace: %w", err)
	case errors.Is(err, ERROR_ACCESS_DENIED):
		return fmt.Errorf("access denied when opening trace: %w", err)
	default:
		return fmt.Errorf("failed to open trace: %w", err)
	}
	// Process the trace. This function blocks until processing ends.
	if err := ProcessTraceFunc(&s.TraceHandler, 1, nil, nil); err != nil {
		return fmt.Errorf("failed to process trace: %w", err)
	}

	return nil
}
