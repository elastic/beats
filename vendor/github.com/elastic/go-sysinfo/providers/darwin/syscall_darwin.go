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

// +build amd64,cgo arm64,cgo

package darwin

/*
#cgo LDFLAGS:-lproc
#include <sys/sysctl.h>
#include <mach/mach_time.h>
#include <mach/mach_host.h>
#include <unistd.h>
*/
import "C"

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sync"
	"syscall"
	"unsafe"

	"github.com/pkg/errors"
)

// Single-word zero for use when we need a valid pointer to 0 bytes.
// See mksyscall.pl.
var _zero uintptr

// Buffer Pool

var bufferPool = sync.Pool{
	New: func() interface{} {
		return &poolMem{
			buf: make([]byte, argMax),
		}
	},
}

type poolMem struct {
	buf  []byte
	pool *sync.Pool
}

func getPoolMem() *poolMem {
	pm := bufferPool.Get().(*poolMem)
	pm.buf = pm.buf[0:cap(pm.buf)]
	pm.pool = &bufferPool
	return pm
}

func (m *poolMem) Release() { m.pool.Put(m) }

// Common errors.

// Do the interface allocations only once for common
// Errno values.
var (
	errEAGAIN error = syscall.EAGAIN
	errEINVAL error = syscall.EINVAL
	errENOENT error = syscall.ENOENT
)

// errnoErr returns common boxed Errno values, to prevent
// allocations at runtime.
func errnoErr(e syscall.Errno) error {
	switch e {
	case 0:
		return nil
	case syscall.EAGAIN:
		return errEAGAIN
	case syscall.EINVAL:
		return errEINVAL
	case syscall.ENOENT:
		return errENOENT
	}
	return e
}

func _sysctl(mib []C.int, old *byte, oldlen *uintptr, new *byte, newlen uintptr) (err error) {
	var _p0 unsafe.Pointer
	if len(mib) > 0 {
		_p0 = unsafe.Pointer(&mib[0])
	} else {
		_p0 = unsafe.Pointer(&_zero)
	}
	_, _, e1 := syscall.Syscall6(syscall.SYS___SYSCTL, uintptr(_p0), uintptr(len(mib)), uintptr(unsafe.Pointer(old)), uintptr(unsafe.Pointer(oldlen)), uintptr(unsafe.Pointer(new)), uintptr(newlen))
	if e1 != 0 {
		err = errnoErr(e1)
	}
	return
}

// Translate "kern.hostname" to []_C_int{0,1,2,3}.
func nametomib(name string) (mib []C.int, err error) {
	const siz = unsafe.Sizeof(mib[0])

	// NOTE(rsc): It seems strange to set the buffer to have
	// size CTL_MAXNAME+2 but use only CTL_MAXNAME
	// as the size. I don't know why the +2 is here, but the
	// kernel uses +2 for its own implementation of this function.
	// I am scared that if we don't include the +2 here, the kernel
	// will silently write 2 words farther than we specify
	// and we'll get memory corruption.
	var buf [C.CTL_MAXNAME + 2]C.int
	n := uintptr(C.CTL_MAXNAME) * siz

	p := (*byte)(unsafe.Pointer(&buf[0]))
	bytes, err := syscall.ByteSliceFromString(name)
	if err != nil {
		return nil, err
	}

	// Magic sysctl: "setting" 0.3 to a string name
	// lets you read back the array of integers form.
	if err = _sysctl([]C.int{0, 3}, p, &n, &bytes[0], uintptr(len(name))); err != nil {
		return nil, err
	}
	return buf[0 : n/siz], nil
}

func sysctl(mib []C.int, value interface{}) error {
	mem := getPoolMem()
	defer mem.Release()

	size := uintptr(len(mem.buf))
	if err := _sysctl(mib, &mem.buf[0], &size, nil, 0); err != nil {
		return err
	}
	data := mem.buf[0:size]

	switch v := value.(type) {
	case *[]byte:
		out := make([]byte, len(data))
		copy(out, data)
		*v = out
		return nil
	default:
		return binary.Read(bytes.NewReader(data), binary.LittleEndian, v)
	}
}

func sysctlByName(name string, out interface{}) error {
	mib, err := nametomib(name)
	if err != nil {
		return err
	}

	return sysctl(mib, out)
}

type cpuUsage struct {
	User   uint32
	System uint32
	Idle   uint32
	Nice   uint32
}

func getHostCPULoadInfo() (*cpuUsage, error) {
	var count C.mach_msg_type_number_t = C.HOST_CPU_LOAD_INFO_COUNT
	var cpu cpuUsage
	status := C.host_statistics(C.host_t(C.mach_host_self()),
		C.HOST_CPU_LOAD_INFO,
		C.host_info_t(unsafe.Pointer(&cpu)),
		&count)

	if status != C.KERN_SUCCESS {
		return nil, errors.Errorf("host_statistics returned status %d", status)
	}

	return &cpu, nil
}

// getClockTicks returns the number of click ticks in one jiffie.
func getClockTicks() int {
	return int(C.sysconf(C._SC_CLK_TCK))
}

func getHostVMInfo64() (*vmStatistics64Data, error) {
	var count C.mach_msg_type_number_t = C.HOST_VM_INFO64_COUNT

	var vmStat vmStatistics64Data
	status := C.host_statistics64(
		C.host_t(C.mach_host_self()),
		C.HOST_VM_INFO64,
		C.host_info_t(unsafe.Pointer(&vmStat)),
		&count)

	if status != C.KERN_SUCCESS {
		return nil, fmt.Errorf("host_statistics64 returned status %d", status)
	}

	return &vmStat, nil
}

func getPageSize() (uint64, error) {
	var pageSize vmSize
	status := C.host_page_size(
		C.host_t(C.mach_host_self()),
		(*C.vm_size_t)(unsafe.Pointer(&pageSize)))
	if status != C.KERN_SUCCESS {
		return 0, errors.Errorf("host_page_size returned status %d", status)
	}

	return uint64(pageSize), nil
}

// From sysctl.h - xsw_usage.
type swapUsage struct {
	Total     uint64
	Available uint64
	Used      uint64
	PageSize  uint64
}

const vmSwapUsageMIB = "vm.swapusage"

func getSwapUsage() (*swapUsage, error) {
	var swap swapUsage
	if err := sysctlByName(vmSwapUsageMIB, &swap); err != nil {
		return nil, err
	}
	return &swap, nil
}
