package wineventlog

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"reflect"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"github.com/elastic/beats/winlogbeat/sys/eventlogging"
	"golang.org/x/sys/windows"
)

// Errors
var (
	// ErrorEvtVarTypeNull is an error that means the content of the EVT_VARIANT
	// data is null.
	ErrorEvtVarTypeNull = errors.New("Null EVT_VARIANT data")
)

// bookmarkTemplate is a parameterized string that requires two parameters,
// the channel name and the record ID. The formatted string can be used to open
// a new event log subscription and resume from the given record ID.
const bookmarkTemplate = `<BookmarkList><Bookmark Channel="%s" RecordId="%d" ` +
	`IsCurrent="True"/></BookmarkList>`

// IsAvailable returns true if the Windows Event Log API is supported by this
// operating system. If not supported then false is returned with the
// accompanying error.
func IsAvailable() (bool, error) {
	err := modwevtapi.Load()
	if err != nil {
		return false, err
	}

	return true, nil
}

// Channels returns a list of channels that are registered on the computer.
func Channels() ([]string, error) {
	handle, err := _EvtOpenChannelEnum(0, 0)
	if err != nil {
		return nil, err
	}
	defer _EvtClose(handle)

	var channels []string
	cpBuffer := make([]uint16, 512)
loop:
	for {
		var used uint32
		err := _EvtNextChannelPath(handle, uint32(len(cpBuffer)), &cpBuffer[0], &used)
		if err != nil {
			errno, ok := err.(syscall.Errno)
			if ok {
				switch errno {
				case ERROR_INSUFFICIENT_BUFFER:
					// Grow buffer.
					newLen := 2 * len(cpBuffer)
					if int(used) > newLen {
						newLen = int(used)
					}
					cpBuffer = make([]uint16, newLen)
					continue
				case ERROR_NO_MORE_ITEMS:
					break loop
				}
			}
			return nil, err
		}
		channels = append(channels, syscall.UTF16ToString(cpBuffer[:used]))
	}

	return channels, nil
}

// Subscribe creates a new subscription to an event log channel.
func Subscribe(
	session EvtHandle,
	event windows.Handle,
	channelPath string,
	query string,
	bookmark EvtHandle,
	flags EvtSubscribeFlag,
) (EvtHandle, error) {
	var err error
	var cp *uint16
	if channelPath != "" {
		cp, err = syscall.UTF16PtrFromString(channelPath)
		if err != nil {
			return 0, err
		}
	}

	var q *uint16
	if query != "" {
		q, err = syscall.UTF16PtrFromString(query)
		if err != nil {
			return 0, err
		}
	}

	eventHandle, err := _EvtSubscribe(session, uintptr(event), cp, q, bookmark,
		0, 0, flags)
	if err != nil {
		return 0, err
	}

	return eventHandle, nil
}

// EventHandles reads the event handles from a subscription. It attempt to read
// at most maxHandles. ErrorNoMoreHandles is returned when there are no more
// handles available to return. Close must be called on each returned EvtHandle
// when finished with the handle.
func EventHandles(subscription EvtHandle, maxHandles int) ([]EvtHandle, error) {
	eventHandles := make([]EvtHandle, maxHandles)
	var numRead uint32

	err := _EvtNext(subscription, uint32(len(eventHandles)),
		&eventHandles[0], 0, 0, &numRead)
	if err != nil {
		// Munge ERROR_INVALID_OPERATION to ERROR_NO_MORE_ITEMS when no handles
		// were read. This happens you call the method and there are no events
		// to read (i.e. polling).
		if err == ERROR_INVALID_OPERATION && numRead == 0 {
			return nil, ERROR_NO_MORE_ITEMS
		}
		return nil, err
	}

	return eventHandles[:numRead], nil
}

