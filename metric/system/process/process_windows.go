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

package process

import (
	"errors"
	"fmt"
	"path/filepath"
	"syscall"
	"unsafe"

	xsyswindows "golang.org/x/sys/windows"

	"github.com/elastic/elastic-agent-libs/opt"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/resolve"
	"github.com/elastic/gosigar/sys/windows"
)

// FetchPids returns a map and array of pids
func (procStats *Stats) FetchPids() (ProcsMap, []ProcState, error) {
	pids, err := windows.EnumProcesses()
	if err != nil {
		return nil, nil, fmt.Errorf("EnumProcesses failed: %w", err)
	}

	procMap := make(ProcsMap, 0)
	var plist []ProcState
	// This is probably the only implementation that doesn't benefit from our
	// little fillPid callback system. We'll need to iterate over everything
	// manually.
	for _, pid := range pids {
		procMap, plist = procStats.pidIter(int(pid), procMap, plist)
	}

	return procMap, plist, nil
}

// GetInfoForPid returns basic info for the process
func GetInfoForPid(_ resolve.Resolver, pid int) (ProcState, error) {
	var err error
	var errs []error
	state := ProcState{Pid: opt.IntWith(pid)}

	name, err := getProcName(pid)
	if err != nil {
		errs = append(errs, fmt.Errorf("error fetching name: %w", err))
	} else {
		state.Name = name
	}

	// system/process doesn't need this here, but system/process_summary does.
	status, err := getPidStatus(pid)
	if err != nil {
		errs = append(errs, fmt.Errorf("error fetching status: %w", err))
	} else {
		state.State = status
	}

	if numThreads, err := FetchNumThreads(pid); err != nil {
		errs = append(errs, fmt.Errorf("error fetching num threads: %w", err))
	} else {
		state.NumThreads = opt.IntWith(numThreads)
	}

	if err := errors.Join(errs...); err != nil {
		return state, fmt.Errorf("could not get all information for PID %d: %w",
			pid, err)
	}

	return state, nil
}

func FetchNumThreads(pid int) (int, error) {
	pHandle, err := syscall.OpenProcess(
		xsyswindows.PROCESS_QUERY_INFORMATION,
		false,
		uint32(pid))
	if err != nil {
		return 0, fmt.Errorf("OpenProcess failed for PID %d: %w", pid, err)
	}
	defer syscall.CloseHandle(pHandle)

	var snapshotHandle syscall.Handle
	err = PssCaptureSnapshot(pHandle, PSSCaptureThreads, 0, &snapshotHandle)
	if err != nil {
		return 0, fmt.Errorf("PssCaptureSnapshot failed: %w", err)
	}

	info := PssThreadInformation{}
	buffSize := unsafe.Sizeof(info)
	err = PssQuerySnapshot(snapshotHandle, PssQueryThreadInformation, &info, uint32(buffSize))
	if err != nil {
		return 0, fmt.Errorf("PssQuerySnapshot failed: %w", err)
	}

	return int(info.ThreadsCaptured), nil
}

// FillPidMetrics is the windows implementation
func FillPidMetrics(_ resolve.Resolver, pid int, state ProcState, _ func(string) bool) (ProcState, error) {
	user, err := getProcCredName(pid)
	if err != nil {
		return state, fmt.Errorf("error fetching username: %w", err)
	}
	state.Username = user

	ppid, _ := getParentPid(pid)
	state.Ppid = opt.IntWith(ppid)

	wss, size, err := procMem(pid)
	if err != nil {
		return state, fmt.Errorf("error fetching memory: %w", err)
	}
	state.Memory.Rss.Bytes = opt.UintWith(wss)
	state.Memory.Size = opt.UintWith(size)

	userTime, sysTime, startTime, err := getProcTimes(pid)
	if err != nil {
		return state, fmt.Errorf("error getting CPU times: %w", err)
	}

	state.CPU.System.Ticks = opt.UintWith(sysTime)
	state.CPU.User.Ticks = opt.UintWith(userTime)
	state.CPU.Total.Ticks = opt.UintWith(userTime + sysTime)

	state.CPU.StartTime = unixTimeMsToTime(startTime)

	argList, err := getProcArgs(pid)
	if err != nil {
		return state, fmt.Errorf("error fetching process args: %w", err)
	}
	state.Args = argList
	return state, nil
}

func getProcArgs(pid int) ([]string, error) {
	handle, err := syscall.OpenProcess(
		windows.PROCESS_QUERY_LIMITED_INFORMATION|
			windows.PROCESS_VM_READ,
		false,
		uint32(pid))
	if err != nil {
		return nil, fmt.Errorf("OpenProcess failed: %w", err)
	}
	defer func() {
		_ = syscall.CloseHandle(handle)
	}()
	pbi, err := windows.NtQueryProcessBasicInformation(handle)
	if err != nil {
		return nil, fmt.Errorf("NtQueryProcessBasicInformation failed: %w", err)
	}

	userProcParams, err := windows.GetUserProcessParams(handle, pbi)
	if err != nil {
		return nil, fmt.Errorf("GetUserProcessParams failed: %w", err)
	}
	argsW, err := windows.ReadProcessUnicodeString(handle, &userProcParams.CommandLine)
	if err != nil {
		return nil, fmt.Errorf("ReadProcessUnicodeString failed: %w", err)
	}

	procList, err := windows.ByteSliceToStringSlice(argsW)
	if err != nil {
		return nil, fmt.Errorf("ByteSliceToStringSlice failed: %w", err)
	}
	return procList, nil
}

