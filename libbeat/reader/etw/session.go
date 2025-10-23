// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build windows

package etw

import (
	"errors"
	"fmt"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/elastic/elastic-agent-libs/logp"
)

// For testing purposes we create a variable to store the function to call
// When running tests, these variables point to a mock function
var (
	guidFromProviderNameFunc = guidFromProviderName

	// EventTraceGuid is the well-known GUID for ETW trace header events.
	// {68FDD900-4A3E-11D1-84F4-0000F80464E3}
	// We need this GUID to identify the event trace session and be able to
	// ignore it when reading an ETL file.
	eventTraceGUID = windows.GUID{
		Data1: 0x68fdd900,
		Data2: 0x4a3e,
		Data3: 0x11d1,
		Data4: [8]byte{0x84, 0xf4, 0x00, 0x00, 0xf8, 0x04, 0x64, 0xe3},
	}
)

type Session struct {
	// Name is the identifier for the session.
	// It is used to identify the session in logs and also for Windows processes.
	Name string
	// Realtime is a flag to know if the consumer reads from a logfile or real-time session.
	Realtime bool
	// NewSession is a flag to indicate whether a new session has been created or attached to an existing one.
	NewSession bool
	// Callback is the pointer to EventRecordCallback which receives and processes event trace events.
	Callback func(*EventRecord) uintptr
	// BufferCallback is the pointer to BufferCallback which processes retrieved metadata about the ETW buffers (optional).
	BufferCallback func(*EventTraceLogfile) uintptr

	// properties of the session that are initialized in newSessionProperties()
	// See https://learn.microsoft.com/en-us/windows/win32/api/evntrace/ns-evntrace-event_trace_properties for more information
	properties *EventTraceProperties
	// handler of the event tracing session for which the provider is being configured.
	// It is obtained from StartTrace when a new trace is started.
	// This handler is needed to enable, query or stop the trace.
	handler uintptr
	// traceHandler is the trace processing handle.
	// It is used to control the trace that receives and processes events.
	traceHandler uint64

	// Pointers to functions that make calls to the Windows API.
	// In tests, these pointers can be replaced with mock functions to simulate API behavior without making actual calls to the Windows API.
	startTrace   func(*uintptr, *uint16, *EventTraceProperties) error
	controlTrace func(traceHandle uintptr, instanceName *uint16, properties *EventTraceProperties, controlCode uint32) error
	enableTrace  func(traceHandle uintptr, providerId *windows.GUID, isEnabled uint32, level uint8, matchAnyKeyword uint64, matchAllKeyword uint64, enableProperty uint32, enableParameters *EnableTraceParameters) error
	closeTrace   func(traceHandle uint64) error
	openTrace    func(elf *EventTraceLogfile) (uint64, error)
	processTrace func(handleArray *uint64, handleCount uint32, startTime *FileTime, endTime *FileTime) error

	config        Config
	metadataCache *metadataCache
}

