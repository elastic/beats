//go:build windows
// +build windows

package etwtest

import (
	"encoding/binary"
	"fmt"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	// Replace with the provider GUID from your manifest
	providerGUID = windows.GUID{
		Data1: 0x1db28f2e,
		Data2: 0x8f80,
		Data3: 0x4027,
		Data4: [8]byte{0x8c, 0x5a, 0xa1, 0x1f, 0x7f, 0x10, 0xf6, 0x2d},
	}
)

func utf16PtrFromString(s string) *uint16 {
	ptr, _ := windows.UTF16PtrFromString(s)
	return ptr
}

func TestSendETWEvents(t *testing.T) {
	var regHandle windows.Handle

	// Register the provider
	status := windows.EventRegister(&providerGUID, 0, 0, &regHandle)
	if status != 0 {
		t.Fatalf("EventRegister failed: %x", status)
	}
	defer windows.EventUnregister(regHandle)

	// Send event ID 1 (TRANSFER_SCHEDULE_EVENT)
	t.Run("SendEvent1", func(t *testing.T) {
		// Template t2: UnicodeString, UInt32, UInt32
		eventData := []windows.EventData{
			{DataPointer: uintptr(unsafe.Pointer(utf16PtrFromString("NightlyBackup"))), Size: 2 * uint16(len("NightlyBackup")+1)},
			{DataPointer: uintptr(unsafe.Pointer(&[]uint32{0x2}[0])), Size: 4}, // Monday
			{DataPointer: uintptr(unsafe.Pointer(&[]uint32{0x1}[0])), Size: 4}, // Download
		}

		eventDescriptor := windows.EventDescriptor{
			Id:      1,
			Version: 0,
			Level:   4, // Informational
			Opcode:  0,
			Task:    2, // Connect
			Keyword: 0x1 | 0x8, // Remote + Read
		}

		status := windows.EventWrite(regHandle, &eventDescriptor, uint32(len(eventData)), &eventData[0])
		if status != 0 {
			t.Errorf("EventWrite Event 1 failed: 0x%x", status)
		}
	})

	// Send event ID 2 (DOWNLOAD_XFER_FAILED_EVENT)
	t.Run("SendEvent2", func(t *testing.T) {
		files := "file1.txt\000file2.txt"
		filesCount := uint16(2)
		buffer := []byte{0xde, 0xad, 0xbe, 0xef}
		cert := [11]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}
		isLocal := byte(0) // false
		errorCode := int32(0x80070005) // E_ACCESSDENIED

		eventData := []windows.EventData{
			{DataPointer: uintptr(unsafe.Pointer(utf16PtrFromString("MyDownloadJob"))), Size: 2 * uint16(len("MyDownloadJob")+1)},
			{DataPointer: uintptr(unsafe.Pointer(&errorCode)), Size: 4},
			{DataPointer: uintptr(unsafe.Pointer(&filesCount)), Size: 2},
			{DataPointer: uintptr(unsafe.Pointer(utf16PtrFromString(files))), Size: 2 * uint16(len(files)+1)},
			{DataPointer: uintptr(unsafe.Pointer(&[]uint32{uint32(len(buffer))}[0])), Size: 4},
			{DataPointer: uintptr(unsafe.Pointer(&buffer[0])), Size: uint16(len(buffer))},
			{DataPointer: uintptr(unsafe.Pointer(&cert[0])), Size: 11},
			{DataPointer: uintptr(unsafe.Pointer(&isLocal)), Size: 1},
			{DataPointer: uintptr(unsafe.Pointer(utf16PtrFromString("C:\\Temp"))), Size: 2 * uint16(len("C:\\Temp")+1)},
			{DataPointer: uintptr(unsafe.Pointer(&[]uint16{1}[0])), Size: 2}, // ValuesCount
			// Values struct: Value + Name
			{DataPointer: uintptr(unsafe.Pointer(&[]uint16{42}[0])), Size: 2},
			{DataPointer: uintptr(unsafe.Pointer(utf16PtrFromString("ExampleName"))), Size: 2 * uint16(len("ExampleName")+1)},
		}

		eventDescriptor := windows.EventDescriptor{
			Id:      2,
			Version: 0,
			Level:   2, // Error
			Opcode:  12, // Initialize
			Task:    1,  // Disconnect
			Keyword: 0x2 | 0x8, // Remote + Write
		}

		status := windows.EventWrite(regHandle, &eventDescriptor, uint32(len(eventData)), &eventData[0])
		if status != 0 {
			t.Errorf("EventWrite Event 2 failed: 0x%x", status)
		}
	})

	// Send event ID 3 (TEMPFILE_CLEANUP_EVENT)
	t.Run("SendEvent3", func(t *testing.T) {
		files := "temp1.log\000temp2.tmp"
		filesCount := uint16(2)

		eventData := []windows.EventData{
			{DataPointer: uintptr(unsafe.Pointer(&filesCount)), Size: 2},
			{DataPointer: uintptr(unsafe.Pointer(utf16PtrFromString(files))), Size: 2 * uint16(len(files)+1)},
			{DataPointer: uintptr(unsafe.Pointer(utf16PtrFromString("C:\\Temp\\Unclean"))), Size: 2 * uint16(len("C:\\Temp\\Unclean")+1)},
		}

		eventDescriptor := windows.EventDescriptor{
			Id:      3,
			Version: 0,
			Level:   16, // NotValid
			Opcode:  13, // Cleanup
			Task:    3,  // Validate
			Keyword: 0x4 | 0x2, // Local + Write
		}

		status := windows.EventWrite(regHandle, &eventDescriptor, uint32(len(eventData)), &eventData[0])
		if status != 0 {
			t.Errorf("EventWrite Event 3 failed: 0x%x", status)
		}
	})
}
