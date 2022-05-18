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
	"fmt"
	"path/filepath"
	"syscall"

	"github.com/elastic/beats/v7/libbeat/metric/system/resolve"
	"github.com/elastic/beats/v7/libbeat/opt"
	"github.com/elastic/gosigar/sys/windows"
)

var (
	processQueryLimitedInfoAccess = windows.PROCESS_QUERY_LIMITED_INFORMATION
)

// FetchPids returns a map and array of pids
func (procStats *Stats) FetchPids() (ProcsMap, []ProcState, error) {
	pids, err := windows.EnumProcesses()
	if err != nil {
		return nil, nil, fmt.Errorf("enumProcesses failed: %w", err)
	}

	procMap := make(ProcsMap, 0)
	var plist []ProcState
	// This is probably the only implementation that doesn't benefit from our little fillPid callback system. We'll need to iterate over everything manually.
	for _, pid := range pids {
		procMap, plist = procStats.pidIter(int(pid), procMap, plist)
	}

	return procMap, plist, nil
}

// GetInfoForPid returns basic info for the process
func GetInfoForPid(_ resolve.Resolver, pid int) (ProcState, error) {
	state := ProcState{}

	name, err := getProcName(pid)
	if err != nil {
		return state, fmt.Errorf("error fetching name: %w", err)
	}
	state.Name = name
	state.Pid = opt.IntWith(pid)

	// system/process doesn't need this here, but system/process_summary does.
	status, err := getPidStatus(pid)
	if err != nil {
		return state, fmt.Errorf("error fetching status: %w", err)
	}
	state.State = status

	return state, nil
}

// FillPidMetrics is the windows implementation
func FillPidMetrics(_ resolve.Resolver, pid int, state ProcState, _ func(string) bool) (ProcState, error) {
	user, err := getProcCredName(pid)
	if err != nil {
		return state, fmt.Errorf("error fetching username: %w", err)
	}
	state.Username = user

	ppid, err := getParentPid(pid)
	if err != nil {
		return state, fmt.Errorf("error fetching parent pid: %w", err)
	}
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

	handle, err := syscall.OpenProcess(processQueryLimitedInfoAccess|windows.PROCESS_VM_READ, false, uint32(pid))
	if err != nil {
		return nil, fmt.Errorf("openProcess failed: %w", err)
	}
	defer func() {
		_ = syscall.CloseHandle(handle)
	}()
	pbi, err := windows.NtQueryProcessBasicInformation(handle)
	if err != nil {
		return nil, fmt.Errorf("ntQueryProcessBasicInformation failed: %w", err)
	}

	userProcParams, err := windows.GetUserProcessParams(handle, pbi)
	if err != nil {
		return nil, fmt.Errorf("getUserProcessParams failed: %w", err)
	}
	argsW, err := windows.ReadProcessUnicodeString(handle, &userProcParams.CommandLine)
	if err != nil {
		return nil, fmt.Errorf("readProcessUnicodeString failed, %w", err)
	}

	procList, err := windows.ByteSliceToStringSlice(argsW)
	if err != nil {
		return nil, fmt.Errorf("byteSliceToStringSlice failed: %w", err)
	}
	return procList, nil
}

func getProcTimes(pid int) (uint64, uint64, uint64, error) {
	handle, err := syscall.OpenProcess(processQueryLimitedInfoAccess, false, uint32(pid))
	if err != nil {
		return 0, 0, 0, fmt.Errorf("openProcess failed for pid=%v: %w", pid, err)
	}
	defer func() {
		_ = syscall.CloseHandle(handle)
	}()

	var cpu syscall.Rusage
	if err := syscall.GetProcessTimes(handle, &cpu.CreationTime, &cpu.ExitTime, &cpu.KernelTime, &cpu.UserTime); err != nil {
		return 0, 0, 0, fmt.Errorf("getProcessTimes failed for pid=%v: %w", pid, err)
	}

	// Everything expects ticks, so we need to go some math.
	return uint64(windows.FiletimeToDuration(&cpu.UserTime).Nanoseconds() / 1e6), uint64(windows.FiletimeToDuration(&cpu.KernelTime).Nanoseconds() / 1e6), uint64(cpu.CreationTime.Nanoseconds() / 1e6), nil
}

