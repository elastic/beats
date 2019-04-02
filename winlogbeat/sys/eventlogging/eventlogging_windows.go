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
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/winlogbeat/sys"
)

// The value of EventID element contains the low-order 16 bits of the event
// identifier and the Qualifier attribute contains the high-order 16 bits of the
// event identifier.
const (
	eventIDLowerMask uint32 = 0xFFFF
	eventIDUpperMask uint32 = 0xFFFF0000
)

// IsAvailable returns true if the Event Logging API is supported by this
// operating system. If not supported then false is returned with the
// accompanying error.
func IsAvailable() (bool, error) {
	err := modadvapi32.Load()
	if err != nil {
		return false, err
	}

	return true, nil
}

// EventLogs returns a list of available event logs on the system.
func EventLogs() ([]string, error) {
	return nil, fmt.Errorf("Not implemented yet.")
}

// OpenEventLog opens the Windows Event Log and returns the handle for it.
func OpenEventLog(uncServerPath, logName string) (Handle, error) {
	// If uncServerPath is nil the local computer is used.
	var server *uint16
	var err error
	if uncServerPath != "" {
		server, err = syscall.UTF16PtrFromString(uncServerPath)
		if err != nil {
			return 0, err
		}
	}

	name, err := syscall.UTF16PtrFromString(logName)
	if err != nil {
		return 0, err
	}

	handle, err := _OpenEventLog(server, name)
	if err != nil {
		return 0, err
	}

	return handle, nil
}

// ReadEventLog takes the handle for the Windows Event Log, and reads through a
// buffer to prevent buffer overflows.
func ReadEventLog(
	handle Handle,
	flags EventLogReadFlag,
	recordID uint32,
	buffer []byte,
) (int, error) {
	var numBytesRead, minBytesRequiredToRead uint32
	err := _ReadEventLog(handle, flags, recordID,
		&buffer[0], uint32(len(buffer)),
		&numBytesRead, &minBytesRequiredToRead)
	if err == syscall.ERROR_INSUFFICIENT_BUFFER {
		return 0, sys.InsufficientBufferError{err, int(minBytesRequiredToRead)}
	}
	if err != nil {
		return 0, err
	}

	if int(numBytesRead) > len(buffer) {
		return 0, fmt.Errorf("Number of bytes read (%d) is greater than the "+
			"buffer length (%d).", numBytesRead, cap(buffer))
	}

	return int(numBytesRead), nil
}

// RenderEvents reads raw events from the provided buffer, formats them into
// structured events, and adds each on to a slice that is returned.
func RenderEvents(
	eventsRaw []byte,
	lang uint32,
	buffer []byte,
	insertStrings *StringInserts,
	pubHandleProvider func(string) sys.MessageFiles,
) ([]sys.Event, int, error) {
	var events []sys.Event
	var offset int
	for {
		if offset >= len(eventsRaw) {
			break
		}

		// Read a single EVENTLOGRECORD from the buffer.
		record, err := parseEventLogRecord(eventsRaw[offset:])
		if err != nil {
			return nil, 0, err
		}

		var qualifier = uint16((record.eventID & eventIDUpperMask) >> 16)
		var eventID = record.eventID & eventIDLowerMask
		event := sys.Event{
			Provider:        sys.Provider{Name: record.sourceName},
			EventIdentifier: sys.EventIdentifier{ID: eventID, Qualifiers: qualifier},
			LevelRaw:        uint8(record.eventType), // Possible overflow
			TaskRaw:         record.eventCategory,
			TimeCreated:     sys.TimeCreated{unixTime(record.timeGenerated)},
			RecordID:        uint64(record.recordNumber),
			Computer:        record.computerName,
			Level:           EventType(record.eventType).String(),
		}

		// Create a slice from the larger buffer only data from the one record.
		// The upper bound has been validated already by parseEventLogRecord.
		recordBuf := eventsRaw[offset : offset+int(record.length)]
		offset += int(record.length)

		// Parse and format the user that logged the event.
		sid, _ := parseSID(record, recordBuf) // TODO: do something with this error
		if sid != nil {
			event.User = *sid
		}

		if record.numStrings > MaxInsertStrings {
			logp.Warn("Record contains %d strings, more than the limit %d. Excess will be ignored.",
				record.numStrings, MaxInsertStrings)
			record.numStrings = MaxInsertStrings
		}
		// Parse the UTF-16 message insert strings.
		if err = insertStrings.Parse(record, recordBuf); err != nil {
			event.RenderErr = append(event.RenderErr, err.Error())
			events = append(events, event)
			continue
		}

		for _, s := range insertStrings.Strings() {
			event.EventData.Pairs = append(event.EventData.Pairs, sys.KeyValue{Value: s})
		}

		// Format the parametrized message using the insert strings.
		event.Message, err = formatMessage(record.sourceName,
			record.eventID, lang, insertStrings.Pointer(), buffer, pubHandleProvider)
		if err != nil {
			event.RenderErr = append(event.RenderErr, err.Error())
			if errno, ok := err.(syscall.Errno); ok {
				event.RenderErrorCode = uint32(errno)
			}
		}

		events = append(events, event)
	}

	return events, 0, nil
}

