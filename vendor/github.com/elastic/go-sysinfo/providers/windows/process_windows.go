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

package windows

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/pkg/errors"
	syswin "golang.org/x/sys/windows"

	windows "github.com/elastic/go-windows"

	"github.com/elastic/go-sysinfo/types"
)

var (
	selfPID   = os.Getpid()
	devMapper = newDeviceMapper()
)

func (s windowsSystem) Processes() (procs []types.Process, err error) {
	pids, err := windows.EnumProcesses()
	if err != nil {
		return nil, errors.Wrap(err, "EnumProcesses")
	}
	procs = make([]types.Process, 0, len(pids))
	var proc types.Process
	for _, pid := range pids {
		if pid == 0 || pid == 4 {
			// The Idle and System processes (PIDs 0 and 4) can never be
			// opened by user-level code (see documentation for OpenProcess).
			continue
		}

		if proc, err = s.Process(int(pid)); err == nil {
			procs = append(procs, proc)
		}
	}
	if len(procs) == 0 {
		return nil, err
	}
	return procs, nil
}

func (s windowsSystem) Process(pid int) (types.Process, error) {
	return newProcess(pid)
}

func (s windowsSystem) Self() (types.Process, error) {
	return newProcess(selfPID)
}

type process struct {
	pid  int
	info types.ProcessInfo
}

func (p *process) PID() int {
	return p.pid
}

func (p *process) Parent() (types.Process, error) {
	info, err := p.Info()
	if err != nil {
		return nil, err
	}

	return newProcess(info.PPID)
}

