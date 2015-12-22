// Examples:
// https://github.com/SublimeText/Pywin32/blob/master/lib/x32/win32/lib/win32evtlogutil.py
// https://msdn.microsoft.com/en-us/library/windows/desktop/bb427356(v=vs.85).aspx
// http://stormcoders.blogspot.com/2005/08/master-mysterious-eventlogrecord.html:w

package eventlog

import (
	"fmt"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/elastic/beats/libbeat/logp"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	maxEventBufferSize         = 0x7ffff                                 // Maximum buffer size supported by ReadEventLog.
	maxFormatMessageBufferSize = 1 << 16                                 // Maximum buffer size supported by FormatMessage.
	winEventLogRecordSize      = int(unsafe.Sizeof(winEventLogRecord{})) // Size in bytes of winEventLogRecord.
)

const (
	noMessageFile = "The description for Event ID (%d) in Source (%s) cannot be found. " +
		"The local computer may not have the necessary registry information or message " +
		"DLL files to display messages from a remote computer. The following " +
		"information is part of the event: %s"
)

// winEventLogRecord is equivalent to EVENTLOGRECORD.
// See https://msdn.microsoft.com/en-us/library/windows/desktop/aa363646(v=vs.85).aspx
type winEventLogRecord struct {
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
}

// IsAvailable returns nil if the Event Logging API is supported by this
// operating system. If not supported then an error is returned.
func IsAvailable() (bool, error) {
	err := modadvapi32.Load()
	if err != nil {
		return false, err
	}

	return true, nil
}

func (el *eventLog) Open(recordNumber uint64) error {
	// If uncServerPath is nil the local computer is used.
	var uncServerPath *uint16
	var err error
	if el.uncServerPath != "" {
		uncServerPath, err = syscall.UTF16PtrFromString(el.uncServerPath)
		if err != nil {
			return err
		}
	}

	providerName, err := syscall.UTF16PtrFromString(el.name)
	if err != nil {
		return err
	}

	detailf("%s Open(recordNumber=%d) calling openEventLog(uncServerPath=%s, providerName=%s)",
		el.logPrefix, recordNumber, el.uncServerPath, el.name)
	handle, err := openEventLog(uncServerPath, providerName)
	if err != nil {
		return err
	}

	numRecords, err := getNumberOfEventLogRecords(handle)
	if err != nil {
		return err
	}

	var oldestRecord, newestRecord uint32
	if numRecords > 0 {
		el.recordNumber = uint32(recordNumber)
		el.seek = true
		el.ignoreFirst = true

		oldestRecord, err = getOldestEventLogRecord(handle)
		if err != nil {
			return err
		}
		newestRecord = oldestRecord + numRecords - 1

		if el.recordNumber < oldestRecord || el.recordNumber > newestRecord {
			el.recordNumber = oldestRecord
			el.ignoreFirst = false
		}
	} else {
		el.recordNumber = 0
		el.seek = false
		el.ignoreFirst = false
	}

	logp.Info("%s contains %d records. Record number range [%d, %d]. Starting "+
		"at %d (ignoringFirst=%t)", el.logPrefix, numRecords, oldestRecord,
		newestRecord, el.recordNumber, el.ignoreFirst)

	el.handle = handle
	el.readBuf = make([]byte, maxEventBufferSize)
	// TODO: Start with this buffer smaller and grow it when needed.
	el.formatBuf = make([]byte, maxFormatMessageBufferSize)
	return nil
}

