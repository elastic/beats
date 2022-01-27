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

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/metric/system/resolve"
	"github.com/elastic/beats/v7/libbeat/opt"
	"github.com/elastic/gosigar/sys/windows"
)

var (
	// version is Windows version of the host OS.
	version = windows.GetWindowsVersion()

	processQueryLimitedInfoAccess = windows.PROCESS_QUERY_LIMITED_INFORMATION
)

// FetchPids returns a map and array of pids
func (procStats *Stats) FetchPids() (ProcsMap, []ProcState, error) {
	pids, err := windows.EnumProcesses()
	if err != nil {
		return nil, nil, errors.Wrap(err, "EnumProcesses failed")
	}

	procMap := make(ProcsMap, 0)
	var plist []ProcState
	// This is probably the only implementation that doesn't benefit from our little fillPid callback system. We'll need to iterate over everything manually.
	for _, pid := range pids {
		status, saved, err := procStats.pidFill(int(pid), true)
		if err != nil {
			procStats.logger.Debugf("Error fetching PID info for %d, skipping: %s", pid, err)
			continue
		}
		if !saved {
			procStats.logger.Debugf("Process name does not matches the provided regex; PID=%d; name=%s", pid, status.Name)
			continue
		}

		procMap[int(pid)] = status
		plist = append(plist, status)
	}

	return procMap, plist, nil
}

// GetInfoForPid returns basic info for the process
func GetInfoForPid(_ resolve.Resolver, pid int) (ProcState, error) {
	state := ProcState{}

	name, err := getProcName(pid)
	if err != nil {
		return state, errors.Wrap(err, "error fetching name")
	}
	state.Name = name
	state.Pid = opt.IntWith(pid)

	return state, nil
}

// FillPidMetrics is the darwin implementation
func FillPidMetrics(_ resolve.Resolver, pid int, state ProcState, _ func(string) bool) (ProcState, error) {
	status, err := getPidStatus(pid)
	if err != nil {
		return state, errors.Wrap(err, "error fetching status")
	}
	state.State = status

	user, err := getProcCredName(pid)
	if err != nil {
		return state, errors.Wrap(err, "error fetching username")
	}
	state.Username = user

	ppid, err := getParentPid(pid)
	state.Ppid = opt.IntWith(ppid)

	wss, size, err := procMem(pid)
	if err != nil {
		return state, errors.Wrap(err, "error fetching memory")
	}
	state.Memory.Rss.Bytes = opt.UintWith(wss)
	state.Memory.Size = opt.UintWith(size)

	userTime, sysTime, startTime, err := getProcTimes(pid)
	if err != nil {
		return state, errors.Wrap(err, "error getting CPU times")
	}

	state.CPU.System.Ticks = opt.UintWith(sysTime)
	state.CPU.User.Ticks = opt.UintWith(userTime)
	state.CPU.Total.Ticks = opt.UintWith(userTime + sysTime)

	state.CPU.StartTime = unixTimeMsToTime(startTime)

	argList, err := getProcArgs(pid)
	if err != nil {
		return state, errors.Wrap(err, "error fetching process args")
	}
	state.Args = argList
	return state, nil
}

func getProcArgs(pid int) ([]string, error) {

	handle, err := syscall.OpenProcess(processQueryLimitedInfoAccess|windows.PROCESS_VM_READ, false, uint32(pid))
	if err != nil {
		return nil, errors.Wrap(err, "OpenProcess failed")
	}
	defer syscall.CloseHandle(handle)
	pbi, err := windows.NtQueryProcessBasicInformation(handle)
	if err != nil {
		return nil, errors.Wrap(err, "NtQueryProcessBasicInformation failed")
	}

	userProcParams, err := windows.GetUserProcessParams(handle, pbi)
	if err != nil {
		return nil, errors.Wrap(err, "GetUserProcessParams failed")
	}
	argsW, err := windows.ReadProcessUnicodeString(handle, &userProcParams.CommandLine)
	if err != nil {
		return nil, errors.Wrap(err, "ReadProcessUnicodeString failed")
	}

	procList, err := windows.ByteSliceToStringSlice(argsW)
	if err != nil {
		return nil, errors.Wrap(err, "ByteSliceToStringSlice failed")
	}
	return procList, nil
}

