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

package raid

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/metric/system/resolve"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
	"github.com/elastic/beats/v7/metricbeat/module/system/raid/blockinfo"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func init() {
	mb.Registry.MustAddMetricSet("system", "raid", New,
		mb.WithHostParser(parse.EmptyHostParser),
	)
}

// MetricSet contains proc fs data.
type MetricSet struct {
	mb.BaseMetricSet
	mod resolve.Resolver
}

// New creates a new instance of the raid metricset.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	sys := base.Module().(resolve.Resolver)
	return &MetricSet{
		BaseMetricSet: base,

		mod: sys,
	}, nil
}

func blockto1024(b int64) int64 {
	//Annoyingly, different linux subsystems report size size using different blocks
	// /proc/mdstat /proc/partitions and mdadm (Via the BLKGETSIZE64 ioctl call) count "size" as the number of 1024-byte blocks.
	// /sys/block/md*/size uses 512-byte blocks. As does /sys/block/md*/md/sync_completed
	//convert the 512-byte blocks to 1024 byte blocks to maintain how this metricset "used to" report size
	return b / 2
}

// Fetch fetches one event for each device
func (m *MetricSet) Fetch(r mb.ReporterV2) error {
	devices, err := blockinfo.ListAll(m.mod.ResolveHostFS("/sys/block"))
	if err != nil {
		return errors.Wrap(err, "failed to parse sysfs")
	}

	for _, blockDev := range devices {

		event := mapstr.M{
			"name":   blockDev.Name,
			"status": blockDev.ArrayState,
			"level":  blockDev.Level,
			"disks": mapstr.M{
				"active": blockDev.DiskStates.Active,
				"total":  blockDev.DiskStates.Total,
				"spare":  blockDev.DiskStates.Spare,
				"failed": blockDev.DiskStates.Failed,
				"states": blockDev.DiskStates.States,
			},
		}
		//emulate the behavior of the previous mdstat parser by using the size when no sync data is available
		if blockDev.SyncStatus.Total == 0 {
			event["blocks"] = mapstr.M{
				"synced": blockto1024(blockDev.Size),
				"total":  blockto1024(blockDev.Size),
			}
		} else {
			event["blocks"] = mapstr.M{
				"synced": blockto1024(blockDev.SyncStatus.Complete),
				"total":  blockto1024(blockDev.SyncStatus.Total),
			}
		}
		//sync action is only available on redundant RAID types
		if blockDev.SyncAction != "" {
			event["sync_action"] = blockDev.SyncAction
		}

		isOpen := r.Event(mb.Event{
			MetricSetFields: event,
		})
		if !isOpen {
			return nil
		}
	}

	return nil
}