func (el *eventLog) Read() ([]LogRecord, error) {
	var numBytesRead, minBytesToRead uint32

	var flags uint32 = EVENTLOG_SEQUENTIAL_READ | EVENTLOG_FORWARDS_READ
	if el.seek {
		flags = EVENTLOG_SEEK_READ | EVENTLOG_FORWARDS_READ
		el.seek = false
	}

	err := retry(
		func() error {
			return readEventLog(
				el.handle,
				flags,
				el.recordNumber,
				&el.readBuf[0],
				uint32(len(el.readBuf)),
				&numBytesRead,
				&minBytesToRead)
		},
		el.readRetryErrorHandler)
	if err != nil {
		debugf("%s ReadEventLog returned error %s", el.logPrefix, err)
		return readErrorHandler(err)
	}
	detailf("%s ReadEventLog read %d bytes", el.logPrefix, numBytesRead)

	var records []LogRecord
	var readOffset uint32
	for readOffset < numBytesRead {
		event := (*winEventLogRecord)(unsafe.Pointer(&el.readBuf[readOffset]))
		singleBuf := el.readBuf[readOffset : readOffset+event.length]
		readOffset += event.length

		sourceName, extraDataOffset, err := UTF16BytesToString(singleBuf[winEventLogRecordSize:])
		if err != nil {
			logp.Warn("%s Failed to read sourceName from event "+
				"data. Skipping event. event=%v, %v", el.logPrefix, event, err)
			continue
		}
		computerName, _, err := UTF16BytesToString(singleBuf[winEventLogRecordSize+extraDataOffset:])
		if err != nil {
			logp.Warn("%s Failed to read computerName from event "+
				"data. Skipping event. event=%v, %v", el.logPrefix, event, err)
			continue
		}

		// TODO: Lookup EventCategory string.
		lr := LogRecord{
			EventLogName:  el.name,
			RecordNumber:  uint64(event.recordNumber),
			EventID:       event.eventID,
			EventType:     EventType(event.eventType).String(),
			TimeGenerated: time.Unix(int64(event.timeGenerated), 0),
			SourceName:    sourceName,
			ComputerName:  computerName,
		}

		if event.userSidLength > 0 {
			sid := (*windows.SID)(unsafe.Pointer(&singleBuf[event.userSidOffset]))
			account, domain, accountType, lookupErr := sid.LookupAccount("")
			if lookupErr != nil {
				logp.Warn("%s Failed to lookup account associated "+
					"with SID. UserSID will be nil. event=%v, %v",
					el.logPrefix, event, err)
				continue
			}

			lr.UserSID = &SID{
				Name:    account,
				Domain:  domain,
				SIDType: SIDType(accountType),
			}
		}

		message, err := el.formatMessage(event, singleBuf, lr)
		if err != nil {
			logp.Warn("%s Failed to format message. Skipping event. "+
				"event=%v, %v", el.logPrefix, event, err)
			continue
		}
		lr.Message = message

		detailf("%s Read log record from buffer[%d:%d] (len=%d). %s",
			el.logPrefix, readOffset-event.length, readOffset, event.length, lr)
		records = append(records, lr)
	}

	if el.ignoreFirst && len(records) > 0 {
		debugf("%s Ignoring first event with record number %d", el.logPrefix,
			records[0].RecordNumber)
		records = records[1:]
		el.ignoreFirst = false
	}

	debugf("%s Read() is returning %d log records", el.logPrefix, len(records))
	return records, nil
}

func (el *eventLog) Close() error {
	debugf("%s Closing handle", el.logPrefix)
	return closeEventLog(el.handle)
}

// formatMessage builds the message text that is associated with an event log
// record. Each EventID has a template that is stored in a library. The event
// contains the parameters used to populate the template. This method evaluates
// the template with the parameters and returns the resulting string.
//
// https://msdn.microsoft.com/en-us/library/windows/desktop/aa363651(v=vs.85).aspx#_win32_description_strings
func (el *eventLog) formatMessage(
	event *winEventLogRecord,
	buf []byte,
	lr LogRecord,
) (string, error) {
	// Get string values and addresses of the inserts:
	stringInserts, stringInsertPtrs, err := getStrings(event, buf)
	if err != nil {
		logp.Warn("%s Failed to get string inserts for "+
			"parameterized message. eventID=%d, recordNumber=%d, %v",
			el.logPrefix, event.eventID, event.recordNumber, err)
		return "", err
	}
	detailf("%s String inserts are [%s]. eventID=%d, recordNumber=%d",
		el.logPrefix, strings.Join(stringInserts, ","), event.eventID,
		event.recordNumber)

	var addr *uintptr
	if stringInsertPtrs != nil && len(stringInsertPtrs) > 0 {
		addr = &stringInsertPtrs[0]
	}

	handles := el.handles.get(lr.SourceName)

	var message string
	for _, handle := range handles {
		numChars, err := formatMessage(
			windows.FORMAT_MESSAGE_FROM_SYSTEM|
				windows.FORMAT_MESSAGE_FROM_HMODULE|
				windows.FORMAT_MESSAGE_ARGUMENT_ARRAY,
			handle,
			event.eventID,
			0, // Language ID
			&el.formatBuf[0],
			uint32(len(el.formatBuf)),
			addr)
		if err != nil {
			detailf("%s Failed to format message for eventID=%d with event "+
				"message file handle=%v. Will try next handle if there are more",
				el.logPrefix, event.eventID, handle)
			continue
		}

		message, _, err = UTF16BytesToString(el.formatBuf[:numChars*2])
		if err != nil {
			detailf("%s Failed to convert UTF16 buffer[:%d] to string. event=%v",
				el.logPrefix, numChars*2, event)
			break
		}

		// Cleanup windows line endings.
		message = strings.Replace(message, "\r\n", "\n", -1)
		message = strings.TrimRight(message, "\n")
		detailf("%s Formatted message. eventID=%d, recordNumber=%d, message='%s'",
			el.logPrefix, event.eventID, event.recordNumber, message)
	}

	if message == "" {
		message = fmt.Sprintf(noMessageFile, lr.EventID, lr.SourceName,
			strings.Join(stringInserts, ", "))
	}

	return message, nil
}

