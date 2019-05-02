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

// +build linux

package common

import (
	"unsafe"

	"golang.org/x/sys/unix"
)

// CapabilitiesData contains the capability sets of a process
type CapabilitiesData struct {
	// Effective is the capability set used for permission checks
	Effective uint64

	// Permitted is the superset of effective capabilities that the thread may assume
	Permitted uint64

	// Inheritable is the set of capabilities inherited to child processes
	Inheritable uint64
}

// Check performs a permission check for a given capabilities set
func (d CapabilitiesData) Check(set uint64) bool {
	return (d.Effective & set) > 0
}

type capData32 [2]struct {
	effective   uint32
	permitted   uint32
	inheritable uint32
}

func (d capData32) to64() CapabilitiesData {
	return CapabilitiesData{
		Effective:   uint32to64(d[1].effective, d[0].effective),
		Permitted:   uint32to64(d[1].permitted, d[0].permitted),
		Inheritable: uint32to64(d[1].inheritable, d[0].inheritable),
	}
}

func uint32to64(a, b uint32) uint64 {
	return uint64(a)<<32 | uint64(b)
}

const (
	capabilityVersion1 = 0x19980330 // Version 1, 32-bit capabilities
	capabilityVersion3 = 0x20080522 // Version 3, 64-bit capabilities (replaced version 2)
)

// GetCapabilities gets the capabilities of this process using system calls to avoid
// depending on procfs or library functions for permission checks
func GetCapabilities() CapabilitiesData {
	header := struct {
		version uint32
		pid     int32
	}{
		version: capabilityVersion3,
		pid:     0, // Self
	}

	// Check compatibility with version 3
	_, _, e := unix.Syscall(unix.SYS_CAPGET, uintptr(unsafe.Pointer(&header)), 0, 0)
	if e != 0 {
		header.version = capabilityVersion1
	}

	var data capData32
	_, _, e = unix.Syscall(unix.SYS_CAPGET, uintptr(unsafe.Pointer(&header)), uintptr(unsafe.Pointer(&data)), 0)
	if e != 0 {
		// If this fails, there are invalid arguments, and all arguments are
		// being created here.
		panic(unix.ErrnoName(e))
	}

	return data.to64()
}
