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

//go:build darwin && cgo

package process

/*
#include <stdlib.h>
#include <sys/sysctl.h>
#include <sys/mount.h>
#include <mach/mach_init.h>
#include <mach/mach_host.h>
#include <mach/host_info.h>
#include <libproc.h>
#include <mach/processor_info.h>
#include <mach/vm_map.h>
*/
import "C"
import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"strconv"
	"syscall"
	"time"
	"unsafe"

	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/opt"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/resolve"
)

// GetSelfPid is the darwin implementation; see the linux version in
// process_linux_common.go for more context.
func GetSelfPid(hostfs resolve.Resolver) (int, error) {
	return os.Getpid(), nil
}

// FetchPids returns a map and array of pids
func (procStats *Stats) FetchPids() (ProcsMap, []ProcState, error) {
	n := C.proc_listpids(C.PROC_ALL_PIDS, 0, nil, 0)
	if n <= 0 {
		return nil, nil, syscall.EINVAL
	}
	buf := make([]byte, n)
	n = C.proc_listpids(C.PROC_ALL_PIDS, 0, unsafe.Pointer(&buf[0]), n)
	if n <= 0 {
		return nil, nil, syscall.ENOMEM
	}

	var pid int32
	num := int(n) / binary.Size(pid)

	bbuf := bytes.NewBuffer(buf)

	procMap := make(ProcsMap, num)
	plist := make([]ProcState, 0, num)
	var wrappedErr error
	var err error

	for i := 0; i < num; i++ {
		if err := binary.Read(bbuf, binary.LittleEndian, &pid); err != nil {
			procStats.logger.Debugf("Errror reading from PROC_ALL_PIDS buffer: %s", err)
			continue
		}
		if pid == 0 {
			continue
		}
		procMap, plist, err = procStats.pidIter(int(pid), procMap, plist)
		wrappedErr = errors.Join(wrappedErr, err)
	}

	return procMap, plist, toNonFatal(wrappedErr)
}

// GetInfoForPid returns basic info for the process
func GetInfoForPid(_ resolve.Resolver, pid int) (ProcState, error) {
	info := C.struct_proc_taskallinfo{}

	size := C.int(unsafe.Sizeof(info))
	ptr := unsafe.Pointer(&info)

	// For docs, see the link below. Check the `proc_taskallinfo` struct, which
	// is a composition of `proc_bsdinfo` and `proc_taskinfo`.
	// https://opensource.apple.com/source/xnu/xnu-1504.3.12/bsd/sys/proc_info.h.auto.html
	n, err := C.proc_pidinfo(C.int(pid), C.PROC_PIDTASKALLINFO, 0, ptr, size)
	if n != size {
		return ProcState{}, fmt.Errorf("could not read process info for pid %d: proc_pidinfo returned %d, err: %w", pid, int(n), err)
	}

	status := ProcState{}

	status.Name = C.GoString(&info.pbsd.pbi_comm[0])

	switch info.pbsd.pbi_status {
	case C.SIDL:
		status.State = Idle
	case C.SRUN:
		status.State = Running
	case C.SSLEEP:
		status.State = Sleeping
	case C.SSTOP:
		status.State = Stopped
	case C.SZOMB:
		status.State = Zombie
	default:
		status.State = Unknown
	}

	status.Ppid = opt.IntWith(int(info.pbsd.pbi_ppid))
	status.Pid = opt.IntWith(pid)
	status.Pgid = opt.IntWith(int(info.pbsd.pbi_pgid))
	status.NumThreads = opt.IntWith(int(info.ptinfo.pti_threadnum))

	// Get process username. Fallback to UID if username is not available.
	uid := strconv.Itoa(int(info.pbsd.pbi_uid))
	user, err := user.LookupId(uid)
	if err == nil && user.Username != "" {
		status.Username = user.Username
	} else {
		status.Username = uid
	}

	// grab memory info + process time while we have it from struct_proc_taskallinfo
	status.Memory.Size = opt.UintWith(uint64(info.ptinfo.pti_virtual_size))
	status.Memory.Rss.Bytes = opt.UintWith(uint64(info.ptinfo.pti_resident_size))

	status.CPU.User.Ticks = opt.UintWith(uint64(info.ptinfo.pti_total_user) / uint64(time.Millisecond))
	status.CPU.System.Ticks = opt.UintWith(uint64(info.ptinfo.pti_total_system) / uint64(time.Millisecond))
	status.CPU.Total.Ticks = opt.UintWith(opt.SumOptUint(status.CPU.User.Ticks, status.CPU.System.Ticks))
	status.CPU.StartTime = unixTimeMsToTime((uint64(info.pbsd.pbi_start_tvsec) * 1000) + (uint64(info.pbsd.pbi_start_tvusec) / 1000))

	return status, nil
}

