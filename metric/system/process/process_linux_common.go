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

//go:build freebsd || linux || solaris

package process

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"strconv"
	"strings"
	"syscall"

	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/opt"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/resolve"
)

// Indulging in one non-const global variable for the sake of storing boot time
// This value obviously won't change while this code is running.
var bootTime uint64 = 0

// system tick multiplier, see C.sysconf(C._SC_CLK_TCK)
const ticks = 100

// GetSelfPid returns the process we're running as.
// for cases of self-monitoring this requires some actual thought;
// if we use os.Getpid() and we're running inside a container, that PID will
// only be valid inside the container, and an attempt to fetch metrics from
// `/hostfs/proc/` for that pid will fail. If we're using a hostfs, revert to _that_ for fetching the pid metrics.
func GetSelfPid(hostfs resolve.Resolver) (int, error) {
	if !hostfs.IsSet() {
		return os.Getpid(), nil
	}

	statRaw, err := os.ReadFile(hostfs.ResolveHostFS("/proc/self/stat"))
	if err != nil {
		return 0, fmt.Errorf("error reading from self/stat while searching for our PID in a container: %w", err)
	}

	parts := strings.Split(string(statRaw), " ")
	pidRaw := parts[0]
	pid, err := strconv.ParseInt(pidRaw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("error parsing int from `stat` while searching for our pid in a container: %w", err)
	}
	return int(pid), nil

}

// FetchPids is the linux implementation of FetchPids
func (procStats *Stats) FetchPids() (ProcsMap, []ProcState, error) {
	dir, err := os.Open(procStats.Hostfs.ResolveHostFS("proc"))
	if err != nil {
		return nil, nil, fmt.Errorf("error reading from procfs %s: %w", procStats.Hostfs.ResolveHostFS("/"), err)
	}
	defer dir.Close()

	const readAllDirnames = -1 // see os.File.Readdirnames doc

	names, err := dir.Readdirnames(readAllDirnames)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading directory names: %w", err)
	}

	procMap := make(ProcsMap, len(names))
	plist := make([]ProcState, 0, len(names))
	var wrappedErr error

	// Iterate over the directory, fetch just enough info so we can filter based on user input.
	for _, name := range names {

		if !dirIsPid(name) {
			continue
		}
		// Will this actually fail?
		pid, err := strconv.Atoi(name)
		if err != nil {
			procStats.logger.Debugf("Error converting PID name %s", name)
			continue
		}
		procMap, plist, err = procStats.pidIter(pid, procMap, plist)
		wrappedErr = errors.Join(wrappedErr, err)
	}

	return procMap, plist, toNonFatal(wrappedErr)
}

func FillPidMetrics(hostfs resolve.Resolver, pid int, state ProcState, filter func(string) bool) (ProcState, error) {
	// Memory Data
	var err error
	state.Memory, err = getMemData(hostfs, pid)
	if err != nil {
		return state, fmt.Errorf("error getting memory data for pid %d: %w", pid, err)
	}

	// CPU Data
	state.CPU, err = getCPUTime(hostfs, pid)
	if err != nil {
		return state, fmt.Errorf("error getting CPU data for pid %d: %w", pid, err)
	}

	// CLI args
	if len(state.Args) == 0 {
		state.Args, err = getArgs(hostfs, pid)
		if err != nil {
			return state, fmt.Errorf("error getting CLI args for pid %d: %w", pid, err)
		}

	}

	// FD metrics
	state.FD, err = getFDStats(hostfs, pid)
	if err != nil {
		return state, fmt.Errorf("error getting FD metrics for pid %d: %w", pid, err)
	}

	if state.Env == nil {
		// env vars
		state.Env, _ = getEnvData(hostfs, pid, filter)
	}

	state.Exe, state.Cwd, err = getProcStringData(hostfs, pid)
	// skip permission errors and file not found errors
	// see https://github.com/elastic/elastic-agent-system-metrics/issues/135 for a bit more context,
	// depending on the permissions/caps that this is running with, the /exe symlink may have different levels of permission restrictions.
	// A kernel proc will also return file not found.
	if err != nil && !errors.Is(err, os.ErrPermission) && !errors.Is(err, os.ErrNotExist) { // ignore permission errors
		return state, fmt.Errorf("error getting metadata for pid %d: %w", pid, err)
	}

	state.Username, err = getUser(hostfs, pid)
	if err != nil {
		return state, fmt.Errorf("error creating username for pid %d: %w", pid, err)
	}

	// the /proc/[pid]/io metrics require SYS_PTRACE when run from inside docker
	state.IO, err = getIOData(hostfs, pid)
	if err != nil {
		return state, NonFatalErr{Err: fmt.Errorf("/io unavailable; if running inside a container, use SYS_PTRACE: %w", err)}
	}

	return state, nil
}

