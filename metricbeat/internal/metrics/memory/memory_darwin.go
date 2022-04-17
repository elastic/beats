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

package memory

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
	"syscall"
	"unsafe"

	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/libbeat/metric/system/resolve"
	"github.com/menderesk/beats/v7/libbeat/opt"
)

type xswUsage struct {
	Total, Avail, Used uint64
}

// get is the darwin implementation for fetching Memory data
func get(_ resolve.Resolver) (Memory, error) {
	var vmstat C.vm_statistics_data_t

	mem := Memory{}

	var total uint64

	if err := sysctlbyname("hw.memsize", &total); err != nil {
		return Memory{}, errors.Wrap(err, "error getting memsize")
	}
	mem.Total = opt.UintWith(total)

	if err := vmInfo(&vmstat); err != nil {
		return Memory{}, errors.Wrap(err, "error getting VM info")
	}

	kern := uint64(vmstat.inactive_count) << 12
	free := uint64(vmstat.free_count) << 12

	mem.Free = opt.UintWith(free)
	mem.Used.Bytes = opt.UintWith(total - free)

	mem.Actual.Free = opt.UintWith(free + kern)
	mem.Actual.Used.Bytes = opt.UintWith((total - free) - kern)

	var err error
	mem.Swap, err = getSwap()
	if err != nil {
		return mem, errors.Wrap(err, "error getting swap memory")
	}

	return mem, nil
}

// Get fetches swap data
func getSwap() (SwapMetrics, error) {
	swUsage := xswUsage{}

	swap := SwapMetrics{}
	if err := sysctlbyname("vm.swapusage", &swUsage); err != nil {
		return swap, errors.Wrap(err, "error getting swap usage")
	}

	swap.Total = opt.UintWith(swUsage.Total)
	swap.Used.Bytes = opt.UintWith(swUsage.Used)
	swap.Free = opt.UintWith(swUsage.Avail)

	return swap, nil
}

// generic Sysctl buffer unmarshalling
func sysctlbyname(name string, data interface{}) (err error) {
	val, err := syscall.Sysctl(name)
	if err != nil {
		return err
	}

	buf := []byte(val)

	switch v := data.(type) {
	case *uint64:
		*v = *(*uint64)(unsafe.Pointer(&buf[0]))
		return
	}

	bbuf := bytes.NewBuffer([]byte(val))
	return binary.Read(bbuf, binary.LittleEndian, data)
}

func vmInfo(vmstat *C.vm_statistics_data_t) error {
	var count C.mach_msg_type_number_t = C.HOST_VM_INFO_COUNT

	status := C.host_statistics(
		C.host_t(C.mach_host_self()),
		C.HOST_VM_INFO,
		C.host_info_t(unsafe.Pointer(vmstat)),
		&count)

	if status != C.KERN_SUCCESS {
		return fmt.Errorf("host_statistics=%d", status)
	}

	return nil
}
