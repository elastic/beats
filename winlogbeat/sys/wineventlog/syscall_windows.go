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

package wineventlog

import (
	"syscall"
)

// EvtHandle is a handle to the event log.
type EvtHandle uintptr

// Event log error codes.
// https://msdn.microsoft.com/en-us/library/windows/desktop/ms681382(v=vs.85).aspx
const (
	ERROR_INSUFFICIENT_BUFFER             syscall.Errno = 122
	ERROR_NO_MORE_ITEMS                   syscall.Errno = 259
	ERROR_NONE_MAPPED                     syscall.Errno = 1332
	RPC_S_INVALID_BOUND                   syscall.Errno = 1734
	ERROR_INVALID_OPERATION               syscall.Errno = 4317
	ERROR_EVT_MESSAGE_NOT_FOUND           syscall.Errno = 15027
	ERROR_EVT_MESSAGE_ID_NOT_FOUND        syscall.Errno = 15028
	ERROR_EVT_UNRESOLVED_VALUE_INSERT     syscall.Errno = 15029
	ERROR_EVT_UNRESOLVED_PARAMETER_INSERT syscall.Errno = 15030
)

// EvtSubscribeFlag defines the possible values that specify when to start subscribing to events.
type EvtSubscribeFlag uint32

// EVT_SUBSCRIBE_FLAGS enumeration
// https://msdn.microsoft.com/en-us/library/windows/desktop/aa385588(v=vs.85).aspx
const (
	EvtSubscribeToFutureEvents      EvtSubscribeFlag = 1
	EvtSubscribeStartAtOldestRecord EvtSubscribeFlag = 2
	EvtSubscribeStartAfterBookmark  EvtSubscribeFlag = 3
	EvtSubscribeOriginMask          EvtSubscribeFlag = 0x3
	EvtSubscribeTolerateQueryErrors EvtSubscribeFlag = 0x1000
	EvtSubscribeStrict              EvtSubscribeFlag = 0x10000
)

// EvtRenderFlag defines the values that specify what to render.
type EvtRenderFlag uint32

// EVT_RENDER_FLAGS enumeration
// https://msdn.microsoft.com/en-us/library/windows/desktop/aa385563(v=vs.85).aspx
const (
	// Render the event properties specified in the rendering context.
	EvtRenderEventValues EvtRenderFlag = iota
	// Render the event as an XML string. For details on the contents of the
	// XML string, see the Event schema.
	EvtRenderEventXml
	// Render the bookmark as an XML string, so that you can easily persist the
	// bookmark for use later.
	EvtRenderBookmark
)

// EvtRenderContextFlag defines the values that specify the type of information
// to access from the event.
type EvtRenderContextFlag uint32

// EVT_RENDER_CONTEXT_FLAGS enumeration
// https://msdn.microsoft.com/en-us/library/windows/desktop/aa385561(v=vs.85).aspx
const (
	// Render specific properties from the event.
	EvtRenderContextValues EvtRenderContextFlag = iota
	// Render the system properties under the System element.
	EvtRenderContextSystem
	// Render all user-defined properties under the UserData or EventData element.
	EvtRenderContextUser
)

// EvtFormatMessageFlag defines the values that specify the message string from
// the event to format.
type EvtFormatMessageFlag uint32

// EVT_FORMAT_MESSAGE_FLAGS enumeration
// https://msdn.microsoft.com/en-us/library/windows/desktop/aa385525(v=vs.85).aspx
const (
	// Format the event's message string.
	EvtFormatMessageEvent EvtFormatMessageFlag = iota + 1
	// Format the message string of the level specified in the event.
	EvtFormatMessageLevel
	// Format the message string of the task specified in the event.
	EvtFormatMessageTask
	// Format the message string of the task specified in the event.
	EvtFormatMessageOpcode
	// Format the message string of the keywords specified in the event. If the
	// event specifies multiple keywords, the formatted string is a list of
	// null-terminated strings. Increment through the strings until your pointer
	// points past the end of the used buffer.
	EvtFormatMessageKeyword
	// Format the message string of the channel specified in the event.
	EvtFormatMessageChannel
	// Format the provider's message string.
	EvtFormatMessageProvider
	// Format the message string associated with a resource identifier. The
	// provider's metadata contains the resource identifiers; the message
	// compiler assigns a resource identifier to each string when it compiles
	// the manifest.
	EvtFormatMessageId
	// Format all the message strings in the event. The formatted message is an
	// XML string that contains the event details and the message strings.
	EvtFormatMessageXml
)