// FillPidMetrics is the darwin implementation
func FillPidMetrics(_ resolve.Resolver, pid int, state ProcState, filter func(string) bool) (ProcState, error) {

	args, exe, env, err := getProcArgs(pid, filter)
	if err != nil {
		return state, fmt.Errorf("error fetching string data from process: %w", err)
	}

	state.Args = args
	state.Exe = exe
	if state.Env == nil {
		state.Env = env
	}

	return state, nil
}

func getProcArgs(pid int, filter func(string) bool) ([]string, string, mapstr.M, error) {
	mib := []C.int{C.CTL_KERN, C.KERN_PROCARGS2, C.int(pid)}
	argmax := uintptr(C.ARG_MAX)
	buf := make([]byte, argmax)
	err := sysctl(mib, &buf[0], &argmax, nil, 0)
	if err != nil {
		return nil, "", nil, fmt.Errorf("error in sysctl: %w", err)
	}

	bbuf := bytes.NewBuffer(buf)
	bbuf.Truncate(int(argmax))

	var argc int32                                    // raw buffer
	_ = binary.Read(bbuf, binary.LittleEndian, &argc) // read length

	path, err := bbuf.ReadBytes(0)
	if err != nil {
		return nil, "", nil, fmt.Errorf("error reading the executable name: %w", err)
	}

	exeName := stripNullByte(path)

	// skip trailing nul bytes
	for {
		c, err := bbuf.ReadByte()
		if err != nil {
			return nil, "", nil, fmt.Errorf("error skipping nul values in KERN_PROCARGS2 buffer: %w", err)
		}
		if c != 0 {
			_ = bbuf.UnreadByte()
			break
		}
	}

	// read CLI args
	argv := make([]string, 0, argc)
	for i := 0; i < int(argc); i++ {
		arg, err := bbuf.ReadBytes(0)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, exeName, nil, fmt.Errorf("error reading args from KERN_PROCARGS2: %w", err)
		}
		argv = append(argv, stripNullByte(arg))
	}

	delim := []byte{61} // "=" for key value pairs

	envVars := mapstr.M{}
	var envErr error
	for {
		line, err := bbuf.ReadBytes(0)
		if err == io.EOF || line[0] == 0 {
			break
		}
		if err != nil {
			return argv, exeName, nil, fmt.Errorf("error reading args from KERN_PROCARGS2 buffer: %w", err)
		}
		pair := bytes.SplitN(stripNullByteRaw(line), delim, 2)

		if len(pair) != 2 {
			// invalid k-v pair encountered, return non-fatal error so that we can continue
			err := fmt.Errorf("error reading process information from KERN_PROCARGS2: encountered invalid env pair for pid %d", pid)
			envErr = errors.Join(envErr, NonFatalErr{Err: err})
			continue
		}
		eKey := string(pair[0])
		if filter == nil || filter(eKey) {
			envVars[string(pair[0])] = string(pair[1])
		}

	}

	return argv, exeName, envVars, envErr
}

func sysctl(mib []C.int, old *byte, oldlen *uintptr,
	new *byte, newlen uintptr) (err error) {
	p0 := unsafe.Pointer(&mib[0])
	_, _, e1 := syscall.Syscall6(syscall.SYS___SYSCTL, uintptr(p0),
		uintptr(len(mib)),
		uintptr(unsafe.Pointer(old)), uintptr(unsafe.Pointer(oldlen)),
		uintptr(unsafe.Pointer(new)), newlen)
	if e1 != 0 {
		err = e1
	}
	return err
}

func FillMetricsRequiringMoreAccess(_ int, state ProcState) (ProcState, error) {
	return state, nil
}
