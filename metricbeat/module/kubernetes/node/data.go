package node

import (
	"encoding/json"
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/module/kubernetes"
)

func eventMapping(content []byte) (common.MapStr, error) {
	var summary kubernetes.Summary
	err := json.Unmarshal(content, &summary)
	if err != nil {
		return nil, fmt.Errorf("Cannot unmarshal json response: %s", err)
	}

	node := summary.Node
	nodeEvent := common.MapStr{
		"name":       node.NodeName,
		"start_time": node.StartTime,

		"cpu": common.MapStr{
			"usage": common.MapStr{
				"nanocores": node.CPU.UsageNanoCores,
				"core": common.MapStr{
					"ns": node.CPU.UsageCoreNanoSeconds,
				},
			},
		},

		"memory": common.MapStr{
			"available": common.MapStr{
				"bytes": node.Memory.AvailableBytes,
			},
			"usage": common.MapStr{
				"bytes": node.Memory.UsageBytes,
			},
			"workingset": common.MapStr{
				"bytes": node.Memory.WorkingSetBytes,
			},
			"rss": common.MapStr{
				"bytes": node.Memory.RssBytes,
			},
			"pagefaults":      node.Memory.PageFaults,
			"majorpagefaults": node.Memory.MajorPageFaults,
		},

		"network": common.MapStr{
			"rx": common.MapStr{
				"bytes":  node.Network.RxBytes,
				"errors": node.Network.RxErrors,
			},
			"tx": common.MapStr{
				"bytes":  node.Network.TxBytes,
				"errors": node.Network.TxErrors,
			},
		},

		"fs": common.MapStr{
			"available": common.MapStr{
				"bytes": node.Fs.AvailableBytes,
			},
			"capacity": common.MapStr{
				"bytes": node.Fs.CapacityBytes,
			},
			"used": common.MapStr{
				"bytes": node.Fs.UsedBytes,
			},
			"inodes": common.MapStr{
				"used":  node.Fs.InodesUsed,
				"free":  node.Fs.InodesFree,
				"count": node.Fs.Inodes,
			},
		},

		"runtime": common.MapStr{
			"imagefs": common.MapStr{
				"available": common.MapStr{
					"bytes": node.Runtime.ImageFs.AvailableBytes,
				},
				"capacity": common.MapStr{
					"bytes": node.Runtime.ImageFs.CapacityBytes,
				},
				"used": common.MapStr{
					"bytes": node.Runtime.ImageFs.UsedBytes,
				},
			},
		},
	}
	return nodeEvent, nil
}
