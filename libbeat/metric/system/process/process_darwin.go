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
	"fmt"
	"os/user"
	"strconv"
	"syscall"
	"time"
	"unsafe"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/metric/system/resolve"
	"github.com/elastic/beats/v7/libbeat/opt"
)

func FetchPid(hostfs resolve.Resolver, pid int, postprocess func(state ProcState) (ProcState, error)) (ProcState, error) {

	return ProcState{}, nil
}

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

	procMap := make(ProcsMap, 0)
	var plist []ProcState

	for i := 0; i < num; i++ {
		if err := binary.Read(bbuf, binary.LittleEndian, &pid); err != nil {
			procStats.logger.Debugf("Errror reading from PROC_ALL_PIDS buffer: %s", err)
			continue
		}
		if pid == 0 {
			continue
		}
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

	return procMap, nil
}

func GetInfoForPid(_ resolve.Resolver, pid int) (ProcState, error) {
	info := C.struct_proc_taskallinfo{}

	err := taskInfo(pid, &info)
	if err != nil {
		return ProcState{}, fmt.Errorf("Could not read task for pid %d", pid)
	}

	status := ProcState{}

	status.Name = C.GoString(&info.pbsd.pbi_comm[0])

	switch info.pbsd.pbi_status {
	case C.SIDL:
		status.State = "idle"
	case C.SRUN:
		status.State = "running"
	case C.SSLEEP:
		status.State = "sleeping"
	case C.SSTOP:
		status.State = "stopped"
	case C.SZOMB:
		status.State = "zombie"
	default:
		status.State = "unknown"
	}

	status.Ppid = opt.IntWith(int(info.pbsd.pbi_ppid))

	status.Pgid = opt.IntWith(int(info.pbsd.pbi_pgid))

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

	status.CPU.StartTime = unixTimeMsToTime((uint64(info.pbsd.pbi_start_tvsec) * 1000) + (uint64(info.pbsd.pbi_start_tvusec) / 1000)).String()

	return status, nil
}

func FillPidMetrics(_ resolve.Resolver, pid int, state ProcState) (ProcState, error) {

	return state, nil
}

func getArgs(_ int) ([]string, error) {

	return nil, nil
}

func taskInfo(pid int, info *C.struct_proc_taskallinfo) error {
	size := C.int(unsafe.Sizeof(*info))
	ptr := unsafe.Pointer(info)

	n := C.proc_pidinfo(C.int(pid), C.PROC_PIDTASKALLINFO, 0, ptr, size)
	if n != size {
		return fmt.Errorf("Could not read process info for pid %d", pid)
	}

	return nil
}
