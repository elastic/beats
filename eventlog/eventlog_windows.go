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
	"unicode/utf16"
	"unsafe"

	"github.com/elastic/libbeat/logp"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	maxEventBufferSize         = 0x7ffff // Maximum buffer size supported by ReadEventLog.
	maxFormatMessageBufferSize = 1 << 16 // Maximum buffer size supported by FormatMessage.
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
	// Then follow:
	//
	// TCHAR SourceName[]
	// TCHAR Computername[]
	// SID   UserSid
	// TCHAR Strings[]
	// BYTE  Data[]
	// CHAR  Pad[]
	// DWORD Length;
}

func (el *eventLog) Open(recordNumber uint32) error {
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

	handle, err := openEventLog(uncServerPath, providerName)
	if err != nil {
		return err
	}

	numRecords, err := getNumberOfEventLogRecords(handle)
	if err != nil {
		logp.Warn("EventLog[%s] Could not obtain total number of records: ", el.name)
	} else {
		logp.Info("EventLog[%s] contains %d records", el.name, numRecords)
	}

	el.handle = handle
	el.recordNumber = recordNumber
	el.readBuf = make([]byte, maxEventBufferSize)
	// TODO: Start with this buffer smaller and grow it when needed.
	el.formatBuf = make([]byte, maxFormatMessageBufferSize)
	return nil
}

func (el *eventLog) Read() ([]LogRecord, error) {
	var numBytesRead, minBytesToRead uint32
	err := readEventLog(el.handle,
		EVENTLOG_SEQUENTIAL_READ|EVENTLOG_FORWARDS_READ, 0,
		&el.readBuf[0], uint32(len(el.readBuf)), &numBytesRead, &minBytesToRead)
	if err != nil {
		errno, ok := err.(syscall.Errno)
		if ok && errno == syscall.ERROR_HANDLE_EOF {
			// Ignore EOF and return empty.
			return []LogRecord{}, nil
		}
		// TODO: Add special handling for other error conditions like
		// ERROR_EVENTLOG_FILE_CHANGED.
		return nil, err
	}

	var records []LogRecord
	var readPtr uint32
	for readPtr < numBytesRead {
		event := (*winEventLogRecord)(unsafe.Pointer(&el.readBuf[readPtr]))
		singleBuf := el.readBuf[readPtr : readPtr+event.length]
		readPtr += event.length

		sourceName, extraDataPtr, err := utf16ToString(singleBuf[56:])
		if err != nil {
			continue
		}
		computerName, extraDataPtr, err := utf16ToString(singleBuf[56+extraDataPtr:])
		if err != nil {
			continue
		}

		lr := LogRecord{
			EventLogName:  el.name,
			RecordNumber:  event.recordNumber,
			EventId:       event.eventID,
			EventType:     EventType(event.eventType),
			EventCategory: "Unknown", // TODO: Lookup category string.
			TimeGenerated: time.Unix(int64(event.timeGenerated), 0),
			TimeWritten:   time.Unix(int64(event.timeWritten), 0),
			SourceName:    sourceName,
			ComputerName:  computerName,
		}

		if event.userSidLength > 0 {
			sid := (*windows.SID)(unsafe.Pointer(&singleBuf[event.userSidOffset]))
			account, domain, accountType, err := sid.LookupAccount("")
			if err != nil {
				continue
			}

			lr.UserSid = &SID{
				Name:    account,
				Domain:  domain,
				SIDType: SIDType(accountType),
			}
		}

		message, err := el.formatMessage(event, singleBuf, lr)
		if err != nil {
			logp.Warn("Error formatting message.", err)
			continue
		}
		lr.Message = message
		logp.Debug("eventlog", "LogRecord %s", lr)
		records = append(records, lr)
	}

	return records, nil
}

func (el *eventLog) Close() error {
	return closeEventLog(el.handle)
}

// UTF16ToString returns the UTF-8 encoding of the UTF-16 sequence s,
// with a terminating NUL removed.
func utf16ToString(b []byte) (string, int, error) {
	if len(b)%2 != 0 {
		return "", 0, fmt.Errorf("Must have even length byte slice")
	}

	offset := len(b)/2 + 2
	s := make([]uint16, len(b)/2)
	for i, _ := range s {
		s[i] = uint16(b[i*2]) + uint16(b[(i*2)+1])<<8

		if s[i] == 0 {
			s = s[0:i]
			offset = i*2 + 2
			break
		}
	}

	return string(utf16.Decode(s)), offset, nil
}

