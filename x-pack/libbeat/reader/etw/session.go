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

	"golang.org/x/sys/windows"
)

// For testing purposes we create a variable to store the function to call
// When running tests, these variables point to a mock function
var (
	guidFromProviderNameFunc = guidFromProviderName
	setSessionGUIDFunc       = setSessionGUID
)

type Session struct {
	// Name is the identifier for the session.
	// It is used to identify the session in logs and also for Windows processes.
	Name string
	// GUID is the provider GUID to configure the session.
	GUID windows.GUID
	// properties of the session that are initialized in newSessionProperties()
	// See https://learn.microsoft.com/en-us/windows/win32/api/evntrace/ns-evntrace-event_trace_properties for more information
	properties *EventTraceProperties
	// handler of the event tracing session for which the provider is being configured.
	// It is obtained from StartTrace when a new trace is started.
	// This handler is needed to enable, query or stop the trace.
	handler uintptr
	// Realtime is a flag to know if the consumer reads from a logfile or real-time session.
	Realtime bool // Real-time flag
	// NewSession is a flag to indicate whether a new session has been created or attached to an existing one.
	NewSession bool
	// TraceLevel sets the maximum level of events that we want the provider to write.
	traceLevel uint8
	// matchAnyKeyword is a 64-bit bitmask of keywords that determine the categories of events that we want the provider to write.
	// The provider writes an event if the event's keyword bits match any of the bits set in this value
	// or if the event has no keyword bits set, in addition to meeting the level and matchAllKeyword criteria.
	matchAnyKeyword uint64
	// matchAllKeyword is a 64-bit bitmask of keywords that restricts the events that we want the provider to write.
	// The provider typically writes an event if the event's keyword bits match all of the bits set in this value
	// or if the event has no keyword bits set, in addition to meeting the level and matchAnyKeyword criteria.
	matchAllKeyword uint64
	// traceHandler is the trace processing handle.
	// It is used to control the trace that receives and processes events.
	traceHandler uint64
	// Callback is the pointer to EventRecordCallback which receives and processes event trace events.
	Callback func(*EventRecord) uintptr
	// BufferCallback is the pointer to BufferCallback which processes retrieved metadata about the ETW buffers (optional).
	BufferCallback func(*EventTraceLogfile) uintptr

	// Pointers to functions that make calls to the Windows API.
	// In tests, these pointers can be replaced with mock functions to simulate API behavior without making actual calls to the Windows API.
	startTrace   func(*uintptr, *uint16, *EventTraceProperties) error
	controlTrace func(traceHandle uintptr, instanceName *uint16, properties *EventTraceProperties, controlCode uint32) error
	enableTrace  func(traceHandle uintptr, providerId *windows.GUID, isEnabled uint32, level uint8, matchAnyKeyword uint64, matchAllKeyword uint64, enableProperty uint32, enableParameters *EnableTraceParameters) error
	closeTrace   func(traceHandle uint64) error
	openTrace    func(elf *EventTraceLogfile) (uint64, error)
	processTrace func(handleArray *uint64, handleCount uint32, startTime *FileTime, endTime *FileTime) error
}

// setSessionName determines the session name based on the provided configuration.
func setSessionName(conf Config) string {
	// Iterate through potential session name values, returning the first non-empty one.
	for _, value := range []string{conf.Logfile, conf.Session, conf.SessionName} {
		if value != "" {
			return value
		}
	}

	if conf.ProviderName != "" {
		return fmt.Sprintf("Elastic-%s", conf.ProviderName)
	}

	return fmt.Sprintf("Elastic-%s", conf.ProviderGUID)
}

// setSessionGUID determines the session GUID based on the provided configuration.
func setSessionGUID(conf Config) (windows.GUID, error) {
	var guid windows.GUID
	var err error

	// If ProviderGUID is not set in the configuration, attempt to resolve it using the provider name.
	if conf.ProviderGUID == "" {
		guid, err = guidFromProviderNameFunc(conf.ProviderName)
		if err != nil {
			return windows.GUID{}, fmt.Errorf("error resolving GUID: %w", err)
		}
	} else {
		// If ProviderGUID is set, parse it into a GUID structure.
		guid, err = windows.GUIDFromString(conf.ProviderGUID)
		if err != nil {
			return windows.GUID{}, fmt.Errorf("error parsing Windows GUID: %w", err)
		}
	}

	return guid, nil
}

// getTraceLevel converts a string representation of a trace level
// to its corresponding uint8 constant value
func getTraceLevel(level string) uint8 {
	switch level {
	case "critical":
		return TRACE_LEVEL_CRITICAL
	case "error":
		return TRACE_LEVEL_ERROR
	case "warning":
		return TRACE_LEVEL_WARNING
	case "information":
		return TRACE_LEVEL_INFORMATION
	case "verbose":
		return TRACE_LEVEL_VERBOSE
	default:
		return TRACE_LEVEL_INFORMATION
	}
}

