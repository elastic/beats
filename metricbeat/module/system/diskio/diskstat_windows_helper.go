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

package diskio

import (
	"syscall"
	"unsafe"

	"github.com/shirou/gopsutil/disk"
)

const (
	fileDeviceDisk               = 0x00000007
	fileAnyAccess                = 0x0000
	errorSuccess   syscall.Errno = 0
)

var (
	// ioctlDiskPerformance :control code - https://docs.microsoft.com/en-us/windows/desktop/api/winioctl/ni-winioctl-ioctl_disk_performance
	ioctlDiskPerformance        = getCtlCode(fileDeviceDisk, 0x0008, 0, fileAnyAccess)
	modkernel32                 = syscall.NewLazyDLL("kernel32.dll")
	procGetLogicalDriveStringsW = modkernel32.NewProc("GetLogicalDriveStringsW")
)

type logicalDrive struct {
	Name    string
	UNCPath string
}

type diskPerformance struct {
	BytesRead           int64
	BytesWritten        int64
	ReadTime            int64
	WriteTime           int64
	IdleTime            int64
	ReadCount           uint32
	WriteCount          uint32
	QueueDepth          uint32
	SplitCount          uint32
	QueryTime           int64
	StorageDeviceNumber uint32
	StorageManagerName  [8]uint16
}

// used to get the specific IOCTL_DISK_PERFORMANCE control code
func getCtlCode(deviceType uint32, function uint32, method uint32, access uint32) uint32 {
	return (deviceType << 16) | (access << 14) | (function << 2) | method
}

// IOCounters gets the diskio counters and maps them to the list of counterstat objects
func iOCounters(names ...string) (map[string]disk.IOCountersStat, error) {
	ret := make(map[string]disk.IOCountersStat, 0)
	logicalDisks, err := getLogicalDriveStrings()
	if err != nil || len(logicalDisks) == 0 {
		return nil, err
	}
	for _, drive := range logicalDisks {
		if len(drive.Name) > 3 { // not get _Total or Harddrive
			continue
		}

		if len(names) > 0 && !containsDrive(names, drive.Name) { //filter by included devices
			continue
		}

		var counter, err = iOCounter(drive.UNCPath)
		if err != nil {
			return nil, err
		}
		ret[drive.Name] = disk.IOCountersStat{
			Name:       drive.Name,
			ReadCount:  uint64(counter.ReadCount),
			WriteCount: uint64(counter.WriteCount),
			ReadBytes:  uint64(counter.BytesRead),
			WriteBytes: uint64(counter.BytesWritten),
			ReadTime:   uint64(counter.ReadTime),
			WriteTime:  uint64(counter.WriteTime),
		}
	}
	return ret, nil
}

// iOCounter calls syscal func CreateFile to generate a handler then executes the DeviceIoControl func in order to retrieve the metrics
func iOCounter(path string) (diskPerformance, error) {
	var diskPerformance diskPerformance
	var diskPerformanceSize uint32
	utfPath, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return diskPerformance, err
	}
	hFile, err := syscall.CreateFile(utfPath,
		syscall.GENERIC_READ|syscall.GENERIC_WRITE,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE,
		nil,
		syscall.OPEN_EXISTING,
		syscall.FILE_FLAG_BACKUP_SEMANTICS,
		0)

	if err != nil {
		return diskPerformance, err
	}
	defer syscall.CloseHandle(hFile)

	err = syscall.DeviceIoControl(hFile,
		ioctlDiskPerformance,
		nil,
		0,
		(*byte)(unsafe.Pointer(&diskPerformance)),
		uint32(unsafe.Sizeof(diskPerformance)),
		&diskPerformanceSize,
		nil)
	if err != nil {
		return diskPerformance, err
	}
	return diskPerformance, nil
}

// getLogicalDriveStrings calls the syscall GetLogicalDriveStrings in order to get the list of logical drives
func getLogicalDriveStrings() ([]logicalDrive, error) {
	lpBuffer := make([]byte, 254)
	logicalDrives := make([]logicalDrive, 0)
	r1, _, e1 := syscall.Syscall(procGetLogicalDriveStringsW.Addr(), 2, uintptr(len(lpBuffer)), uintptr(unsafe.Pointer(&lpBuffer[0])), 0)
	if r1 == 0 {
		if e1 != errorSuccess {
			return nil, e1
		} else {
			return nil, syscall.EINVAL
		}
	}

	for _, v := range lpBuffer {
		if v >= 65 && v <= 90 {
			path := string(v) + ":"
			if path == "A:" || path == "B:" {
				continue
			}

			drive := logicalDrive{path, "\\\\.\\" + path}
			logicalDrives = append(logicalDrives, drive)
		}
	}
	return logicalDrives, nil
}

func containsDrive(devices []string, disk string) bool {
	for _, vv := range devices {
		if vv == disk {
			return true
		}
	}
	return false
}
