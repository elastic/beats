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

package pdh

import (
	"strconv"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Windows API calls
//sys _PdhOpenQuery(dataSource *uint16, userData uintptr, query *PdhQueryHandle) (errcode error) [failretval!=0] = pdh.PdhOpenQueryW
//sys _PdhAddEnglishCounter(query PdhQueryHandle, counterPath string, userData uintptr, counter *PdhCounterHandle) (errcode error) [failretval!=0] = pdh.PdhAddEnglishCounterW
//sys _PdhAddCounter(query PdhQueryHandle, counterPath string, userData uintptr, counter *PdhCounterHandle) (errcode error) [failretval!=0] = pdh.PdhAddCounterW
//sys _PdhRemoveCounter(counter PdhCounterHandle) (errcode error) [failretval!=0] = pdh.PdhRemoveCounter
//sys _PdhCollectQueryData(query PdhQueryHandle) (errcode error) [failretval!=0] = pdh.PdhCollectQueryData
//sys _PdhGetFormattedCounterValueDouble(counter PdhCounterHandle, format PdhCounterFormat, counterType *uint32, value *PdhCounterValueDouble) (errcode error) [failretval!=0] = pdh.PdhGetFormattedCounterValue
//sys _PdhGetFormattedCounterValueLarge(counter PdhCounterHandle, format PdhCounterFormat, counterType *uint32, value *PdhCounterValueLarge) (errcode error) [failretval!=0] = pdh.PdhGetFormattedCounterValue
//sys _PdhGetFormattedCounterValueLong(counter PdhCounterHandle, format PdhCounterFormat, counterType *uint32, value *PdhCounterValueLong) (errcode error) [failretval!=0]= pdh.PdhGetFormattedCounterValue
//sys _PdhCloseQuery(query PdhQueryHandle) (errcode error) [failretval!=0] = pdh.PdhCloseQuery
//sys _PdhExpandWildCardPath(dataSource *uint16, wildcardPath *uint16, expandedPathList *uint16, pathListLength *uint32) (errcode error) [failretval!=0] = pdh.PdhExpandWildCardPathW
//sys _PdhExpandCounterPath(wildcardPath *uint16, expandedPathList *uint16, pathListLength *uint32) (errcode error) [failretval!=0] = pdh.PdhExpandCounterPathW
//sys _PdhGetCounterInfo(counter PdhCounterHandle, text uint16, size *uint32, lpBuffer *byte) (errcode error) [failretval!=0] = pdh.PdhGetCounterInfoW
//sys _PdhEnumObjectItems(dataSource uint16, machineName uint16, objectName *uint16, counterList *uint16, counterListSize *uint32, instanceList *uint16, instanceListSize *uint32, detailLevel uint32, flags uint32) (errcode error) [failretval!=0] = pdh.PdhEnumObjectItemsW

type PdhQueryHandle uintptr

var InvalidQueryHandle = ^PdhQueryHandle(0)

type PdhCounterHandle uintptr

var InvalidCounterHandle = ^PdhCounterHandle(0)

// PerformanceDetailWizard is the counter detail level
const PerformanceDetailWizard = 400

// PdhCounterInfo struct contains the performance counter details
type PdhCounterInfo struct {
	DwLength         uint32
	DwType           uint32
	CVersion         uint32
	CStatus          uint32
	LScale           int32
	LDefaultScale    int32
	DwUserData       *uint32
	DwQueryUserData  *uint32
	SzFullPath       *uint16 // pointer to a string
	SzMachineName    *uint16 // pointer to a string
	SzObjectName     *uint16 // pointer to a string
	SzInstanceName   *uint16 // pointer to a string
	SzParentInstance *uint16 // pointer to a string
	DwInstanceIndex  uint32  // pointer to a string
	SzCounterName    *uint16 // pointer to a string
	Padding          [4]byte
	SzExplainText    *uint16   // pointer to a string
	DataBuffer       [1]uint32 // pointer to an extra space
}

// PdhCounterValueDouble  for double values
type PdhCounterValueDouble struct {
	CStatus   uint32
	Pad_cgo_0 [4]byte
	Value     float64
	Pad_cgo_1 [4]byte
}

// PdhCounterValueLarge for 64 bit integer values
type PdhCounterValueLarge struct {
	CStatus   uint32
	Pad_cgo_0 [4]byte
	Value     int64
	Pad_cgo_1 [4]byte
}

// PdhCounterValueLong for long values
type PdhCounterValueLong struct {
	CStatus   uint32
	Pad_cgo_0 [4]byte
	Value     int32
	Pad_cgo_1 [4]byte
}

// PdhOpenQuery creates a new query.
func PdhOpenQuery(dataSource string, userData uintptr) (PdhQueryHandle, error) {
	var dataSourcePtr *uint16
	if dataSource != "" {
		var err error
		dataSourcePtr, err = syscall.UTF16PtrFromString(dataSource)
		if err != nil {
			return InvalidQueryHandle, err
		}
	}

	var handle PdhQueryHandle
	if err := _PdhOpenQuery(dataSourcePtr, userData, &handle); err != nil {
		return InvalidQueryHandle, PdhErrno(err.(syscall.Errno))
	}
	return handle, nil
}

// PdhAddEnglishCounter adds the specified counter to the query.
func PdhAddEnglishCounter(query PdhQueryHandle, counterPath string, userData uintptr) (PdhCounterHandle, error) {
	var handle PdhCounterHandle
	if err := _PdhAddEnglishCounter(query, counterPath, userData, &handle); err != nil {
		return InvalidCounterHandle, PdhErrno(err.(syscall.Errno))
	}

	return handle, nil
}

// PdhAddCounter adds the specified counter to the query.
func PdhAddCounter(query PdhQueryHandle, counterPath string, userData uintptr) (PdhCounterHandle, error) {
	var handle PdhCounterHandle
	if err := _PdhAddCounter(query, counterPath, userData, &handle); err != nil {
		return InvalidCounterHandle, PdhErrno(err.(syscall.Errno))
	}

	return handle, nil
}

// PdhRemoveCounter removes the specified counter to the query.
func PdhRemoveCounter(counter PdhCounterHandle) error {
	if err := _PdhRemoveCounter(counter); err != nil {
		return PdhErrno(err.(syscall.Errno))
	}

	return nil
}

// PdhCollectQueryData collects the current raw data value for all counters in the specified query.
func PdhCollectQueryData(query PdhQueryHandle) error {
	if err := _PdhCollectQueryData(query); err != nil {
		return PdhErrno(err.(syscall.Errno))
	}

	return nil
}

// PdhGetFormattedCounterValueDouble computes a displayable double value for the specified counter.
func PdhGetFormattedCounterValueDouble(counter PdhCounterHandle) (uint32, *PdhCounterValueDouble, error) {
	var counterType uint32
	var value PdhCounterValueDouble
	if err := _PdhGetFormattedCounterValueDouble(counter, PdhFmtDouble|PdhFmtNoCap100, &counterType, &value); err != nil {
		return 0, &value, PdhErrno(err.(syscall.Errno))
	}

	return counterType, &value, nil
}

// PdhGetFormattedCounterValueLarge computes a displayable large value for the specified counter.
func PdhGetFormattedCounterValueLarge(counter PdhCounterHandle) (uint32, *PdhCounterValueLarge, error) {
	var counterType uint32
	var value PdhCounterValueLarge
	if err := _PdhGetFormattedCounterValueLarge(counter, PdhFmtLarge|PdhFmtNoCap100, &counterType, &value); err != nil {
		return 0, &value, PdhErrno(err.(syscall.Errno))
	}

	return counterType, &value, nil
}

// PdhGetFormattedCounterValueLong computes a displayable long value for the specified counter.
func PdhGetFormattedCounterValueLong(counter PdhCounterHandle) (uint32, *PdhCounterValueLong, error) {
	var counterType uint32
	var value PdhCounterValueLong
	if err := _PdhGetFormattedCounterValueLong(counter, PdhFmtLong|PdhFmtNoCap100, &counterType, &value); err != nil {
		return 0, &value, PdhErrno(err.(syscall.Errno))
	}

	return counterType, &value, nil
}

// PdhExpandWildCardPath returns counter paths that match the given counter path.
func PdhExpandWildCardPath(utfPath *uint16) ([]uint16, error) {
	var bufferSize uint32
	if err := _PdhExpandWildCardPath(nil, utfPath, nil, &bufferSize); err != nil {
		if PdhErrno(err.(syscall.Errno)) != PDH_MORE_DATA {
			return nil, PdhErrno(err.(syscall.Errno))
		}
		expandPaths := make([]uint16, bufferSize)
		if err := _PdhExpandWildCardPath(nil, utfPath, &expandPaths[0], &bufferSize); err != nil {
			return nil, PdhErrno(err.(syscall.Errno))
		}
		return expandPaths, nil
	}
	return nil, nil
}

// PdhExpandCounterPath returns counter paths that match the given counter path, for 32 bit windows.
func PdhExpandCounterPath(utfPath *uint16) ([]uint16, error) {
	var bufferSize uint32
	if err := _PdhExpandCounterPath(utfPath, nil, &bufferSize); err != nil {
		if PdhErrno(err.(syscall.Errno)) != PDH_MORE_DATA {
			return nil, PdhErrno(err.(syscall.Errno))
		}
		expandPaths := make([]uint16, bufferSize)
		if err := _PdhExpandCounterPath(utfPath, &expandPaths[0], &bufferSize); err != nil {
			return nil, PdhErrno(err.(syscall.Errno))
		}
		return expandPaths, nil
	}
	return nil, nil
}

// PdhGetCounterInfo returns the counter information for given handle
func PdhGetCounterInfo(handle PdhCounterHandle) (*PdhCounterInfo, error) {
	var bufSize uint32
	var buff []byte
	if err := _PdhGetCounterInfo(handle, 0, &bufSize, nil); err != nil {
		if PdhErrno(err.(syscall.Errno)) != PDH_MORE_DATA {
			return nil, PdhErrno(err.(syscall.Errno))
		}
		buff = make([]byte, bufSize)
		bufSize = uint32(len(buff))

		if err = _PdhGetCounterInfo(handle, 0, &bufSize, &buff[0]); err == nil {
			counterInfo := (*PdhCounterInfo)(unsafe.Pointer(&buff[0]))
			if counterInfo != nil {
				return counterInfo, nil
			}
		}
	}
	return nil, nil
}

// PdhCloseQuery closes all counters contained in the specified query.
func PdhCloseQuery(query PdhQueryHandle) error {
	if err := _PdhCloseQuery(query); err != nil {
		return PdhErrno(err.(syscall.Errno))
	}

	return nil
}

// PdhEnumObjectItems returns the counters and instance info for given object
func PdhEnumObjectItems(objectName string) ([]uint16, []uint16, error) {
	var (
		cBuff     = make([]uint16, 1)
		cBuffSize = uint32(0)
		iBuff     = make([]uint16, 1)
		iBuffSize = uint32(0)
	)
	obj := windows.StringToUTF16Ptr(objectName)
	if err := _PdhEnumObjectItems(
		0,
		0,
		obj,
		&cBuff[0],
		&cBuffSize,
		&iBuff[0],
		&iBuffSize,
		PerformanceDetailWizard,
		0); err != nil {
		if PdhErrno(err.(syscall.Errno)) != PDH_MORE_DATA {
			return nil, nil, PdhErrno(err.(syscall.Errno))
		}
		cBuff = make([]uint16, cBuffSize)
		iBuff = make([]uint16, iBuffSize)

		if err = _PdhEnumObjectItems(
			0,
			0,
			obj,
			&cBuff[0],
			&cBuffSize,
			&iBuff[0],
			&iBuffSize,
			PerformanceDetailWizard,
			0); err != nil {
			return nil, nil, err
		}
		return cBuff, iBuff, nil
	}
	return nil, nil, nil
}

// Error returns a more explicit error message.
func (e PdhErrno) Error() string {
	// If the value is not one of the known PDH errors then assume its a
	// general windows error.
	if _, found := pdhErrors[e]; !found {
		return syscall.Errno(e).Error()
	}

	// Use FormatMessage to convert the PDH errno to a string.
	// Example: https://msdn.microsoft.com/en-us/library/windows/desktop/aa373046(v=vs.85).aspx
	var flags uint32 = windows.FORMAT_MESSAGE_FROM_HMODULE | windows.FORMAT_MESSAGE_ARGUMENT_ARRAY | windows.FORMAT_MESSAGE_IGNORE_INSERTS
	b := make([]uint16, 300)
	n, err := windows.FormatMessage(flags, modpdh.Handle(), uint32(e), 0, b, nil)
	if err != nil {
		return "pdh error #" + strconv.Itoa(int(e))
	}

	// Trim terminating \r and \n
	for ; n > 0 && (b[n-1] == '\n' || b[n-1] == '\r'); n-- {
	}
	return string(utf16.Decode(b[:n]))
}
