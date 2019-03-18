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
	"os"
	"strings"
)

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

//RaidData is an interface that provides a human-friendly way to access raid data via ioctl
type RaidData interface {
	Close() error
	GetArrayInfo() (MDArrayInfo, error)
}

//NewDevice opens a file to a /dev/md* device
func NewDevice(dev string, procfs string) (RaidData, error) {

	//create a mock test device to see if we're in some integration/unit test mode
	if strings.Contains(procfs, "testdata") {
		return MockData(""), nil
	}

	//we're expecting the name as it comes from /proc/mdstat
	mdDev := "/dev/" + dev
	f, err := os.Open(mdDev)
	if err != nil {
		return MDDevice{}, err
	}

	return MDDevice{dev: f}, nil
}
