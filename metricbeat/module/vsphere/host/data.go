package host

import (
	"github.com/elastic/beats/libbeat/common"

	"github.com/vmware/govmomi/vim25/mo"
)

func eventMapping(hs mo.HostSystem) common.MapStr {
	totalCpu := int64(hs.Summary.Hardware.CpuMhz) * int64(hs.Summary.Hardware.NumCpuCores)
	freeCpu := int64(totalCpu) - int64(hs.Summary.QuickStats.OverallCpuUsage)
	usedMemory := int64(hs.Summary.QuickStats.OverallMemoryUsage) * 1024 * 1024
	freeMemory := int64(hs.Summary.Hardware.MemorySize) - usedMemory

	event := common.MapStr{
		"name": hs.Summary.Config.Name,
		"cpu": common.MapStr{
			"used": common.MapStr{
				"mhz": hs.Summary.QuickStats.OverallCpuUsage,
			},
			"total": common.MapStr{
				"mhz": totalCpu,
			},
			"free": common.MapStr{
				"mhz": freeCpu,
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
