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

package blockinfo

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
)

// SyncStatus represents the status of a sync action as Complete/Total. Will be 0/0 if no sync action is going on
type SyncStatus struct {
	Complete int64
	Total    int64
}

// MDDevice represents /sys/block/[device] for an md device
type MDDevice struct {
	Name        string     //the name of the device
	Size        int64      //Size, as count of 512 byte blocks
	ActiveDisks int64      //Active disks
	ArrayState  string     //State of the Array
	Devices     []Disk     //Disks in the Array
	SyncAction  string     //The current sync action, if any
	SyncStatus  SyncStatus //the current sync status, if any
}

// Disk represents a single dis component, found at  /sys/block/[device]/md/dev-* for an md device
type Disk struct {
	Size  int64
	State string
}

// DiskStates summarizes the state of all the devices in the array
type DiskStates struct {
	Active  int
	Total   int
	Failed  int
	Spare   int
	Unknown int
	States  common.MapStr
}

// ListAllMDDevices returns a string array of the paths to all the md devices under the root
func ListAllMDDevices(path string) ([]string, error) {
	//I'm not convinced that using /sys/block/md* is a reliable glob, as you should be able to make those whatever you want.
	dir, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, errors.Wrap(err, "could not read directory")
	}
	var mds []string
	for _, item := range dir {
		testpath := filepath.Join(path, item.Name())
		if !isMD(testpath) {
			continue
		}
		mds = append(mds, testpath)
	}

	if len(mds) == 0 {
		return nil, fmt.Errorf("no matches from path %s,", path)
	}

	return mds, nil
}

// GetMDDevice returns a MDDevice object representing a multi-disk device, or error if it's not a "real" md device
func GetMDDevice(path string) (MDDevice, error) {
	_, err := os.Stat(path)
	if err != nil {
		return MDDevice{}, errors.Wrap(err, "path does not exist")
	}

	//This is the best heuristic I've found so far for identifying an md device.
	if !isMD(path) {
		return MDDevice{}, err
	}
	return newMD(path)
}

// ReduceDisks disks on linux uses the raw states to provide a common status
//see https://www.kernel.org/doc/html/v4.15/admin-guide/md.html#md-devices-in-sysfs
func (dev MDDevice) ReduceDisks() DiskStates {
	var disks DiskStates
	disks.States = common.MapStr{}
	for _, disk := range dev.Devices {
		switch disk.State {
		case "faulty", "blocked", "write_error", "want_replacement":
			disks.Failed++
		case "in_sync", "writemostly", "replacement":
			disks.Active++
		case "spare":
			disks.Spare++
		default:
			disks.Unknown++
		}

		if _, ok := disks.States[disk.State]; !ok {
			disks.States[disk.State] = 1
		} else {
			disks.States[disk.State] = disks.States[disk.State].(int) + 1
		}

		disks.Total++
	}

	return disks
}

//check if a block device directory looks like an MD device
//Right now, we're doing this by looking for an `md` directory in the device dir.
func isMD(path string) bool {
	_, err := os.Stat(filepath.Join(path, "md"))
	if err != nil {
		return false
	}
	return true
}
