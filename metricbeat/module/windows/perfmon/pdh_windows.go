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

package perfmon

import (
	"syscall"
	"unsafe"
)

// Windows API calls
//sys _PdhOpenQuery(dataSource *uint16, userData uintptr, query *PdhQueryHandle) (errcode error) [failretval!=0] = pdh.PdhOpenQueryW
//sys _PdhAddCounter(query PdhQueryHandle, counterPath string, userData uintptr, counter *PdhCounterHandle) (errcode error) [failretval!=0] = pdh.PdhAddEnglishCounterW
//sys _PdhCollectQueryData(query PdhQueryHandle) (errcode error) [failretval!=0] = pdh.PdhCollectQueryData
//sys _PdhGetFormattedCounterValue(counter PdhCounterHandle, format PdhCounterFormat, counterType *uint32, value *PdhCounterValue) (errcode error) [failretval!=0] = pdh.PdhGetFormattedCounterValue
//sys _PdhGetFormattedCounterArray(counter PdhCounterHandle, format PdhCounterFormat, bufferSize *uint32, bufferCount *uint32, itemBuffer *byte) (errcode error) [failretval!=0] = pdh.PdhGetFormattedCounterArrayW
//sys _PdhGetRawCounterValue(counter PdhCounterHandle, counterType *uint32, value *PdhRawCounter) (errcode error) [failretval!=0] = pdh.PdhGetRawCounterValue
//sys _PdhGetRawCounterArray(counter PdhCounterHandle, bufferSize *uint32, bufferCount *uint32, itemBuffer *pdhRawCounterItem) (errcode error) [failretval!=0] = pdh.PdhGetRawCounterArray
//sys _PdhCalculateCounterFromRawValue(counter PdhCounterHandle, format PdhCounterFormat, rawValue1 *PdhRawCounter, rawValue2 *PdhRawCounter, value *PdhCounterValue) (errcode error) [failretval!=0] = pdh.PdhCalculateCounterFromRawValue
//sys _PdhFormatFromRawValue(counterType uint32, format PdhCounterFormat, timeBase *uint64, rawValue1 *PdhRawCounter, rawValue2 *PdhRawCounter, value *PdhCounterValue) (errcode error) [failretval!=0] = pdh.PdhFormatFromRawValue
//sys _PdhCloseQuery(query PdhQueryHandle) (errcode error) [failretval!=0] = pdh.PdhCloseQuery
//sys _PdhExpandWildCardPath(dataSource *uint16, wildcardPath *uint16, expandedPathList *uint16, pathListLength *uint32) (errcode error) [failretval!=0] = pdh.PdhExpandWildCardPathW

var (
	sizeofPdhCounterValueItem = (int)(unsafe.Sizeof(pdhCounterValueItem{}))
)

type PdhQueryHandle uintptr

var InvalidQueryHandle = ^PdhQueryHandle(0)

type PdhCounterHandle uintptr

var InvalidCounterHandle = ^PdhCounterHandle(0)

type pdhCounterValueItem struct {
	SzName   uintptr
	FmtValue PdhCounterValue
}

type pdhRawCounterItem struct {
	SzName   uintptr
	RawValue PdhRawCounter
}

type CounterValueItem struct {
	Name  string
	Value PdhCounterValue
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

// PdhAddCounter adds the specified counter to the query.
func PdhAddCounter(query PdhQueryHandle, counterPath string, userData uintptr) (PdhCounterHandle, error) {
	var handle PdhCounterHandle
	if err := _PdhAddCounter(query, counterPath, userData, &handle); err != nil {
		return InvalidCounterHandle, PdhErrno(err.(syscall.Errno))
	}

	return handle, nil
}

// PdhCollectQueryData collects the current raw data value for all counters in the specified query.
func PdhCollectQueryData(query PdhQueryHandle) error {
	if err := _PdhCollectQueryData(query); err != nil {
		return PdhErrno(err.(syscall.Errno))
	}

	return nil
}

// PdhGetFormattedCounterValue computes a displayable value for the specified counter.
func PdhGetFormattedCounterValue(counter PdhCounterHandle, format PdhCounterFormat) (uint32, *PdhCounterValue, error) {
	var counterType uint32
	var value PdhCounterValue
	if err := _PdhGetFormattedCounterValue(counter, format, &counterType, &value); err != nil {
		return 0, nil, PdhErrno(err.(syscall.Errno))
	}

	return counterType, &value, nil
}

// PdhGetRawCounterValue returns the current raw value of the counter.
func PdhGetRawCounterValue(counter PdhCounterHandle) (uint32, *PdhRawCounter, error) {
	var counterType uint32
	var value PdhRawCounter
	if err := _PdhGetRawCounterValue(counter, &counterType, &value); err != nil {
		return 0, nil, PdhErrno(err.(syscall.Errno))
	}

	return counterType, &value, nil
}

// PdhCalculateCounterFromRawValue calculates the displayable value of two raw counter values.
func PdhCalculateCounterFromRawValue(counter PdhCounterHandle, format PdhCounterFormat, rawValue1 *PdhRawCounter, rawValue2 *PdhRawCounter) (*PdhCounterValue, error) {
	var value PdhCounterValue
	if err := _PdhCalculateCounterFromRawValue(counter, format, rawValue1, rawValue2, &value); err != nil {
		return nil, PdhErrno(err.(syscall.Errno))
	}

	return &value, nil
}

// PdhExpandWildCardPath returns counter paths that match the given counter path.
func PdhExpandWildCardPath(wildCardPath string) ([]uint16, error) {
	utfPath, err := syscall.UTF16PtrFromString(wildCardPath)
	if err != nil {
		return nil, err
	}
	var bufferSize uint32
	if err := _PdhExpandWildCardPath(nil, utfPath, nil, &bufferSize); err != nil {
		if PdhErrno(err.(syscall.Errno)) != PDH_MORE_DATA {
			return nil, PdhErrno(err.(syscall.Errno))
		}

		expdPaths := make([]uint16, bufferSize)
		bufferSize = uint32(len(expdPaths))
		if err := _PdhExpandWildCardPath(nil, utfPath, &expdPaths[0], &bufferSize); err != nil {
			return nil, PdhErrno(err.(syscall.Errno))
		}
		return expdPaths, nil
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
