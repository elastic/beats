// Copyright 2018 Elasticsearch Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build windows

package windows

import (
	"fmt"
	"syscall"

	"github.com/pkg/errors"
)

// Syscalls
//sys   _GetNativeSystemInfo(systemInfo *SystemInfo) (err error) = kernel32.GetNativeSystemInfo
//sys   _GetTickCount64() (millis uint64, err error) = kernel32.GetTickCount64

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

// GetNativeSystemInfo retrieves information about the current system to an
// application running under WOW64. If the function is called from a 64-bit
// application, it is equivalent to the GetSystemInfo function.
// https://msdn.microsoft.com/en-us/library/ms724340%28v=vs.85%29.aspx?f=255&MSPPError=-2147217396
func GetNativeSystemInfo() (SystemInfo, error) {
	var systemInfo SystemInfo
	if err := _GetNativeSystemInfo(&systemInfo); err != nil {
		return SystemInfo{}, errors.Wrap(err, "GetNativeSystemInfo failed")
	}
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
