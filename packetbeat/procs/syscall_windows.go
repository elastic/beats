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

//go:build windows

//nolint:structcheck // Struct fields reflect Windows layout.
package procs

// Add -trace to enable debug prints around syscalls.
//go:generate go run $GOROOT/src/syscall/mksyscall_windows.go -output zsyscall_windows.go syscall_windows.go

import (
	"syscall"
	"unsafe"
)

const (
	UDP_TABLE_OWNER_PID     = 1
	TCP_TABLE_OWNER_PID_ALL = 5

	sizeOfDWORD           = int(unsafe.Sizeof(uint32(0)))
	sizeOfTCPRowOwnerPID  = int(unsafe.Sizeof(TCPRowOwnerPID{}))
	sizeOfTCP6RowOwnerPID = int(unsafe.Sizeof(TCP6RowOwnerPID{}))
)

func _() {
	// Make sure the structs in Go have the expected size.

	// An invalid array index indicates that the size of the Go struct does not match
	// the expected size according to the Microsoft documentation.
	var x [1]struct{}
	_ = x[sizeOfDWORD-4]
	_ = x[sizeOfTCPRowOwnerPID-24]
	_ = x[sizeOfTCP6RowOwnerPID-56]
}

// https://docs.microsoft.com/en-us/windows/win32/api/tcpmib/ns-tcpmib-mib_tcprow_owner_pid
type TCPRowOwnerPID struct {
	state      uint32
	localAddr  uint32
	localPort  uint32
	remoteAddr uint32
	remotePort uint32
	owningPID  uint32
}

// https://docs.microsoft.com/en-us/windows/win32/api/tcpmib/ns-tcpmib-mib_tcp6row_owner_pid
type TCP6RowOwnerPID struct {
	localAddr     [16]byte
	localScopeID  uint32
	localPort     uint32
	remoteAddr    [16]byte
	remoteScopeID uint32
	remotePort    uint32
	state         uint32
	owningPID     uint32
}

// https://docs.microsoft.com/en-us/windows/win32/api/udpmib/ns-udpmib-mib_udprow_owner_pid
type UDPRowOwnerPID struct {
	localAddr uint32
	localPort uint32
	owningPID uint32
}

// https://docs.microsoft.com/en-us/windows/win32/api/udpmib/ns-udpmib-mib_udp6row_owner_pid
type UDP6RowOwnerPID struct {
	localAddr    [16]byte
	localScopeID uint32
	localPort    uint32
	owningPID    uint32
}

// GetExtendedTableFn is the prototype for GetExtendedTcpTable and GetExtendedUdpTable
type GetExtendedTableFn func(pTcpTable uintptr, pdwSize *uint32, bOrder bool, ulAf uint32, tableClass uint32, reserved uint32) (code syscall.Errno, err error)

// Windows API calls
//sys _GetExtendedTcpTable(pTcpTable uintptr, pdwSize *uint32, bOrder bool, ulAf uint32, tableClass uint32, reserved uint32) (code syscall.Errno, err error) = iphlpapi.GetExtendedTcpTable
//sys _GetExtendedUdpTable(pTcpTable uintptr, pdwSize *uint32, bOrder bool, ulAf uint32, tableClass uint32, reserved uint32) (code syscall.Errno, err error) = iphlpapi.GetExtendedUdpTable
