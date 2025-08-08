// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package etw

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// eventFilterEventIDHeader represents the fixed-size portion of the
// EVENT_FILTER_EVENT_ID structure, which defines an event ID filter.
type eventFilterEventIDHeader struct {
	FilterIn uint8  // 1 to allow events in EventIDs, 0 to block them.
	Reserved uint8  // Must be 0.
	Count    uint16 // The number of event IDs in the EventIDs array.
}

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
	for _, provider := range s.config.Providers {
		// Set additional parameters for trace enabling.
		// See https://learn.microsoft.com/en-us/windows/win32/api/evntrace/ns-evntrace-enable_trace_parameters#members
		params := EnableTraceParameters{
			EnableProperty: computeStringEnableProperty(provider.EnableProperty),
			Version:        2, // ENABLE_TRACE_PARAMETERS_VERSION_2
		}

		// The filterData slice is declared here in the parent scope to ensure its
		// memory remains valid for the duration of the enableTrace call.
		var filterData [][]byte
		descriptors, err := buildEventDescriptor(&filterData, provider.EventFilter)
		if err != nil {
			return fmt.Errorf("failed to configure event filters for provider %s: %w", provider.Name, err)
		}

		// If any descriptors were created, point the params to them.
		if len(descriptors) > 0 {
			params.EnableFilterDesc = &descriptors[0]
			params.FilterDescrCount = uint32(len(descriptors)) // #nosec
		}

		// Zero timeout means asynchronous enablement
		const timeout = 0

		guid, err := getProviderGUID(provider)
		if err != nil {
			return fmt.Errorf("error parsing GUID: %w", err)
		}

		// Enable the trace session with extended options.
		err = s.enableTrace(s.handler, &guid, EVENT_CONTROL_CODE_ENABLE_PROVIDER, getTraceLevel(provider.TraceLevel), provider.MatchAnyKeyword, provider.MatchAllKeyword, timeout, &params)
		switch {
		case err == nil:
			continue
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
	return nil
}

// StopSession closes the ETW session and associated handles if they were created.
func (s *Session) StopSession() error {
	if !s.Realtime {
		return nil
	}

	// try to flush all buffer before stopping the session
	_ = s.controlTrace(
		s.handler,
		nil,
		s.properties,
		EVENT_TRACE_CONTROL_FLUSH,
	)

	// give time to process any flushed events
	time.Sleep(time.Second)

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

// getProviderGUID determines the GUID based on the provided configuration.
func getProviderGUID(conf ProviderConfig) (windows.GUID, error) {
	var guid windows.GUID
	var err error

	// If ProviderGUID is not set in the configuration, attempt to resolve it using the provider name.
	if conf.GUID == "" {
		guid, err = guidFromProviderNameFunc(conf.Name)
		if err != nil {
			return windows.GUID{}, fmt.Errorf("error resolving GUID: %w", err)
		}
	} else {
		// If ProviderGUID is set, parse it into a GUID structure.
		guid, err = windows.GUIDFromString(conf.GUID)
		if err != nil {
			return windows.GUID{}, fmt.Errorf("error parsing Windows GUID: %w", err)
		}
	}

	return guid, nil
}

// buildEventDescriptor populates the provided filterData slice and returns a
// descriptor that point to the data within it.
func buildEventDescriptor(buf *[][]byte, filter EventFilter) ([]EventFilterDescriptor, error) {
	if len(filter.EventIDs) == 0 {
		return nil, nil
	}

	payload, err := createEventIDFilter(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to create event ID filter payload: %w", err)
	}
	*buf = append(*buf, payload)

	// The descriptor points to the payload we just created.
	// We take a pointer to the first element of the most recently added payload.
	descriptor := EventFilterDescriptor{
		Ptr:  uint64(uintptr(unsafe.Pointer(&(*buf)[len(*buf)-1][0]))),
		Size: uint32(len(payload)), // #nosec
		Type: EVENT_FILTER_TYPE_EVENT_ID,
	}

	return []EventFilterDescriptor{descriptor}, nil
}

// createEventIDFilter builds the specific byte payload needed for an EVENT_ID filter
// by serializing a header struct and the list of event IDs.
func createEventIDFilter(filter EventFilter) ([]byte, error) {
	header := eventFilterEventIDHeader{
		Count: uint16(len(filter.EventIDs)), // #nosec
	}
	if filter.FilterIn {
		header.FilterIn = 1
	}

	buf := new(bytes.Buffer)

	// Write the fixed-size header struct to the buffer.
	if err := binary.Write(buf, binary.LittleEndian, header); err != nil {
		return nil, fmt.Errorf("failed to write filter header: %w", err)
	}

	// Write the dynamic part (the event IDs slice) to the buffer.
	if err := binary.Write(buf, binary.LittleEndian, filter.EventIDs); err != nil {
		return nil, fmt.Errorf("failed to write filter event IDs: %w", err)
	}

	return buf.Bytes(), nil
}
