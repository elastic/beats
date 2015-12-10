package wineventlog

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"reflect"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/elastic/beats/winlogbeat/eventlog"
	"golang.org/x/sys/windows"
)

// Errors
var (
	ErrorEvtVarTypeNULL = errors.New("NULL event variant data")
)

const bookmarkTemplate = `<BookmarkList><Bookmark Channel="%s" RecordId="%d" ` +
	`IsCurrent="True"/></BookmarkList>`

// IsAvailable returns nil if the Windows Event Log API is supported by this
// operating system. If not supported then an error is returned.
func IsAvailable() (bool, error) {
	err := modwevtapi.Load()
	if err != nil {
		return false, err
	}

	return true, nil
}

// Channels returns a list of channels that are registered on the computer.
func Channels() ([]string, error) {
	handle, err := _EvtOpenChannelEnum(NullEvtHandle, 0)
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
			return NullEvtHandle, err
		}
	}

	var q *uint16
	if query != "" {
		q, err = syscall.UTF16PtrFromString(query)
		if err != nil {
			return NullEvtHandle, err
		}
	}

	eventHandle, err := _EvtSubscribe(session, uintptr(event), cp, q, bookmark,
		uintptr(0), NullHandle, flags)
	if err != nil {
		return NullEvtHandle, err
	}

	return eventHandle, nil
}

// EventHandles reads the event handles from a subscription. It attempt to read
// at most maxHandles. ErrorNoMoreHandles is returned when there are no more
// handles available to return.
func EventHandles(subscription EvtHandle, maxHandles int) ([]EvtHandle, error) {
	eventHandles := make([]EvtHandle, maxHandles)
	var numRead uint32

	err := _EvtNext(subscription, uint32(len(eventHandles)),
		&eventHandles[0], 0, 0, &numRead)
	if err != nil {
		if err == ERROR_INVALID_OPERATION && numRead == 0 {
			return nil, ERROR_NO_MORE_ITEMS
		}
		return nil, err
	}

	return eventHandles[:numRead], nil
}

func RenderEvent(
	h EvtHandle,
	systemContext EvtHandle,
	lang uint32,
	renderBuf []byte,
	pubHandleProvider func(string) EvtHandle,
) (Event, int, error) {
	var err error

	// Create a render context for local machine.
	if systemContext == NullEvtHandle {
		systemContext, err = _EvtCreateRenderContext(0, nil, EvtRenderContextSystem)
		if err != nil {
			return Event{}, 0, err
		}
	}

	var bufferUsed, propertyCount uint32
	err = _EvtRender(systemContext, h, EvtRenderEventValues,
		uint32(len(renderBuf)), &renderBuf[0], &bufferUsed,
		&propertyCount)
	if err != nil {
		if isInsufficientBuffer(err) {
			return Event{}, int(bufferUsed), err
		}

		return Event{}, 0, err
	}

	// Validate bufferUsed set by Windows.
	if int(bufferUsed) > len(renderBuf) {
		return Event{}, 0, fmt.Errorf("Bytes used (%d) is greater than the buffer "+
			"size (%d)", bufferUsed, len(renderBuf))
	}

	// Ignore any additional unknown properties that might exist.
	if propertyCount > uint32(EvtSystemPropertyIdEND) {
		propertyCount = uint32(EvtSystemPropertyIdEND)
	}

	var e Event
	err = parseRenderEventBuffer(renderBuf[:bufferUsed], &e)
	if err != nil {
		return Event{}, 0, err
	}

	publisherHandle := NullEvtHandle
	if pubHandleProvider != nil {
		publisherHandle = pubHandleProvider(e.ProviderName)
	}

	// Populate strings that must be looked up.
	var requiredSize int
	requiredSize, err = populateStrings(h, publisherHandle, lang, renderBuf, &e)
	if err != nil {
		if isInsufficientBuffer(err) {
			return Event{}, requiredSize, err
		}

		return Event{}, 0, err
	}

	return e, 0, nil
}

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
			evt.TimeCreated, err = readFiletime(buffer, reader)
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

func populateStrings(
	eventHandle EvtHandle,
	providerHandle EvtHandle,
	lang uint32,
	buffer []byte,
	event *Event,
) (int, error) {
	strs, size, err := FormatEventString(EvtFormatMessageEvent,
		eventHandle, event.ProviderName, providerHandle, lang, buffer)
	if len(strs) > 0 {
		event.Message = strs[0]
	}
	if err != nil && !isRecoverable(err) {
		return size, err
	}

	strs, size, err = FormatEventString(EvtFormatMessageLevel,
		eventHandle, event.ProviderName, providerHandle, lang, buffer)
	if len(strs) > 0 {
		event.Level = strs[0]
	}
	if err != nil && !isRecoverable(err) {
		return size, err
	}

	strs, size, err = FormatEventString(EvtFormatMessageTask,
		eventHandle, event.ProviderName, providerHandle, lang, buffer)
	if len(strs) > 0 {
		event.Task = strs[0]
	}
	if err != nil && !isRecoverable(err) {
		return size, err
	}

	strs, size, err = FormatEventString(EvtFormatMessageOpcode,
		eventHandle, event.ProviderName, providerHandle, lang, buffer)
	if len(strs) > 0 {
		event.Opcode = strs[0]
	}
	if err != nil && !isRecoverable(err) {
		return size, err
	}

	event.Keywords, size, err = FormatEventString(EvtFormatMessageKeyword,
		eventHandle, event.ProviderName, providerHandle, lang, buffer)
	if err != nil && !isRecoverable(err) {
		return size, err
	}

	return 0, nil
}