// RenderEvent reads the event data associated with the EvtHandle and renders
// the data so that it can used.
func RenderEvent(
	eventHandle EvtHandle,
	systemContext EvtHandle,
	lang uint32,
	renderBuf []byte,
	pubHandleProvider func(string) eventlogging.MessageFiles,
) (Event, error) {
	var err error

	// Create a render context for local machine.
	if systemContext == 0 {
		systemContext, err = _EvtCreateRenderContext(0, nil, EvtRenderContextSystem)
		if err != nil {
			return Event{}, err
		}
		defer _EvtClose(systemContext)
	}

	var bufferUsed, propertyCount uint32
	err = _EvtRender(systemContext, eventHandle, EvtRenderEventValues,
		uint32(len(renderBuf)), &renderBuf[0], &bufferUsed,
		&propertyCount)
	if err == ERROR_INSUFFICIENT_BUFFER {
		return Event{}, eventlogging.InsufficientBufferError{err, int(bufferUsed)}
	}
	if err != nil {
		return Event{}, err
	}

	// Validate bufferUsed set by Windows.
	if int(bufferUsed) > len(renderBuf) {
		return Event{}, fmt.Errorf("Bytes used (%d) is greater than the "+
			"buffer size (%d)", bufferUsed, len(renderBuf))
	}

	// Ignore any additional unknown properties that might exist.
	if propertyCount > uint32(EvtSystemPropertyIdEND) {
		propertyCount = uint32(EvtSystemPropertyIdEND)
	}

	var e Event
	err = parseRenderEventBuffer(renderBuf[:bufferUsed], &e)
	if err != nil {
		return Event{}, err
	}

	var publisherHandle uintptr
	if pubHandleProvider != nil {
		messageFiles := pubHandleProvider(e.ProviderName)
		if messageFiles.Err == nil {
			// There is only ever a single handle when using the Windows Event
			// Log API.
			publisherHandle = messageFiles.Handles[0].Handle
		}
	}

	// Populate strings that must be looked up.
	populateStrings(eventHandle, EvtHandle(publisherHandle), lang, renderBuf, &e)

	return e, nil
}

// parseRenderEventBuffer parses the system context data from buffer. This
// function can be used on the data written by the EvtRender system call.
func parseRenderEventBuffer(buffer []byte, evt *Event) error {
	reader := bytes.NewReader(buffer)

	for i := 0; i < int(EvtSystemPropertyIdEND); i++ {
		// Each EVT_VARIANT is 16 bytes.
		_, err := reader.Seek(int64(16*i), 0)
		if err != nil {
			return fmt.Errorf("Error seeking to read %s: %v",
				EvtSystemPropertyID(i), err)
		}

		switch EvtSystemPropertyID(i) {
		case EvtSystemKeywords, EvtSystemLevel, EvtSystemOpcode, EvtSystemTask:
			// These are rendered as strings so ignore them here.
			continue
		case EvtSystemProviderName:
			evt.ProviderName, err = readString(buffer, reader)
		case EvtSystemComputer:
			evt.Computer, err = readString(buffer, reader)
		case EvtSystemChannel:
			evt.Channel, err = readString(buffer, reader)
		case EvtSystemVersion:
			err = binary.Read(reader, binary.LittleEndian, &evt.Version)
		case EvtSystemEventID:
			err = binary.Read(reader, binary.LittleEndian, &evt.EventID)
		case EvtSystemQualifiers:
			err = binary.Read(reader, binary.LittleEndian, &evt.Qualifiers)
		case EvtSystemThreadID:
			err = binary.Read(reader, binary.LittleEndian, &evt.ThreadID)
		case EvtSystemProcessID:
			err = binary.Read(reader, binary.LittleEndian, &evt.ProcessID)
		case EvtSystemEventRecordId:
			err = binary.Read(reader, binary.LittleEndian, &evt.RecordID)
		case EvtSystemTimeCreated:
			evt.TimeCreated, err = readFiletime(reader)
		case EvtSystemActivityID:
			evt.ActivityID, err = readGUID(buffer, reader)
		case EvtSystemRelatedActivityID:
			evt.RelatedActivityID, err = readGUID(buffer, reader)
		case EvtSystemProviderGuid:
			evt.ProviderGUID, err = readGUID(buffer, reader)
		case EvtSystemUserID:
			evt.UserSID, err = readSID(buffer, reader)
		}

		if err != nil {
			return fmt.Errorf("Error reading %s: %v", EvtSystemPropertyID(i), err)
		}
	}

	return nil
}

// populateStrings populates the string fields of the Event that require
// formatting (Message, Level, Task, Opcode, and Keywords). It attempts to
// populate each field even if an error occurs. Any errors that occur are
// written to the Event (see MessageErr, LevelErr, TaskErr, OpcodeErr, and
// KeywordsErr).
func populateStrings(
	eventHandle EvtHandle,
	providerHandle EvtHandle,
	lang uint32,
	buffer []byte,
	event *Event,
) {
	var strs []string
	strs, event.MessageErr = FormatEventString(EvtFormatMessageEvent,
		eventHandle, event.ProviderName, providerHandle, lang, buffer)
	if len(strs) > 0 {
		event.Message = strs[0]
	}
	// TODO: Populate the MessageInserts when there is a MessageErr.

	strs, event.LevelErr = FormatEventString(EvtFormatMessageLevel,
		eventHandle, event.ProviderName, providerHandle, lang, buffer)
	if len(strs) > 0 {
		event.Level = strs[0]
	}

	strs, event.TaskErr = FormatEventString(EvtFormatMessageTask,
		eventHandle, event.ProviderName, providerHandle, lang, buffer)
	if len(strs) > 0 {
		event.Task = strs[0]
	}

	strs, event.OpcodeErr = FormatEventString(EvtFormatMessageOpcode,
		eventHandle, event.ProviderName, providerHandle, lang, buffer)
	if len(strs) > 0 {
		event.Opcode = strs[0]
	}

	event.Keywords, event.KeywordsError = FormatEventString(
		EvtFormatMessageKeyword, eventHandle, event.ProviderName,
		providerHandle, lang, buffer)
}