func newProcess(pid int) (*process, error) {
	p := &process{pid: pid}
	if err := p.init(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *process) init() error {
	handle, err := p.open()
	if err != nil {
		return err
	}
	defer syscall.CloseHandle(handle)

	var path string
	if imgf, err := windows.GetProcessImageFileName(handle); err == nil {
		path, err = devMapper.DevicePathToDrivePath(imgf)
		if err != nil {
			path = imgf
		}
	}

	var creationTime, exitTime, kernelTime, userTime syscall.Filetime
	if err := syscall.GetProcessTimes(handle, &creationTime, &exitTime, &kernelTime, &userTime); err != nil {
		return err
	}

	// Try to read the RTL_USER_PROCESS_PARAMETERS struct from the target process
	// memory. This can fail due to missing access rights or when we are running
	// as a 32bit process in a 64bit system (WOW64).
	// Don't make this a fatal error: If it fails, `args` and `cwd` fields will
	// be missing.
	var args []string
	var cwd string
	var ppid int
	pbi, err := getProcessBasicInformation(handle)
	if err == nil {
		ppid = int(pbi.InheritedFromUniqueProcessID)
		userProcParams, err := getUserProcessParams(handle, pbi)
		if err == nil {
			if argsW, err := readProcessUnicodeString(handle, &userProcParams.CommandLine); err == nil {
				args, err = splitCommandline(argsW)
				if err != nil {
					args = nil
				}
			}
			if cwdW, err := readProcessUnicodeString(handle, &userProcParams.CurrentDirectoryPath); err == nil {
				cwd, _, err = windows.UTF16BytesToString(cwdW)
				if err != nil {
					cwd = ""
				}
				// Remove trailing separator
				cwd = strings.TrimRight(cwd, "\\")
			}
		}
	}

	p.info = types.ProcessInfo{
		Name:      filepath.Base(path),
		PID:       p.pid,
		PPID:      ppid,
		Exe:       path,
		Args:      args,
		CWD:       cwd,
		StartTime: time.Unix(0, creationTime.Nanoseconds()),
	}
	return nil
}

func getProcessBasicInformation(handle syscall.Handle) (pbi windows.ProcessBasicInformationStruct, err error) {
	actualSize, err := windows.NtQueryInformationProcess(handle, windows.ProcessBasicInformation, unsafe.Pointer(&pbi), uint32(windows.SizeOfProcessBasicInformationStruct))
	if actualSize < uint32(windows.SizeOfProcessBasicInformationStruct) {
		return pbi, errors.New("bad size for PROCESS_BASIC_INFORMATION")
	}
	return pbi, err
}

func getUserProcessParams(handle syscall.Handle, pbi windows.ProcessBasicInformationStruct) (params windows.RtlUserProcessParameters, err error) {
	const is32bitProc = unsafe.Sizeof(uintptr(0)) == 4

	// Offset of params field within PEB structure.
	// This structure is different in 32 and 64 bit.
	paramsOffset := 0x20
	if is32bitProc {
		paramsOffset = 0x10
	}

	// Read the PEB from the target process memory
	pebSize := paramsOffset + 8
	peb := make([]byte, pebSize)
	nRead, err := windows.ReadProcessMemory(handle, pbi.PebBaseAddress, peb)
	if err != nil {
		return params, err
	}
	if nRead != uintptr(pebSize) {
		return params, errors.Errorf("PEB: short read (%d/%d)", nRead, pebSize)
	}

	// Get the RTL_USER_PROCESS_PARAMETERS struct pointer from the PEB
	paramsAddr := *(*uintptr)(unsafe.Pointer(&peb[paramsOffset]))

	// Read the RTL_USER_PROCESS_PARAMETERS from the target process memory
	paramsBuf := make([]byte, windows.SizeOfRtlUserProcessParameters)
	nRead, err = windows.ReadProcessMemory(handle, paramsAddr, paramsBuf)
	if err != nil {
		return params, err
	}
	if nRead != uintptr(windows.SizeOfRtlUserProcessParameters) {
		return params, errors.Errorf("RTL_USER_PROCESS_PARAMETERS: short read (%d/%d)", nRead, windows.SizeOfRtlUserProcessParameters)
	}

	params = *(*windows.RtlUserProcessParameters)(unsafe.Pointer(&paramsBuf[0]))
	return params, nil
}

// read an UTF-16 string from another process memory. Result is an []byte
// with the UTF-16 data.
func readProcessUnicodeString(handle syscall.Handle, s *windows.UnicodeString) ([]byte, error) {
	// Allocate an extra UTF-16 null character at the end in case the read string
	// is not terminated.
	extra := 2
	if s.Size&1 != 0 {
		extra = 3 // If size is odd, need 3 nulls to terminate.
	}
	buf := make([]byte, int(s.Size)+extra)
	nRead, err := windows.ReadProcessMemory(handle, s.Buffer, buf[:s.Size])
	if err != nil {
		return nil, err
	}
	if nRead != uintptr(s.Size) {
		return nil, errors.Errorf("unicode string: short read: (%d/%d)", nRead, s.Size)
	}
	return buf, nil
}

// Use Windows' CommandLineToArgv API to split an UTF-16 command line string
// into a list of parameters.
func splitCommandline(utf16 []byte) ([]string, error) {
	n := len(utf16)
	// Discard odd byte
	if n&1 != 0 {
		n--
		utf16 = utf16[:n]
	}
	if n == 0 {
		return nil, nil
	}
	terminated := false
	for i := 0; i < n && !terminated; i += 2 {
		terminated = utf16[i] == 0 && utf16[i+1] == 0
	}
	if !terminated {
		// Append a null uint16 at the end if terminator is missing
		utf16 = append(utf16, 0, 0)
	}
	var numArgs int32
	argsWide, err := syscall.CommandLineToArgv((*uint16)(unsafe.Pointer(&utf16[0])), &numArgs)
	if err != nil {
		return nil, err
	}

	// Free memory allocated for CommandLineToArgvW arguments.
	defer syscall.LocalFree((syscall.Handle)(unsafe.Pointer(argsWide)))

	args := make([]string, numArgs)
	for idx := range args {
		args[idx] = syscall.UTF16ToString(argsWide[idx][:])
	}
	return args, nil
}

func (p *process) open() (handle syscall.Handle, err error) {
	if p.pid == selfPID {
		return syscall.GetCurrentProcess()
	}

	// Try different access rights, from broader to more limited.
	// PROCESS_VM_READ is needed to get command-line and working directory
	// PROCESS_QUERY_LIMITED_INFORMATION is only available in Vista+
	for _, permissions := range [4]uint32{
		syscall.PROCESS_QUERY_INFORMATION | windows.PROCESS_VM_READ,
		windows.PROCESS_QUERY_LIMITED_INFORMATION | windows.PROCESS_VM_READ,
		syscall.PROCESS_QUERY_INFORMATION,
		windows.PROCESS_QUERY_LIMITED_INFORMATION,
	} {
		if handle, err = syscall.OpenProcess(permissions, false, uint32(p.pid)); err == nil {
			break
		}
	}
	return handle, err
}

func (p *process) Info() (types.ProcessInfo, error) {
	return p.info, nil
}

func (p *process) User() (types.UserInfo, error) {
	handle, err := p.open()
	if err != nil {
		return types.UserInfo{}, errors.Wrap(err, "OpenProcess failed")
	}
	defer syscall.CloseHandle(handle)

	var accessToken syswin.Token
	err = syswin.OpenProcessToken(syswin.Handle(handle), syscall.TOKEN_QUERY, &accessToken)
	if err != nil {
		return types.UserInfo{}, errors.Wrap(err, "OpenProcessToken failed")
	}
	defer accessToken.Close()

	tokenUser, err := accessToken.GetTokenUser()
	if err != nil {
		return types.UserInfo{}, errors.Wrap(err, "GetTokenUser failed")
	}

	sid, err := tokenUser.User.Sid.String()
	if err != nil {
		return types.UserInfo{}, errors.Wrap(err, "failed to look up user SID")
	}

	tokenGroup, err := accessToken.GetTokenPrimaryGroup()
	if err != nil {
		return types.UserInfo{}, errors.Wrap(err, "GetTokenPrimaryGroup failed")
	}
	gsid, err := tokenGroup.PrimaryGroup.String()
	if err != nil {
		return types.UserInfo{}, errors.Wrap(err, "failed to look up primary group SID")
	}

	return types.UserInfo{
		UID: sid,
		GID: gsid,
	}, nil
}

func (p *process) Memory() (types.MemoryInfo, error) {
	handle, err := p.open()
	if err != nil {
		return types.MemoryInfo{}, err
	}
	defer syscall.CloseHandle(handle)

	counters, err := windows.GetProcessMemoryInfo(handle)
	if err != nil {
		return types.MemoryInfo{}, err
	}

	return types.MemoryInfo{
		Resident: uint64(counters.WorkingSetSize),
		Virtual:  uint64(counters.PrivateUsage),
	}, nil
}

func (p *process) CPUTime() (types.CPUTimes, error) {
	handle, err := p.open()
	if err != nil {
		return types.CPUTimes{}, err
	}
	defer syscall.CloseHandle(handle)

	var creationTime, exitTime, kernelTime, userTime syscall.Filetime
	if err := syscall.GetProcessTimes(handle, &creationTime, &exitTime, &kernelTime, &userTime); err != nil {
		return types.CPUTimes{}, err
	}

	return types.CPUTimes{
		User:   windows.FiletimeToDuration(&userTime),
		System: windows.FiletimeToDuration(&kernelTime),
	}, nil
}

// OpenHandles returns the number of open handles of the process.
func (p *process) OpenHandleCount() (int, error) {
	handle, err := p.open()
	if err != nil {
		return 0, err
	}
	defer syscall.CloseHandle(handle)

	count, err := windows.GetProcessHandleCount(handle)
	return int(count), err
}
