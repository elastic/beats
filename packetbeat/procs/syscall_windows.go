// +build windows

package procs

import (
	"syscall"
)

const (
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

// GetExtendedTableFn is the prototype for GetExtendedTcpTable and GetExtendedUdpTable
type GetExtendedTableFn func(pTcpTable uintptr, pdwSize *uint32, bOrder bool, ulAf uint32, tableClass uint32, reserved uint32) (code syscall.Errno, err error)

// Add -trace to enable debug prints around syscalls.
//go:generate go run $GOROOT/src/syscall/mksyscall_windows.go -output zsyscall_windows.go syscall_windows.go

// Windows API calls
//sys _GetExtendedTcpTable(pTcpTable uintptr, pdwSize *uint32, bOrder bool, ulAf uint32, tableClass uint32, reserved uint32) (code syscall.Errno, err error) = iphlpapi.GetExtendedTcpTable
