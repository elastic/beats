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

//go:build openbsd || freebsd
// +build openbsd freebsd

package numcpu

/*
#include <sys/param.h>
#include <sys/types.h>
#include <sys/sysctl.h>
#include <stdlib.h>
#include <unistd.h>
*/
import "C"

import (
	"syscall"
	"unsafe"
)

// getCPU implements NumCPU on openbsd
// This is just using the HW_NCPU sysctl value.
func getCPU() (int, bool, error) {

	// Get count of available CPUs
	ncpuMIB := [2]int32{C.CTL_HW, C.HW_NCPU}
	callSize := uintptr(0)
	var ncpu int
	// Get size of return value.
	_, _, errno := syscall.Syscall6(syscall.SYS___SYSCTL, uintptr(unsafe.Pointer(&ncpuMIB[0])), 2, 0, uintptr(unsafe.Pointer(&callSize)), 0, 0)

	if errno != 0 || callSize == 0 {
		return -1, false, errno
	}

	// Get CPU count
	_, _, errno = syscall.Syscall6(syscall.SYS___SYSCTL, uintptr(unsafe.Pointer(&ncpuMIB[0])), 2, uintptr(unsafe.Pointer(&ncpu)), uintptr(unsafe.Pointer(&callSize)), 0, 0)

	return ncpu, true, nil
}
