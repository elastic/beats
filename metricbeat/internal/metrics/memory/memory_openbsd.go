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
#include <sys/param.h>
#include <sys/types.h>
#include <sys/sysctl.h>
#include <sys/mount.h>
#include <sys/sched.h>
#include <sys/swap.h>
#include <stdlib.h>
#include <unistd.h>
*/
import "C"

import (
	"syscall"
	"unsafe"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/metric/system/resolve"
	"github.com/elastic/beats/v7/libbeat/opt"
)

// Uvmexp wraps memory data from sysctl
type Uvmexp struct {
	pagesize           uint32
	pagemask           uint32
	pageshift          uint32
	npages             uint32
	free               uint32
	active             uint32
	inactive           uint32
	paging             uint32
	wired              uint32
	zeropages          uint32
	reserve_pagedaemon uint32
	reserve_kernel     uint32
	anonpages          uint32
	vnodepages         uint32
	vtextpages         uint32
	freemin            uint32
	freetarg           uint32
	inactarg           uint32
	wiredmax           uint32
	anonmin            uint32
	vtextmin           uint32
	vnodemin           uint32
	anonminpct         uint32
	vtextmi            uint32
	npct               uint32
	vnodeminpct        uint32
	nswapdev           uint32
	swpages            uint32
	swpginuse          uint32
	swpgonly           uint32
	nswget             uint32
	nanon              uint32
	nanonneeded        uint32
	nfreeanon          uint32
	faults             uint32
	traps              uint32
	intrs              uint32
	swtch              uint32
	softs              uint32
	syscalls           uint32
	pageins            uint32
	obsolete_swapins   uint32
	obsolete_swapouts  uint32
	pgswapin           uint32
	pgswapout          uint32
	forks              uint32
	forks_ppwait       uint32
	forks_sharevm      uint32
	pga_zerohit        uint32
	pga_zeromiss       uint32
	zeroaborts         uint32
	fltnoram           uint32
	fltnoanon          uint32
	fltpgwait          uint32
	fltpgrele          uint32
	fltrelck           uint32
	fltrelckok         uint32
	fltanget           uint32
	fltanretry         uint32
	fltamcopy          uint32
	fltnamap           uint32
	fltnomap           uint32
	fltlget            uint32
	fltget             uint32
	flt_anon           uint32
	flt_acow           uint32
	flt_obj            uint32
	flt_prcopy         uint32
	flt_przero         uint32
	pdwoke             uint32
	pdrevs             uint32
	pdswout            uint32
	pdfreed            uint32
	pdscans            uint32
	pdanscan           uint32
	pdobscan           uint32
	pdreact            uint32
	pdbusy             uint32
	pdpageouts         uint32
	pdpending          uint32
	pddeact            uint32
	pdreanon           uint32
	pdrevnode          uint32
	pdrevtext          uint32
	fpswtch            uint32
	kmapent            uint32
}

// Bcachestats reports cache stats from sysctl
type Bcachestats struct {
	numbufs        uint64
	numbufpages    uint64
	numdirtypages  uint64
	numcleanpages  uint64
	pendingwrites  uint64
	pendingreads   uint64
	numwrites      uint64
	numreads       uint64
	cachehits      uint64
	busymapped     uint64
	dmapages       uint64
	highpages      uint64
	delwribufs     uint64
	kvaslots       uint64
	kvaslots_avail uint64
}

// Swapent reports swap metrics from sysctl
type Swapent struct {
	se_dev      C.dev_t
	se_flags    int32
	se_nblks    int32
	se_inuse    int32
	se_priority int32
	sw_path     []byte
}

