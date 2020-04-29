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

// +build windows

package windows

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"

	"github.com/pkg/errors"
)

// Syscalls
//sys   _GetNativeSystemInfo(systemInfo *SystemInfo) = kernel32.GetNativeSystemInfo
//sys   _GetTickCount64() (millis uint64, err error) = kernel32.GetTickCount64
//sys   _GetSystemTimes(idleTime *syscall.Filetime, kernelTime *syscall.Filetime, userTime *syscall.Filetime) (err error) = kernel32.GetSystemTimes
//sys   _GlobalMemoryStatusEx(buffer *MemoryStatusEx) (err error) = kernel32.GlobalMemoryStatusEx
//sys   _ReadProcessMemory(handle syscall.Handle, baseAddress uintptr, buffer uintptr, size uintptr, numRead *uintptr) (err error) = kernel32.ReadProcessMemory
//sys   _GetProcessHandleCount(handle syscall.Handle, pdwHandleCount *uint32) (err error) = kernel32.GetProcessHandleCount

var (
	sizeofMemoryStatusEx = uint32(unsafe.Sizeof(MemoryStatusEx{}))
)

// SystemInfo is an equivalent representation of SYSTEM_INFO in the Windows API.
// https://msdn.microsoft.com/en-us/library/ms724958%28VS.85%29.aspx?f=255&MSPPError=-2147217396
type SystemInfo struct {
	ProcessorArchitecture     ProcessorArchitecture
	Reserved                  uint16
	PageSize                  uint32
	MinimumApplicationAddress uintptr
	MaximumApplicationAddress uintptr
	ActiveProcessorMask       uint64
	NumberOfProcessors        uint32
	ProcessorType             ProcessorType
	AllocationGranularity     uint32
	ProcessorLevel            uint16
	ProcessorRevision         uint16
}

// ProcessorArchitecture specifies the processor architecture that the OS requires.
type ProcessorArchitecture uint16

// List of processor architectures associated with SystemInfo.
const (
	ProcessorArchitectureAMD64   ProcessorArchitecture = 9
	ProcessorArchitectureARM     ProcessorArchitecture = 5
	ProcessorArchitectureARM64   ProcessorArchitecture = 12
	ProcessorArchitectureIA64    ProcessorArchitecture = 6
	ProcessorArchitectureIntel   ProcessorArchitecture = 0
	ProcessorArchitectureUnknown ProcessorArchitecture = 0xFFFF
)

// ErrReadFailed is returned by ReadProcessMemory on failure
var ErrReadFailed = errors.New("ReadProcessMemory failed")

func (a ProcessorArchitecture) String() string {
	names := map[ProcessorArchitecture]string{
		ProcessorArchitectureAMD64: "x86_64",
		ProcessorArchitectureARM:   "arm",
		ProcessorArchitectureARM64: "arm64",
		ProcessorArchitectureIA64:  "ia64",
		ProcessorArchitectureIntel: "x86",
	}

	name, found := names[a]
	if !found {
		return "unknown"
	}
	return name
}

// ProcessorType specifies the type of processor.
type ProcessorType uint32

// List of processor types associated with SystemInfo.
const (
	ProcessorTypeIntel386     ProcessorType = 386
	ProcessorTypeIntel486     ProcessorType = 486
	ProcessorTypeIntelPentium ProcessorType = 586
	ProcessorTypeIntelIA64    ProcessorType = 2200
	ProcessorTypeAMDX8664     ProcessorType = 8664
)

func (t ProcessorType) String() string {
	names := map[ProcessorType]string{
		ProcessorTypeIntel386:     "386",
		ProcessorTypeIntel486:     "486",
		ProcessorTypeIntelPentium: "586",
		ProcessorTypeIntelIA64:    "ia64",
		ProcessorTypeAMDX8664:     "x64_64",
	}

	name, found := names[t]
	if !found {
		return "unknown"
	}
	return name
}

