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

type CapData struct {
	Effective   uint64
	Permitted   uint64
	Inheritable uint64
}

type capData32 struct {
	effective   uint32
	permitted   uint32
	inheritable uint32
}

func uint32to64(a, b uint32) uint64 {
	return uint64(a)<<32 | uint64(b)
}

const (
	capabilityVersion1 = 0x19980330 // Version 1, 32-bit capabilities
	capabilityVersion3 = 0x20080522 // Version 3 (replaced V2), 64-bit capabilities
)

func GetCapabilities() CapData {
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

	var data [2]capData32
	_, _, e = unix.Syscall(unix.SYS_CAPGET, uintptr(unsafe.Pointer(&header)), uintptr(unsafe.Pointer(&data)), 0)
	if e != 0 {
		// If this fails, there are invalid arguments
		panic(unix.ErrnoName(e))
	}

	return CapData{
		Effective:   uint32to64(data[1].effective, data[0].effective),
		Permitted:   uint32to64(data[1].permitted, data[0].permitted),
		Inheritable: uint32to64(data[1].inheritable, data[0].inheritable),
	}
}