// GetInfoForPid fetches and parses the process information of the process
// identified by pid from /proc/[PID]/stat
func GetInfoForPid(hostFS resolve.Resolver, pid int) (ProcState, error) {
	path := hostFS.Join("proc", strconv.Itoa(pid), "stat")
	data, err := os.ReadFile(path)
	// Transform the error into a more sensible error in cases where the directory doesn't exist, i.e the process is gone
	if err != nil {
		if os.IsNotExist(err) {
			return ProcState{}, syscall.ESRCH
		}
		return ProcState{}, fmt.Errorf("error reading procdir %s: %w", path, err)
	}

	state, err := parseProcStat(data)
	state.Pid = opt.IntWith(pid)
	if err != nil {
		return state, fmt.Errorf("failed to parse information for pid %d': %w", pid, err)
	}

	return state, nil
}

func parseProcStat(data []byte) (ProcState, error) {
	const minFields = 36
	state := ProcState{}

	// Extract the comm value with is surrounded by parentheses.
	lIdx := bytes.Index(data, []byte("("))
	rIdx := bytes.LastIndex(data, []byte(")"))
	if lIdx < 0 || rIdx < 0 || lIdx >= rIdx || rIdx+2 >= len(data) {
		return state, fmt.Errorf("failed to extract 'comm' field from '%v'",
			string(data))
	}
	state.Name = string(data[lIdx+1 : rIdx])

	// Extract the rest of the fields that we are interested in.
	fields := bytes.Fields(data[rIdx+2:])
	if len(fields) <= minFields {
		return state, fmt.Errorf("expected at least %d stat fields from '%v'",
			minFields, string(data))
	}

	// See https://man7.org/linux/man-pages/man5/proc.5.html for all fields.
	interests := bytes.Join([][]byte{
		fields[0],  // state
		fields[1],  // ppid
		fields[2],  // pgrp
		fields[17], // num_threads
	}, []byte(" "))

	var procState string
	var ppid, pgid, numThreads int
	_, err := fmt.Fscan(bytes.NewBuffer(interests),
		&procState,
		&ppid,
		&pgid,
		&numThreads,
	)
	if err != nil {
		return state, fmt.Errorf("failed to parse stat fields from '%s': %w",
			string(data), err)
	}

	state.State = getProcState(procState[0])
	state.Ppid = opt.IntWith(ppid)
	state.Pgid = opt.IntWith(pgid)
	state.NumThreads = opt.IntWith(numThreads)

	return state, nil
}

func getProcStringData(hostfs resolve.Resolver, pid int) (string, string, error) {
	exe, err := os.Readlink(hostfs.Join("proc", strconv.Itoa(pid), "exe"))
	if errors.Is(err, os.ErrPermission) || errors.Is(err, os.ErrNotExist) { // pass through errors
		return "", "", err
	} else if err != nil {
		return "", "", fmt.Errorf("error fetching exe from pid %d: %w", pid, err)
	}

	cwd, err := os.Readlink(hostfs.Join("proc", strconv.Itoa(pid), "cwd"))
	if errors.Is(err, os.ErrPermission) {
		return "", "", err
	} else if err != nil {
		return "", "", fmt.Errorf("error fetching cwd for pid %d: %w", pid, err)
	}

	return exe, cwd, nil
}

func dirIsPid(name string) bool {
	if name[0] < '0' || name[0] > '9' {
		return false
	}
	return true
}

func getUser(hostfs resolve.Resolver, pid int) (string, error) {
	status, err := getProcStatus(hostfs, pid)
	if err != nil {
		return "", fmt.Errorf("error fetching user ID for pid %d: %w", pid, err)
	}
	uidValues, ok := status["Uid"]
	if !ok {
		return "", fmt.Errorf("field Uid not found in proc status: %w", err)
	}
	uidStrings := strings.Fields(uidValues)
	var userFinal string
	user, err := user.LookupId(uidStrings[0])
	if err == nil {
		userFinal = user.Username
	} else {
		userFinal = uidStrings[0]
	}

	return userFinal, nil
}

