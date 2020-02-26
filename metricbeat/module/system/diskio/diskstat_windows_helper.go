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
	"strings"
	"syscall"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/disk"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"github.com/elastic/beats/libbeat/logp"
)

const (
	errorSuccess            syscall.Errno = 0
	ioctlDiskPerformance                  = 0x70020
	ioctlDiskPerformanceOff               = 0x70060
)

var (
	modkernel32                 = windows.NewLazySystemDLL("kernel32.dll")
	procGetLogicalDriveStringsW = modkernel32.NewProc("GetLogicalDriveStringsW")
	procGetDriveTypeW           = modkernel32.NewProc("GetDriveTypeW")
)

type logicalDrive struct {
	Name    string
	UNCPath string
}

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
}

// ioCounters gets the diskio counters and maps them to the list of counterstat objects.
func ioCounters(names ...string) (map[string]disk.IOCountersStat, error) {
	if err := enablePerformanceCounters(); err != nil {
		return nil, err
	}
	logicalDisks, err := getLogicalDriveStrings()
	if err != nil || len(logicalDisks) == 0 {
		return nil, err
	}
	ret := make(map[string]disk.IOCountersStat)
	for _, drive := range logicalDisks {
		// not get _Total or Harddrive
		if len(drive.Name) > 3 {
			continue
		}
		// filter by included devices
		if len(names) > 0 && !containsDrive(names, drive.Name) {
			continue
		}
		var counter diskPerformance
		err = ioCounter(drive.UNCPath, &counter)
		if err != nil {
			logp.Err("Could not return any performance counter values for %s .Error: %v", drive.UNCPath, err)
			continue
		}
		ret[drive.Name] = disk.IOCountersStat{
			Name:       drive.Name,
			ReadCount:  uint64(counter.ReadCount),
			WriteCount: uint64(counter.WriteCount),
			ReadBytes:  uint64(counter.BytesRead),
			WriteBytes: uint64(counter.BytesWritten),
			// Ticks (which is equal to 100 nanoseconds) will be converted to milliseconds for consistency reasons for both ReadTime and WriteTime (https://docs.microsoft.com/en-us/dotnet/api/system.timespan.ticks?redirectedfrom=MSDN&view=netframework-4.8#remarks)
			ReadTime:  uint64(counter.ReadTime / 10000),
			WriteTime: uint64(counter.WriteTime / 10000),
		}
	}
	return ret, nil
}

// ioCounter calls syscall func CreateFile to generate a handler then executes the DeviceIoControl func in order to retrieve the metrics.
func ioCounter(path string, diskPerformance *diskPerformance) error {
	utfPath, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	hFile, err := syscall.CreateFile(utfPath,
		syscall.GENERIC_READ|syscall.GENERIC_WRITE,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE,
		nil,
		syscall.OPEN_EXISTING,
		syscall.FILE_FLAG_BACKUP_SEMANTICS,
		0)

	if err != nil {
		return err
	}
	defer syscall.CloseHandle(hFile)
	var diskPerformanceSize uint32
	return syscall.DeviceIoControl(hFile,
		ioctlDiskPerformance,
		nil,
		0,
		(*byte)(unsafe.Pointer(diskPerformance)),
		uint32(unsafe.Sizeof(*diskPerformance)),
		&diskPerformanceSize,
		nil)
}

// enablePerformanceCounters will enable performance counters by adding the EnableCounterForIoctl registry key
func enablePerformanceCounters() error {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, "SYSTEM\\CurrentControlSet\\Services\\partmgr", registry.READ|registry.WRITE)
	// closing handler for the registry key. If the key is not one of the predefined registry keys (which is the case here), a call the RegCloseKey function should be executed after using the handle.
	defer func() {
		if key != 0 {
			clErr := key.Close()
			if clErr != nil {
				logp.L().Named("diskio").Errorf("cannot close handler for HKLM:SYSTEM\\CurrentControlSet\\Services\\Partmgr\\EnableCounterForIoctl key in the registry: %s", clErr)
			}
		}
	}()

	if err != nil {
		return errors.Errorf("cannot open new key in the registry in order to enable the performance counters: %s", err)
	}
	val, _, err := key.GetIntegerValue("EnableCounterForIoctl")
	if val != 1 || err != nil {
		if err = key.SetDWordValue("EnableCounterForIoctl", 1); err != nil {
			return errors.Errorf("cannot create HKLM:SYSTEM\\CurrentControlSet\\Services\\Partmgr\\EnableCounterForIoctl key in the registry in order to enable the performance counters: %s", err)
		}
		logp.L().Named("diskio").Info("The registry key EnableCounterForIoctl at HKLM:SYSTEM\\CurrentControlSet\\Services\\Partmgr has been created in order to enable the performance counters")
	}

	return nil
}

