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
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/opt"
	"github.com/elastic/gosigar"
	"github.com/pkg/errors"
)

// Indulging in one non-const global variable for the sake of storing boot time
// This value obviously won't change while this code is running.
var bootTime uint64 = 0

// system tick multiplier, see C.sysconf(C._SC_CLK_TCK)
const ticks = 100

// GetSelfPid returns the PID for this process
func GetSelfPid() (int, error) {
	pid, err := os.Readlink(path.Join(gosigar.Procd, "self"))

	if err != nil {
		return 0, err
	}

	return strconv.Atoi(pid)
}

// FetchPids is the linux implementation of FetchPids
func FetchPids(hostfs string, filter func(name string) bool) ([]ProcState, error) {
	dir, err := os.Open(hostfs)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading from procfs %s", hostfs)
	}
	defer dir.Close()

	const readAllDirnames = -1 // see os.File.Readdirnames doc

	names, err := dir.Readdirnames(readAllDirnames)
	if err != nil {
		return nil, errors.Wrap(err, "error reading directory names")
	}

	list := make([]ProcState, 0)

	// Iterate over the directory, fetch just enough info so we can filter based on user input.
	logger := logp.L()
	for _, name := range names {

		if !dirIsPid(name) {
			continue
		}
		// Will this actually fail?
		pid, err := strconv.Atoi(name)
		if err != nil {
			logger.Debugf("Error converting PID name %s", name)
			continue
		}
		// Fetch proc state so we can get the name for filtering based on user's filter.
		status, err := GetInfoForPid(hostfs, pid)
		if err != nil {
			logger.Debugf("Skipping over PID=%d, due to: %d", pid, err)
			continue
		}
		// Filter based on user-supplied func
		if !filter(status.Name) {
			logger.Debugf("Process name does not matches the provided regex; PID=%d; name=%s", pid, status.Name)
			continue
		}

		//If we've passed the filter, continue to fill out the rest of the metrics
		status, err = FillPidMetrics(hostfs, pid, status)
		if err != nil {
			logger.Debugf("Skipping over PID=%d, due to: %d", pid, err)
			continue
		}
		list = append(list, status)
	}

	return list, nil
}

func FillPidMetrics(hostfs string, pid int, state ProcState) (ProcState, error) {
	// Memory Data
	var err error
	state.Memory, err = getMemData(hostfs, pid)
	if err != nil {
		return state, errors.Wrapf(err, "error getting memory data for pid %d", pid)
	}

	// CPU Data
	state.CPU, err = getCPUTime(hostfs, pid)
	if err != nil {
		return state, errors.Wrapf(err, "error getting CPU data for pid %d", pid)
	}

	// CLI args
	state.Args, err = getArgs(hostfs, pid)
	if err != nil {
		return state, errors.Wrapf(err, "error getting CLI args for pid %d", pid)
	}

	return state, nil
}

// GetInfoForPid fetches the basic hostinfo from /proc/[PID]/stat
func GetInfoForPid(hostfs string, pid int) (ProcState, error) {
	path := filepath.Join(hostfs, strconv.Itoa(pid), "stat")
	data, err := ioutil.ReadFile(path)
	// Transform the error into a more sensible error in cases where the directory doesn't exist, i.e the process is gone
	if err != nil {
		if os.IsNotExist(err) {
			return ProcState{}, syscall.ESRCH
		} else {
			return ProcState{}, errors.Wrapf(err, "error reading procdir %s", path)
		}
	}

	state := ProcState{}

	// Extract the comm value with is surrounded by parentheses.
	lIdx := bytes.Index(data, []byte("("))
	rIdx := bytes.LastIndex(data, []byte(")"))
	if lIdx < 0 || rIdx < 0 || lIdx >= rIdx || rIdx+2 >= len(data) {
		return state, fmt.Errorf("failed to extract comm for pid %d from '%v'", pid, string(data))
	}
	state.Name = string(data[lIdx+1 : rIdx])

	// Extract the rest of the fields that we are interested in.
	fields := bytes.Fields(data[rIdx+2:])
	if len(fields) <= 36 {
		return state, fmt.Errorf("expected more stat fields for pid %d from '%v'", pid, string(data))
	}

	interests := bytes.Join([][]byte{
		fields[0], // state
		fields[1], // ppid
		fields[2], // pgrp
	}, []byte(" "))

	var procState string
	var ppid, pgid int

	_, err = fmt.Fscan(bytes.NewBuffer(interests),
		&procState,
		&ppid,
		&pgid,
	)
	if err != nil {
		return state, fmt.Errorf("failed to parse stat fields for pid %d from '%v': %v", pid, string(data), err)
	}
	state.State = getProcState(procState[0])
	state.Ppid = opt.IntWith(ppid)
	state.Pgid = opt.IntWith(pgid)

	return state, nil
}

