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

	"golang.org/x/sys/windows"

	"github.com/elastic/beats/winlogbeat/sys"
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

var providerNameContext EvtHandle

func init() {
	if avail, _ := IsAvailable(); avail {
		providerNameContext, _ = CreateRenderContext([]string{"Event/System/Provider/@Name"}, EvtRenderContextValues)
	}
}

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
	if maxHandles < 1 {
		return nil, fmt.Errorf("maxHandles must be greater than 0")
	}

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
// the data as XML. An error and XML can be returned by this method if an error
// occurs while rendering the XML with RenderingInfo and the method is able to
// recover by rendering the XML without RenderingInfo.
func RenderEvent(
	eventHandle EvtHandle,
	lang uint32,
	renderBuf []byte,
	pubHandleProvider func(string) sys.MessageFiles,
	out io.Writer,
) error {
	providerName, err := evtRenderProviderName(renderBuf, eventHandle)
	if err != nil {
		return err
	}

	var publisherHandle uintptr
	if pubHandleProvider != nil {
		messageFiles := pubHandleProvider(providerName)
		if messageFiles.Err == nil {
			// There is only ever a single handle when using the Windows Event
			// Log API.
			publisherHandle = messageFiles.Handles[0].Handle
		}
	}

	// Only a single string is returned when rendering XML.
	err = FormatEventString(EvtFormatMessageXml,
		eventHandle, providerName, EvtHandle(publisherHandle), lang, renderBuf, out)

	// Recover by rendering the XML without the RenderingInfo (message string).
	if err != nil {
		// Do not try to recover from InsufficientBufferErrors because these
		// can be retried with a larger buffer.
		if _, ok := err.(sys.InsufficientBufferError); ok {
			return err
		}

		err = RenderEventXML(eventHandle, renderBuf, out)
	}

	return err
}

// RenderEventXML renders the event as XML. If the event is already rendered, as
// in a forwarded event whose content type is "RenderedText", then the XML will
// include the RenderingInfo (message). If the event is not rendered then the
// XML will not include the message, and in this case RenderEvent should be
// used.
func RenderEventXML(eventHandle EvtHandle, renderBuf []byte, out io.Writer) error {
	return renderXML(eventHandle, EvtRenderEventXml, renderBuf, out)
}

// RenderBookmarkXML renders a bookmark as XML.
func RenderBookmarkXML(bookmarkHandle EvtHandle, renderBuf []byte, out io.Writer) error {
	return renderXML(bookmarkHandle, EvtRenderBookmark, renderBuf, out)
}

// CreateBookmarkFromRecordID creates a new bookmark pointing to the given recordID
// within the supplied channel. Close must be called on returned EvtHandle when
// finished with the handle.
func CreateBookmarkFromRecordID(channel string, recordID uint64) (EvtHandle, error) {
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

// CreateBookmarkFromEvent creates a new bookmark pointing to the given event.
// Close must be called on returned EvtHandle when finished with the handle.
func CreateBookmarkFromEvent(handle EvtHandle) (EvtHandle, error) {
	h, err := _EvtCreateBookmark(nil)
	if err != nil {
		return 0, err
	}
	if err = _EvtUpdateBookmark(h, handle); err != nil {
		return 0, err
	}
	return h, nil
}

// CreateBookmarkFromXML creates a new bookmark from the serialised representation
// of an existing bookmark. Close must be called on returned EvtHandle when
// finished with the handle.
func CreateBookmarkFromXML(bookmarkXML string) (EvtHandle, error) {
	xml, err := syscall.UTF16PtrFromString(bookmarkXML)
	if err != nil {
		return 0, err
	}
	return _EvtCreateBookmark(xml)
}

// CreateRenderContext creates a render context. Close must be called on
// returned EvtHandle when finished with the handle.
func CreateRenderContext(valuePaths []string, flag EvtRenderContextFlag) (EvtHandle, error) {
	var paths []uintptr
	for _, path := range valuePaths {
		utf16, err := syscall.UTF16FromString(path)
		if err != nil {
			return 0, err
		}

		paths = append(paths, reflect.ValueOf(&utf16[0]).Pointer())
	}

	var pathsAddr uintptr
	if len(paths) > 0 {
		pathsAddr = reflect.ValueOf(&paths[0]).Pointer()
	}

	context, err := _EvtCreateRenderContext(uint32(len(paths)), pathsAddr, flag)
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
	out io.Writer,
) error {
	// Open a publisher handle if one was not provided.
	ph := publisherHandle
	if ph == 0 {
		ph, err := OpenPublisherMetadata(0, publisher, 0)
		if err != nil {
			return err
		}
		defer _EvtClose(ph)
	}

	// Create a buffer if one was not provided.
	var bufferUsed uint32
	if buffer == nil {
		err := _EvtFormatMessage(ph, eventHandle, 0, 0, 0, messageFlag,
			0, nil, &bufferUsed)
		if err != nil && err != ERROR_INSUFFICIENT_BUFFER {
			return err
		}

		bufferUsed *= 2
		buffer = make([]byte, bufferUsed)
		bufferUsed = 0
	}

	err := _EvtFormatMessage(ph, eventHandle, 0, 0, 0, messageFlag,
		uint32(len(buffer)/2), &buffer[0], &bufferUsed)
	bufferUsed *= 2
	if err == ERROR_INSUFFICIENT_BUFFER {
		return sys.InsufficientBufferError{err, int(bufferUsed)}
	}
	if err != nil {
		return err
	}

	// This assumes there is only a single string value to read. This will
	// not work to read keys (when messageFlag == EvtFormatMessageKeyword).
	return sys.UTF16ToUTF8Bytes(buffer[:bufferUsed], out)
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
	str, _, err := sys.UTF16BytesToString(buffer[offset:])
	return str, err
}

// evtRenderProviderName renders the ProviderName of an event.
func evtRenderProviderName(renderBuf []byte, eventHandle EvtHandle) (string, error) {
	var bufferUsed, propertyCount uint32
	err := _EvtRender(providerNameContext, eventHandle, EvtRenderEventValues,
		uint32(len(renderBuf)), &renderBuf[0], &bufferUsed, &propertyCount)
	if err == ERROR_INSUFFICIENT_BUFFER {
		return "", sys.InsufficientBufferError{err, int(bufferUsed)}
	}
	if err != nil {
		return "", fmt.Errorf("evtRenderProviderName %v", err)
	}

	reader := bytes.NewReader(renderBuf)
	return readString(renderBuf, reader)
}

func renderXML(eventHandle EvtHandle, flag EvtRenderFlag, renderBuf []byte, out io.Writer) error {
	var bufferUsed, propertyCount uint32
	err := _EvtRender(0, eventHandle, flag, uint32(len(renderBuf)),
		&renderBuf[0], &bufferUsed, &propertyCount)
	if err == ERROR_INSUFFICIENT_BUFFER {
		return sys.InsufficientBufferError{err, int(bufferUsed)}
	}
	if err != nil {
		return err
	}

	if int(bufferUsed) > len(renderBuf) {
		return fmt.Errorf("Windows EvtRender reported that wrote %d bytes "+
			"to the buffer, but the buffer can only hold %d bytes",
			bufferUsed, len(renderBuf))
	}
	return sys.UTF16ToUTF8Bytes(renderBuf[:bufferUsed], out)
}