func getEnvData(hostfs resolve.Resolver, pid int, filter func(string) bool) (mapstr.M, error) {
	path := hostfs.Join("proc", strconv.Itoa(pid), "environ")
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrPermission) { // pass through permission errors
		return nil, err
	} else if err != nil {
		return nil, fmt.Errorf("error opening file %s: %w", path, err)
	}
	env := mapstr.M{}

	pairs := bytes.Split(data, []byte{0})
	for _, kv := range pairs {
		parts := bytes.SplitN(kv, []byte{'='}, 2)
		if len(parts) != 2 {
			continue
		}

		key := string(bytes.TrimSpace(parts[0]))
		if key == "" {
			continue
		}

		if filter == nil || filter(key) {
			env[key] = string(bytes.TrimSpace(parts[1]))
		}
	}
	return env, nil
}

func getMemData(hostfs resolve.Resolver, pid int) (ProcMemInfo, error) {
	// Memory data
	state := ProcMemInfo{}
	path := hostfs.Join("proc", strconv.Itoa(pid), "statm")
	data, err := os.ReadFile(path)
	if err != nil {
		return state, fmt.Errorf("error opening file %s: %w", path, err)
	}

	fields := strings.Fields(string(data))

	size, err := strconv.ParseUint(fields[0], 10, 64)
	if err != nil {
		return state, fmt.Errorf("error parsing memory size %s: %w", fields[0], err)
	}
	state.Size = opt.UintWith(size << 12)

	rss, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return state, fmt.Errorf("error parsing memory rss %s: %w", fields[1], err)
	}
	state.Rss.Bytes = opt.UintWith(rss << 12)

	share, _ := strconv.ParseUint(fields[2], 10, 64)
	state.Share = opt.UintWith(share << 12)

	return state, nil
}

func getIOData(hostfs resolve.Resolver, pid int) (ProcIOInfo, error) {
	state := ProcIOInfo{}
	path := hostfs.Join("proc", strconv.Itoa(pid), "io")
	data, err := os.ReadFile(path)
	if err != nil {
		return state, fmt.Errorf("error fetching IO metrics: %w", err)
	}

	for _, metric := range strings.Split(string(data), "\n") {
		raw := strings.Split(metric, ": ")
		if len(raw) < 2 {
			continue
		}
		value, err := strconv.ParseUint(raw[1], 10, 64)
		if err != nil {
			return state, fmt.Errorf("error converting counters '%s' in io stat file: %w", raw, err)
		}

		switch raw[0] {
		case "rchar":
			state.ReadChar = opt.UintWith(value)
		case "wchar":
			state.WriteChar = opt.UintWith(value)
		case "syscr":
			state.ReadSyscalls = opt.UintWith(value)
		case "syscw":
			state.WriteSyscalls = opt.UintWith(value)
		case "read_bytes":
			state.ReadBytes = opt.UintWith(value)
		case "write_bytes":
			state.WriteBytes = opt.UintWith(value)
		case "cancelled_write_bytes":
			state.CancelledWriteBytes = opt.UintWith(value)
		}
	}
	return state, nil
}

