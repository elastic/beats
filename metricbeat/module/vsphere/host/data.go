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

package host

import (
	"github.com/menderesk/beats/v7/libbeat/common"

	"github.com/vmware/govmomi/vim25/mo"
)

func eventMapping(hs mo.HostSystem) common.MapStr {
	totalCPU := int64(hs.Summary.Hardware.CpuMhz) * int64(hs.Summary.Hardware.NumCpuCores)
	freeCPU := int64(totalCPU) - int64(hs.Summary.QuickStats.OverallCpuUsage)
	usedMemory := int64(hs.Summary.QuickStats.OverallMemoryUsage) * 1024 * 1024
	freeMemory := int64(hs.Summary.Hardware.MemorySize) - usedMemory

	event := common.MapStr{
		"name": hs.Summary.Config.Name,
		"cpu": common.MapStr{
			"used": common.MapStr{
				"mhz": hs.Summary.QuickStats.OverallCpuUsage,
			},
			"total": common.MapStr{
				"mhz": totalCPU,
			},
			"free": common.MapStr{
				"mhz": freeCPU,
			},
		},
		"memory": common.MapStr{
			"used": common.MapStr{
				"bytes": usedMemory,
			},
			"total": common.MapStr{
				"bytes": hs.Summary.Hardware.MemorySize,
			},
			"free": common.MapStr{
				"bytes": freeMemory,
			},
		},
	}

	return event
}