// CreateBookmark creates a new handle to a bookmark. Close must be called on
// returned EvtHandle when finished with the handle.
func CreateBookmark(channel string, recordID uint64) (EvtHandle, error) {
	xml := fmt.Sprintf(bookmarkTemplate, channel, recordID)
	p, err := syscall.UTF16PtrFromString(xml)
	if err != nil {
		return 0, err
	}

	h, err := _EvtCreateBookmark(p)
	if err != nil {
		return 0, err
	}

	return h, nil
}

// Create a render context. Close must be called on returned EvtHandle when
// finished with the handle.
func CreateRenderContext(valuePaths []string, flag EvtRenderContextFlag) (EvtHandle, error) {
	context, err := _EvtCreateRenderContext(0, nil, EvtRenderContextSystem)
	if err != nil {
		return 0, err
	}

	return context, nil
}

// OpenPublisherMetadata opens a handle to the publisher's metadata. Close must
// be called on returned EvtHandle when finished with the handle.
func OpenPublisherMetadata(
	session EvtHandle,
	publisherName string,
	lang uint32,
) (EvtHandle, error) {
	p, err := syscall.UTF16PtrFromString(publisherName)
	if err != nil {
		return 0, err
	}

	h, err := _EvtOpenPublisherMetadata(session, p, nil, lang, 0)
	if err != nil {
		return 0, err
	}

	return h, nil
}

// Close closes an EvtHandle.
func Close(h EvtHandle) error {
	return _EvtClose(h)
}

// FormatEventString formats part of the event as a string.
// messageFlag determines what part of the event is formatted as as string.
// eventHandle is the handle to the event.
// publisher is the name of the event's publisher.
// publisherHandle is a handle to the publisher's metadata as provided by
// EvtOpenPublisherMetadata.
// lang is the language ID.
// buffer is optional and if not provided it will be allocated. If the provided
// buffer is not large enough then an InsufficientBufferError will be returned.
func FormatEventString(
	messageFlag EvtFormatMessageFlag,
	eventHandle EvtHandle,
	publisher string,
	publisherHandle EvtHandle,
	lang uint32,
	buffer []byte,
) ([]string, error) {
	p, err := syscall.UTF16PtrFromString(publisher)
	if err != nil {
		return nil, err
	}

	// Open a publisher handle if one was not provided.
	ph := publisherHandle
	if ph == 0 {
		ph, err = _EvtOpenPublisherMetadata(0, p, nil, lang, 0)
		if err != nil {
			return nil, err
		}
		defer _EvtClose(ph)
	}

	// Create a buffer if one was not provider.
	var bufferUsed uint32
	if buffer == nil {
		err = _EvtFormatMessage(ph, eventHandle, 0, 0, 0, messageFlag,
			0, nil, &bufferUsed)
		bufferUsed *= 2 // It returns the number of utf-16 chars.
		if err != nil && err != ERROR_INSUFFICIENT_BUFFER {
			return nil, err
		}

		buffer = make([]byte, bufferUsed)
		bufferUsed = 0
	}

	err = _EvtFormatMessage(ph, eventHandle, 0, 0, 0, messageFlag,
		uint32(len(buffer)/2), &buffer[0], &bufferUsed)
	bufferUsed *= 2 // It returns the number of utf-16 chars.
	if err == ERROR_INSUFFICIENT_BUFFER {
		return nil, eventlogging.InsufficientBufferError{err, int(bufferUsed)}
	}
	if err != nil {
		return nil, err
	}

	var value string
	var offset int
	var size int
	var values []string
	for {
		value, size, err = eventlogging.UTF16BytesToString(buffer[offset:bufferUsed])
		if err != nil {
			return nil, err
		}
		offset += size
		values = append(values, eventlogging.RemoveWindowsLineEndings(value))

		if offset >= int(bufferUsed) {
			break
		}
	}

	return values, nil
}

