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
	"github.com/vmware/govmomi/vim25/mo"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

func eventMapping(hs mo.HostSystem) mapstr.M {
	totalCPU := int64(hs.Summary.Hardware.CpuMhz) * int64(hs.Summary.Hardware.NumCpuCores)
	freeCPU := int64(totalCPU) - int64(hs.Summary.QuickStats.OverallCpuUsage)
	usedMemory := int64(hs.Summary.QuickStats.OverallMemoryUsage) * 1024 * 1024
	freeMemory := int64(hs.Summary.Hardware.MemorySize) - usedMemory

	event := mapstr.M{
		"name": hs.Summary.Config.Name,
		"cpu": mapstr.M{
			"used": mapstr.M{
				"mhz": hs.Summary.QuickStats.OverallCpuUsage,
			},
			"total": mapstr.M{
				"mhz": totalCPU,
			},
			"free": mapstr.M{
				"mhz": freeCPU,
			},
		},
		"memory": mapstr.M{
			"used": mapstr.M{
				"bytes": usedMemory,
			},
			"total": mapstr.M{
				"bytes": hs.Summary.Hardware.MemorySize,
			},
			"free": mapstr.M{
				"bytes": freeMemory,
			},
		},
	}

	return event
}
