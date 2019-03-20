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

//+build darwin freebsd linux openbsd

package raid

import (
	"os"
	"syscall"
	"unsafe"

	"github.com/pkg/errors"
)

//newMDDevice is a function used for returning a new instance of an ioctl reader
var newMDDevice = makenewMDDevice

func makenewMDDevice(dev string) (MDData, error) {

	//we're expecting the name as it comes from /proc/mdstat
	f, err := os.Open(dev)
	if err != nil {
		return MDDevice{}, err
	}

	return MDDevice{dev: f}, nil
}

//IoctlGetArrayInfo is the ioctl device code for GET_ARRAY_INFO
//On linux, this is generated via the _IOR() macro.
//Specifically, _IOR(9,17,mdu_array_info_t)
//9 is the block device major number, 17 is our magic number,
//and the last value is the struct we pass via pointer to ioctl.
var IoctlGetArrayInfo uint64 = 0x80480911

//MDArrayInfo contains the MD data from iotctl()
type MDArrayInfo struct {
	MajorVersion int32
	MinorVersion int32
	PatchVersion int32
	//RAID array creation time
	Ctime uint32
	Level int32
	Size  int32
	//Total devices
	NrDisks int32
	//Raid Devices
	RAIDDisks     int32
	MDMinor       int32
	NotPersistent int32
	//superblock update time
	Utime uint32
	//state bitmask
	State        int32
	ActiveDisks  int32
	WorkingDisks int32
	FailedDisks  int32
	SpareDisks   int32
	Layout       int32
	//This value is normally divided by 1024, presumably to get block size.
	//If you want the raid size in bytes, you need the BLKGETSIZE64 ioctl call.
	ChunkSize int32
}

//MDData is an interface that provides a human-friendly way to access raid data via ioctl
type MDData interface {
	Close() error
	GetArrayInfo() (MDArrayInfo, error)
}

//MDDevice is a multi-disk device to make status queries on
type MDDevice struct {
	dev *os.File
}

//Close closes the connection to the /dev/md device
func (dev MDDevice) Close() error {
	return dev.dev.Close()
}

//GetArrayInfo returns a struct describing the state of the RAID array.
func (dev MDDevice) GetArrayInfo() (MDArrayInfo, error) {

	var dat MDArrayInfo
	//Users beware, there's a lot of undocumented things going on when it comes to actually parsing this struct and making it useful.
	//The first return is just 0/-1 for an error, but we can get the same info from errno.
	//r2 doesn't seem to be used.
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(dev.dev.Fd()), uintptr(IoctlGetArrayInfo), uintptr(unsafe.Pointer(&dat)))
	if errno != 0 {
		return dat, errors.Wrap(errno, "ioctl failed")
	}
	return dat, nil
}

//NewDevice opens a file to a /dev/md* device
func NewDevice(dev string) (MDData, error) {

	return newMDDevice(dev)
}