// unixTime takes a time which is an unsigned 32-bit integer, and converts it
// into a Golang time.Time pointer formatted as a unix time.
func unixTime(sec uint32) time.Time {
	t := time.Unix(int64(sec), 0)
	return t
}

// formatMessage takes event data and formats the event message into a
// normalized format.
func formatMessage(
	sourceName string,
	eventID uint32,
	lang uint32,
	stringInserts uintptr,
	buffer []byte,
	pubHandleProvider func(string) sys.MessageFiles,
) (string, error) {
	messageFiles := pubHandleProvider(sourceName)

	var lastErr error
	var fh sys.FileHandle
	var message string
	for _, fh = range messageFiles.Handles {
		if fh.Err != nil {
			lastErr = fh.Err
			continue
		}

		numChars, err := _FormatMessage(
			windows.FORMAT_MESSAGE_FROM_HMODULE|
				windows.FORMAT_MESSAGE_ARGUMENT_ARRAY,
			Handle(fh.Handle),
			eventID,
			lang,
			&buffer[0],            // Max size allowed is 64k bytes.
			uint32(len(buffer)/2), // Size of buffer in TCHARS
			stringInserts)
		// bufferUsed = numChars * sizeof(TCHAR) + sizeof(null-terminator)
		bufferUsed := int(numChars*2 + 2)
		if err == syscall.ERROR_INSUFFICIENT_BUFFER {
			return "", err
		}
		if err != nil {
			lastErr = err
			continue
		}

		if bufferUsed > len(buffer) {
			return "", fmt.Errorf("Windows FormatMessage reported that "+
				"message contains %d characters plus a null-terminator "+
				"(%d bytes), but the buffer can only hold %d bytes",
				numChars, bufferUsed, len(buffer))
		}
		message, _, err = sys.UTF16BytesToString(buffer[:bufferUsed])
		if err != nil {
			return "", err
		}
	}

	if message == "" {
		switch lastErr {
		case nil:
			return "", messageFiles.Err
		case ERROR_MR_MID_NOT_FOUND:
			return "", fmt.Errorf("The system cannot find message text for "+
				"message number %d in the message file for %s.", eventID, fh.File)
		default:
			return "", fh.Err
		}
	}

	return message, nil
}