// EvtSystemPropertyID defines the identifiers that identify the system-specific
// properties of an event.
type EvtSystemPropertyID uint32

// EVT_SYSTEM_PROPERTY_ID enumeration
// https://msdn.microsoft.com/en-us/library/windows/desktop/aa385606(v=vs.85).aspx
const (
	// Identifies the Name attribute of the provider element.
	// The variant type for this property is EvtVarTypeString.
	EvtSystemProviderName EvtSystemPropertyID = iota
	// Identifies the Guid attribute of the provider element.
	// The variant type for this property is EvtVarTypeGuid.
	EvtSystemProviderGuid
	// Identifies the EventID element.
	// The variant type for this property is EvtVarTypeUInt16.
	EvtSystemEventID
	// Identifies the Qualifiers attribute of the EventID element.
	// The variant type for this property is EvtVarTypeUInt16.
	EvtSystemQualifiers
	// Identifies the Level element.
	// The variant type for this property is EvtVarTypeUInt8.
	EvtSystemLevel
	// Identifies the Task element.
	// The variant type for this property is EvtVarTypeUInt16.
	EvtSystemTask
	// Identifies the Opcode element.
	// The variant type for this property is EvtVarTypeUInt8.
	EvtSystemOpcode
	// Identifies the Keywords element.
	// The variant type for this property is EvtVarTypeInt64.
	EvtSystemKeywords
	// Identifies the SystemTime attribute of the TimeCreated element.
	// The variant type for this property is EvtVarTypeFileTime.
	EvtSystemTimeCreated
	// Identifies the EventRecordID element.
	// The variant type for this property is EvtVarTypeUInt64.
	EvtSystemEventRecordId
	// Identifies the ActivityID attribute of the Correlation element.
	// The variant type for this property is EvtVarTypeGuid.
	EvtSystemActivityID
	// Identifies the RelatedActivityID attribute of the Correlation element.
	// The variant type for this property is EvtVarTypeGuid.
	EvtSystemRelatedActivityID
	// Identifies the ProcessID attribute of the Execution element.
	// The variant type for this property is EvtVarTypeUInt32.
	EvtSystemProcessID
	// Identifies the ThreadID attribute of the Execution element.
	// The variant type for this property is EvtVarTypeUInt32.
	EvtSystemThreadID
	// Identifies the Channel element.
	// The variant type for this property is EvtVarTypeString.
	EvtSystemChannel
	// Identifies the Computer element.
	// The variant type for this property is EvtVarTypeString.
	EvtSystemComputer
	// Identifies the UserID element.
	// The variant type for this property is EvtVarTypeSid.
	EvtSystemUserID
	// Identifies the Version element.
	// The variant type for this property is EvtVarTypeUInt8.
	EvtSystemVersion
	// This enumeration value marks the end of the enumeration values.
	EvtSystemPropertyIdEND
)

