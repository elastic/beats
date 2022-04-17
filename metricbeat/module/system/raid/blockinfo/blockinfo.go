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

import "github.com/menderesk/beats/v7/libbeat/common"

// SyncStatus represents the status of a sync action as Complete/Total. Will be 0/0 if no sync action is going on
type SyncStatus struct {
	Complete int64
	Total    int64
}

// MDDevice represents /sys/block/[device] for an md device
type MDDevice struct {
	Name       string     //the name of the device
	Level      string     //The RAID level of the device
	Size       int64      //Size, as count of 512 byte blocks
	ArrayState string     //State of the Array
	DiskStates DiskStates //Disks in the Array
	SyncAction string     //The current sync action, if any
	SyncStatus SyncStatus //the current sync status, if any
}

// DiskStates summarizes the state of all the devices in the array
type DiskStates struct {
	Active int
	Total  int
	Failed int
	Spare  int
	States common.MapStr
}