func CreateBookmark(channel string, recordID uint64) (EvtHandle, error) {
	xml := fmt.Sprintf(bookmarkTemplate, channel, recordID)
	p, err := syscall.UTF16PtrFromString(xml)
	if err != nil {
		return NullEvtHandle, err
	}

	h, err := _EvtCreateBookmark(p)
	if err != nil {
		return NullEvtHandle, err
	}

	return h, nil
}

// OpenPublisherMetadata opens a handle to the publisher's metadata. The handle
// needs to be closed when finished.
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
// buffer is optional and if not provided it will be allocated.
func FormatEventString(
	messageFlag EvtFormatMessageFlag,
	eventHandle EvtHandle,
	publisher string,
	publisherHandle EvtHandle,
	lang uint32,
	buffer []byte,
) ([]string, int, error) {
	p, err := syscall.UTF16PtrFromString(publisher)
	if err != nil {
		return nil, 0, err
	}

	// Open a publisher handle if one was not provided.
	ph := publisherHandle
	if ph == NullEvtHandle {
		ph, err = _EvtOpenPublisherMetadata(NullEvtHandle, p, nil, lang, 0)
		if err != nil {
			return nil, 0, err
		}
		defer _EvtClose(ph)
	}

	// Create a buffer if one was not provider.
	var bufferUsed uint32
	if buffer == nil {
		err = _EvtFormatMessage(ph, eventHandle, 0, 0, 0, messageFlag,
			0, nil, &bufferUsed)
		bufferUsed *= 2 // It returns the number of utf-16 chars.
		if err != nil && !isInsufficientBuffer(err) {
			return nil, 0, err
		}

		buffer = make([]byte, bufferUsed)
		bufferUsed = 0
	}

	err = _EvtFormatMessage(ph, eventHandle, 0, 0, 0, messageFlag,
		uint32(len(buffer)/2), &buffer[0], &bufferUsed)
	bufferUsed *= 2 // It returns the number of utf-16 chars.
	if err != nil {
		if isInsufficientBuffer(err) {
			return nil, int(bufferUsed), err
		}
		return nil, 0, err
	}

	var value string
	var offset int
	var size int
	var values []string
	for {
		value, size, err = eventlog.UTF16BytesToString(buffer[offset:bufferUsed])
		if err != nil {
			return nil, 0, err
		}
		offset += size
		values = append(values, removeWindowsLineEndings(value))

		if offset >= int(bufferUsed) {
			break
		}
	}

	return values, 0, nil
}

// isInsufficientBuffer returns true iff the error is ERROR_INSUFFICIENT_BUFFER.
func isInsufficientBuffer(err error) bool {
	if err == nil {
		return false
	}
	errno, ok := err.(syscall.Errno)
	return ok && errno == syscall.ERROR_INSUFFICIENT_BUFFER
}

func isRecoverable(err error) bool {
	if err == nil {
		return true
	}
	errno, ok := err.(syscall.Errno)
	if !ok {
		return false
	}

	switch errno {
	case ERROR_EVT_MESSAGE_NOT_FOUND, ERROR_EVT_MESSAGE_ID_NOT_FOUND:
		//fmt.Printf("Error Errno=%d MSG NOT FOUND\n", errno)
		return true
	case syscall.ERROR_FILE_NOT_FOUND:
		//fmt.Printf("Error Errno=%d FILE NOT FOUND\n", errno)
		return true
	case ERROR_EVT_UNRESOLVED_VALUE_INSERT,
		ERROR_EVT_UNRESOLVED_PARAMETER_INSERT:
		//fmt.Printf("Error Errno=%d UNRESOLVED\n", errno)
		return false
	}

	return false
}

// removeWindowsLineEndings replaces CRLF with LF and trims any newline
// character that may exist at the end of the string.
func removeWindowsLineEndings(s string) string {
	s = strings.Replace(s, "\r\n", "\n", -1)
	return strings.TrimRight(s, "\n")
}

// offset reads a pointer value from the reader then calculates an offset from
// the start of the buffer to the pointer location. If the pointer value is
// NULL or is outside of the bounds of the buffer then an error is returned.
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
		return 0, ErrorEvtVarTypeNULL
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

func readString(buffer []byte, reader io.Reader) (string, error) {
	offset, err := offset(buffer, reader)
	if err != nil {
		// Ignore NULL values.
		if err == ErrorEvtVarTypeNULL {
			return "", nil
		}
		return "", err
	}
	str, _, err := eventlog.UTF16BytesToString(buffer[offset:])
	return str, err
}

func readFiletime(buffer []byte, reader io.Reader) (*time.Time, error) {
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

func readSID(buffer []byte, reader io.Reader) (*eventlog.SID, error) {
	offset, err := offset(buffer, reader)
	if err != nil {
		// Ignore NULL values.
		if err == ErrorEvtVarTypeNULL {
			return nil, nil
		}
		return nil, err
	}
	sid := (*windows.SID)(unsafe.Pointer(&buffer[offset]))
	account, domain, accountType, err := sid.LookupAccount("")
	if err != nil {
		return nil, err
	}

	return &eventlog.SID{
		Name:    account,
		Domain:  domain,
		SIDType: eventlog.SIDType(accountType),
	}, nil
}

func readGUID(buffer []byte, reader io.ReadSeeker) (string, error) {
	offset, err := offset(buffer, reader)
	if err != nil {
		// Ignore NULL values.
		if err == ErrorEvtVarTypeNULL {
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