// offset reads a pointer value from the reader then calculates an offset from
// the start of the buffer to the pointer location. If the pointer value is
// NULL or is outside of the bounds of the buffer then an error is returned.
// reader will be advanced by the size of a uintptr.
func offset(buffer []byte, reader io.Reader) (uint64, error) {
	// Handle 32 and 64-bit pointer size differences.
	var dataPtr uint64
	var err error
	switch runtime.GOARCH {
	default:
		return 0, fmt.Errorf("Unhandled architecture: %s", runtime.GOARCH)
	case "amd64":
		err = binary.Read(reader, binary.LittleEndian, &dataPtr)
		if err != nil {
			return 0, err
		}
	case "386":
		var p uint32
		err = binary.Read(reader, binary.LittleEndian, &p)
		if err != nil {
			return 0, err
		}
		dataPtr = uint64(p)
	}

	if dataPtr == 0 {
		return 0, ErrorEvtVarTypeNull
	}

	bufferPtr := uint64(reflect.ValueOf(&buffer[0]).Pointer())
	offset := dataPtr - bufferPtr

	if offset < 0 || offset > uint64(len(buffer)) {
		return 0, fmt.Errorf("Invalid pointer %x. Cannot dereference an "+
			"address outside of the buffer [%x:%x].", dataPtr, bufferPtr,
			bufferPtr+uint64(len(buffer)))
	}

	return offset, nil
}

// readString reads a pointer using the reader then parses the UTF-16 string
// that the pointer addresses within the buffer.
func readString(buffer []byte, reader io.Reader) (string, error) {
	offset, err := offset(buffer, reader)
	if err != nil {
		// Ignore NULL values.
		if err == ErrorEvtVarTypeNull {
			return "", nil
		}
		return "", err
	}
	str, _, err := eventlogging.UTF16BytesToString(buffer[offset:])
	return str, err
}

// readFiletime reads a Windows Filetime struct and converts it to a
// time.Time value with a UTC timezone.
func readFiletime(reader io.Reader) (*time.Time, error) {
	var filetime syscall.Filetime
	err := binary.Read(reader, binary.LittleEndian, &filetime.LowDateTime)
	if err != nil {
		return nil, err
	}
	err = binary.Read(reader, binary.LittleEndian, &filetime.HighDateTime)
	if err != nil {
		return nil, err
	}
	t := time.Unix(0, filetime.Nanoseconds()).UTC()
	return &t, nil
}

// readSID reads a pointer using the reader then parses the Windows SID
// data that the pointer addresses within the buffer.
func readSID(buffer []byte, reader io.Reader) (*eventlogging.SID, error) {
	offset, err := offset(buffer, reader)
	if err != nil {
		// Ignore NULL values.
		if err == ErrorEvtVarTypeNull {
			return nil, nil
		}
		return nil, err
	}
	sid := (*windows.SID)(unsafe.Pointer(&buffer[offset]))
	identifier, err := sid.String()
	if err != nil {
		return nil, err
	}

	account, domain, accountType, err := sid.LookupAccount("")
	if err != nil {
		// Ignore the error and return a partially populated SID.
		return &eventlogging.SID{Identifier: identifier}, nil
	}

	return &eventlogging.SID{
		Identifier: identifier,
		Name:       account,
		Domain:     domain,
		Type:       eventlogging.SIDType(accountType),
	}, nil
}

// readGUID reads a pointer using the reader then parses the Windows GUID
// data that the pointer addresses within the buffer.
func readGUID(buffer []byte, reader io.ReadSeeker) (string, error) {
	offset, err := offset(buffer, reader)
	if err != nil {
		// Ignore NULL values.
		if err == ErrorEvtVarTypeNull {
			return "", nil
		}
		return "", err
	}

	guid := &syscall.GUID{}
	_, err = reader.Seek(int64(offset), 0)
	if err != nil {
		return "", err
	}
	err = binary.Read(reader, binary.LittleEndian, &guid.Data1)
	if err != nil {
		return "", err
	}
	err = binary.Read(reader, binary.LittleEndian, &guid.Data2)
	if err != nil {
		return "", err
	}
	err = binary.Read(reader, binary.LittleEndian, &guid.Data3)
	if err != nil {
		return "", err
	}
	err = binary.Read(reader, binary.LittleEndian, &guid.Data4)
	if err != nil {
		return "", err
	}

	guidStr, err := StringFromGUID(guid)
	if err != nil {
		return "", err
	}

	return guidStr, nil
}

// StringFromGUID returns a displayable GUID string from the GUID struct.
func StringFromGUID(guid *syscall.GUID) (string, error) {
	if guid == nil {
		return "", nil
	}

	buf := make([]uint16, 40)
	err := _StringFromGUID2(guid, &buf[0], uint32(len(buf)))
	if err != nil {
		return "", err
	}

	return syscall.UTF16ToString(buf), nil
}
