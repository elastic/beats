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
	"encoding/binary"
	"errors"
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

type extractor interface {
	// Extract extracts useful information from the pointed-to structure
	Extract(unsafe.Pointer)
	// Size of the structure
	Size() int
}

type callbackFn func(uint16, int)
type extractorFactory func(fn callbackFn) extractor

type tcpRowOwnerPIDExtractor callbackFn
type tcp6RowOwnerPIDExtractor callbackFn

var tables = []struct {
	family    uint32
	function  GetExtendedTableFn
	class     uint32
	extractor extractorFactory
}{
	{windows.AF_INET, _GetExtendedTcpTable, TCP_TABLE_OWNER_PID_ALL, extractTCPRowOwnerPID},
	{windows.AF_INET6, _GetExtendedTcpTable, TCP_TABLE_OWNER_PID_ALL, extractTCP6RowOwnerPID},
}

// GetLocalPortToPIDMapping returns the list of local port numbers and the PID
// that owns them.
func (proc *ProcessesWatcher) GetLocalPortToPIDMapping() (ports map[uint16]int, err error) {
	storeResults := func(localPort uint16, pid int) {
		ports[localPort] = pid
	}

	ports = make(map[uint16]int)
	for _, table := range tables {
		data, err := getNetTable(table.function, false, table.family, table.class)
		if err != nil {
			return nil, err
		}
		err = parseTable(data, table.extractor(storeResults))
		if err != nil {
			return nil, err
		}
	}
	return ports, nil
}

func getNetTable(fn GetExtendedTableFn, order bool, family uint32, tableClass uint32) ([]byte, error) {
	// Call the winapi function with an increasing buffer until the required
	// size is satisfied
	for size, ptr, addr := uint32(0), []byte(nil), uintptr(0); ; {
		code, err := fn(addr, &size, order, family, tableClass, 0)
		if code == syscall.Errno(0) {
			return ptr, nil
		} else if code == syscall.ERROR_INSUFFICIENT_BUFFER {
			ptr = make([]byte, size)
			addr = uintptr(unsafe.Pointer(&ptr[0]))
		} else {
			return nil, fmt.Errorf("getNetTable failed: code=%v err=%v", code, err)
		}
	}
}

func parseTable(data []byte, extractor extractor) error {
	lim := len(data)
	if lim < sizeOfDWORD {
		return errors.New("data table too small for length")
	}
	rowSize := extractor.Size()
	n := int(*(*uint32)(unsafe.Pointer(&data[0])))
	if lim < n*rowSize+sizeOfDWORD {
		return errors.New("data table too small for its contents")
	}
	for i := 0; i < n; i++ {
		ptr := unsafe.Pointer(&data[sizeOfDWORD+i*rowSize])
		extractor.Extract(ptr)
	}
	return nil
}

// The MIB_TCP_ROW_xxx structures uses a 32-bit field to store ports:
// The first 16 bits contain the port in big-endian encoding
// The last 16 bits are unused.
func uint32FieldToPort(be uint32) uint16 {
	return binary.BigEndian.Uint16((*[2]byte)(unsafe.Pointer(&be))[:])
}

func extractTCPRowOwnerPID(fn callbackFn) extractor {
	return tcpRowOwnerPIDExtractor(fn)
}

func extractTCP6RowOwnerPID(fn callbackFn) extractor {
	return tcp6RowOwnerPIDExtractor(fn)
}

// Extract will parse a row of Size() bytes pointed to by ptr
func (e tcpRowOwnerPIDExtractor) Extract(ptr unsafe.Pointer) {
	row := (*TCPRowOwnerPID)(ptr)
	e(uint32FieldToPort(row.localPort), int(row.owningPID))
}

// Size returns the size of a table row
func (tcpRowOwnerPIDExtractor) Size() int {
	return int(unsafe.Sizeof(TCPRowOwnerPID{}))
}

// Extract will parse a row of Size() bytes pointed to by ptr
func (e tcp6RowOwnerPIDExtractor) Extract(ptr unsafe.Pointer) {
	row := (*TCP6RowOwnerPID)(ptr)
	e(uint32FieldToPort(row.localPort), int(row.owningPID))
}

// Size returns the size of a table row
func (tcp6RowOwnerPIDExtractor) Size() int {
	return int(unsafe.Sizeof(TCP6RowOwnerPID{}))
}
