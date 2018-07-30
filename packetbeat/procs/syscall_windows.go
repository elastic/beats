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

// +build windows

package procs

import (
	"syscall"
)

const (
	UDP_TABLE_OWNER_PID     = 1
	TCP_TABLE_OWNER_PID_ALL = 5

	sizeOfDWORD           = 4
	sizeOfTCPRowOwnerPID  = 24
	sizeOfTCP6RowOwnerPID = 56
)

type TCPRowOwnerPID struct {
	state      uint32
	localAddr  uint32
	localPort  uint32
	remoteAddr uint32
	remotePort uint32
	owningPID  uint32
}

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

type UDPRowOwnerPID struct {
	localAddr uint32
	localPort uint32
	owningPID uint32
}

type UDP6RowOwnerPID struct {
	localAddr    [16]byte
	localScopeID uint32
	localPort    uint32
	owningPID    uint32
}

// GetExtendedTableFn is the prototype for GetExtendedTcpTable and GetExtendedUdpTable
type GetExtendedTableFn func(pTcpTable uintptr, pdwSize *uint32, bOrder bool, ulAf uint32, tableClass uint32, reserved uint32) (code syscall.Errno, err error)

// Add -trace to enable debug prints around syscalls.
//go:generate go run $GOROOT/src/syscall/mksyscall_windows.go -output zsyscall_windows.go syscall_windows.go

// Windows API calls
//sys _GetExtendedTcpTable(pTcpTable uintptr, pdwSize *uint32, bOrder bool, ulAf uint32, tableClass uint32, reserved uint32) (code syscall.Errno, err error) = iphlpapi.GetExtendedTcpTable
//sys _GetExtendedUdpTable(pTcpTable uintptr, pdwSize *uint32, bOrder bool, ulAf uint32, tableClass uint32, reserved uint32) (code syscall.Errno, err error) = iphlpapi.GetExtendedUdpTable
