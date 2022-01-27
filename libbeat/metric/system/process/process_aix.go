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
	"io"
	"os"
	"os/user"
	"strconv"
	"syscall"
	"unsafe"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/metric/system/resolve"
	"github.com/elastic/beats/v7/libbeat/opt"
)

// FetchPids returns a map and array of pids
func (procStats *Stats) FetchPids() (ProcsMap, []ProcState, error) {

	info := C.struct_procsinfo64{}
	pid := C.pid_t(0)

	procMap := make(ProcsMap, 0)
	var plist []ProcState
	for {
		// getprocs first argument is a void*
		num, err := C.getprocs(unsafe.Pointer(&info), C.sizeof_struct_procsinfo64, nil, 0, &pid, 1)
		if err != nil {
			return nil, nil, errors.Wrap(err, "error fetching PIDs")
		}

		status, saved, err := procStats.pidFill(int(info.pi_pid), true)
		if err != nil {
			procStats.logger.Debugf("Error fetching PID info for %d, skipping: %s", pid, err)
			continue
		}
		if !saved {
			procStats.logger.Debugf("Process name does not matches the provided regex; PID=%d; name=%s", pid, status.Name)
			continue
		}
		procMap[int(info.pi_pid)] = status
		plist = append(plist, status)

		if num == 0 {
			break
		}
	}
	return procMap, plist, nil
}

// GetInfoForPid returns basic info for the process
func GetInfoForPid(_ resolve.Resolver, pid int) (ProcState, error) {
	info := C.struct_procsinfo64{}
	cpid := C.pid_t(pid)

	num, err := C.getprocs(unsafe.Pointer(&info), C.sizeof_struct_procsinfo64, nil, 0, &cpid, 1)
	if err != nil {
		return ProcState{}, errors.Wrap(err, "error in getprocs")
	}
	if num != 1 {
		return ProcState{}, syscall.ESRCH
	}

	state := ProcState{}
	state.Pid = opt.IntWith(pid)

	state.Name = C.GoString(&info.pi_comm[0])
	state.Ppid = opt.IntWith(int(info.pi_ppid))
	state.Pgid = opt.IntWith(int(info.pi_pgrp))

	switch info.pi_state {
	case C.SACTIVE:
		state.State = "running"
	case C.SIDL:
		state.State = "idle"
	case C.SSTOP:
		state.State = "stopped"
	case C.SZOMB:
		state.State = "zombie"
	case C.SSWAP:
		state.State = "sleeping"
	default:
		state.State = "unknown"
	}

	// Get process username. Fallback to UID if username is not available.
	uid := strconv.Itoa(int(info.pi_uid))
	userID, err := user.LookupId(uid)
	if err == nil && userID.Username != "" {
		state.Username = userID.Username
	} else {
		state.Username = uid
	}

	return state, nil
}

// FillPidMetrics is the aix implementation
func FillPidMetrics(_ resolve.Resolver, pid int, state ProcState, filter func(string) bool) (ProcState, error) {
	pagesize := uint64(os.Getpagesize())
	info := C.struct_procsinfo64{}
	cpid := C.pid_t(pid)

	num, err := C.getprocs(unsafe.Pointer(&info), C.sizeof_struct_procsinfo64, nil, 0, &cpid, 1)
	if err != nil {
		return state, errors.Wrap(err, "error in getprocs")
	}
	if num != 1 {
		return state, syscall.ESRCH
	}

	state.Memory.Size = opt.UintWith(uint64(info.pi_size) * pagesize)
	state.Memory.Share = opt.UintWith(uint64(info.pi_sdsize) * pagesize)
	state.Memory.Rss.Bytes = opt.UintWith(uint64(info.pi_drss+info.pi_trss) * pagesize)

	state.CPU.StartTime = unixTimeMsToTime(uint64(info.pi_start) * 1000)
	state.CPU.User.Ticks = opt.UintWith(uint64(info.pi_utime) * 1000)
	state.CPU.System.Ticks = opt.UintWith(uint64(info.pi_stime) * 1000)
	state.CPU.Total.Ticks = opt.UintWith(opt.SumOptUint(state.CPU.User.Ticks, state.CPU.System.Ticks))

	// Get Proc Args
	/* If buffer is not large enough, args are truncated */
	buf := make([]byte, 8192)
	info.pi_pid = C.pid_t(pid)

	if _, err := C.getargs(unsafe.Pointer(&info), C.sizeof_struct_procsinfo64, (*C.char)(&buf[0]), 8192); err != nil {
		return state, errors.Wrap(err, "error in gitargs")
	}

	bbuf := bytes.NewBuffer(buf)
	var args []string

	for {
		arg, err := bbuf.ReadBytes(0)
		if err == io.EOF || arg[0] == 0 {
			break
		}
		if err != nil {
			return state, errors.Wrap(err, "error reading args buffer")
		}

		args = append(args, stripNullByte(arg))
	}
	state.Args = args
	state.Exe = args[0]

	// get env vars
	buf = make([]byte, 8192)

	if _, err := C.getevars(unsafe.Pointer(&info), C.sizeof_struct_procsinfo64, (*C.char)(&buf[0]), 8192); err != nil {
		return state, errors.Wrap(err, "error in getevars")
	}

	if state.Env != nil {
		return state, nil
	}

	bbuf = bytes.NewBuffer(buf)
	delim := []byte{61} // "="
	vars := map[string]string{}
	for {
		line, err := bbuf.ReadBytes(0)
		if err == io.EOF || line[0] == 0 {
			break
		}
		if err != nil {
			return state, errors.Wrap(err, "error")
		}

		pair := bytes.SplitN(stripNullByteRaw(line), delim, 2)
		if len(pair) != 2 {
			return state, errors.Wrap(err, "error reading environment")
		}
		eKey := string(pair[0])
		if filter == nil || filter(eKey) {
			vars[string(pair[0])] = string(pair[1])
		}

	}
	state.Env = vars

	return state, nil
}