func getProcTimes(pid int) (uint64, uint64, uint64, error) {
	handle, err := syscall.OpenProcess(processQueryLimitedInfoAccess, false, uint32(pid))
	if err != nil {
		return 0, 0, 0, errors.Wrapf(err, "OpenProcess failed for pid=%v", pid)
	}
	defer syscall.CloseHandle(handle)

	var cpu syscall.Rusage
	if err := syscall.GetProcessTimes(handle, &cpu.CreationTime, &cpu.ExitTime, &cpu.KernelTime, &cpu.UserTime); err != nil {
		return 0, 0, 0, errors.Wrapf(err, "GetProcessTimes failed for pid=%v", pid)
	}

	// Everything expects ticks, so we need to go some math.
	return uint64(windows.FiletimeToDuration(&cpu.UserTime).Nanoseconds() / 1e6), uint64(windows.FiletimeToDuration(&cpu.KernelTime).Nanoseconds() / 1e6), uint64(cpu.CreationTime.Nanoseconds() / 1e6), nil
}

func procMem(pid int) (uint64, uint64, error) {
	handle, err := syscall.OpenProcess(processQueryLimitedInfoAccess|windows.PROCESS_VM_READ, false, uint32(pid))
	if err != nil {
		return 0, 0, errors.Wrapf(err, "OpenProcess failed for pid=%v", pid)
	}
	defer syscall.CloseHandle(handle)

	counters, err := windows.GetProcessMemoryInfo(handle)
	if err != nil {
		return 0, 0, errors.Wrapf(err, "GetProcessMemoryInfo failed for pid=%v", pid)
	}
	return uint64(counters.WorkingSetSize), uint64(counters.PrivateUsage), nil
}

// getProcName returns the process name associated with the PID.
func getProcName(pid int) (string, error) {
	handle, err := syscall.OpenProcess(processQueryLimitedInfoAccess, false, uint32(pid))
	if err != nil {
		return "", errors.Wrapf(err, "OpenProcess failed for pid=%v", pid)
	}
	defer syscall.CloseHandle(handle)

	filename, err := windows.GetProcessImageFileName(handle)
	if err != nil {
		return "", errors.Wrapf(err, "GetProcessImageFileName failed for pid=%v", pid)
	}

	return filepath.Base(filename), nil
}

// getProcStatus returns the status of a process.
func getPidStatus(pid int) (string, error) {
	handle, err := syscall.OpenProcess(processQueryLimitedInfoAccess, false, uint32(pid))
	if err != nil {
		return "unknown", errors.Wrapf(err, "OpenProcess failed for pid=%v", pid)
	}
	defer syscall.CloseHandle(handle)

	var exitCode uint32
	err = syscall.GetExitCodeProcess(handle, &exitCode)
	if err != nil {
		return "unknown", errors.Wrapf(err, "GetExitCodeProcess failed for pid=%v", pid)
	}

	if exitCode == 259 { //still active
		return "running", nil
	}
	return "sleeping", nil
}

// getParentPid returns the parent process ID of a process.
func getParentPid(pid int) (int, error) {
	handle, err := syscall.OpenProcess(processQueryLimitedInfoAccess, false, uint32(pid))
	if err != nil {
		return 0, errors.Wrapf(err, "OpenProcess failed for pid=%v", pid)
	}
	defer syscall.CloseHandle(handle)

	procInfo, err := windows.NtQueryProcessBasicInformation(handle)
	if err != nil {
		return 0, errors.Wrapf(err, "NtQueryProcessBasicInformation failed for pid=%v", pid)
	}

	return int(procInfo.InheritedFromUniqueProcessID), nil
}

func getProcCredName(pid int) (string, error) {
	handle, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		return "", errors.Wrapf(err, "OpenProcess failed for pid=%v", pid)
	}
	defer syscall.CloseHandle(handle)

	// Find process token via win32.
	var token syscall.Token
	err = syscall.OpenProcessToken(handle, syscall.TOKEN_QUERY, &token)
	if err != nil {
		return "", errors.Wrapf(err, "OpenProcessToken failed for pid=%v", pid)
	}
	// Close token to prevent handle leaks.
	defer token.Close()

	// Find the token user.
	tokenUser, err := token.GetTokenUser()
	if err != nil {
		return "", errors.Wrapf(err, "GetTokenInformation failed for pid=%v", pid)
	}

	// Look up domain account by SID.
	account, domain, _, err := tokenUser.User.Sid.LookupAccount("")
	if err != nil {
		sid, sidErr := tokenUser.User.Sid.String()
		if sidErr != nil {
			return "", errors.Wrapf(err, "failed while looking up account name for pid=%v", pid)
		}
		return "", errors.Wrapf(err, "failed while looking up account name for SID=%v of pid=%v", sid, pid)
	}

	return fmt.Sprintf(`%s\%s`, domain, account), nil
}