func get(_ resolve.Resolver) (Memory, error) {

	memData := Memory{}

	n := uintptr(0)
	var uvmexp Uvmexp
	mib := [2]int32{C.CTL_VM, C.VM_UVMEXP}
	n = uintptr(0)
	// First we determine how much memory we'll need to pass later on (via `n`)
	_, _, errno := syscall.Syscall6(syscall.SYS___SYSCTL, uintptr(unsafe.Pointer(&mib[0])), 2, 0, uintptr(unsafe.Pointer(&n)), 0, 0)
	if errno != 0 || n == 0 {
		return memData, errors.Errorf("Error in size VM_UVMEXP sysctl call, errno %d", errno)
	}

	_, _, errno = syscall.Syscall6(syscall.SYS___SYSCTL, uintptr(unsafe.Pointer(&mib[0])), 2, uintptr(unsafe.Pointer(&uvmexp)), uintptr(unsafe.Pointer(&n)), 0, 0)
	if errno != 0 || n == 0 {
		return memData, errors.Errorf("Error in VM_UVMEXP sysctl call, errno %d", errno)
	}

	var bcachestats Bcachestats
	mib3 := [3]int32{C.CTL_VFS, C.VFS_GENERIC, C.VFS_BCACHESTAT}
	n = uintptr(0)
	_, _, errno = syscall.Syscall6(syscall.SYS___SYSCTL, uintptr(unsafe.Pointer(&mib3[0])), 3, 0, uintptr(unsafe.Pointer(&n)), 0, 0)
	if errno != 0 || n == 0 {
		return memData, errors.Errorf("Error in size VFS_BCACHESTAT sysctl call, errno %d", errno)
	}
	_, _, errno = syscall.Syscall6(syscall.SYS___SYSCTL, uintptr(unsafe.Pointer(&mib3[0])), 3, uintptr(unsafe.Pointer(&bcachestats)), uintptr(unsafe.Pointer(&n)), 0, 0)
	if errno != 0 || n == 0 {
		return memData, errors.Errorf("Error in VFS_BCACHESTAT sysctl call, errno %d", errno)
	}

	memFree := uint64(uvmexp.free) << uvmexp.pageshift
	memUsed := uint64(uvmexp.npages-uvmexp.free) << uvmexp.pageshift

	memData.Total = opt.UintWith(uint64(uvmexp.npages) << uvmexp.pageshift)
	memData.Used.Bytes = opt.UintWith(memUsed)
	memData.Free = opt.UintWith(memFree)

	memData.Actual.Free = opt.UintWith(memFree + (uint64(bcachestats.numbufpages) << uvmexp.pageshift))
	memData.Actual.Used.Bytes = opt.UintWith(memUsed - (uint64(bcachestats.numbufpages) << uvmexp.pageshift))

	var err error
	memData.Swap, err = getSwap()
	if err != nil {
		return memData, errors.Wrap(err, "error getting swap data")
	}

	return memData, nil
}

func getSwap() (SwapMetrics, error) {
	swapData := SwapMetrics{}
	nswap := C.swapctl(C.SWAP_NSWAP, unsafe.Pointer(uintptr(0)), 0)

	// If there are no swap devices, nothing to do here.
	if nswap == 0 {
		return swapData, nil
	}

	swdev := make([]Swapent, nswap)

	rnswap := C.swapctl(C.SWAP_STATS, unsafe.Pointer(&swdev[0]), nswap)
	if rnswap == 0 {
		return swapData, errors.Errorf("error in SWAP_STATS sysctl, swapctl returned %d", rnswap)
	}

	for i := 0; i < int(nswap); i++ {
		if swdev[i].se_flags&C.SWF_ENABLE == 2 {
			swapData.Used.Bytes = opt.UintWith(swapData.Used.Bytes.ValueOr(0) + uint64(swdev[i].se_inuse/(1024/C.DEV_BSIZE)))
			swapData.Total = opt.UintWith(swapData.Total.ValueOr(0) + uint64(swdev[i].se_nblks/(1024/C.DEV_BSIZE)))
		}
	}

	swapData.Free = opt.UintWith(swapData.Total.ValueOr(0) - swapData.Used.Bytes.ValueOr(0))

	return swapData, nil
}