// setSessionName determines the session name based on the provided configuration.
func setSessionName(conf Config) string {
	// Iterate through potential session name values, returning the first non-empty one.
	for _, value := range []string{conf.Logfile, conf.Session, conf.SessionName} {
		if value != "" {
			return value
		}
	}

	var nameGUIDs []string
	for _, provider := range conf.Providers {
		if provider.Name != "" {
			nameGUIDs = append(nameGUIDs, provider.Name)
		} else if provider.GUID != "" {
			nameGUIDs = append(nameGUIDs, provider.GUID)
		}
	}

	return fmt.Sprintf("Elastic-%s", strings.Join(nameGUIDs, "-"))
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
func newSessionProperties(sessionName string, conf Config) *EventTraceProperties {
	// Calculate buffer size for session properties.
	sessionNameSize := uintptr(len(sessionName)+1) * 2
	bufSize := uint32(sessionNameSize + unsafe.Sizeof(EventTraceProperties{}))

	// Allocate buffer and cast to EventTraceProperties.
	propertiesBuf := make([]byte, bufSize)
	sessionProperties := (*EventTraceProperties)(unsafe.Pointer(&propertiesBuf[0]))

	// Initialize mandatory fields of the EventTraceProperties struct.
	// Filled based on https://learn.microsoft.com/en-us/windows/win32/etw/wnode-header
	sessionProperties.Wnode.BufferSize = bufSize
	sessionProperties.Wnode.Guid = windows.GUID{} // GUID not required for non-private/kernel sessions
	// ClientContext is used for timestamp resolution
	// Not used unless adding PROCESS_TRACE_MODE_RAW_TIMESTAMP flag to EVENT_TRACE_LOGFILE struct
	// See https://learn.microsoft.com/en-us/windows/win32/etw/wnode-header
	sessionProperties.Wnode.ClientContext = 1
	sessionProperties.Wnode.Flags = WNODE_FLAG_TRACED_GUID
	// Set logging mode to real-time
	// See https://learn.microsoft.com/en-us/windows/win32/etw/logging-mode-constants
	sessionProperties.LogFileMode = EVENT_TRACE_REAL_TIME_MODE
	sessionProperties.LogFileNameOffset = 0 // Can be specified to log to a file as well as to a real-time session
	if conf.BufferSize == 0 {
		conf.BufferSize = 64 // Default buffer size if not specified
	}
	sessionProperties.BufferSize = conf.BufferSize
	sessionProperties.MinimumBuffers = conf.MinimumBuffers
	sessionProperties.MaximumBuffers = conf.MaximumBuffers
	sessionProperties.LoggerNameOffset = uint32(unsafe.Sizeof(EventTraceProperties{})) // Offset to the logger name
	return sessionProperties
}

// NewSession initializes and returns a new ETW Session based on the provided configuration.
func NewSession(conf Config) (*Session, error) {
	session := &Session{
		config: conf,
	}

	// Assign ETW Windows API functions
	session.startTrace = _StartTrace
	session.controlTrace = _ControlTrace
	session.enableTrace = _EnableTraceEx2
	session.openTrace = _OpenTrace
	session.processTrace = _ProcessTrace
	session.closeTrace = _CloseTrace

	session.metadataCache = newMetadataCache(logp.NewLogger("etw_session"))
	session.Name = setSessionName(conf)
	session.Realtime = true

	// If a current session is configured, set up the session properties and return.
	if conf.Session != "" {
		session.properties = newSessionProperties(session.Name, conf)
		return session, nil
	} else if conf.Logfile != "" {
		// If a logfile is specified, set up for non-realtime session.
		session.Realtime = false
		return session, nil
	}

	session.NewSession = true // Indicate this is a new session

	// Initialize additional session properties.
	session.properties = newSessionProperties(session.Name, conf)

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

var ErrUnprocessableEvent = errors.New("unprocessable event")

// GetEventProperties extracts and returns properties from an ETW event record.
func (s *Session) RenderEvent(r *EventRecord) (e RenderedEtwEvent, err error) {
	if !s.Realtime {
		if r.EventHeader.ProviderId == eventTraceGUID {
			// Ignore event trace session events when reading from a logfile.
			return RenderedEtwEvent{}, ErrUnprocessableEvent
		}
	}
	providerCache, err := s.metadataCache.getProviderCache(r.EventHeader.ProviderId)
	if err != nil {
		return RenderedEtwEvent{}, fmt.Errorf("failed to get provider cache: %w", err)
	}
	// Initialize a new property parser for the event record.
	p := newEventRenderer(providerCache, r, s.metadataCache.bufferPools)
	event, err := p.render()
	if err != nil {
		return RenderedEtwEvent{}, fmt.Errorf("failed to parse event properties: %w", err)
	}
	return event, nil
}

func uintptrToBytes(ptr uintptr, length uint16) []byte {
	return unsafe.Slice((*byte)(unsafe.Pointer(ptr)), length)
}