func dirIsPid(name string) bool {
	if name[0] < '0' || name[0] > '9' {
		return false
	}
	return true
}

func getMemData(hostfs string, pid int) (ProcMemInfo, error) {
	// Memory data
	state := ProcMemInfo{}
	path := filepath.Join(hostfs, strconv.Itoa(pid), "statm")
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return state, errors.Wrapf(err, "error opening file %s", path)
	}

	fields := strings.Fields(string(data))

	size, err := strconv.ParseUint(fields[0], 10, 64)
	if err != nil {
		return state, errors.Wrapf(err, "error parsing memory size %s", fields[0])
	}
	state.Size = opt.UintWith(size << 12)

	rss, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return state, errors.Wrapf(err, "error parsing memory rss %s", fields[1])
	}
	state.Rss.Bytes = opt.UintWith(rss << 12)

	share, err := strconv.ParseUint(fields[2], 10, 64)
	state.Share = opt.UintWith(share << 12)

	return state, nil
}

func getCPUTime(hostfs string, pid int) (ProcCPUInfo, error) {
	state := ProcCPUInfo{}

	pathCPU := filepath.Join(hostfs, strconv.Itoa(pid), "stat")
	data, err := ioutil.ReadFile(pathCPU)
	if err != nil {
		return state, errors.Wrapf(err, "error opening file %s", pathCPU)
	}

	fields := strings.Fields(string(data))

	user, err := strconv.ParseUint(fields[13], 10, 64)
	if err != nil {
		return state, errors.Wrapf(err, "error parsing user CPU times for pid %d", pid)
	}
	sys, err := strconv.ParseUint(fields[14], 10, 64)
	if err != nil {
		return state, errors.Wrapf(err, "error parsing system CPU times for pid %d", pid)
	}

	btime, err := getLinuxBootTime(hostfs)
	if err != nil {
		return state, errors.Wrapf(err, "error feting boot time for pid %d", pid)
	}

	// convert to millis
	state.User.Ticks = opt.UintWith(user * (1000 / ticks))
	state.System.Ticks = opt.UintWith(sys * (1000 / ticks))
	state.Total.Ticks = opt.UintWith(opt.SumOptUint(state.User.Ticks, state.System.Ticks))

	// convert to millis

	startTime, err := strconv.ParseUint(fields[21], 10, 64)
	if err != nil {
		return state, errors.Wrapf(err, "error parsing start time value %s for pid %d", fields[21], pid)
	}

	startTime /= ticks
	startTime += btime
	startTime *= 1000

	state.StartTime = unixTimeMsToTime(startTime)

	return state, nil
}

func getArgs(hostfs string, pid int) ([]string, error) {
	path := filepath.Join(hostfs, strconv.Itoa(pid), "cmdline")
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "error opening file %s", path)
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

func getFDStats(hostfs string, pid int) (ProcFDInfo, error) {
	state := ProcFDInfo{}

	path := filepath.Join(hostfs, strconv.Itoa(pid), "limits")
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return state, errors.Wrapf(err, "error opening file %s", path)
	}

	statVals := strings.Split(string(data), "\n")

	for _, line := range statVals {
		if strings.HasPrefix(line, "Max open files") {
			fields := strings.Fields(line)
			if len(fields) == 6 {

				softLimit, err := strconv.ParseUint(fields[3], 10, 64)
				if err != nil {
					return state, errors.Wrapf(err, "error parsing limits value %s for pid %d", fields[3], pid)
				}
				state.Limit.Soft = opt.UintWith(softLimit)

				hardLimit, err := strconv.ParseUint(fields[4], 10, 64)
				if err != nil {
					return state, errors.Wrapf(err, "error parsing limits value %s for pid %d", fields[3], pid)
				}
				state.Limit.Hard = opt.UintWith(hardLimit)
			}

		}
	}

	pathFD := filepath.Join(hostfs, strconv.Itoa(pid), "fd")
	fds, err := ioutil.ReadDir(pathFD)
	if err != nil {
		return state, errors.Wrapf(err, "error reading FD directory for pid %d", pid)
	}
	state.Open = opt.UintWith(uint64(len(fds)))
	return state, nil
}

// getLinuxBootTime fetches the static unix time for when the system was booted.
func getLinuxBootTime(hostfs string) (uint64, error) {
	if bootTime != 0 {
		return bootTime, nil
	}

	path := filepath.Join(hostfs, "stat")
	// grab system boot time
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return 0, errors.Wrapf(err, "error opening file %s", path)
	}

	statVals := strings.Split(string(data), "\n")

	for _, line := range statVals {
		if strings.HasPrefix(line, "btime") {
			btime, err := strconv.ParseUint(line[6:], 10, 64)
			if err != nil {
				return 0, errors.Wrap(err, "error reading boot time")
			}
			bootTime = btime
			return btime, nil
		}
	}

	return 0, errors.Wrapf(err, "no boot time find in file %s", path)
}