var evtSystemMap = map[EvtSystemPropertyID]string{
	EvtSystemProviderName:      "Provider Name",
	EvtSystemProviderGuid:      "Provider GUID",
	EvtSystemEventID:           "Event ID",
	EvtSystemQualifiers:        "Qualifiers",
	EvtSystemLevel:             "Level",
	EvtSystemTask:              "Task",
	EvtSystemOpcode:            "Opcode",
	EvtSystemKeywords:          "Keywords",
	EvtSystemTimeCreated:       "Time Created",
	EvtSystemEventRecordId:     "Record ID",
	EvtSystemActivityID:        "Activity ID",
	EvtSystemRelatedActivityID: "Related Activity ID",
	EvtSystemProcessID:         "Process ID",
	EvtSystemThreadID:          "Thread ID",
	EvtSystemChannel:           "Channel",
	EvtSystemComputer:          "Computer",
	EvtSystemUserID:            "User ID",
	EvtSystemVersion:           "Version",
}

func (e EvtSystemPropertyID) String() string {
	s, found := evtSystemMap[e]
	if !found {
		return "Unknown"
	}
	return s
}

// EventLevel identifies the six levels of events that can be logged
type EventLevel uint16

// EventLevel values.
const (
	// Do not reorder.
	EVENTLOG_LOGALWAYS_LEVEL EventLevel = iota
	EVENTLOG_CRITICAL_LEVEL
	EVENTLOG_ERROR_LEVEL
	EVENTLOG_WARNING_LEVEL
	EVENTLOG_INFORMATION_LEVEL
	EVENTLOG_VERBOSE_LEVEL
)

// Mapping of event levels to their string representations.
var EventLevelToString = map[EventLevel]string{
	EVENTLOG_LOGALWAYS_LEVEL:   "Information",
	EVENTLOG_INFORMATION_LEVEL: "Information",
	EVENTLOG_CRITICAL_LEVEL:    "Critical",
	EVENTLOG_ERROR_LEVEL:       "Error",
	EVENTLOG_WARNING_LEVEL:     "Warning",
	EVENTLOG_VERBOSE_LEVEL:     "Verbose",
}

// String returns string representation of EventLevel.
func (et EventLevel) String() string {
	return EventLevelToString[et]
}

// EvtQueryFlag defines the values that specify how to return the query results
// and whether you are query against a channel or log file.
type EvtQueryFlag uint32

const (
	// EvtQueryChannelPath specifies that the query is against one or more
	// channels. The Path parameter of the EvtQuery function must specify the
	// name of a channel or NULL.
	EvtQueryChannelPath EvtQueryFlag = 0x1
	// EvtQueryFilePath specifies that the query is against one or more log
	// files. The Path parameter of the EvtQuery function must specify the full
	// path to a log file or NULL.
	EvtQueryFilePath EvtQueryFlag = 0x2
	// EvtQueryForwardDirection specifies that the events in the query result
	// are ordered from oldest to newest. This is the default.
	EvtQueryForwardDirection EvtQueryFlag = 0x100
	// EvtQueryReverseDirection specifies that the events in the query result
	// are ordered from newest to oldest.
	EvtQueryReverseDirection EvtQueryFlag = 0x200
	// EvtQueryTolerateQueryErrors specifies that EvtQuery should run the query
	// even if the part of the query generates an error (is not well formed).
	EvtQueryTolerateQueryErrors EvtQueryFlag = 0x1000
)

// EvtOpenLogFlag defines the values that specify whether to open a channel or
// exported log file. This maps to EVT_OPEN_LOG_FLAGS in Windows.
type EvtOpenLogFlag uint32

const (
	// EvtOpenChannelPath opens a channel.
	EvtOpenChannelPath EvtOpenLogFlag = 0x1
	// EvtOpenFilePath opens an exported log file.
	EvtOpenFilePath EvtOpenLogFlag = 0x2
)

// EvtSeekFlag defines the relative position in the result set from which to seek.
type EvtSeekFlag uint32