// parseEventLogRecord parses a single Windows EVENTLOGRECORD struct from the
// buffer.
func parseEventLogRecord(buffer []byte) (eventLogRecord, error) {
	var record eventLogRecord
	reader := bytes.NewReader(buffer)

	// Length
	err := binary.Read(reader, binary.LittleEndian, &record.length)
	if err != nil {
		return record, err
	}
	if len(buffer) < int(record.length) {
		return record, fmt.Errorf("Decoded EVENTLOGRECORD length (%d) is "+
			"greater than the buffer length (%d)", record.length, len(buffer))
	}

	// Reserved
	err = binary.Read(reader, binary.LittleEndian, &record.reserved)
	if err != nil {
		return record, err
	}
	if record.reserved != uint32(0x654c664c) {
		return record, fmt.Errorf("Buffer does not contain ELF_LOG_SIGNATURE. "+
			"The data is invalid. Value is %X", record.reserved)
	}

	// Buffer appears to be value so slice it to the adjust length.
	buffer = buffer[:record.length]
	reader = bytes.NewReader(buffer)
	reader.Seek(8, 0)

	// RecordNumber
	err = binary.Read(reader, binary.LittleEndian, &record.recordNumber)
	if err != nil {
		return record, err
	}

	// TimeGenerated
	err = binary.Read(reader, binary.LittleEndian, &record.timeGenerated)
	if err != nil {
		return record, err
	}

	// TimeWritten
	err = binary.Read(reader, binary.LittleEndian, &record.timeWritten)
	if err != nil {
		return record, err
	}

	// EventID
	err = binary.Read(reader, binary.LittleEndian, &record.eventID)
	if err != nil {
		return record, err
	}

	// EventType
	err = binary.Read(reader, binary.LittleEndian, &record.eventType)
	if err != nil {
		return record, err
	}

	// NumStrings
	err = binary.Read(reader, binary.LittleEndian, &record.numStrings)
	if err != nil {
		return record, err
	}

	// EventCategory
	err = binary.Read(reader, binary.LittleEndian, &record.eventCategory)
	if err != nil {
		return record, err
	}

	// ReservedFlags (2 bytes), ClosingRecordNumber (4 bytes)
	_, err = reader.Seek(6, 1)
	if err != nil {
		return record, err
	}

	// StringOffset
	err = binary.Read(reader, binary.LittleEndian, &record.stringOffset)
	if err != nil {
		return record, err
	}
	if record.numStrings > 0 && record.stringOffset > record.length {
		return record, fmt.Errorf("StringOffset value (%d) is invalid "+
			"because it is greater than the Length (%d)", record.stringOffset,
			record.length)
	}

	// UserSidLength
	err = binary.Read(reader, binary.LittleEndian, &record.userSidLength)
	if err != nil {
		return record, err
	}

	// UserSidOffset
	err = binary.Read(reader, binary.LittleEndian, &record.userSidOffset)
	if err != nil {
		return record, err
	}
	if record.userSidLength > 0 && record.userSidOffset > record.length {
		return record, fmt.Errorf("UserSidOffset value (%d) is invalid "+
			"because it is greater than the Length (%d)", record.userSidOffset,
			record.length)
	}

	// DataLength
	err = binary.Read(reader, binary.LittleEndian, &record.dataLength)
	if err != nil {
		return record, err
	}

	// DataOffset
	err = binary.Read(reader, binary.LittleEndian, &record.dataOffset)
	if err != nil {
		return record, err
	}

	// SourceName (null-terminated UTF-16 string)
	begin, _ := reader.Seek(0, 1)
	sourceName, length, err := sys.UTF16BytesToString(buffer[begin:])
	if err != nil {
		return record, err
	}
	record.sourceName = sourceName
	begin, err = reader.Seek(int64(length), 1)
	if err != nil {
		return record, err
	}

	// ComputerName (null-terminated UTF-16 string)
	computerName, length, err := sys.UTF16BytesToString(buffer[begin:])
	if err != nil {
		return record, err
	}
	record.computerName = computerName
	_, err = reader.Seek(int64(length), 1)
	if err != nil {
		return record, err
	}

	return record, nil
}

func parseSID(record eventLogRecord, buffer []byte) (*sys.SID, error) {
	if record.userSidLength == 0 {
		return nil, nil
	}

	sid := (*windows.SID)(unsafe.Pointer(&buffer[record.userSidOffset]))
	identifier, err := sid.String()
	if err != nil {
		return nil, err
	}

	return &sys.SID{Identifier: identifier}, nil
}

// ClearEventLog takes an event log file handle and empties the log. If a backup
// filename is provided, this will back up the event log before clearing the logs.
func ClearEventLog(handle Handle, backupFileName string) error {
	var name *uint16
	if backupFileName != "" {
		var err error
		name, err = syscall.UTF16PtrFromString(backupFileName)
		if err != nil {
			return err
		}
	}

	return _ClearEventLog(handle, name)
}

