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

package wineventlog

import (
	"fmt"

	"golang.org/x/sys/windows"

	"github.com/elastic/beats/v9/winlogbeat/sys"
)

// getMessageStringFromHandle returns the message for the given eventHandle.
func getMessageStringFromHandle(metadata *PublisherMetadata, eventHandle EvtHandle, values []EvtVariant) (string, error) {
	return getMessageString(metadata, eventHandle, 0, values)
}

// getMessageStringFromMessageID returns the message associated with the given
// message ID.
func getMessageStringFromMessageID(metadata *PublisherMetadata, messageID uint32, values []EvtVariant) (string, error) {
	return getMessageString(metadata, NilHandle, messageID, values)
}

// getMessageString returns an event's message. Don't use this directly. Instead
// use either getMessageStringFromHandle or getMessageStringFromMessageID.
func getMessageString(metadata *PublisherMetadata, eventHandle EvtHandle, messageID uint32, values []EvtVariant) (string, error) {
	var flags EvtFormatMessageFlag
	if eventHandle > 0 {
		flags = EvtFormatMessageEvent
	} else {
		flags = EvtFormatMessageId
	}

	metadataHandle := NilHandle
	if metadata != nil {
		metadataHandle = metadata.Handle
	}

	return evtFormatMessage(metadataHandle, eventHandle, messageID, values, flags)
}

// getEventXML returns all data in the event as XML.
func getEventXML(metadata *PublisherMetadata, eventHandle EvtHandle) (string, error) {
	metadataHandle := NilHandle
	if metadata != nil {
		metadataHandle = metadata.Handle
	}
	return evtFormatMessage(metadataHandle, eventHandle, 0, nil, EvtFormatMessageXml)
}

// evtFormatMessage uses EvtFormatMessage to generate a string.
func evtFormatMessage(metadataHandle EvtHandle, eventHandle EvtHandle, messageID uint32, values []EvtVariant, messageFlag EvtFormatMessageFlag) (string, error) {
	var (
		valuesCount = uint32(len(values))
		valuesPtr   *EvtVariant
	)
	if len(values) != 0 {
		valuesPtr = &values[0]
	}

	// best guess render buffer size, to avoid rendering message twice in most cases
	const bestGuessRenderBufferSize = 1 << 19 // 512KB, 256K wide characters

	// EvtFormatMessage operates with WCHAR buffer, assuming the size of the buffer in characters.
	// https://docs.microsoft.com/en-us/windows/win32/api/winevt/nf-winevt-evtformatmessage
	var wcharBufferUsed uint32
	wcharBufferSize := uint32(bestGuessRenderBufferSize / 2)

	// Get a buffer from the pool and adjust its length.
	bb := sys.NewPooledByteBuffer()
	defer bb.Free()
	bb.Reserve(int(wcharBufferSize * 2))

	err := _EvtFormatMessage(metadataHandle, eventHandle, messageID, valuesCount, valuesPtr, messageFlag, wcharBufferSize, bb.PtrAt(0), &wcharBufferUsed)
	switch err { //nolint:errorlint // This is an errno or nil.
	// Ignore some errors so it can tolerate missing or mismatched parameter values.
	case nil, // OK
		windows.ERROR_EVT_UNRESOLVED_VALUE_INSERT,
		windows.ERROR_EVT_UNRESOLVED_PARAMETER_INSERT,
		windows.ERROR_EVT_MAX_INSERTS_REACHED:
		// wcharBufferUsed indicates the size used internally to render the message. When called with nil buffer
		// EvtFormatMessage returns ERROR_INSUFFICIENT_BUFFER, but otherwise succeeds copying only up to
		// wcharBufferSize to our buffer, truncating the message if our buffer was too small.
		if wcharBufferUsed <= wcharBufferSize {
			return sys.UTF16BytesToString(bb.Bytes())
		}
		fallthrough

	case windows.ERROR_INSUFFICIENT_BUFFER:
		bb.Reserve(int(wcharBufferUsed * 2))
		wcharBufferSize = wcharBufferUsed

	default:
		return "", fmt.Errorf("failed in EvtFormatMessage: %w", err)
	}

	err = _EvtFormatMessage(metadataHandle, eventHandle, messageID, valuesCount, valuesPtr, messageFlag, wcharBufferSize, bb.PtrAt(0), &wcharBufferUsed)
	switch err { //nolint:errorlint // This is an errno or nil.
	case nil: // OK

	// Ignore some errors so it can tolerate missing or mismatched parameter values.
	case windows.ERROR_EVT_UNRESOLVED_VALUE_INSERT,
		windows.ERROR_EVT_UNRESOLVED_PARAMETER_INSERT,
		windows.ERROR_EVT_MAX_INSERTS_REACHED:

	default:
		return "", fmt.Errorf("failed in EvtFormatMessage: %w", err)
	}

	return sys.UTF16BytesToString(bb.Bytes())
}