const (
	// EvtSeekRelativeToFirst seeks to the specified offset from the first entry
	// in the result set. The offset must be a positive value.
	EvtSeekRelativeToFirst EvtSeekFlag = 1
	// EvtSeekRelativeToLast seeks to the specified offset from the last entry
	// in the result set. The offset must be a negative value.
	EvtSeekRelativeToLast EvtSeekFlag = 2
	// EvtSeekRelativeToCurrent seeks to the specified offset from the current
	// entry in the result set. The offset can be a positive or negative value.
	EvtSeekRelativeToCurrent EvtSeekFlag = 3
	// EvtSeekRelativeToBookmark seek to the specified offset from the
	// bookmarked entry in the result set. The offset can be a positive or
	// negative value.
	EvtSeekRelativeToBookmark EvtSeekFlag = 4
	// EvtSeekOriginMask is a bitmask that you can use to determine which of the
	// following flags is set:
	EvtSeekOriginMask EvtSeekFlag = 7
	// EvtSeekStrict forces the function to fail if the event does not exist.
	EvtSeekStrict EvtSeekFlag = 0x10000
)

// Add -trace to enable debug prints around syscalls.
//go:generate go run $GOROOT/src/syscall/mksyscall_windows.go -output zsyscall_windows.go syscall_windows.go

// Windows API calls
//sys   _EvtOpenLog(session EvtHandle, path *uint16, flags uint32) (handle EvtHandle, err error) = wevtapi.EvtOpenLog
//sys   _EvtQuery(session EvtHandle, path *uint16, query *uint16, flags uint32) (handle EvtHandle, err error) = wevtapi.EvtQuery
//sys   _EvtSubscribe(session EvtHandle, signalEvent uintptr, channelPath *uint16, query *uint16, bookmark EvtHandle, context uintptr, callback syscall.Handle, flags EvtSubscribeFlag) (handle EvtHandle, err error) = wevtapi.EvtSubscribe
//sys   _EvtCreateBookmark(bookmarkXML *uint16) (handle EvtHandle, err error) = wevtapi.EvtCreateBookmark
//sys   _EvtUpdateBookmark(bookmark EvtHandle, event EvtHandle) (err error) = wevtapi.EvtUpdateBookmark
//sys   _EvtCreateRenderContext(ValuePathsCount uint32, valuePaths uintptr, flags EvtRenderContextFlag) (handle EvtHandle, err error) = wevtapi.EvtCreateRenderContext
//sys   _EvtRender(context EvtHandle, fragment EvtHandle, flags EvtRenderFlag, bufferSize uint32, buffer *byte, bufferUsed *uint32, propertyCount *uint32) (err error) = wevtapi.EvtRender
//sys   _EvtClose(object EvtHandle) (err error) = wevtapi.EvtClose
//sys   _EvtSeek(resultSet EvtHandle, position int64, bookmark EvtHandle, timeout uint32, flags uint32) (success bool, err error) [!success] = wevtapi.EvtSeek
//sys   _EvtNext(resultSet EvtHandle, eventArraySize uint32, eventArray *EvtHandle, timeout uint32, flags uint32, numReturned *uint32) (err error) = wevtapi.EvtNext
//sys   _EvtOpenChannelEnum(session EvtHandle, flags uint32) (handle EvtHandle, err error) = wevtapi.EvtOpenChannelEnum
//sys   _EvtNextChannelPath(channelEnum EvtHandle, channelPathBufferSize uint32, channelPathBuffer *uint16, channelPathBufferUsed *uint32) (err error) = wevtapi.EvtNextChannelPath
//sys   _EvtFormatMessage(publisherMetadata EvtHandle, event EvtHandle, messageID uint32, valueCount uint32, values uintptr, flags EvtFormatMessageFlag, bufferSize uint32, buffer *byte, bufferUsed *uint32) (err error) = wevtapi.EvtFormatMessage
//sys   _EvtOpenPublisherMetadata(session EvtHandle, publisherIdentity *uint16, logFilePath *uint16, locale uint32, flags uint32) (handle EvtHandle, err error) = wevtapi.EvtOpenPublisherMetadata

//sys   _StringFromGUID2(rguid *syscall.GUID, pStr *uint16, strSize uint32) (err error) = ole32.StringFromGUID2