// newSessionProperties initializes and returns a pointer to EventTraceProperties
// with the necessary settings for starting an ETW session.
// See https://learn.microsoft.com/en-us/windows/win32/api/evntrace/ns-evntrace-event_trace_properties
func newSessionProperties(sessionName string) *EventTraceProperties {
	// Calculate buffer size for session properties.
	sessionNameSize := (len(sessionName) + 1) * 2
	bufSize := sessionNameSize + int(unsafe.Sizeof(EventTraceProperties{}))

	// Allocate buffer and cast to EventTraceProperties.
	propertiesBuf := make([]byte, bufSize)
	sessionProperties := (*EventTraceProperties)(unsafe.Pointer(&propertiesBuf[0]))

	// Initialize mandatory fields of the EventTraceProperties struct.
	// Filled based on https://learn.microsoft.com/en-us/windows/win32/etw/wnode-header
	sessionProperties.Wnode.BufferSize = uint32(bufSize)
	sessionProperties.Wnode.Guid = windows.GUID{} // GUID not required for non-private/kernel sessions
	// ClientContext is used for timestamp resolution
	// Not used unless adding PROCESS_TRACE_MODE_RAW_TIMESTAMP flag to EVENT_TRACE_LOGFILE struct
	// See https://learn.microsoft.com/en-us/windows/win32/etw/wnode-header
	sessionProperties.Wnode.ClientContext = 1
	sessionProperties.Wnode.Flags = WNODE_FLAG_TRACED_GUID
	// Set logging mode to real-time
	// See https://learn.microsoft.com/en-us/windows/win32/etw/logging-mode-constants
	sessionProperties.LogFileMode = EVENT_TRACE_REAL_TIME_MODE
	sessionProperties.LogFileNameOffset = 0                                            // Can be specified to log to a file as well as to a real-time session
	sessionProperties.BufferSize = 64                                                  // Default buffer size, can be configurable
	sessionProperties.LoggerNameOffset = uint32(unsafe.Sizeof(EventTraceProperties{})) // Offset to the logger name

	return sessionProperties
}

// NewSession initializes and returns a new ETW Session based on the provided configuration.
func NewSession(conf Config) (*Session, error) {
	session := &Session{}

	// Assign ETW Windows API functions
	session.startTrace = _StartTrace
	session.controlTrace = _ControlTrace
	session.enableTrace = _EnableTraceEx2
	session.openTrace = _OpenTrace
	session.processTrace = _ProcessTrace
	session.closeTrace = _CloseTrace

	session.Name = setSessionName(conf)
	session.Realtime = true

	// If a current session is configured, set up the session properties and return.
	if conf.Session != "" {
		session.properties = newSessionProperties(session.Name)
		return session, nil
	} else if conf.Logfile != "" {
		// If a logfile is specified, set up for non-realtime session.
		session.Realtime = false
		return session, nil
	}

	session.NewSession = true // Indicate this is a new session

	var err error
	session.GUID, err = setSessionGUIDFunc(conf)
	if err != nil {
		return nil, fmt.Errorf("error when initializing session '%s': %w", session.Name, err)
	}

	// Initialize additional session properties.
	session.properties = newSessionProperties(session.Name)
	session.traceLevel = getTraceLevel(conf.TraceLevel)
	session.matchAnyKeyword = conf.MatchAnyKeyword
	session.matchAllKeyword = conf.MatchAllKeyword

	return session, nil
}

// StartConsumer initializes and starts the ETW event tracing session.
func (s *Session) StartConsumer() error {
	var elf EventTraceLogfile
	var err error

	// Configure EventTraceLogfile based on the session type (realtime or not).
	if !s.Realtime {
		elf.LogFileMode = PROCESS_TRACE_MODE_EVENT_RECORD
		logfilePtr, err := syscall.UTF16PtrFromString(s.Name)
		if err != nil {
			return fmt.Errorf("failed to convert logfile name: %w", err)
		}
		elf.LogFileName = logfilePtr
	} else {
		elf.LogFileMode = PROCESS_TRACE_MODE_EVENT_RECORD | PROCESS_TRACE_MODE_REAL_TIME
		sessionPtr, err := syscall.UTF16PtrFromString(s.Name)
		if err != nil {
			return fmt.Errorf("failed to convert session name: %w", err)
		}
		elf.LoggerName = sessionPtr
	}

	// Set callback and context for the session.
	if s.Callback == nil {
		return fmt.Errorf("error loading callback")
	}
	elf.Callback = syscall.NewCallback(s.Callback)
	elf.Context = 0

	// Open an ETW trace processing handle for consuming events
	// from an ETW real-time trace session or an ETW log file.
	s.traceHandler, err = s.openTrace(&elf)

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
	if err := s.processTrace(&s.traceHandler, 1, nil, nil); err != nil {
		return fmt.Errorf("failed to process trace: %w", err)
	}

	return nil
}
