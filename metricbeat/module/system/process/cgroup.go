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

package process

import (
	"strconv"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/gosigar/cgroup"
)

// cgroupStatsToMap returns a MapStr containing the data from the stats object.
// If stats is nil then nil is returned.
func cgroupStatsToMap(stats *cgroup.Stats, perCPU bool) common.MapStr {
	if stats == nil {
		return nil
	}

	cgroup := common.MapStr{}

	// id and path are only available when all subsystems share a common path.
	if stats.ID != "" {
		cgroup["id"] = stats.ID
	}
	if stats.Path != "" {
		cgroup["path"] = stats.Path
	}

	if cpu := cgroupCPUToMapStr(stats.CPU); cpu != nil {
		cgroup["cpu"] = cpu
	}
	if cpuacct := cgroupCPUAccountingToMapStr(stats.CPUAccounting, perCPU); cpuacct != nil {
		cgroup["cpuacct"] = cpuacct
	}
	if memory := cgroupMemoryToMapStr(stats.Memory); memory != nil {
		cgroup["memory"] = memory
	}
	if blkio := cgroupBlockIOToMapStr(stats.BlockIO); blkio != nil {
		cgroup["blkio"] = blkio
	}

	return cgroup
}

// cgroupCPUToMapStr returns a MapStr containing CPUSubsystem data. If the
// cpu parameter is nil then nil is returned.
func cgroupCPUToMapStr(cpu *cgroup.CPUSubsystem) common.MapStr {
	if cpu == nil {
		return nil
	}

	return common.MapStr{
		"id":   cpu.ID,
		"path": cpu.Path,
		"cfs": common.MapStr{
			"period": common.MapStr{
				"us": cpu.CFS.PeriodMicros,
			},
			"quota": common.MapStr{
				"us": cpu.CFS.QuotaMicros,
			},
			"shares": cpu.CFS.Shares,
		},
		"rt": common.MapStr{
			"period": common.MapStr{
				"us": cpu.RT.PeriodMicros,
			},
			"runtime": common.MapStr{
				"us": cpu.RT.RuntimeMicros,
			},
		},
		"stats": common.MapStr{
			"periods": cpu.Stats.Periods,
			"throttled": common.MapStr{
				"periods": cpu.Stats.ThrottledPeriods,
				"ns":      cpu.Stats.ThrottledTimeNanos,
			},
		},
	}
}

// cgroupCPUAccountingToMapStr returns a MapStr containing
// CPUAccountingSubsystem data. If the cpuacct parameter is nil then nil is
// returned.
func cgroupCPUAccountingToMapStr(cpuacct *cgroup.CPUAccountingSubsystem, perCPU bool) common.MapStr {
	if cpuacct == nil {
		return nil
	}

	event := common.MapStr{
		"id":   cpuacct.ID,
		"path": cpuacct.Path,
		"total": common.MapStr{
			"ns": cpuacct.TotalNanos,
		},
		"stats": common.MapStr{
			"system": common.MapStr{
				"ns": cpuacct.Stats.SystemNanos,
			},
			"user": common.MapStr{
				"ns": cpuacct.Stats.UserNanos,
			},
		},
	}

	if perCPU {
		perCPUUsage := common.MapStr{}
		for i, usage := range cpuacct.UsagePerCPU {
			perCPUUsage[strconv.Itoa(i+1)] = usage
		}
		event["percpu"] = perCPUUsage
	}

	return event
}

// cgroupMemoryToMapStr returns a MapStr containing MemorySubsystem data. If the
// memory parameter is nil then nil is returned.
func cgroupMemoryToMapStr(memory *cgroup.MemorySubsystem) common.MapStr {
	if memory == nil {
		return nil
	}

	addMemData := func(key string, m common.MapStr, data cgroup.MemoryData) {
		m[key] = common.MapStr{
			"failures": data.FailCount,
			"limit": common.MapStr{
				"bytes": data.Limit,
			},
			"usage": common.MapStr{
				"bytes": data.Usage,
				"max": common.MapStr{
					"bytes": data.MaxUsage,
				},
			},
		}
	}

	memMap := common.MapStr{
		"id":   memory.ID,
		"path": memory.Path,
	}
	addMemData("mem", memMap, memory.Mem)
	addMemData("memsw", memMap, memory.MemSwap)
	addMemData("kmem", memMap, memory.Kernel)
	addMemData("kmem_tcp", memMap, memory.KernelTCP)
	memMap["stats"] = common.MapStr{
		"active_anon": common.MapStr{
			"bytes": memory.Stats.ActiveAnon,
		},
		"active_file": common.MapStr{
			"bytes": memory.Stats.ActiveFile,
		},
		"cache": common.MapStr{
			"bytes": memory.Stats.Cache,
		},
		"hierarchical_memory_limit": common.MapStr{
			"bytes": memory.Stats.HierarchicalMemoryLimit,
		},
		"hierarchical_memsw_limit": common.MapStr{
			"bytes": memory.Stats.HierarchicalMemswLimit,
		},
		"inactive_anon": common.MapStr{
			"bytes": memory.Stats.InactiveAnon,
		},
		"inactive_file": common.MapStr{
			"bytes": memory.Stats.InactiveFile,
		},
		"mapped_file": common.MapStr{
			"bytes": memory.Stats.MappedFile,
		},
		"page_faults":       memory.Stats.PageFaults,
		"major_page_faults": memory.Stats.MajorPageFaults,
		"pages_in":          memory.Stats.PagesIn,
		"pages_out":         memory.Stats.PagesOut,
		"rss": common.MapStr{
			"bytes": memory.Stats.RSS,
		},
		"rss_huge": common.MapStr{
			"bytes": memory.Stats.RSSHuge,
		},
		"swap": common.MapStr{
			"bytes": memory.Stats.Swap,
		},
		"unevictable": common.MapStr{
			"bytes": memory.Stats.Unevictable,
		},
	}

	return memMap
}

// cgroupBlockIOToMapStr returns a MapStr containing BlockIOSubsystem data.
// If the blockIO parameter is nil then nil is returned.
func cgroupBlockIOToMapStr(blockIO *cgroup.BlockIOSubsystem) common.MapStr {
	if blockIO == nil {
		return nil
	}

	return common.MapStr{
		"id":   blockIO.ID,
		"path": blockIO.Path,
		"total": common.MapStr{
			"bytes": blockIO.Throttle.TotalBytes,
			"ios":   blockIO.Throttle.TotalIOs,
		},
	}
}