// GetNumberOfEventLogRecords retrieves the number of events within a Windows
// log file handle.
func GetNumberOfEventLogRecords(handle Handle) (uint32, error) {
	var numRecords uint32
	err := _GetNumberOfEventLogRecords(handle, &numRecords)
	if err != nil {
		return 0, err
	}

	return numRecords, nil
}

// GetOldestEventLogRecord retrieves the oldest event within a Windows log file
// handle and returns the raw event.
func GetOldestEventLogRecord(handle Handle) (uint32, error) {
	var oldestRecord uint32
	err := _GetOldestEventLogRecord(handle, &oldestRecord)
	if err != nil {
		return 0, err
	}

	return oldestRecord, nil
}

// FreeLibrary frees the loaded dynamic-link library (DLL) module and,
// if necessary, decrements its reference count. When the reference count
// reaches zero, the module is unloaded from the address space of the calling
// process and the handle is no longer valid.
func FreeLibrary(handle uintptr) error {
	// Wrap the method so that we can stub it out and use our own Handle type.
	return windows.FreeLibrary(windows.Handle(handle))
}

// CloseEventLog takes an event log file handle, and closes the handle via
// _CloseEventLog
func CloseEventLog(handle Handle) error {
	return _CloseEventLog(handle)
}

// QueryEventMessageFiles queries the registry to get the value of
// the EventMessageFile key that points to a DLL or EXE containing parameterized
// event log messages. If found, it loads the libraries as a datafiles and
// returns a slice of Handles to the libraries. Those handles must be closed
// by the caller.
func QueryEventMessageFiles(providerName, sourceName string) sys.MessageFiles {
	mf := sys.MessageFiles{SourceName: sourceName}

	// Open key in registry:
	registryKeyName := fmt.Sprintf(
		"SYSTEM\\CurrentControlSet\\Services\\EventLog\\%s\\%s",
		providerName, sourceName)
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, registryKeyName,
		registry.QUERY_VALUE)
	if err != nil {
		mf.Err = fmt.Errorf("Failed to open HKLM\\%s", registryKeyName)
		return mf
	}
	defer func() {
		err := key.Close()
		if err != nil {
			logp.Warn("Failed to close registry key. key=%s err=%v",
				registryKeyName, err)
		}
	}()
	logp.Debug("eventlog", "RegOpenKey opened handle to HKLM\\%s, key=%v",
		registryKeyName, key)

	// Read value from registry:
	value, _, err := key.GetStringValue("EventMessageFile")
	if err != nil {
		mf.Err = fmt.Errorf("Failed querying EventMessageFile from "+
			"HKLM\\%s. %v", registryKeyName, err)
		return mf
	}
	value, err = registry.ExpandString(value)
	if err != nil {
		mf.Err = fmt.Errorf("Failed to expand strings in '%s'. %v", value, err)
		return mf
	}

	// Split the value in case there is more than one file in the value.
	eventMessageFiles := strings.Split(value, ";")
	logp.Debug("eventlog", "RegQueryValueEx queried EventMessageFile from "+
		"HKLM\\%s and got [%s]", registryKeyName,
		strings.Join(eventMessageFiles, ","))

	// Load the libraries:
	var files []sys.FileHandle
	for _, eventMessageFile := range eventMessageFiles {
		sPtr, err := syscall.UTF16PtrFromString(eventMessageFile)
		if err != nil {
			logp.Debug("eventlog", "Failed to get UTF16Ptr for '%s'. "+
				"Skipping. %v", eventMessageFile, err)
			continue
		}

		handle, err := _LoadLibraryEx(sPtr, 0, LOAD_LIBRARY_AS_DATAFILE)
		if err != nil {
			logp.Debug("eventlog", "Failed to load library '%s' as data file. "+
				"Skipping. %v", eventMessageFile, err)
		}

		f := sys.FileHandle{File: eventMessageFile, Handle: uintptr(handle), Err: err}
		files = append(files, f)
	}

	logp.Debug("eventlog", "Returning message files %+v for sourceName %s", files,
		sourceName)
	mf.Handles = files
	return mf
}