func procMem(pid int) (uint64, uint64, error) {
	handle, err := syscall.OpenProcess(processQueryLimitedInfoAccess|windows.PROCESS_VM_READ, false, uint32(pid))
	if err != nil {
		return 0, 0, fmt.Errorf("openProcess failed for pid=%v: %w", pid, err)
	}
	defer func() {
		_ = syscall.CloseHandle(handle)
	}()

	counters, err := windows.GetProcessMemoryInfo(handle)
	if err != nil {
		return 0, 0, fmt.Errorf("getProcessMemoryInfo failed for pid=%v: %w", pid, err)
	}
	return uint64(counters.WorkingSetSize), uint64(counters.PrivateUsage), nil
}

// getProcName returns the process name associated with the PID.
func getProcName(pid int) (string, error) {
	handle, err := syscall.OpenProcess(processQueryLimitedInfoAccess, false, uint32(pid))
	if err != nil {
		return "", fmt.Errorf("openProcess failed for pid=%v: %w", pid, err)
	}
	defer func() {
		_ = syscall.CloseHandle(handle)
	}()

	filename, err := windows.GetProcessImageFileName(handle)
	if err != nil {
		return "", fmt.Errorf("getProcessImageFileName failed for pid=%v: %w", pid, err)
	}

	return filepath.Base(filename), nil
}

// getProcStatus returns the status of a process.
func getPidStatus(pid int) (PidState, error) {
	handle, err := syscall.OpenProcess(processQueryLimitedInfoAccess, false, uint32(pid))
	if err != nil {
		return Unknown, fmt.Errorf("openProcess failed for pid=%v: %w", pid, err)
	}
	defer func() {
		_ = syscall.CloseHandle(handle)
	}()

	var exitCode uint32
	err = syscall.GetExitCodeProcess(handle, &exitCode)
	if err != nil {
		return Unknown, fmt.Errorf("getExitCodeProcess failed for pid=%v: %w", pid, err)
	}

	if exitCode == 259 { //still active
		return Running, nil
	}
	return Sleeping, nil
}

// getParentPid returns the parent process ID of a process.
func getParentPid(pid int) (int, error) {
	handle, err := syscall.OpenProcess(processQueryLimitedInfoAccess, false, uint32(pid))
	if err != nil {
		return 0, fmt.Errorf("openProcess failed for pid=%v: %w", pid, err)
	}
	defer func() {
		_ = syscall.CloseHandle(handle)
	}()

	procInfo, err := windows.NtQueryProcessBasicInformation(handle)
	if err != nil {
		return 0, fmt.Errorf("ntQueryProcessBasicInformation failed for pid=%v: %w", pid, err)
	}

	return int(procInfo.InheritedFromUniqueProcessID), nil
}

func getProcCredName(pid int) (string, error) {
	handle, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		return "", fmt.Errorf("openProcess failed for pid=%v: %w", pid, err)
	}
	defer func() {
		_ = syscall.CloseHandle(handle)
	}()

	// Find process token via win32.
	var token syscall.Token
	err = syscall.OpenProcessToken(handle, syscall.TOKEN_QUERY, &token)
	if err != nil {
		return "", fmt.Errorf("openProcessToken failed for pid=%v: %w", pid, err)
	}
	// Close token to prevent handle leaks.
	defer token.Close()

	// Find the token user.
	tokenUser, err := token.GetTokenUser()
	if err != nil {
		return "", fmt.Errorf("getTokenInformation failed for pid=%v: %w", pid, err)
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