func getProcTimes(pid int) (uint64, uint64, uint64, error) {
	handle, err := syscall.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return 0, 0, 0, fmt.Errorf("OpenProcess failed for pid=%v: %w", pid, err)
	}
	defer func() {
		_ = syscall.CloseHandle(handle)
	}()

	var cpu syscall.Rusage
	if err := syscall.GetProcessTimes(handle, &cpu.CreationTime, &cpu.ExitTime, &cpu.KernelTime, &cpu.UserTime); err != nil {
		return 0, 0, 0, fmt.Errorf("GetProcessTimes failed for pid=%v: %w", pid, err)
	}

	// Everything expects ticks, so we need to go some math.
	return uint64(windows.FiletimeToDuration(&cpu.UserTime).Nanoseconds() / 1e6), uint64(windows.FiletimeToDuration(&cpu.KernelTime).Nanoseconds() / 1e6), uint64(cpu.CreationTime.Nanoseconds() / 1e6), nil
}

func procMem(pid int) (uint64, uint64, error) {
	handle, err := syscall.OpenProcess(
		windows.PROCESS_QUERY_LIMITED_INFORMATION|
			windows.PROCESS_VM_READ,
		false,
		uint32(pid))
	if err != nil {
		return 0, 0, fmt.Errorf("OpenProcess failed for pid=%v: %w", pid, err)
	}
	defer func() {
		_ = syscall.CloseHandle(handle)
	}()

	counters, err := windows.GetProcessMemoryInfo(handle)
	if err != nil {
		return 0, 0, fmt.Errorf("GetProcessMemoryInfo failed for pid=%v: %w", pid, err)
	}
	return uint64(counters.WorkingSetSize), uint64(counters.PrivateUsage), nil
}

// getProcName returns the process name associated with the PID.
func getProcName(pid int) (string, error) {
	handle, err := syscall.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return "", fmt.Errorf("OpenProcess failed for pid=%v: %w", pid, err)
	}
	defer func() {
		_ = syscall.CloseHandle(handle)
	}()

	filename, err := windows.GetProcessImageFileName(handle)
	if err != nil {
		return "", fmt.Errorf("GetProcessImageFileName failed for pid=%v: %w", pid, err)
	}

	return filepath.Base(filename), nil
}

// getProcStatus returns the status of a process.
func getPidStatus(pid int) (PidState, error) {
	handle, err := syscall.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return Unknown, fmt.Errorf("OpenProcess failed for pid=%v: %w", pid, err)
	}
	defer func() {
		_ = syscall.CloseHandle(handle)
	}()

	var exitCode uint32
	err = syscall.GetExitCodeProcess(handle, &exitCode)
	if err != nil {
		return Unknown, fmt.Errorf("GetExitCodeProcess failed for pid=%v: %w", pid, err)
	}

	if exitCode == 259 { // still active
		return Running, nil
	}
	return Sleeping, nil
}

// getParentPid returns the parent process ID of a process.
func getParentPid(pid int) (int, error) {
	handle, err := syscall.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return 0, fmt.Errorf("OpenProcess failed for pid=%v: %w", pid, err)
	}
	defer func() {
		_ = syscall.CloseHandle(handle)
	}()

	procInfo, err := windows.NtQueryProcessBasicInformation(handle)
	if err != nil {
		return 0, fmt.Errorf("NtQueryProcessBasicInformation failed for pid=%v: %w", pid, err)
	}

	return int(procInfo.InheritedFromUniqueProcessID), nil
}

func getProcCredName(pid int) (string, error) {
	handle, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		return "", fmt.Errorf("OpenProcess failed for pid=%v: %w", pid, err)
	}
	defer func() {
		_ = syscall.CloseHandle(handle)
	}()

	// Find process token via win32.
	var token syscall.Token
	err = syscall.OpenProcessToken(handle, syscall.TOKEN_QUERY, &token)
	if err != nil {
		return "", fmt.Errorf("OpenProcessToken failed for pid=%v: %w", pid, err)
	}
	// Close token to prevent handle leaks.
	defer token.Close()

	// Find the token user.
	tokenUser, err := token.GetTokenUser()
	if err != nil {
		return "", fmt.Errorf("GetTokenInformation failed for pid=%v: %w", pid, err)
	}

	// Look up domain account by SID.
	account, domain, _, err := tokenUser.User.Sid.LookupAccount("")
	if err != nil {
		sid, sidErr := tokenUser.User.Sid.String()
		if sidErr != nil {
			return "", fmt.Errorf("failed while looking up account name for pid=%v: %w", pid, err)
		}
		return "", fmt.Errorf("failed while looking up account name for SID=%v of pid=%v: %w", sid, pid, err)
	}

	return fmt.Sprintf(`%s\%s`, domain, account), nil
}
