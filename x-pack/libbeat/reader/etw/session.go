// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package etw

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
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

// Sets the session name used for identify the created session
func setSessionName(conf Config) string {
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

func setSessionGUID(conf Config) (GUID, error) {
	var guid GUID
	var err error
	if conf.ProviderGUID == "" {
		guid, err = GUIDFromProviderName(conf.ProviderName)
		if err != nil {
			return GUID{}, fmt.Errorf("error resolving GUID from %s: %v", conf.ProviderName, err)
		}
	} else {
		winGUID, err := windows.GUIDFromString(conf.ProviderGUID)
		if err != nil {
			return GUID{}, fmt.Errorf("error parsing Windows GUID: %v", err)
		}
		guid = convertWindowsGUID(winGUID)
	}

	return guid, nil
}

func convertWindowsGUID(windowsGUID windows.GUID) GUID {
	return GUID{
		Data1: windowsGUID.Data1,
		Data2: windowsGUID.Data2,
		Data3: windowsGUID.Data3,
		Data4: windowsGUID.Data4,
	}
}

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

// See https://learn.microsoft.com/en-us/windows/win32/api/evntrace/ns-evntrace-event_trace_properties for more options when creating sessions
func NewSessionProperties(sessionName string) *EventTraceProperties {
	sessionNameSize := (len(sessionName) + 1) * 2
	bufSize := sessionNameSize + int(unsafe.Sizeof(EventTraceProperties{}))

	propertiesBuf := make([]byte, bufSize)
	sessionProperties := (*EventTraceProperties)(unsafe.Pointer(&propertiesBuf[0]))

	// Mandatory fields for SessionProperties struct
	sessionProperties.Wnode.BufferSize = uint32(bufSize)
	sessionProperties.Wnode.Guid = GUID{}     // Not needed to create GUID if other than private or kernel session
	sessionProperties.Wnode.ClientContext = 1 // Clock resolution for timestamp (defaults to QPC)
	sessionProperties.Wnode.Flags = WNODE_FLAG_TRACED_GUID
	sessionProperties.LogFileMode = EVENT_TRACE_REAL_TIME_MODE // See https://learn.microsoft.com/en-us/windows/win32/etw/logging-mode-constants
	sessionProperties.LogFileNameOffset = 0                    // Can be specified to log to a file as well as to a real-time session
	sessionProperties.BufferSize = 64                          // This is a default value, may be part of the configuration
	sessionProperties.LoggerNameOffset = uint32(unsafe.Sizeof(EventTraceProperties{}))

	return sessionProperties
}

// Initialise a session
func NewSession(conf Config) (Session, error) {
	var session Session
	var err error
	session.Name = setSessionName(conf)
	session.Realtime = true

	if conf.Session != "" {
		// Whether reading from an existing session, there is no need to initialise parameters below
		return session, nil
	} else if conf.Logfile != "" {
		// Same when reading from a logfile
		session.Realtime = false
		return session, nil
	}

	session.NewSession = true

	session.GUID, err = setSessionGUID(conf)
	if err != nil {
		return Session{}, fmt.Errorf("error resolving GUID from %s: %v", conf.ProviderName, err)
	}

	session.Properties = NewSessionProperties(session.Name)
	session.TraceLevel = getTraceLevel(conf.TraceLevel)
	session.MatchAnyKeyword = conf.MatchAnyKeyword
	session.MatchAllKeyword = conf.MatchAllKeyword

	if session.Callback == 0 {
		session.Callback = syscall.NewCallback(DefaultCallback)
	}
	session.BufferCallback = syscall.NewCallback(DefaultBufferCallback)

	return session, nil
}
