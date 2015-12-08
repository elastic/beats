package eventlog

import (
	"syscall"

	"golang.org/x/sys/windows"
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

// Read flags that indicate how to read events.
// https://msdn.microsoft.com/en-us/library/windows/desktop/aa363674(v=vs.85).aspx
const (
	EVENTLOG_SEQUENTIAL_READ = 1 << iota
	EVENTLOG_SEEK_READ
	EVENTLOG_FORWARDS_READ
	EVENTLOG_BACKWARDS_READ
)

// Event Log Error Codes
// https://msdn.microsoft.com/en-us/library/windows/desktop/ms681385(v=vs.85).aspx
const (
	ERROR_EVENTLOG_FILE_CORRUPT syscall.Errno = 1500
	ERROR_EVENTLOG_FILE_CHANGED syscall.Errno = 1503
)

// Handle to a the OS specific event log.
type Handle uintptr

// Frees the loaded dynamic-link library (DLL) module and, if necessary,
// decrements its reference count. When the reference count reaches zero, the
// module is unloaded from the address space of the calling process and the
// handle is no longer valid.
func freeLibrary(handle Handle) error {
	// Wrap the method so that we can stub it out and use our own Handle type.
	return windows.FreeLibrary(windows.Handle(handle))
}

// Add -trace to enable debug prints around syscalls.
//go:generate go run $GOROOT/src/syscall/mksyscall_windows.go -output zsyscall_windows.go syscall_windows.go

// Windows API calls
//sys   openEventLog(uncServerName *uint16, sourceName *uint16) (handle Handle, err error) = advapi32.OpenEventLogW
//sys   closeEventLog(eventLog Handle) (err error) = advapi32.CloseEventLog
//sys   readEventLog(eventLog Handle, readFlags uint32, recordOffset uint32, buffer *byte, numberOfBytesToRead uint32, bytesRead *uint32, minNumberOfBytesNeeded *uint32) (err error) = advapi32.ReadEventLogW
//sys   loadLibraryEx(filename *uint16, file Handle, flags uint32) (handle Handle, err error) = kernel32.LoadLibraryExW
//sys   formatMessage(flags uint32, source Handle, messageID uint32, languageID uint32, buffer *byte, bufferSize uint32, arguments *uintptr) (numChars uint32, err error) = kernel32.FormatMessageW
//sys   _clearEventLog(eventLog Handle, backupFileName *uint16) (err error) = advapi32.ClearEventLogW
//sys   _getNumberOfEventLogRecords(eventLog Handle, numberOfRecords *uint32) (err error) = advapi32.GetNumberOfEventLogRecords
//sys   _getOldestEventLogRecord(eventLog Handle, oldestRecord *uint32) (err error) = advapi32.GetOldestEventLogRecord

func clearEventLog(handle Handle, backupFileName string) error {
	var name *uint16
	if backupFileName != "" {
		var err error
		name, err = syscall.UTF16PtrFromString(backupFileName)
		if err != nil {
			return err
		}
	}

	return _clearEventLog(handle, name)
}

func getNumberOfEventLogRecords(handle Handle) (uint32, error) {
	var numRecords uint32
	err := _getNumberOfEventLogRecords(handle, &numRecords)
	if err != nil {
		return 0, err
	}

	return numRecords, nil
}

func getOldestEventLogRecord(handle Handle) (uint32, error) {
	var oldestRecord uint32
	err := _getOldestEventLogRecord(handle, &oldestRecord)
	if err != nil {
		return 0, err
	}

	return oldestRecord, nil
}
