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
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/logp"
)

var debugf = logp.MakeDebug("system.raid")

//get the raid level and use that to determine how we fill out the array
//Only data-reduntant RIAD levels (1,4,5,6,10) have some of these fields
func isRedundant(raidStr string) bool {
	if raidStr == "raid1" || raidStr == "raid4" || raidStr == "raid5" ||
		raidStr == "raid6" || raidStr == "raid10" {
		return true
	}

	return false
}

func parseIntVal(path string) (int64, error) {
	var value int64
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return value, err
	}
	strVal := strings.TrimSpace(string(raw))

	value, err = strconv.ParseInt(string(strVal), 10, 64)
	if err != nil {
		return value, err
	}

	return value, nil
}

//get the current sync status as it exists under md/sync_completed
//if there's no sync operation in progress, the file will just have 'none'
//in which case, default to to the overall size
func getSyncStatus(path string, size int64) (SyncStatus, error) {
	raw, err := ioutil.ReadFile(filepath.Join(path, "md", "sync_completed"))
	if err != nil {
		return SyncStatus{}, errors.Wrap(err, "could not open sync_completed")
	}
	completedState := strings.TrimSpace(string(raw))
	if completedState == "none" {
		return SyncStatus{Complete: size, Total: size}, nil
	}

	matches := strings.SplitN(completedState, " / ", 2)

	if len(matches) != 2 {
		return SyncStatus{}, fmt.Errorf("could not get data from sync_completed")
	}

	current, err := strconv.ParseInt(matches[0], 10, 64)
	if err != nil {
		return SyncStatus{}, errors.Wrap(err, "could not parse data sync_completed")
	}

	total, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return SyncStatus{}, errors.Wrap(err, "could not parse data sync_completed")
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
	dev.DiskStates = disks

	level, err := ioutil.ReadFile(filepath.Join(path, "md", "level"))
	if err != nil {
		return dev, errors.Wrap(err, "could not get raid level")
	}
	dev.Level = strings.TrimSpace(string(level))

	//sync action and sync status will only exist for redundant raid levels
	if isRedundant(dev.Level) {

		//Get the sync action
		//Will be idle if nothing is going on
		syncAction, err := ioutil.ReadFile(filepath.Join(path, "md", "sync_action"))
		if err != nil {
			return dev, errors.Wrap(err, "could not open sync_action")
		}
		dev.SyncAction = strings.TrimSpace(string(syncAction))

		//sync status
		syncStats, err := getSyncStatus(path, dev.Size)
		if err != nil {
			return dev, errors.Wrap(err, "error getting sync data")
		}

		dev.SyncStatus = syncStats
	}

	return dev, nil
}

//get all the disks associated with an MD device
func getDisks(path string) (DiskStates, error) {
	//so far, haven't found a less hacky way to do this.
	devices, err := filepath.Glob(filepath.Join(path, "md", "dev-*"))
	if err != nil {
		return DiskStates{}, errors.Wrap(err, "could not get device list")
	}

	var disks DiskStates
	disks.States = common.MapStr{}
	//This is meant to provide a 'common status' for disks in the array
	//see https://www.kernel.org/doc/html/v4.15/admin-guide/md.html#md-devices-in-sysfs
	for _, disk := range devices {
		disk, err := getDisk(disk)
		if err != nil {
			return DiskStates{}, err
		}

		switch disk {
		case "faulty", "blocked", "write_error", "want_replacement":
			disks.Failed++
		case "in_sync", "writemostly", "replacement":
			disks.Active++
		case "spare":
			disks.Spare++
		default:
			debugf("Unknown disk state %s", disk)
		}

		if _, ok := disks.States[disk]; !ok {
			disks.States[disk] = 1
		} else {
			disks.States[disk] = disks.States[disk].(int) + 1
		}

		disks.Total++

	}

	return disks, nil
}

func getDisk(path string) (string, error) {
	state, err := ioutil.ReadFile(filepath.Join(path, "state"))
	if err != nil {
		return "", errors.Wrap(err, "error getting disk state")
	}

	return strings.TrimSpace(string(state)), nil

}