func getCPUTime(hostfs resolve.Resolver, pid int) (ProcCPUInfo, error) {
	state := ProcCPUInfo{}

	pathCPU := hostfs.Join("proc", strconv.Itoa(pid), "stat")
	data, err := os.ReadFile(pathCPU)
	if err != nil {
		return state, fmt.Errorf("error opening file %s: %w", pathCPU, err)
	}
	fields := strings.Fields(string(data))

	user, err := strconv.ParseUint(fields[13], 10, 64)
	if err != nil {
		return state, fmt.Errorf("error parsing user CPU times for pid %d: %w", pid, err)
	}
	sys, err := strconv.ParseUint(fields[14], 10, 64)
	if err != nil {
		return state, fmt.Errorf("error parsing system CPU times for pid %d: %w", pid, err)
	}

	btime, err := getLinuxBootTime(hostfs)
	if err != nil {
		return state, fmt.Errorf("error feting boot time for pid %d: %w", pid, err)
	}

	// convert to milliseconds from USER_HZ
	// This effectively means our definition of "ticks" throughout the process code is a millisecond
	state.User.Ticks = opt.UintWith(user * (1000 / ticks))
	state.System.Ticks = opt.UintWith(sys * (1000 / ticks))
	state.Total.Ticks = opt.UintWith(opt.SumOptUint(state.User.Ticks, state.System.Ticks))

	startTime, err := strconv.ParseUint(fields[21], 10, 64)
	if err != nil {
		return state, fmt.Errorf("error parsing start time value %s for pid %d: %w", fields[21], pid, err)
	}

	startTime /= ticks
	startTime += btime
	startTime *= 1000

	state.StartTime = unixTimeMsToTime(startTime)
	return state, nil
}

func getArgs(hostfs resolve.Resolver, pid int) ([]string, error) {
	path := hostfs.Join("proc", strconv.Itoa(pid), "cmdline")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error opening file %s: %w", path, err)
	}
	bbuf := bytes.NewBuffer(data)

	var args []string

	for {
		arg, err := bbuf.ReadBytes(0)
		if err == io.EOF {
			break
		}
		trimmedArg := string(arg[0 : len(arg)-1])
		args = append(args, trimmedArg)
	}

	return args, nil
}

func getFDStats(hostfs resolve.Resolver, pid int) (ProcFDInfo, error) {
	state := ProcFDInfo{}

	path := hostfs.Join("proc", strconv.Itoa(pid), "limits")
	data, err := os.ReadFile(path)
	if err != nil {
		return state, fmt.Errorf("error opening file %s: %w", path, err)
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "Max open files") {
			fields := strings.Fields(line)
			if len(fields) == 6 {

				softLimit, err := strconv.ParseUint(fields[3], 10, 64)
				if err != nil {
					return state, fmt.Errorf("error parsing limits value %s for pid %d: %w", fields[3], pid, err)
				}
				state.Limit.Soft = opt.UintWith(softLimit)

				hardLimit, err := strconv.ParseUint(fields[4], 10, 64)
				if err != nil {
					return state, fmt.Errorf("error parsing limits value %s for pid %d: %w", fields[3], pid, err)
				}
				state.Limit.Hard = opt.UintWith(hardLimit)
			}

		}
	}

	pathFD := hostfs.Join("proc", strconv.Itoa(pid), "fd")
	fds, err := os.ReadDir(pathFD)
	if errors.Is(err, os.ErrPermission) { // ignore permission errors, passthrough other data
		return state, nil
	} else if err != nil {
		return state, fmt.Errorf("error reading FD directory for pid %d: %w", pid, err)
	}
	state.Open = opt.UintWith(uint64(len(fds)))
	return state, nil
}

// getLinuxBootTime fetches the static unix time for when the system was booted.
func getLinuxBootTime(hostfs resolve.Resolver) (uint64, error) {
	if bootTime != 0 {
		return bootTime, nil
	}

	path := hostfs.Join("proc", "stat")
	// grab system boot time
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("error opening file %s: %w", path, err)
	}

	statVals := strings.Split(string(data), "\n")

	for _, line := range statVals {
		if strings.HasPrefix(line, "btime") {
			btime, err := strconv.ParseUint(line[6:], 10, 64)
			if err != nil {
				return 0, fmt.Errorf("error reading boot time: %w", err)
			}
			bootTime = btime
			return btime, nil
		}
	}

	return 0, fmt.Errorf("no boot time find in file %s: %w", path, err)
}

func getProcStatus(hostfs resolve.Resolver, pid int) (map[string]string, error) {
	status := make(map[string]string, 42)
	path := hostfs.Join("proc", strconv.Itoa(pid), "status")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error opening file %s: %w", path, err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.SplitN(line, ":", 2)
		if len(fields) == 2 {
			status[fields[0]] = strings.TrimSpace(fields[1])
		}
	}

	return status, err
}

func getProcState(b byte) PidState {
	state, ok := PidStates[b]
	if ok {
		return state
	}
	return Unknown
}

func FillMetricsRequiringMoreAccess(_ int, state ProcState) (ProcState, error) {
	return state, nil
}
