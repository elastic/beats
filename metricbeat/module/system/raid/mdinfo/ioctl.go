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

package mdinfo

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

//IoctlGetArrayInfo is the ioctl device code for GET_ARRAY_INFO
//On linux, this is generated via the _IOR() macro.
//Specifically, _IOR(9,17,mdu_array_info_t)
//9 is the block device major number, 17 is our magic number,
//and the last value is the struct we pass via pointer to ioctl.
var IoctlGetArrayInfo uint64 = 0x80480911

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
		return dat, fmt.Errorf("Got error from syscall: %d", errno)
	}
	return dat, nil
}
