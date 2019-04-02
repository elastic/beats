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
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

//get the raid level and use that to determine how we fill out the array
//Only data-reduntant RIAD levels (1,4,5,6,10) have some of these fields
func isRedundant(path string) (bool, error) {

	raw, err := ioutil.ReadFile(filepath.Join(path, "md", "level"))
	if err != nil {
		return false, err
	}
	raidStr := strings.TrimSpace(string(raw))
	if raidStr == "raid1" || raidStr == "raid4" || raidStr == "raid5" ||
		raidStr == "raid6" || raidStr == "raid10" {
		return true, nil
	}

	return false, nil
}

func parseIntVal(path string) (int64, error) {

	var value int64
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return value, err
	}
	strVal := strings.TrimSpace(string(raw))
	//size in 512 byte blocks
	value, err = strconv.ParseInt(string(strVal), 10, 64)
	if err != nil {
		return value, err
	}

	return value, nil
}

//get all the disks associated with an MD device
func getDisks(path string) ([]Disk, error) {

	//so far, haven't found a less hacky way to do this.
	devices, err := filepath.Glob(filepath.Join(path, "md", "dev-*"))
	if err != nil {
		return nil, errors.Wrap(err, "could not get device list")
	}

	var diskList []Disk
	for _, disk := range devices {
		d, err := getDisk(disk)
		if err != nil {
			return nil, err
		}
		diskList = append(diskList, d)
	}

	return diskList, nil
}

func getDisk(path string) (Disk, error) {
	size, err := parseIntVal(filepath.Join(path, "size"))
	if err != nil {
		return Disk{}, errors.Wrap(err, "error getting disk size")
	}

	state, err := ioutil.ReadFile(filepath.Join(path, "state"))
	if err != nil {
		return Disk{}, errors.Wrap(err, "error getting disk state")
	}

	return Disk{Size: size, State: strings.TrimSpace(string(state))}, nil

}

//get the current sync status as it exists under md/sync_completed
//if there's no sync operation in progress, the file will just have 'none'
//in which case, default to to the overall size
func getSyncStatus(path string, size int64) (SyncStatus, error) {

	re := regexp.MustCompile(`^(\d*) \/ (\d*)`)

	raw, err := ioutil.ReadFile(filepath.Join(path, "md", "sync_completed"))
	if err != nil {
		return SyncStatus{}, errors.Wrap(err, "could not open sync_completed")
	}
	if string(raw) == "none\n" {
		return SyncStatus{Complete: size, Total: size}, nil
	}

	matches := re.FindStringSubmatch(string(raw))

	if len(matches) < 3 {
		return SyncStatus{}, fmt.Errorf("Could not get data from sync_completed")
	}

	current, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return SyncStatus{}, errors.Wrap(err, "Could not parse data sync_completed")
	}

	total, err := strconv.ParseInt(matches[2], 10, 64)
	if err != nil {
		return SyncStatus{}, errors.Wrap(err, "Could not parse data sync_completed")
	}

	return SyncStatus{Complete: current, Total: total}, nil

}

//Create a new disk object, parsing any needed fields
func newMD(path string) (MDDevice, error) {
	var dev MDDevice

	dev.Name = filepath.Base(path)
	size, err := parseIntVal(filepath.Join(path, "size"))
	if err != nil {
		return dev, errors.Wrap(err, "could not get device size")
	}
	dev.Size = size

	//This is the count of 'active' disks
	active, err := parseIntVal(filepath.Join(path, "md", "raid_disks"))
	if err != nil {
		return dev, errors.Wrap(err, "could not get raid_disks")
	}
	dev.ActiveDisks = active

	//RAID array state
	state, err := ioutil.ReadFile(filepath.Join(path, "md", "array_state"))
	if err != nil {
		return dev, errors.Wrap(err, "could not open array_state")
	}
	dev.ArrayState = strings.TrimSpace(string(state))

	//get total disks
	disks, err := getDisks(path)
	if err != nil {
		return dev, errors.Wrap(err, "could not get disk data")
	}
	dev.Devices = disks

	//sync action and sync status will only exist for redundant raid levels
	redundant, err := isRedundant(path)
	if err != nil {
		return dev, errors.Wrap(err, "error getting raid level")
	}
	if redundant {

		//Get the sync action
		//Will be idle if nothing is going on
		syncAction, err := ioutil.ReadFile(filepath.Join(path, "md", "sync_action"))
		if err != nil {
			return dev, errors.Wrap(err, "could not open sync_action")
		}
		dev.SyncAction = strings.TrimSpace(string(syncAction))

		//sync status
		syncComp, err := getSyncStatus(path, dev.Size)
		if err != nil {
			return dev, errors.Wrap(err, "error getting sync data")
		}

		dev.SyncStatus = syncComp
	}

	return dev, nil
}