// disablePerformanceCounters will disable performance counters using the IOCTL_DISK_PERFORMANCE_OFF IOCTL control code
func disablePerformanceCounters(path string) error {
	utfPath, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	hFile, err := syscall.CreateFile(utfPath,
		syscall.GENERIC_READ|syscall.GENERIC_WRITE,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE,
		nil,
		syscall.OPEN_EXISTING,
		syscall.FILE_FLAG_BACKUP_SEMANTICS,
		0)

	if err != nil {
		return err
	}
	defer syscall.CloseHandle(hFile)
	var diskPerformanceSize uint32
	return syscall.DeviceIoControl(hFile,
		ioctlDiskPerformanceOff,
		nil,
		0,
		nil,
		0,
		&diskPerformanceSize,
		nil)
}

// getLogicalDriveStrings calls the syscall GetLogicalDriveStrings in order to get the list of logical drives
func getLogicalDriveStrings() ([]logicalDrive, error) {
	lpBuffer := make([]byte, 254)
	r1, _, e1 := syscall.Syscall(procGetLogicalDriveStringsW.Addr(), 2, uintptr(len(lpBuffer)), uintptr(unsafe.Pointer(&lpBuffer[0])), 0)
	if r1 == 0 {
		err := e1
		if e1 != errorSuccess {
			err = syscall.EINVAL
		}
		return nil, err
	}
	var logicalDrives []logicalDrive
	for _, v := range lpBuffer {
		if v >= 65 && v <= 90 {
			s := string(v)
			if s == "A" || s == "B" {
				continue
			}
			path := s + ":"
			drive := logicalDrive{path, `\\.\` + path}
			if isValidLogicalDrive(path) {
				logicalDrives = append(logicalDrives, drive)
			}
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

// isValidLogicalDrive should filter CD-ROM type drives based on https://docs.microsoft.com/en-us/windows/desktop/api/fileapi/nf-fileapi-getdrivetypew
func isValidLogicalDrive(path string) bool {
	utfPath, err := syscall.UTF16PtrFromString(path + `\`)
	if err != nil {
		return false
	}
	ret, _, err := syscall.Syscall(procGetDriveTypeW.Addr(), 1, uintptr(unsafe.Pointer(utfPath)), 0, 0)

	//DRIVE_NO_ROOT_DIR = 1 DRIVE_CDROM = 5 DRIVE_UNKNOWN = 0 DRIVE_RAMDISK = 6
	if ret == 1 || ret == 5 || ret == 0 || ret == 6 || err != errorSuccess {
		return false
	}

	//check for ramdisk label as the drive type is fixed in this case
	volumeLabel, err := GetVolumeLabel(utfPath)
	if err != nil {
		return false
	}
	if strings.ToLower(volumeLabel) == "ramdisk" {
		return false
	}
	return true
}

// GetVolumeLabel function will retrieve the volume label
func GetVolumeLabel(path *uint16) (string, error) {
	lpVolumeNameBuffer := make([]uint16, 256)
	lpVolumeSerialNumber := uint32(0)
	lpMaximumComponentLength := uint32(0)
	lpFileSystemFlags := uint32(0)
	lpFileSystemNameBuffer := make([]uint16, 256)
	err := windows.GetVolumeInformation(
		path,
		&lpVolumeNameBuffer[0],
		uint32(len(lpVolumeNameBuffer)),
		&lpVolumeSerialNumber,
		&lpMaximumComponentLength,
		&lpFileSystemFlags,
		&lpFileSystemNameBuffer[0],
		uint32(len(lpFileSystemNameBuffer)))
	if err != nil {
		return "", err
	}
	return syscall.UTF16ToString(lpVolumeNameBuffer), nil
}
