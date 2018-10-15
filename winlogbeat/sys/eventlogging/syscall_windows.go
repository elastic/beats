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

package eventlogging

import (
	"syscall"
)

// Handle to an OS specific object.
type Handle uintptr

const (
	// MaxEventBufferSize is the maximum buffer size supported by ReadEventLog.
	MaxEventBufferSize = 0x7ffff

	// MaxFormatMessageBufferSize is the maximum buffer size supported by FormatMessage.
	MaxFormatMessageBufferSize = 1 << 16
)

// Event Log Error Codes
// https://msdn.microsoft.com/en-us/library/windows/desktop/ms681385(v=vs.85).aspx
const (
	ERROR_MR_MID_NOT_FOUND      syscall.Errno = 317
	ERROR_EVENTLOG_FILE_CORRUPT syscall.Errno = 1500
	ERROR_EVENTLOG_FILE_CHANGED syscall.Errno = 1503
)

// Flags to use with LoadLibraryEx.
// https://msdn.microsoft.com/en-us/library/windows/desktop/ms684179(v=vs.85).aspx
const (
	DONT_RESOLVE_DLL_REFERENCES         uint32 = 0x0001
	LOAD_LIBRARY_AS_DATAFILE            uint32 = 0x0002
	LOAD_WITH_ALTERED_SEARCH_PATH       uint32 = 0x0008
	LOAD_IGNORE_CODE_AUTHZ_LEVEL        uint32 = 0x0010
	LOAD_LIBRARY_AS_IMAGE_RESOURCE      uint32 = 0x0020
	LOAD_LIBRARY_AS_DATAFILE_EXCLUSIVE  uint32 = 0x0040
	LOAD_LIBRARY_SEARCH_DLL_LOAD_DIR    uint32 = 0x0100
	LOAD_LIBRARY_SEARCH_APPLICATION_DIR uint32 = 0x0200
	LOAD_LIBRARY_SEARCH_USER_DIRS       uint32 = 0x0400
	LOAD_LIBRARY_SEARCH_SYSTEM32        uint32 = 0x0800
	LOAD_LIBRARY_SEARCH_DEFAULT_DIRS    uint32 = 0x1000
)

// EventLogReadFlag indicates how to read the log file.
type EventLogReadFlag uint32

// EventLogReadFlag values.
// https://msdn.microsoft.com/en-us/library/windows/desktop/aa363674(v=vs.85).aspx
const (
	EVENTLOG_SEQUENTIAL_READ EventLogReadFlag = 1 << iota
	EVENTLOG_SEEK_READ
	EVENTLOG_FORWARDS_READ
	EVENTLOG_BACKWARDS_READ
)

// EventType identifies the five types of events that can be logged by
// applications.
type EventType uint16

// EventType values.
const (
	// Do not reorder.
	EVENTLOG_SUCCESS    EventType = 0
	EVENTLOG_ERROR_TYPE           = 1 << (iota - 1)
	EVENTLOG_WARNING_TYPE
	EVENTLOG_INFORMATION_TYPE
	EVENTLOG_AUDIT_SUCCESS
	EVENTLOG_AUDIT_FAILURE
)

// Mapping of event types to their string representations.
var eventTypeToString = map[EventType]string{
	EVENTLOG_SUCCESS:          "Success",
	EVENTLOG_ERROR_TYPE:       "Error",
	EVENTLOG_AUDIT_FAILURE:    "Audit Failure",
	EVENTLOG_AUDIT_SUCCESS:    "Audit Success",
	EVENTLOG_INFORMATION_TYPE: "Information",
	EVENTLOG_WARNING_TYPE:     "Warning",
}

// String returns string representation of EventType.
func (et EventType) String() string {
	return eventTypeToString[et]
}

// winEventLogRecord is equivalent to EVENTLOGRECORD.
// See https://msdn.microsoft.com/en-us/library/windows/desktop/aa363646(v=vs.85).aspx
type eventLogRecord struct {
	length        uint32 // The size of this event record, in bytes
	reserved      uint32 // value that is always set to ELF_LOG_SIGNATURE (the value is 0x654c664c), which is ASCII for eLfL
	recordNumber  uint32 // The number of the record.
	timeGenerated uint32 // time at which this entry was submitted
	timeWritten   uint32 // time at which this entry was received by the service to be written to the log
	eventID       uint32 // The event identifier. The value is specific to the event source for the event, and is used
	// with source name to locate a description string in the message file for the event source.
	eventType           uint16 // The type of event
	numStrings          uint16 // number of strings present in the log
	eventCategory       uint16 // category for this event
	reservedFlags       uint16 // Reserved
	closingRecordNumber uint32 // Reserved
	stringOffset        uint32 // offset of the description strings within this event log record
	userSidLength       uint32 // size of the UserSid member, in bytes. This value can be zero if no security identifier was provided
	userSidOffset       uint32 // offset of the security identifier (SID) within this event log record
	dataLength          uint32 // size of the event-specific data in bytes
	dataOffset          uint32 // offset of the event-specific information within this event log record, in bytes

	//
	// Then follows the extra data.
	//
	// TCHAR SourceName[]
	// TCHAR Computername[]
	// SID   UserSid
	// TCHAR Strings[]
	// BYTE  Data[]
	// CHAR  Pad[]
	// DWORD Length;

	sourceName   string
	computerName string
	userSid      []byte
}

// Add -trace to enable debug prints around syscalls.
//go:generate go run $GOROOT/src/syscall/mksyscall_windows.go -output zsyscall_windows.go syscall_windows.go

// Windows API calls
//sys   _OpenEventLog(uncServerName *uint16, sourceName *uint16) (handle Handle, err error) = advapi32.OpenEventLogW
//sys   _CloseEventLog(eventLog Handle) (err error) = advapi32.CloseEventLog
//sys   _ReadEventLog(eventLog Handle, readFlags EventLogReadFlag, recordOffset uint32, buffer *byte, numberOfBytesToRead uint32, bytesRead *uint32, minNumberOfBytesNeeded *uint32) (err error) = advapi32.ReadEventLogW
//sys   _LoadLibraryEx(filename *uint16, file Handle, flags uint32) (handle Handle, err error) = kernel32.LoadLibraryExW
//sys   _FormatMessage(flags uint32, source Handle, messageID uint32, languageID uint32, buffer *byte, bufferSize uint32, arguments uintptr) (numChars uint32, err error) = kernel32.FormatMessageW
//sys   _ClearEventLog(eventLog Handle, backupFileName *uint16) (err error) = advapi32.ClearEventLogW
//sys   _GetNumberOfEventLogRecords(eventLog Handle, numberOfRecords *uint32) (err error) = advapi32.GetNumberOfEventLogRecords
//sys   _GetOldestEventLogRecord(eventLog Handle, oldestRecord *uint32) (err error) = advapi32.GetOldestEventLogRecord
