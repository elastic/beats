// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package etw

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// For testing purposes we create a variable to store the function to call
// When running tests, these variables point to a mock function
var (
	GUIDFromProviderNameFunc = GUIDFromProviderName
	SetSessionGUIDFunc       = setSessionGUID
)

type Session struct {
	Name            string // Identifier of the session
	GUID            GUID
	Properties      *EventTraceProperties // Session properties
	Handler         uintptr               // Session handler
	Realtime        bool                  // Real-time flag
	NewSession      bool                  // Flag to indicate whether a new session has been created or attached to an existing one
	TraceLevel      uint8                 // Trace level
	MatchAnyKeyword uint64
	MatchAllKeyword uint64
	TraceHandler    uint64 // Trace processing handle

	Callback       uintptr // Pointer to EventRecordCallback to process ETW events
	BufferCallback uintptr // Pointer to BufferCallback which processes retrieved metadata about the ETW buffers
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
func setSessionGUID(conf Config) (GUID, error) {
	var guid GUID
	var err error

	// If ProviderGUID is not set in the configuration, attempt to resolve it using the provider name.
	if conf.ProviderGUID == "" {
		guid, err = GUIDFromProviderNameFunc(conf.ProviderName)
		if err != nil {
			return GUID{}, fmt.Errorf("error resolving GUID: %w", err)
		}
	} else {
		// If ProviderGUID is set, parse it into a GUID structure.
		winGUID, err := windows.GUIDFromString(conf.ProviderGUID)
		if err != nil {
			return GUID{}, fmt.Errorf("error parsing Windows GUID: %w", err)
		}
		guid = convertWindowsGUID(winGUID)
	}

	return guid, nil
}

// convertWindowsGUID converts a Windows GUID structure to a custom GUID structure.
func convertWindowsGUID(windowsGUID windows.GUID) GUID {
	return GUID{
		Data1: windowsGUID.Data1,
		Data2: windowsGUID.Data2,
		Data3: windowsGUID.Data3,
		Data4: windowsGUID.Data4,
	}
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
	sessionProperties.Wnode.Guid = GUID{} // GUID not required for non-private/kernel sessions
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
func NewSession(conf Config) (Session, error) {
	var session Session
	var err error
	session.Name = setSessionName(conf)
	session.Realtime = true

	// Set default callbacks if not already specified.
	if session.Callback == 0 {
		session.Callback = syscall.NewCallback(DefaultCallback)
	}
	session.BufferCallback = syscall.NewCallback(DefaultBufferCallback)

	// If a current session is configured, set up the session properties and return.
	if conf.Session != "" {
		session.Properties = newSessionProperties(session.Name)
		return session, nil
	} else if conf.Logfile != "" {
		// If a logfile is specified, set up for non-realtime session.
		session.Realtime = false
		return session, nil
	}

	session.NewSession = true // Indicate this is a new session

	session.GUID, err = SetSessionGUIDFunc(conf)
	if err != nil {
		return Session{}, err
	}

	// Initialize additional session properties.
	session.Properties = newSessionProperties(session.Name)
	session.TraceLevel = getTraceLevel(conf.TraceLevel)
	session.MatchAnyKeyword = conf.MatchAnyKeyword
	session.MatchAllKeyword = conf.MatchAllKeyword

	return session, nil
}