// formatMessage builds the message text that is associated with an event log
// record. Each EventID has a template that is stored in a library. The event
// contains the parameters used to populate the template. This method evaluates
// the template with the parameters and returns the resulting string.
//
// https://msdn.microsoft.com/en-us/library/windows/desktop/aa363651(v=vs.85).aspx#_win32_description_strings
func (el *eventLog) formatMessage(event *winEventLogRecord, buf []byte, lr LogRecord) (string, error) {
	// Get string values and addresses of the inserts:
	stringInserts, stringInsertPtrs, err := getStrings(event, buf)
	if err != nil {
		logp.Warn("Failed to get string inserts.", err)
		return "", err
	}

	handles := el.handles.get(lr.SourceName)
	if handles == nil || len(handles) == 0 {
		message := fmt.Sprintf(noMessageFile, lr.EventId, lr.SourceName,
			strings.Join(stringInserts, ", "))
		return message, nil
	}

	var addr *uintptr
	if stringInsertPtrs != nil && len(stringInsertPtrs) > 0 {
		addr = &stringInsertPtrs[0]
	}

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
			// Try the next handle to see if a message can be found.
			logp.Debug("eventlog", "Failed to find message. Trying next handle.")
			continue
		}

		message, _, err = utf16ToString(el.formatBuf[:numChars*2])
		if err != nil {
			// Found a handle that provides the message.
			break
		}
	}

	if message == "" {
		message = fmt.Sprintf(noMessageFile, lr.EventId, lr.SourceName,
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
		evtStr, length, err := utf16ToString(buf[offset:])
		if err != nil {
			logp.Warn("Failed to convert from UTF16 to string %v", err)
			return nil, nil, err
		}
		inserts[i] = evtStr
		insertPtrs[i] = bufPtr + uintptr(offset)
		offset += length
	}

	return inserts, insertPtrs, nil
}

// queryEventMessageFiles queries the registry to get the value of
// the EventMessageFile key that points to a DLL or EXE containing templated
// event log messages. If found, it loads the libraries as a datafiles and
// returns a slice of Handles.
func queryEventMessageFiles(eventLogName, sourceName string) ([]Handle, error) {
	// Attempt to find the event message file in the registry and then store
	// a Handle to it in the cache, or store nil if an event message file does
	// not exist for the source name.

	// Open key in registry:
	registryKeyName := fmt.Sprintf(
		"SYSTEM\\CurrentControlSet\\Services\\EventLog\\%s\\%s",
		eventLogName, sourceName)
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, registryKeyName,
		registry.QUERY_VALUE)
	if err != nil {
		logp.Debug("eventlog", "Failed to open HKLM\\%s", registryKeyName)
		return nil, err
	}
	defer key.Close()
	logp.Debug("eventlog", "RegOpenKey opened handle to HKLM\\%s, %v",
		registryKeyName, key)

	// Read value from registry:
	value, _, err := key.GetStringValue("EventMessageFile")
	if err != nil {
		logp.Debug("eventlog", "Failed querying EventMessageFile from HKLM\\%s", registryKeyName)
		return nil, err
	}
	value, err = registry.ExpandString(value)
	if err != nil {
		return nil, err
	}

	// Split the value in case there is more than one file specified.
	eventMessageFiles := strings.Split(value, ";")
	logp.Debug("eventlog", "RegQueryValueEx queried EventMessageFile from "+
		"HKLM\\%s and got %v", registryKeyName, eventMessageFiles)

	var handles []Handle
	for _, eventMessageFile := range eventMessageFiles {
		sPtr, err := syscall.UTF16PtrFromString(eventMessageFile)
		if err != nil {
			logp.Debug("Failed to get UTF16Ptr for '%s' (%v). Skipping",
				eventMessageFile, err)
			continue
		}
		handle, err := loadLibraryEx(sPtr, 0, LOAD_LIBRARY_AS_DATAFILE)
		if err != nil {
			logp.Debug("eventlog", "Failed to load library '%s' as data file:"+
				"%v", eventMessageFile, err)
			continue
		}
		handles = append(handles, handle)
	}

	logp.Debug("eventlog", "Returning handles %v for sourceName %s",
		handles, sourceName)
	return handles, nil
}