func getStrings(event *winEventLogRecord, buf []byte) ([]string, []uintptr, error) {
	inserts := make([]string, event.numStrings)
	insertPtrs := make([]uintptr, event.numStrings)

	bufPtr := uintptr(unsafe.Pointer(&buf[0]))
	offset := int(event.stringOffset)
	for i := 0; i < int(event.numStrings); i++ {
		evtStr, length, err := UTF16BytesToString(buf[offset:])
		if err != nil {
			return nil, nil, err
		}
		inserts[i] = evtStr
		insertPtrs[i] = bufPtr + uintptr(offset)
		offset += length
	}

	return inserts, insertPtrs, nil
}

// queryEventMessageFiles queries the registry to get the value of
// the EventMessageFile key that points to a DLL or EXE containing parameterized
// event log messages. If found, it loads the libraries as a datafiles and
// returns a slice of Handles to the libraries.
func queryEventMessageFiles(providerName, sourceName string) ([]Handle, error) {
	// Open key in registry:
	registryKeyName := fmt.Sprintf(
		"SYSTEM\\CurrentControlSet\\Services\\EventLog\\%s\\%s",
		providerName, sourceName)
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, registryKeyName,
		registry.QUERY_VALUE)
	if err != nil {
		return nil, fmt.Errorf("Failed to open HKLM\\%s", registryKeyName)
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
		return nil, fmt.Errorf("Failed querying EventMessageFile from "+
			"HKLM\\%s. %v", registryKeyName, err)
	}
	value, err = registry.ExpandString(value)
	if err != nil {
		return nil, err
	}

	// Split the value in case there is more than one file in the value.
	eventMessageFiles := strings.Split(value, ";")
	logp.Debug("eventlog", "RegQueryValueEx queried EventMessageFile from "+
		"HKLM\\%s and got [%s]", registryKeyName,
		strings.Join(eventMessageFiles, ","))

	// Load the libraries:
	var handles []Handle
	for _, eventMessageFile := range eventMessageFiles {
		sPtr, err := syscall.UTF16PtrFromString(eventMessageFile)
		if err != nil {
			logp.Debug("eventlog", "Failed to get UTF16Ptr for '%s'. "+
				"Skipping. %v", eventMessageFile, err)
			continue
		}
		handle, err := loadLibraryEx(sPtr, 0, LOAD_LIBRARY_AS_DATAFILE)
		if err != nil {
			logp.Debug("eventlog", "Failed to load library '%s' as data file. "+
				"Skipping. %v", eventMessageFile, err)
			continue
		}
		handles = append(handles, handle)
	}

	logp.Debug("eventlog", "Returning handles %v for sourceName %s", handles,
		sourceName)
	return handles, nil
}

// readRetryErrorHandler handles errors returned from the readEventLog function
// by attempting to correct the error through closing and reopening the event
// log.
func (el *eventLog) readRetryErrorHandler(err error) error {
	if errno, ok := err.(syscall.Errno); ok {
		var reopen bool

		switch errno {
		case ERROR_EVENTLOG_FILE_CHANGED:
			debugf("Re-opening event log because event log file was changed")
			reopen = true
		case ERROR_EVENTLOG_FILE_CORRUPT:
			debugf("Re-opening event log because event log file is corrupt")
			reopen = true
		}

		if reopen {
			el.Close()
			return el.Open(uint64(el.recordNumber))
		}
	}
	return err
}

// readErrorHandler handles errors returned by the readEventLog function.
func readErrorHandler(err error) ([]LogRecord, error) {
	if errno, ok := err.(syscall.Errno); ok {
		switch errno {
		case syscall.ERROR_HANDLE_EOF,
			ERROR_EVENTLOG_FILE_CHANGED,
			ERROR_EVENTLOG_FILE_CORRUPT:
			return []LogRecord{}, nil
		}
	}
	return nil, err
}
