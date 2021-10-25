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
// +build windows

package diskio

// diskPerformance struct provides disk performance information. It is used by the IOCTL_DISK_PERFORMANCE control code.
// DeviceIoControl() will fail with ERROR_INSUFFICIENT_BUFFER (The data area passed to a system call is too small) on 32 bit systems.
// The memory layout is different for 32 bit vs 64 bit so an alignment (AlignmentPadding) is necessary in order to increase the buffer size
type diskPerformance struct {
	BytesRead    int64
	BytesWritten int64
	// Contains a cumulative time, expressed in increments of 100 nanoseconds (or ticks).
	ReadTime int64
	// Contains a cumulative time, expressed in increments of 100 nanoseconds (or ticks).
	WriteTime int64
	//Contains a cumulative time, expressed in increments of 100 nanoseconds (or ticks).
	IdleTime            int64
	ReadCount           uint32
	WriteCount          uint32
	QueueDepth          uint32
	SplitCount          uint32
	QueryTime           int64
	StorageDeviceNumber uint32
	StorageManagerName  [8]uint16
	AlignmentPadding    uint32
}
