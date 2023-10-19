// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package etw

import (
	"fmt"
	"syscall"
)

func (s Session) StartConsumer() error {
	var elf EventTraceLogfile
	var err error

	if !s.Realtime {
		elf.LogFileMode = PROCESS_TRACE_MODE_EVENT_RECORD | PROCESS_TRACE_MODE_RAW_TIMESTAMP
		logfilePtr, err := syscall.UTF16PtrFromString(s.Name)
		if err != nil {
			return fmt.Errorf("failed to convert logfile name '%s'", s.Name)
		}
		elf.LogFileName = logfilePtr
	} else {
		elf.LogFileMode = PROCESS_TRACE_MODE_EVENT_RECORD | PROCESS_TRACE_MODE_RAW_TIMESTAMP | PROCESS_TRACE_MODE_REAL_TIME
		sessionPtr, err := syscall.UTF16PtrFromString(s.Name)
		if err != nil {
			return fmt.Errorf("failed to convert session '%s'", s.Name)
		}
		elf.LoggerName = sessionPtr
	}

	elf.BufferCallback = s.BufferCallback
	elf.Callback = s.Callback
	elf.Context = 0

	s.TraceHandler, err = _OpenTrace(&elf)
	if err != nil {
		return fmt.Errorf("failed to open trace for session %s", s.Name)
	}
	if err := _ProcessTrace(&s.TraceHandler, 1, nil, nil); err != nil {
		return fmt.Errorf("failed to process trace for session '%s'", s.Name)
	}

	return nil
}