// MemoryStatusEx is an equivalent representation of MEMORYSTATUSEX in the
// Windows API. It contains information about the current state of both physical
// and virtual memory, including extended memory.
// https://msdn.microsoft.com/en-us/library/windows/desktop/aa366770
type MemoryStatusEx struct {
	length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

// GetNativeSystemInfo retrieves information about the current system to an
// application running under WOW64. If the function is called from a 64-bit
// application, it is equivalent to the GetSystemInfo function.
// https://msdn.microsoft.com/en-us/library/ms724340%28v=vs.85%29.aspx?f=255&MSPPError=-2147217396
func GetNativeSystemInfo() (SystemInfo, error) {
	var systemInfo SystemInfo
	_GetNativeSystemInfo(&systemInfo)
	return systemInfo, nil
}

// Version identifies a Windows version by major, minor, and build number.
type Version struct {
	Major int
	Minor int
	Build int
}

// GetWindowsVersion returns the Windows version information. Applications not
// manifested for Windows 8.1 or Windows 10 will return the Windows 8 OS version
// value (6.2).
//
// For a table of version numbers see:
// https://msdn.microsoft.com/en-us/library/windows/desktop/ms724833(v=vs.85).aspx
func GetWindowsVersion() Version {
	// https://msdn.microsoft.com/en-us/library/windows/desktop/ms724439(v=vs.85).aspx
	ver, err := syscall.GetVersion()
	if err != nil {
		// GetVersion should never return an error.
		panic(fmt.Errorf("GetVersion failed: %v", err))
	}

	return Version{
		Major: int(ver & 0xFF),
		Minor: int(ver >> 8 & 0xFF),
		Build: int(ver >> 16),
	}
}

// IsWindowsVistaOrGreater returns true if the Windows version is Vista or
// greater.
func (v Version) IsWindowsVistaOrGreater() bool {
	// Vista is 6.0.
	return v.Major >= 6 && v.Minor >= 0
}

// GetTickCount64 retrieves the number of milliseconds that have elapsed since
// the system was started.
// This function is available on Windows Vista and newer.
// https://msdn.microsoft.com/en-us/library/windows/desktop/ms724411(v=vs.85).aspx
func GetTickCount64() (uint64, error) {
	return _GetTickCount64()
}

// GetSystemTimes retrieves system timing information. On a multiprocessor
// system, the values returned are the sum of the designated times across all
// processors. The returned kernel time does not include the system idle time.
// https://msdn.microsoft.com/en-us/library/windows/desktop/ms724400(v=vs.85).aspx
func GetSystemTimes() (idle, kernel, user time.Duration, err error) {
	var idleTime, kernelTime, userTime syscall.Filetime
	err = _GetSystemTimes(&idleTime, &kernelTime, &userTime)
	if err != nil {
		return 0, 0, 0, errors.Wrap(err, "GetSystemTimes failed")
	}

	idle = FiletimeToDuration(&idleTime)
	kernel = FiletimeToDuration(&kernelTime) // Kernel time includes idle time so we subtract it out.
	user = FiletimeToDuration(&userTime)

	return idle, kernel - idle, user, nil
}

// FiletimeToDuration converts a Filetime to a time.Duration. Do not use this
// method to convert a Filetime to an actual clock time, for that use
// Filetime.Nanosecond().
func FiletimeToDuration(ft *syscall.Filetime) time.Duration {
	n := int64(ft.HighDateTime)<<32 + int64(ft.LowDateTime) // in 100-nanosecond intervals
	return time.Duration(n * 100)
}

// GlobalMemoryStatusEx retrieves information about the system's current usage
// of both physical and virtual memory.
// https://msdn.microsoft.com/en-us/library/windows/desktop/aa366589(v=vs.85).aspx
func GlobalMemoryStatusEx() (MemoryStatusEx, error) {
	memoryStatusEx := MemoryStatusEx{length: sizeofMemoryStatusEx}
	err := _GlobalMemoryStatusEx(&memoryStatusEx)
	if err != nil {
		return MemoryStatusEx{}, errors.Wrap(err, "GlobalMemoryStatusEx failed")
	}

	return memoryStatusEx, nil
}

// ReadProcessMemory reads from another process memory. The Handle needs to have
// the PROCESS_VM_READ right.
// A zero-byte read is a no-op, no error is returned.
func ReadProcessMemory(handle syscall.Handle, baseAddress uintptr, dest []byte) (numRead uintptr, err error) {
	n := len(dest)
	if n == 0 {
		return 0, nil
	}
	if err = _ReadProcessMemory(handle, baseAddress, uintptr(unsafe.Pointer(&dest[0])), uintptr(n), &numRead); err != nil {
		return 0, err
	}
	return numRead, nil
}

// GetProcessHandleCount retrieves the number of open handles of a process.
// https://docs.microsoft.com/en-us/windows/desktop/api/processthreadsapi/nf-processthreadsapi-getprocesshandlecount
func GetProcessHandleCount(process syscall.Handle) (uint32, error) {
	var count uint32
	if err := _GetProcessHandleCount(process, &count); err != nil {
		return 0, errors.Wrap(err, "GetProcessHandleCount failed")
	}
	return count, nil
}
